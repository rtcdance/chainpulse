// Command loadtest is a minimal p99/throughput load harness for the §8
// performance data path. It uses the Go standard library only — no third-party
// load tooling; the in-process mode links the repo's own pkg/evm decode hot
// path (the same path the pkg/evm micro benchmarks exercise).
//
// Usage:
//
//	go run ./scripts/bench/loadtest -workers 4 -duration 2s -payload-size 32
//	BASE_URL=http://localhost:8080/health go run ./scripts/bench/loadtest -workers 8 -duration 5s
//
// Modes:
//   - default (no BASE_URL): in-process — each worker repeatedly calls
//     evm.DecodeEventData on a fixed ERC-20 Transfer log; -payload-size sets the
//     event data buffer in bytes (rounded up to a whole 32-byte ABI word, min 32).
//   - BASE_URL set: each worker issues GET requests to that URL and counts
//     transport failures and HTTP >= 400 as errors (-payload-size is unused).
//
// Output: one JSON object on stdout with p50_ms, p95_ms, p99_ms (nearest-rank
// over sorted per-op latency samples), throughput_ops (ops/s), errors,
// duration_s and workers. Nothing is wired into CI workflows.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/evm"
)

type summary struct {
	P50Ms         float64 `json:"p50_ms"`
	P95Ms         float64 `json:"p95_ms"`
	P99Ms         float64 `json:"p99_ms"`
	ThroughputOps float64 `json:"throughput_ops"`
	Errors        int64   `json:"errors"`
	DurationS     float64 `json:"duration_s"`
	Workers       int     `json:"workers"`
}

// operation performs one load-test iteration. Latency is sampled around it
// regardless of the returned error, because an errored op still costs time.
type operation func() error

func main() {
	workers := flag.Int("workers", 4, "number of concurrent workers")
	duration := flag.Duration("duration", 2*time.Second, "how long to run the load")
	payloadSize := flag.Int("payload-size", 32, "event data buffer size in bytes (in-process mode; rounded up to a 32-byte ABI word)")
	flag.Parse()

	if *workers < 1 {
		fatalf("-workers must be >= 1")
	}
	if *duration <= 0 {
		fatalf("-duration must be > 0")
	}
	if *payloadSize < 1 {
		fatalf("-payload-size must be >= 1")
	}

	op, mode, err := buildOperation(*payloadSize)
	if err != nil {
		fatalf("%v", err)
	}
	fmt.Fprintf(os.Stderr, "loadtest: mode=%s workers=%d duration=%s payload_size=%d\n",
		mode, *workers, *duration, *payloadSize)

	result := run(op, *workers, *duration)

	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(result); err != nil {
		fatalf("encode summary: %v", err)
	}
}

func buildOperation(payloadSize int) (operation, string, error) {
	if baseURL := os.Getenv("BASE_URL"); baseURL != "" {
		client := &http.Client{Timeout: 10 * time.Second}
		return func() error {
			resp, err := client.Get(baseURL)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if _, err := io.Copy(io.Discard, resp.Body); err != nil {
				return err
			}
			if resp.StatusCode >= http.StatusBadRequest {
				return fmt.Errorf("HTTP %d", resp.StatusCode)
			}
			return nil
		}, "http", nil
	}

	// Same log shape as BenchmarkDecodeEventDataTransfer in pkg/evm.
	topics := []common.Hash{
		common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"),
		common.HexToHash("0x000000000000000000000000f39fd6e51aad88f6f4ce6ab8827279cfffb92266"),
		common.HexToHash("0x00000000000000000000000070997970c51812dc3a010c7d01b50e0d17dc79c8"),
	}
	// Whole ABI words only: a ragged length makes abi.Unpack reject the buffer,
	// which would measure the logger instead of the decode path.
	words := (payloadSize + 31) / 32
	if words < 1 {
		words = 1
	}
	data := make([]byte, words*32)
	data[len(data)-1] = 0xe8 // value = 1000 when the buffer is exactly one word

	return func() error {
		if evm.DecodeEventData("Transfer", topics, data) == nil {
			return fmt.Errorf("DecodeEventData returned nil")
		}
		return nil
	}, "decode", nil
}

func run(op operation, workers int, duration time.Duration) summary {
	deadline := time.Now().Add(duration)
	latencies := make([][]time.Duration, workers)
	errCounts := make([]int64, workers)

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			local := latencies[w]
			for time.Now().Before(deadline) {
				start := time.Now()
				err := op()
				local = append(local, time.Since(start))
				if err != nil {
					errCounts[w]++
				}
			}
			latencies[w] = local
		}(w)
	}
	wg.Wait()

	var (
		all     []time.Duration
		errs    int64
		total   int
		elapsed = time.Since(deadline.Add(-duration))
	)
	for w := 0; w < workers; w++ {
		all = append(all, latencies[w]...)
		errs += errCounts[w]
		total += len(latencies[w])
	}
	sort.Slice(all, func(i, j int) bool { return all[i] < all[j] })

	secs := elapsed.Seconds()
	if secs <= 0 {
		secs = duration.Seconds()
	}
	return summary{
		P50Ms:         percentileMs(all, 0.50),
		P95Ms:         percentileMs(all, 0.95),
		P99Ms:         percentileMs(all, 0.99),
		ThroughputOps: float64(total) / secs,
		Errors:        errs,
		DurationS:     secs,
		Workers:       workers,
	}
}

// percentileMs is nearest-rank on an ascending-sorted sample: the value at
// ceil(q*n)-1, so p99 of a small sample is a real observed latency, never an
// interpolation between two of them.
func percentileMs(sorted []time.Duration, q float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(q*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return float64(sorted[idx]) / float64(time.Millisecond)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "loadtest: "+format+"\n", args...)
	os.Exit(1)
}
