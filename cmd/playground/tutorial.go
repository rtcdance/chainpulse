package main

import (
	"net/http"
	"strings"
)

// tutorialStep represents one step in the interactive Go learning tutorial.
type tutorialStep struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Curl        string `json:"curl"`
	Concept     string `json:"concept"`
	Category    string `json:"category"`
}

var tutorialSteps = []tutorialStep{
	{
		Title:       "1. Hello, Blockchain Event",
		Description: "Generate mock ERC-20 Transfer events in memory. No blockchain needed!",
		Curl:        "curl http://localhost:PORT/generate",
		Concept:     "Go struct mapping: Solidity event → Go blockchain.BlockchainEvent",
		Category:    "Web3 Basics",
	},
	{
		Title:       "2. Listing Events",
		Description: "Query all stored events — shows how the in-memory database works.",
		Curl:        "curl http://localhost:PORT/events",
		Concept:     "JSON serialization: Go struct → JSON via json.Marshal",
		Category:    "Data Access",
	},
	{
		Title:       "3. AMM Math (Uniswap Swap)",
		Description: "Generate a Swap event using core.ConstantProductAMM — x·y=k invariant.",
		Curl:        "curl http://localhost:PORT/generate-swap",
		Concept:     "DeFi primitives in Go: core.ConstantProductAMM.SpotPrice()",
		Category:    "DeFi",
	},
	{
		Title:       "4. Account Abstraction (ERC-4337)",
		Description: "Generate a UserOperation event — sender, paymaster, nonce, gas.",
		Curl:        "curl http://localhost:PORT/generate-aa",
		Concept:     "EIP-4337 in Go: UserOperation struct, bundler mempool",
		Category:    "AA / L2",
	},
	{
		Title:       "5. EventBus Pub-Sub",
		Description: "Subscribe to SSE stream, then publish events from another terminal.",
		Curl:        "curl -N http://localhost:PORT/subscribe  # Terminal 1\ncurl 'http://localhost:PORT/publish?topic=events&msg=CSP!'  # Terminal 2",
		Concept:     "Go CSP model: channels, goroutines, select — no Kafka needed",
		Category:    "Concurrency",
	},
	{
		Title:       "6. Result[T] — Error Handling",
		Description: "core.Result[T] wraps a value-or-error — like Rust/TypeScript Results.",
		Curl:        "curl 'http://localhost:PORT/stats'",
		Concept:     "Generics: core.Result[T] = Go's (T, error) with functional Map/OrElse",
		Category:    "Generics",
	},
	{
		Title:       "7. Middleware Chain",
		Description: "Every request goes through: recovery → logging → rate limiter → handler.",
		Curl:        "curl -v http://localhost:PORT/stats  # Look at X-* headers",
		Concept:     "Go HTTP middleware: func(http.Handler) http.Handler — onion pattern",
		Category:    "Patterns",
	},
	{
		Title:       "8. Graceful Shutdown",
		Description: "Press Ctrl+C — the server drains connections before exiting cleanly.",
		Curl:        "# Start server, then: curl http://localhost:PORT/stats & kill -INT <pid>",
		Concept:     "Go shutdown: signal.NotifyContext + server.Shutdown() + ctx.Deadline",
		Category:    "Patterns",
	},
	{
		Title:       "9. pprof Performance",
		Description: "CPU/heap/goroutine profiling — Go's answer to debug_traceTransaction.",
		Curl:        "go tool pprof -http=:8081 http://localhost:6060/debug/pprof/heap",
		Concept:     "Performance: pprof = EVM gas profiler for Go programs",
		Category:    "Performance",
	},
	{
		Title:       "10. Code Generation",
		Description: "Generate Go event structs from Solidity ABI JSON (like typechain).",
		Curl:        "chainpulse gen-abi -abi contracts/EventEmitter.abi.json -out events_gen.go",
		Concept:     "go generate: ABI JSON → Go struct with type-safe Solidity type mapping",
		Category:    "Tooling",
	},
}

func (p *playground) handleTutorial(w http.ResponseWriter, r *http.Request) {
	step := r.URL.Query().Get("step")

	if step != "" {
		// Return single step
		for _, s := range tutorialSteps {
			if strings.HasPrefix(s.Title, step+".") || strings.EqualFold(s.Concept, step) {
				writeJSON(w, map[string]any{
					"step":    s,
					"concept": s.Concept,
					"curl":    strings.ReplaceAll(s.Curl, "PORT", r.Host),
				})
				return
			}
		}
		jsonError(w, http.StatusNotFound, "step not found. Try /tutorial (no query) for full list.")
		return
	}

	// Format curl commands with correct port
	steps := make([]tutorialStep, len(tutorialSteps))
	copy(steps, tutorialSteps)
	host := r.Host
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}
	port := "9099"
	for i := range steps {
		steps[i].Curl = strings.ReplaceAll(steps[i].Curl, "PORT", host+":"+port)
	}

	// Group by category
	categories := make(map[string][]tutorialStep)
	for _, s := range steps {
		categories[s.Category] = append(categories[s.Category], s)
	}

	writeJSON(w, map[string]any{
		"title":      "ChainPulse Go Learning Path",
		"steps":      len(steps),
		"categories": categories,
		"next":       "Pick a step: curl .../tutorial?step=1",
	})
}

func (p *playground) handleConcepts(w http.ResponseWriter, r *http.Request) {
	concepts := []map[string]string{
		{"name": "goroutine", "go": "go func() { ... }()", "web3": "setTimeout / async function"},
		{"name": "channel", "go": "ch := make(chan Event, 64)", "web3": "EventEmitter .on() / .emit()"},
		{"name": "select", "go": "select { case <-ch: ... case <-ctx.Done(): ... }", "web3": "Promise.race([event, timeout])"},
		{"name": "context", "go": "ctx, cancel := context.WithTimeout(parent, 5*time.Second)", "web3": "AbortController + setTimeout"},
		{"name": "interface", "go": "type Plugin interface { Start() error }", "web3": "interface IPlugin { function start(); }"},
		{"name": "errgroup", "go": "g, ctx := errgroup.WithContext(ctx)", "web3": "Promise.all([...]).catch(...)"},
		{"name": "defer", "go": "defer f.Close()", "web3": "try { ... } finally { ... }"},
		{"name": "Result[T]", "go": "core.WrapResult(fetchBlock(100)).Map(...)", "web3": "result.map(...) // Rust/neverthrow"},
		{"name": "sync.Mutex", "go": "mu.Lock(); defer mu.Unlock()", "web3": "ReentrancyGuard / mutex pattern"},
		{"name": "go generate", "go": "//go:generate chainpulse gen-abi -abi ...", "web3": "typechain / hardhat typechain"},
	}

	writeJSON(w, map[string]any{
		"title":    "Go ↔ Web3 Concept Map",
		"count":    len(concepts),
		"concepts": concepts,
	})
}
