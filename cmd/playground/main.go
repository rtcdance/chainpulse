// Command playground starts ChainPulse in playground mode:
// a zero-dependency in-memory environment with mock blockchain events
// and a REST API — no PostgreSQL, Kafka, or Redis required.
//
// Ideal for learners exploring the Web3 → Go data flow.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"chainpulse/pkg/application/bootstrap"
	"chainpulse/pkg/core"

	"github.com/ethereum/go-ethereum/common"
)

// --- mock puller ---

type mockPuller struct {
	mu       sync.Mutex
	events   []core.BlockchainEvent
	nextID   atomic.Uint64
	blockNum atomic.Uint64
}

func newMockPuller() *mockPuller {
	p := &mockPuller{}
	p.blockNum.Store(17_000_000)
	return p
}

// generate creates count mock ERC-20 Transfer events.
func (p *mockPuller) generate(count int) []core.BlockchainEvent {
	p.mu.Lock()
	defer p.mu.Unlock()

	block := p.blockNum.Add(uint64(count))
	now := time.Now()
	generated := make([]core.BlockchainEvent, 0, count)

	for i := 0; i < count; i++ {
		id := p.nextID.Add(1)
		bn := block - uint64(count) + uint64(i) + 1
		generated = append(generated, core.BlockchainEvent{
			ID:              fmt.Sprintf("mock_evt_%d", id),
			EventHash:       fmt.Sprintf("0x%064x", id),
			EventSignature:  common.HexToHash(fmt.Sprintf("0x%064x", id)),
			BlockNumber:     bn,
			BlockHash:       common.HexToHash(fmt.Sprintf("0x%064x", bn)),
			BlockTimestamp:  now.Unix(),
			TransactionHash: common.HexToHash(fmt.Sprintf("0x%064x", id)),
			LogIndex:        uint64(i),
			ContractAddress: common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
			EventName:       "Transfer",
			EventData:       mockTransferData(1000 + id),
			DecodedData: map[string]any{
				"from":  "0x1234",
				"to":    "0x5678",
				"value": fmt.Sprintf("%d", 1000+id),
			},
			ChainID:   "1",
			Network:   "mainnet",
			Status:    core.EventStatusConfirmed,
			CreatedAt: now,
		})
	}
	p.events = append(p.events, generated...)
	return generated
}

func mockTransferData(amount uint64) []byte {
	v := new(big.Int).SetUint64(amount)
	b := make([]byte, 32)
	v.FillBytes(b)
	return b
}

// --- playground ---

type playground struct {
	db       *bootstrap.MonolithicMemoryDatabase
	puller   *mockPuller
	eventBus *core.ChannelEventBus
}

func newPlayground(logger core.Logger) *playground {
	pg := &playground{
		db:       bootstrap.NewMonolithicMemoryDatabase(logger),
		puller:   newMockPuller(),
		eventBus: core.NewChannelEventBus(),
	}

	// Demonstrate ChannelEventBus: subscribe to events and print them
	pg.eventBus.SubscribeNamed(context.Background(), "events", "printer", func(event any) {
		if ev, ok := event.(core.BlockchainEvent); ok {
			log.Printf("[eventbus] received: %s (block=%d, network=%s)", ev.EventName, ev.BlockNumber, ev.Network)
		}
	})

	return pg
}

func (p *playground) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		handleHome(w, r)
	case "/generate":
		p.handleGenerate(w, r)
	case "/generate-swap":
		p.handleGenerateSwap(w, r)
	case "/generate-aa":
		p.handleGenerateAA(w, r)
	case "/events":
		p.handleListEvents(w, r)
	case "/stats":
		p.handleStats(w, r)
	case "/subscribe":
		p.handleSubscribe(w, r)
	case "/publish":
		p.handlePublish(w, r)
	case "/tutorial":
		p.handleTutorial(w, r)
	case "/concepts":
		p.handleConcepts(w, r)
	case "/pool":
		p.handlePoolDemo(w, r)
	case "/replay-check":
		p.handleReplayCheck(w, r)
	default:
		jsonError(w, http.StatusNotFound, "unknown path")
	}
}

