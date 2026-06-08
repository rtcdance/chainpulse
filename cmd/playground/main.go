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
	"log/slog"
	"math/big"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/core/replay"
	"github.com/rtcdance/chainpulse/pkg/defi"

	"github.com/ethereum/go-ethereum/common"
)

// --- mock puller ---

type mockPuller struct {
	events   []blockchain.BlockchainEvent
	nextID   atomic.Uint64
	blockNum atomic.Uint64
}

func newMockPuller() *mockPuller {
	p := &mockPuller{}
	p.blockNum.Store(17_000_000)
	return p
}

// generate creates count mock ERC-20 Transfer events.
func (p *mockPuller) generate(count int) []blockchain.BlockchainEvent {
	block := p.blockNum.Add(uint64(count))
	now := time.Now()
	generated := make([]blockchain.BlockchainEvent, 0, count)

	for i := 0; i < count; i++ {
		id := p.nextID.Add(1)
		bn := block - uint64(count) + uint64(i) + 1
		generated = append(generated, blockchain.BlockchainEvent{
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
			Status:    blockchain.EventStatusConfirmed,
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

// --- in-memory database (self-contained, no bootstrap dependency) ---

type memoryDB struct {
	mu     sync.RWMutex
	events map[string]*blockchain.BlockchainEvent
}

func newMemoryDB() *memoryDB {
	return &memoryDB{events: make(map[string]*blockchain.BlockchainEvent)}
}

func (db *memoryDB) StoreEvent(ctx context.Context, event *blockchain.BlockchainEvent) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.events[event.ID] = event
	return nil
}

func (db *memoryDB) GetAllEvents(ctx context.Context) ([]*blockchain.BlockchainEvent, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	result := make([]*blockchain.BlockchainEvent, 0, len(db.events))
	for _, ev := range db.events {
		result = append(result, ev)
	}
	return result, nil
}

// --- playground ---

type playground struct {
	db       *memoryDB
	puller   *mockPuller
	eventBus *core.ChannelEventBus
}

func newPlayground(logger core.Logger) *playground {
	db := newMemoryDB()

	pg := &playground{
		db:       db,
		puller:   newMockPuller(),
		eventBus: core.NewChannelEventBus(),
	}

	pg.eventBus.SubscribeNamed(context.Background(), "events", "printer", func(_ context.Context, event any) error {
		if ev, ok := event.(blockchain.BlockchainEvent); ok {
			slog.Info(
				"eventbus received",
				"event", ev.EventName,
				"block", ev.BlockNumber,
				"network", ev.Network,
			)
		}
		return nil
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
		_ = p.db.StoreEvent(context.Background(), &ev)
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
		all = []*blockchain.BlockchainEvent{}
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
// from defi.ConstantProductAMM (defi_primitives.go).
func (p *mockPuller) generateSwap() blockchain.BlockchainEvent {
	id := p.nextID.Add(1)
	bn := p.blockNum.Add(1)
	now := time.Now()

	// Simulate a USDC/WETH swap with 0.3% fee
	amm := defi.NewConstantProductAMM(
		big.NewInt(5_000_000_000_000),         // 5000 USDC reserve (6 decimals)
		big.NewInt(1_000_000_000_000_000_000), // 1 WETH reserve (18 decimals)
		30,                                    // 0.3% fee
	)

	return blockchain.BlockchainEvent{
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
		Status:    blockchain.EventStatusConfirmed,
		CreatedAt: now,
	}
}

func amtToString(f *big.Float) string {
	return f.String()
}

// generateAA creates a mock ERC-4337 UserOperation event demonstrating
// the Account Abstraction types in blockchain.BlockchainEvent.
func (p *mockPuller) generateAA() blockchain.BlockchainEvent {
	id := p.nextID.Add(1)
	bn := p.blockNum.Add(1)
	now := time.Now()

	return blockchain.BlockchainEvent{
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
		Status:    blockchain.EventStatusConfirmed,
		CreatedAt: now,
	}
}

func (p *playground) handleGenerateSwap(w http.ResponseWriter, r *http.Request) {
	ev := p.puller.generateSwap()
	_ = p.db.StoreEvent(context.Background(), &ev)
	_ = p.eventBus.Publish(context.Background(), "events", ev)
	writeJSON(w, map[string]any{
		"generated": 1,
		"event_id":  ev.ID,
		"event":     "Swap (via defi.ConstantProductAMM)",
	})
}

func (p *playground) handleGenerateAA(w http.ResponseWriter, r *http.Request) {
	ev := p.puller.generateAA()
	_ = p.db.StoreEvent(context.Background(), &ev)
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
	_, err := p.eventBus.SubscribeNamed(ctx, topic, "", func(_ context.Context, event any) error {
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		return nil
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

	signerType := replay.InferEIP155SignerType(big.NewInt(int64(v)))
	isVulnerable := replay.IsReplayVulnerable(v)
	extractedChainID := replay.ExtractChainIDFromV(v)

	result := map[string]any{
		"v_value":         v,
		"signer_type":     signerType.String(),
		"is_vulnerable":   isVulnerable,
		"extracted_chain": extractedChainID,
		"explanation":     explainEIP155(v, isVulnerable, extractedChainID, signerType),
	}
	writeJSON(w, result)
}

func explainEIP155(v uint64, vulnerable bool, chainID *big.Int, signerType replay.SignerType) string {
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

const defaultPort = "9099"

func main() {
	port := defaultPort
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

	fmt.Printf("\033]8;;http://localhost:%s\033\\🔗 http://localhost:%s\033]8;;\033\\\n", port, port)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  ChainPulse Playground")
	fmt.Println("  Next step: open the URL above in your browser")
	fmt.Println("  → /generate  — create mock blockchain events")
	fmt.Println("  → /events    — view all indexed events")
	fmt.Println("  → /stats     — see event statistics")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  Quick start:")
	fmt.Printf("    curl http://localhost:%s/generate\n", port)
	fmt.Printf("    curl http://localhost:%s/events?network=ethereum | jq\n", port)
	fmt.Printf("    curl http://localhost:%s/events?network=polygon | jq\n", port)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Auto-open browser on supported platforms
	if port == defaultPort {
		_ = openBrowser("http://localhost:" + port)
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Warn("playground server error", "error", err)
		}
	}()

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
		slog.Info("pprof debug server", "addr", pprofSrv.Addr+"/debug/pprof/")
		if err := pprofSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Warn("pprof server error", "error", err)
		}
	}()

	// Block until signal received
	<-notifyCtx.Done()
	slog.Info("shutdown signal received, draining connections...")

	// Shutdown with deadline
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Warn("shutdown API server forced close", "error", err)
	}
	if err := pprofSrv.Shutdown(shutdownCtx); err != nil {
		slog.Warn("shutdown pprof server forced close", "error", err)
	}
}

// openBrowser opens the given URL in the default browser.
// Silently ignores errors (e.g., no browser available in headless environments).
func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}
