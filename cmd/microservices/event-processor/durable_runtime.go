package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/rtcdance/chainpulse/pkg/application/bootstrap"
	appindexing "github.com/rtcdance/chainpulse/pkg/application/indexing"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/chainid"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/env"
	"github.com/rtcdance/chainpulse/pkg/plugins/database"
	"github.com/rtcdance/chainpulse/pkg/plugins/mq"
	"github.com/rtcdance/chainpulse/pkg/ports"
	"github.com/rtcdance/chainpulse/pkg/services/finality"
)

type durableIndexingRuntime interface {
	ProcessCompleteBlock(context.Context, appindexing.CompletedBlock) error
	Health(context.Context) (core.HealthStatus, error)
	Status() appindexing.RuntimeStatus
}

type durableProcessorStore interface {
	RecordOrphanedBlock(context.Context, string, string, []byte) error
	CompleteReorgsThrough(context.Context, string, uint64) error
}

type durableOwnership interface {
	RotateOwnership(context.Context, string) error
	ReleaseAll(context.Context) error
}

type durableProcessor struct {
	ownership         durableOwnership
	runtime           durableIndexingRuntime
	store             durableProcessorStore
	validateCanonical func(context.Context, string, *blockchain.Block) error
	failed            atomic.Int64
}

func (p *durableProcessor) AssignOwnership(ctx context.Context, owner string) error {
	if p.ownership == nil {
		return fmt.Errorf("durable processor has no ownership controller")
	}
	return p.ownership.RotateOwnership(ctx, owner)
}

func (p *durableProcessor) RevokeOwnership(ctx context.Context) error {
	if p.ownership == nil {
		return fmt.Errorf("durable processor has no ownership controller")
	}
	return p.ownership.ReleaseAll(ctx)
}

func (p *durableProcessor) ProcessEvent(context.Context, *blockchain.BlockchainEvent) error {
	return fmt.Errorf("PostgreSQL ingestion requires a complete block frame")
}

func (p *durableProcessor) ProcessCompleteBlock(ctx context.Context, frame appindexing.CompletedBlock) error {
	if err := p.runtime.ProcessCompleteBlock(ctx, frame); err != nil {
		var gap *appindexing.BlockGapError
		if errors.As(err, &gap) {
			// Queued old-branch blocks above the rewind point cannot pass the
			// contiguous checkpoint check. Probe only that rejection; normal blocks
			// retain their canonical check inside the fenced transaction.
			if p.validateCanonical == nil {
				return fmt.Errorf("canonical gap requires a configured RPC verifier")
			}
			block := &blockchain.Block{Number: frame.Header.Number.Uint64(), Hash: frame.Header.Hash(), ParentHash: frame.Header.ParentHash}
			if canonicalErr := p.validateCanonical(ctx, frame.ChainID, block); canonicalErr != nil {
				err = canonicalErr
			}
		}
		p.failed.Add(1)
		var orphan *bootstrap.NonCanonicalBlockError
		if errors.As(err, &orphan) {
			payload, encodeErr := json.Marshal(frame)
			if encodeErr != nil {
				return fmt.Errorf("encode orphan frame: %w", encodeErr)
			}
			if auditErr := p.store.RecordOrphanedBlock(ctx, frame.ChainID, frame.Header.Hash().Hex(), payload); auditErr != nil {
				return auditErr
			}
			return &appindexing.OrphanedBlockError{ChainID: frame.ChainID, Hash: frame.Header.Hash().Hex()}
		}
		return err
	}
	return p.store.CompleteReorgsThrough(ctx, frame.ChainID, frame.Header.Number.Uint64())
}

func (p *durableProcessor) Health() *core.HealthStatus {
	status, err := p.runtime.Health(context.Background())
	if err != nil {
		return &core.HealthStatus{Status: "unhealthy", Message: err.Error()}
	}
	return &status
}
func (p *durableProcessor) GetProcessedCount() int64 { return p.runtime.Status().ProcessedEvents }
func (p *durableProcessor) GetFailedCount() int64    { return p.failed.Load() }
func (p *durableProcessor) GetDuplicateCount() int64 { return p.runtime.Status().SkippedDuplicates }

type durableCheckpointReader struct {
	store *database.PostgreSQLPluginAdapter
}

func (s durableCheckpointReader) Load(ctx context.Context, chain string) (appindexing.Checkpoint, error) {
	height, hash, err := s.store.GetCheckpointForChain(ctx, chain)
	if err != nil {
		return appindexing.Checkpoint{}, fmt.Errorf("load durable checkpoint for chain %s: %w", chain, err)
	}
	return appindexing.Checkpoint{ChainID: chain, BlockNumber: height, BlockHash: hash}, err
}