func (p *playground) handleGenerate(w http.ResponseWriter, r *http.Request) {
	events := p.puller.generate(5)
	for _, ev := range events {
		_ = p.db.StoreEvent(context.Background(), ev)
		_ = p.eventBus.Publish(context.Background(), "events", ev)
	}
	writeJSON(w, map[string]any{
		"generated": len(events),
		"message":   "mock events created",
	})
}

func (p *playground) handleListEvents(w http.ResponseWriter, r *http.Request) {
	all, err := p.db.GetAllEvents(context.Background())
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if all == nil {
		all = []*core.BlockchainEvent{}
	}
	writeJSON(w, map[string]any{
		"total":  len(all),
		"events": all,
	})
}

func (p *playground) handleStats(w http.ResponseWriter, r *http.Request) {
	all, _ := p.db.GetAllEvents(context.Background())
	count := 0
	if all != nil {
		count = len(all)
	}
	writeJSON(w, map[string]any{
		"version":              "chainpulse-playground",
		"mode":                 "in-memory",
		"total_events":         count,
		"current_block":        p.puller.blockNum.Load(),
		"requires_external_db": false,
	})
}

// generateSwap creates a mock Uniswap Swap event demonstrating AMM math
// from core.ConstantProductAMM (defi_primitives.go).
func (p *mockPuller) generateSwap() core.BlockchainEvent {
	id := p.nextID.Add(1)
	bn := p.blockNum.Add(1)
	now := time.Now()

	// Simulate a USDC/WETH swap with 0.3% fee
	amm := core.NewConstantProductAMM(
		big.NewInt(5_000_000_000_000),         // 5000 USDC reserve (6 decimals)
		big.NewInt(1_000_000_000_000_000_000), // 1 WETH reserve (18 decimals)
		30,                                    // 0.3% fee
	)

	return core.BlockchainEvent{
		ID:              fmt.Sprintf("mock_swap_%d", id),
		EventHash:       fmt.Sprintf("0x%064x", id),
		EventSignature:  common.HexToHash("0xd78ad95fa46c994b6551d0da85fc275fe613ce37657fb8d5e3d130840159d822"),
		BlockNumber:     bn,
		BlockHash:       common.HexToHash(fmt.Sprintf("0x%064x", bn)),
		BlockTimestamp:  now.Unix(),
		TransactionHash: common.HexToHash(fmt.Sprintf("0x%064x", id)),
		LogIndex:        0,
		ContractAddress: common.HexToAddress("0x88e6A0c2dDD26FEEb64F039a2c41296FcB3f5640"),
		EventName:       "Swap",
		EventData:       []byte{},
		DecodedData: map[string]any{
			"sender":       "0x1234",
			"amount0In":    "1000000",
			"amount1Out":   amtToString(amm.SpotPrice()),
			"sqrtPriceX96": "1982923847293874293",
		},
		ChainID:   "1",
		Network:   "mainnet",
		Status:    core.EventStatusConfirmed,
		CreatedAt: now,
	}
}

func amtToString(f *big.Float) string {
	return f.String()
}

// generateAA creates a mock ERC-4337 UserOperation event demonstrating
// the Account Abstraction types in core.BlockchainEvent.
func (p *mockPuller) generateAA() core.BlockchainEvent {
	id := p.nextID.Add(1)
	bn := p.blockNum.Add(1)
	now := time.Now()

	return core.BlockchainEvent{
		ID:              fmt.Sprintf("mock_aa_%d", id),
		EventHash:       fmt.Sprintf("0x%064x", id),
		EventSignature:  common.HexToHash("0x49628e147c1ffba5e4c8a0b077908f2c04d246a911fc3da0f2d5e80eecae9148"),
		BlockNumber:     bn,
		BlockHash:       common.HexToHash(fmt.Sprintf("0x%064x", bn)),
		BlockTimestamp:  now.Unix(),
		TransactionHash: common.HexToHash(fmt.Sprintf("0x%064x", id)),
		LogIndex:        0,
		ContractAddress: common.HexToAddress("0x5FF137D4b0FDCD49DcA30c7d57e7E7188e1a3E0a"),
		EventName:       "UserOperationEvent",
		EventData:       []byte{},
		DecodedData: map[string]any{
			"userOpHash":    fmt.Sprintf("0x%064x", id),
			"sender":        "0x1234",
			"paymaster":     "0x0000000000000000000000000000000000000000",
			"nonce":         fmt.Sprintf("%d", id),
			"success":       true,
			"actualGasCost": "100000",
			"actualGasUsed": "50000",
		},
		ChainID:   "1",
		Network:   "mainnet",
		Status:    core.EventStatusConfirmed,
		CreatedAt: now,
	}
}

