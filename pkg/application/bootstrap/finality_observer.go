package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/ports"
	"github.com/rtcdance/chainpulse/pkg/services/finality"
)

type FinalityObserverOptions struct {
	Chains   []string
	RPCs     []string
	Registry *finality.Registry
}

type finalityObservationWriter interface {
	RecordFinalityObservation(context.Context, string, string, *blockchain.FinalityEvidence) error
}
type finalityEvidenceReader interface {
	ObserveFinality(context.Context, string, string) (*blockchain.FinalityEvidence, error)
}

// StartFinalityObserver owns at most one poller per configured chain and closes
// its RPC clients only after cancellation has joined all in-flight pollers.
// Confirmations/latest policies do not produce consensus-finality evidence.
func StartFinalityObserver(ctx context.Context, database ports.DatabasePlugin, options FinalityObserverOptions, logger ports.Logger, metrics ports.MetricsCollector) (func(), error) {
	if options.Registry == nil || len(options.Chains) == 0 || len(options.Chains) > 50 {
		return nil, fmt.Errorf("finality observer requires a policy and 1..50 declared chains")
	}
	selected := make([]string, 0, len(options.Chains))
	tags := make(map[string]string)
	for _, chain := range options.Chains {
		strategy := options.Registry.StrategyFor(chain)
		if strategy.Mode == finality.StrategyTag {
			if _, exists := tags[chain]; exists {
				return nil, fmt.Errorf("duplicate finality observer chain")
			}
			selected = append(selected, chain)
			tags[chain] = strategy.Tag
		}
	}
	if len(selected) == 0 {
		return func() {}, nil
	}
	store, ok := unwrapCanonicalSource(database).(finalityObservationWriter)
	if !ok {
		return nil, fmt.Errorf("tag finality requires durable canonical observation storage")
	}
	entries := make([]string, 0, len(selected))
	for _, entry := range options.RPCs {
		chain, _, _ := strings.Cut(entry, "=")
		if _, wanted := tags[chain]; wanted {
			entries = append(entries, entry)
		}
	}
	verifier, err := NewCanonicalRPCVerifier(ctx, selected, entries)
	if err != nil {
		return nil, fmt.Errorf("initialize finality RPC identity: %w", err)
	}
	loopCtx, cancel := context.WithCancel(ctx)
	var workers sync.WaitGroup
	for _, chain := range selected {
		tag := tags[chain]
		workers.Add(1)
		go func() {
			defer workers.Done()
			lastStatus := ""
			runFinalityPollLoop(loopCtx, 5*time.Second, func(ctx context.Context) {
				started := time.Now()
				known, err := observeFinalityOnce(ctx, verifier, store, chain, tag)
				if ctx.Err() != nil {
					return
				}
				status := "known"
				if !known {
					status = "degraded"
				}
				if err != nil && lastStatus != status && logger != nil {
					logger.Warn("Finality observation degraded", "chain_id", chain, "operation", tag)
				}
				lastStatus = status
				if metrics != nil {
					labels := map[string]string{"chain_id": chain, "operation": tag, "status": status}
					metrics.RecordCounter("finality_observation_total", 1, labels)
					metrics.RecordHistogram("finality_observation_duration_seconds", time.Since(started).Seconds(), labels)
					available := 0.0
					if known {
						available = 1
					}
					metrics.RecordGauge("finality_observation_available", available, map[string]string{"chain_id": chain, "operation": tag})
				}
			})
		}()
	}
	var once sync.Once
	return func() { once.Do(func() { cancel(); workers.Wait(); verifier.Close() }) }, nil
}

func observeFinalityOnce(ctx context.Context, reader finalityEvidenceReader, store finalityObservationWriter, chain, tag string) (bool, error) {
	proof, err := reader.ObserveFinality(ctx, chain, tag)
	if err == nil {
		if proof == nil {
			err = fmt.Errorf("finality RPC returned no evidence")
		} else {
			err = store.RecordFinalityObservation(ctx, chain, tag, proof)
		}
	}
	if err == nil {
		return true, nil
	}
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	if degradedErr := store.RecordFinalityObservation(ctx, chain, tag, nil); degradedErr != nil {
		return false, errors.Join(err, fmt.Errorf("record degraded finality: %w", degradedErr))
	}
	return false, err
}

func runFinalityPollLoop(ctx context.Context, interval time.Duration, poll func(context.Context)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for ctx.Err() == nil {
		poll(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