func (durableCheckpointReader) Save(context.Context, appindexing.Checkpoint) error {
	return fmt.Errorf("durable checkpoint writes require the complete block transaction")
}

// runDurableEventProcessor wires the shared block transaction into the native
// Kafka service. Mongo and per-event metadata are not authorities on this path.
func runDurableEventProcessor(config EventProcessorConfig, logger ports.Logger, metrics core.MetricsCollector) (result error) {
	chains := env.GetCSV("CHAINS", nil)
	if len(chains) == 0 {
		return fmt.Errorf("PostgreSQL event processor requires CHAINS")
	}
	seen := make(map[string]bool)
	for i, chain := range chains {
		id := chainid.ResolveChainID(chain)
		if id <= 0 {
			return fmt.Errorf("durable processor requires positive EVM chain identifiers")
		}
		chains[i] = fmt.Sprint(id)
		if seen[chains[i]] {
			return fmt.Errorf("durable processor chain aliases resolve to a duplicate identity")
		}
		seen[chains[i]] = true
	}
	if _, set := os.LookupEnv("KAFKA_INPUT_TOPICS"); !set {
		config.InputTopics = []string{"chainpulse.blocks.v1"}
	}
	if len(config.InputTopics) != 1 {
		return fmt.Errorf("durable processor requires exactly one authoritative block input topic")
	}
	dsn := env.Get("DATABASE_URL", "")
	if dsn == "" {
		return fmt.Errorf("PostgreSQL event processor requires DATABASE_URL")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store := database.NewPostgreSQLPluginAdapter(logger, metrics).WithChainScope(chains...)
	if err := store.Initialize(ctx, core.Config{DatabaseURL: dsn}); err != nil {
		return fmt.Errorf("initialize durable storage: %w", err)
	}
	defer func() {
		if err := store.Stop(context.Background()); err != nil {
			logger.Error("Close durable storage", "error", err)
			result = errors.Join(result, fmt.Errorf("close durable storage: %w", err))
		}
	}()
	if err := store.Start(ctx); err != nil {
		return fmt.Errorf("start durable storage: %w", err)
	}
	gate := bootstrap.NewDrainController()
	committer, err := bootstrap.BuildMonolithicBatchCommitter(store, gate, chains)
	if err != nil {
		return err
	}
	if committer == nil {
		return fmt.Errorf("durable storage cannot fence chain ownership")
	}
	defer func() {
		releaseCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if err := committer.ReleaseAll(releaseCtx); err != nil {
			logger.Error("Release chain ownership", "error", err)
			result = errors.Join(result, fmt.Errorf("release durable chain ownership: %w", err))
		}
	}()
	verifier, err := bootstrap.NewCanonicalRPCVerifier(ctx, chains, env.GetCSV("BLOCKCHAIN_RPCS", nil))
	if err != nil {
		return err
	}
	defer verifier.Close()
	committer.SetBlockValidator(verifier.ValidateBlock)

	finalityPolicy, err := finality.NewRegistryFromConfig(bootstrap.FinalityConfigFromEnv())
	if err != nil {
		return fmt.Errorf("configure finality observation: %w", err)
	}
	stopFinality, err := bootstrap.StartFinalityObserver(ctx, store, bootstrap.FinalityObserverOptions{Chains: chains, RPCs: env.GetCSV("BLOCKCHAIN_RPCS", nil), Registry: finalityPolicy}, logger, metrics)
	if err != nil {
		return err
	}
	defer stopFinality()

	runtime, err := bootstrap.BuildMonolithicIndexingRuntimeWithOptions(logger, store, nil, chains, bootstrap.InMemoryIndexingRuntimeOptions{Committer: committer, CheckpointStore: durableCheckpointReader{store: store}})
	if err != nil {
		return err
	}
	if err := runtime.Initialize(ctx); err != nil {
		return err
	}
	if err := runtime.Start(ctx); err != nil {
		return err
	}
	defer func() {
		if err := runtime.Stop(context.Background()); err != nil {
			result = errors.Join(result, fmt.Errorf("stop durable runtime: %w", err))
		}
	}()
	processor := &durableProcessor{runtime: runtime, store: store, validateCanonical: verifier.ValidateBlock, ownership: committer}
	kafka := mq.NewKafkaMQPlugin("durable-blocks", "1", &core.Config{}, logger, metrics, nil, config.KafkaBrokers, config.ConsumerGroup)
	if err := kafka.Initialize(ctx, core.Config{}); err != nil {
		return err
	}
	if err := kafka.Start(ctx); err != nil {
		return err
	}
	defer func() {
		stopCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		if err := kafka.Stop(stopCtx); err != nil {
			logger.Error("Stop durable Kafka", "error", err)
			result = errors.Join(result, fmt.Errorf("stop durable Kafka: %w", err))
		}
	}()
	if _, set := os.LookupEnv("KAFKA_INPUT_TOPICS"); !set {
		config.InputTopics = []string{"chainpulse.blocks.v1"}
	}
	if _, set := os.LookupEnv("KAFKA_OUTPUT_TOPICS"); !set {
		config.OutputTopics = []string{"chainpulse.committed-blocks.v1"}
	}
	consume := newEventProcessorConsumeRuntime(logger, metrics, kafka, processor, kafka, config.InputTopics, config.OutputTopics, nil)
	var workers sync.WaitGroup
	consume.Start(ctx, &workers)
	defer func() { cancel(); workers.Wait() }()
	server := newEventProcessorRuntimeHTTPServer(config.Port, nil, metrics, func(*http.Request) *eventProcessorRuntimeSummaryResponse {
		return newDurableRuntimeSummary(map[string]any{"indexing": runtime.Status(), "consumer": consume.Snapshot()})
	}, consume)
	base := server.Handler
	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health/live" {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]any{"alive": true, "status": "alive", "service": "event-processor"}); err != nil {
				logger.Error("Encode durable liveness response", "error", err)
			}
			return
		}
		if r.URL.Path == "/health" || r.URL.Path == "/health/ready" || r.URL.Path == "/health/live" {
			probeCtx, stop := context.WithTimeout(r.Context(), 3*time.Second)
			defer stop()
			_, _, err := store.GetCheckpointForChain(probeCtx, chains[0])
			state := consume.Snapshot()
			ownershipReady := !state.OwnershipRequired || (state.ConfiguredTopics > 0 && state.OwnedTopics == state.ConfiguredTopics)
			runtimeReady := processor.Health().Status == "healthy"
			ready := err == nil && runtimeReady && state.Running && ownershipReady && state.LastError == ""
			w.Header().Set("Content-Type", "application/json")
			if !ready {
				w.WriteHeader(http.StatusServiceUnavailable)
			}
			if err := json.NewEncoder(w).Encode(map[string]any{"ready": ready, "status": map[bool]string{true: "healthy", false: "unhealthy"}[ready], "service": "event-processor", "checks": map[string]bool{"storage": err == nil, "runtime": runtimeReady, "ownership": ownershipReady, "consumer": state.Running, "effects": state.LastError == ""}}); err != nil {
				logger.Error("Encode durable health response", "error", err)
			}
			return
		}
		base.ServeHTTP(w, r)
	})
	auth, limits, err := bootstrap.BuildSecurityControls(bootstrap.SecurityControlsConfig{AuthEnabled: config.AuthEnabled, AuthJWTSecret: config.AuthJWTSecret, AuthAPIKeys: config.AuthAPIKeys, RateLimitEnabled: config.RateLimitEnabled, RateLimitPerMinute: config.RateLimitPerMinute, ServiceName: "event processor", EnvPrefix: "EVENT_PROCESSOR"}, logger, metrics)
	if err != nil {
		return err
	}
	server.Handler = wrapEventProcessorRuntimeSecurityHandler(server.Handler, auth, limits)
	serverErrors := make(chan error, 1)
	go func() { serverErrors <- server.ListenAndServe() }()
	defer func() {
		stopCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if err := server.Shutdown(stopCtx); err != nil {
			result = errors.Join(result, fmt.Errorf("shutdown durable HTTP: %w", err))
		}
	}()
	logger.Info("Durable block processor started", "chains", chains, "topics", config.InputTopics)
	signalCtx, stopSignal := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stopSignal()
	select {
	case <-signalCtx.Done():
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("durable HTTP server: %w", err)
		}
	}
	if !gate.Drain(25 * time.Second) {
		return fmt.Errorf("durable block transaction failed to drain")
	}
	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("durable HTTP server: %w", err)
		}
	default:
	}
	return nil
}

func newDurableRuntimeSummary(processor map[string]any) *eventProcessorRuntimeSummaryResponse {
	return &eventProcessorRuntimeSummaryResponse{
		Service: "event-processor", Timestamp: time.Now().Unix(),
		DeploymentMode: core.DeploymentModeMicroservice,
		RuntimeMode:    "runtime-wired", RuntimePosture: "runtime-wired", ComponentState: "runtime-wired",
		Processor: processor,
	}
}