func (p *playground) handleGenerateSwap(w http.ResponseWriter, r *http.Request) {
	ev := p.puller.generateSwap()
	_ = p.db.StoreEvent(context.Background(), ev)
	_ = p.eventBus.Publish(context.Background(), "events", ev)
	writeJSON(w, map[string]any{
		"generated": 1,
		"event_id":  ev.ID,
		"event":     "Swap (via core.ConstantProductAMM)",
	})
}

func (p *playground) handleGenerateAA(w http.ResponseWriter, r *http.Request) {
	ev := p.puller.generateAA()
	_ = p.db.StoreEvent(context.Background(), ev)
	_ = p.eventBus.Publish(context.Background(), "events", ev)
	writeJSON(w, map[string]any{
		"generated": 1,
		"event_id":  ev.ID,
		"event":     "UserOperationEvent (ERC-4337)",
	})
}

// handleSubscribe demonstrates Go's CSP concurrency model:
// a channel-based pub-sub pattern — no Kafka needed.
func (p *playground) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	topic := r.URL.Query().Get("topic")
	if topic == "" {
		topic = "events"
	}

	// Flusher for SSE (Server-Sent Events)
	flusher, ok := w.(http.Flusher)
	if !ok {
		jsonError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ctx := r.Context()
	_, err := p.eventBus.SubscribeNamed(ctx, topic, "", func(event any) {
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	<-ctx.Done()
}

// handlePublish demonstrates direct EventBus publishing.
func (p *playground) handlePublish(w http.ResponseWriter, r *http.Request) {
	topic := r.URL.Query().Get("topic")
	if topic == "" {
		topic = "events"
	}
	msg := r.URL.Query().Get("msg")
	if msg == "" {
		msg = fmt.Sprintf("manual_publish_%d", time.Now().UnixNano())
	}

	err := p.eventBus.Publish(r.Context(), topic, map[string]any{
		"topic":     topic,
		"message":   msg,
		"timestamp": time.Now().Unix(),
	})
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	subs := p.eventBus.SubscriberCount(topic)
	writeJSON(w, map[string]any{
		"status":      "published",
		"topic":       topic,
		"message":     msg,
		"subscribers": subs,
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

// handleReplayCheck demonstrates EIP-155 replay protection:
// Given a signature V value, it extracts and validates the chain ID
// encoded within, showing how Ethereum prevents cross-chain replay attacks.
func (p *playground) handleReplayCheck(w http.ResponseWriter, r *http.Request) {
	vStr := r.URL.Query().Get("v")
	if vStr == "" {
		vStr = "37"
	}
	v, err := strconv.ParseUint(vStr, 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, fmt.Sprintf("invalid v value: %v", err))
		return
	}

	signerType := core.InferEIP155SignerType(big.NewInt(int64(v)))
	isVulnerable := core.IsReplayVulnerable(v)
	extractedChainID := core.ExtractChainIDFromV(v)

	result := map[string]any{
		"v_value":          v,
		"signer_type":      signerType.String(),
		"is_vulnerable":    isVulnerable,
		"extracted_chain":  extractedChainID,
		"explanation":      explainEIP155(v, isVulnerable, extractedChainID, signerType),
	}
	writeJSON(w, result)
}

func explainEIP155(v uint64, vulnerable bool, chainID *big.Int, signerType core.SignerType) string {
	if vulnerable {
		return fmt.Sprintf("V=%d 是 Homestead 签名（27 或 28），没有链 ID 保护，存在跨链重放攻击风险。", v)
	}
	if chainID != nil {
		id := chainID.Int64()
		return fmt.Sprintf("V=%d 是 EIP-155 签名，编码了链 ID=%d。"+
			"公式: chainID = (V - 35) / 2 = (%d - 35) / 2 = %d。"+
			"这笔交易只能在链 %d 上重放，其他链上的重放会被拒绝。",
			v, id, v, id, id)
	}
	return fmt.Sprintf("V=%d, 签名类型: %s。无法提取链 ID。", v, signerType.String())
}

func jsonError(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	writeJSON(w, map[string]string{"error": msg})
}

func main() {
	port := "9099"
	if p := os.Getenv("PLAYGROUND_PORT"); p != "" {
		port = p
	}

	logger := core.NewDefaultLogger(core.LogLevelInfo)
	pg := newPlayground(logger)

	// Generate initial batch
	pg.puller.generate(10)

	mux := http.NewServeMux()
	mux.Handle("/", pg)

	// Build middleware chain (Go's classic HTTP middleware pattern)
	rl := newRateLimiter(100, time.Minute)
	chain := newMiddlewareChain()
	chain.Use(recoveryMiddleware)
	chain.Use(loggingMiddleware)
	chain.Use(rl.middleware)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: chain.Then(mux),
	}

	// Graceful shutdown using the standard Go pattern:
	//   1. signal.NotifyContext — cancel on SIGINT/SIGTERM
	//   2. server.Shutdown(ctx) — drain connections
	//   3. context.WithTimeout — enforce a deadline
	//
	// This is the same pattern used by Kubernetes, Docker, and every
	// major Go web framework. Mastering it is essential.
	notifyCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start pprof debug server for performance analysis
	pprofSrv := &http.Server{Addr: "localhost:6060", Handler: nil}
	go func() {
		log.Printf("[pprof] debug server on %s/debug/pprof/", pprofSrv.Addr)
		if err := pprofSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[pprof] server error: %v", err)
		}
	}()

	go func() {
		printBanner(port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[server] error: %v", err)
		}
	}()

	// Block until signal received
	<-notifyCtx.Done()
	log.Println("[shutdown] signal received, draining connections...")

	// Shutdown with deadline
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("[shutdown] API server forced close: %v", err)
	}
	if err := pprofSrv.Shutdown(shutdownCtx); err != nil {
		log.Printf("[shutdown] pprof server forced close: %v", err)
	}

	log.Println("[shutdown] all connections drained — goodbye")
}

func printBanner(port string) {
	fmt.Printf("\n")
	fmt.Printf("╔══════════════════════════════════════════════════╗\n")
	fmt.Printf("║     ChainPulse Playground                        ║\n")
	fmt.Printf("║     Zero-dependency in-memory mode               ║\n")
	fmt.Printf("╠══════════════════════════════════════════════════╣\n")
	fmt.Printf("║  API: http://localhost:%s                       ║\n", port)
	fmt.Printf("║  pprof: http://localhost:6060/debug/pprof/      ║\n")
	fmt.Printf("║                                                  ║\n")
	fmt.Printf("║  Try:                                            ║\n")
	fmt.Printf("║   curl http://localhost:%s/stats     — stats     ║\n", port)
	fmt.Printf("║   curl http://localhost:%s/generate  — gen events║\n", port)
	fmt.Printf("║   curl http://localhost:%s/events    — list all  ║\n", port)
	fmt.Printf("║   curl http://localhost:%s/subscribe — SSE stream║\n", port)
	fmt.Printf("║   curl http://localhost:%s/publish   — pub event ║\n", port)
	fmt.Printf("║   curl http://localhost:%s/tutorial — 10-step Go guide║\n", port)
	fmt.Printf("║   curl http://localhost:%s/concepts — Go↔Web3 map   ║\n", port)
		fmt.Printf("║   curl http://localhost:%s/pool     — sync.Pool demo║\n", port)
	fmt.Printf("║   curl http://localhost:%s/replay-check — EIP-155 replay║\n", port)
	fmt.Printf("║   open http://localhost:%s          — Web UI        ║\n", port)
	fmt.Printf("╚══════════════════════════════════════════════════╝\n")
	fmt.Printf("\n")
}
