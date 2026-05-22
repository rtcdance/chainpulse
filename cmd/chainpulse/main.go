package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/rtcdance/chainpulse/pkg/codegen"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	switch os.Args[1] {
	case "run":
		runMonolithic(ctx)
	case "playground":
		runPlayground(ctx)
	case "status":
		runStatus(ctx)
	case "gen-abi":
		runGenABI(ctx)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`ChainPulse — Web3 Event Indexer

Usage:
  chainpulse run          Start monolithic mode (requires env config)
  chainpulse playground   Start zero-dependency in-memory playground
  chainpulse status       Query a running instance health
  chainpulse gen-abi      Generate Go structs from Solidity ABI JSON

Examples:
  chainpulse playground
  chainpulse gen-abi -abi ./abi.json -out ./events_gen.go
  curl http://localhost:9099/stats
`)
}

func runMonolithic(ctx context.Context) {
	args := []string{"run", "./cmd/monolithic/chainpulse/"}
	if len(os.Args) > 2 {
		args = append(args, "--")
		args = append(args, os.Args[2:]...)
	}
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "monolithic failed: %v\n", err)
		os.Exit(1)
	}
}

func runPlayground(ctx context.Context) {
	cmd := exec.Command("go", "run", "./cmd/playground/")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "playground failed: %v\n", err)
		os.Exit(1)
	}
}

func runGenABI(_ context.Context) {
	abiPath := ""
	outFile := ""
	pkg := "contracts"

	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-abi":
			if i+1 < len(args) {
				abiPath = args[i+1]
				i++
			}
		case "-out":
			if i+1 < len(args) {
				outFile = args[i+1]
				i++
			}
		case "-pkg":
			if i+1 < len(args) {
				pkg = args[i+1]
				i++
			}
		}
	}

	if abiPath == "" || outFile == "" {
		fmt.Fprintf(os.Stderr, "Usage: chainpulse gen-abi -abi <path> -out <path> [-pkg <name>]\n")
		os.Exit(1)
	}

	if err := codegen.GenerateABIFromFile(abiPath, outFile, pkg); err != nil {
		fmt.Fprintf(os.Stderr, "gen-abi: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Generated Go event structs → %s\n", outFile)
}

func runStatus(ctx context.Context) {
	port := "9099"
	if p := os.Getenv("PLAYGROUND_PORT"); p != "" {
		port = p
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%s/stats", port))
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, os.ErrDeadlineExceeded) {
			fmt.Fprintf(os.Stderr, "no instance responding on :%s\n", port)
		} else {
			fmt.Fprintf(os.Stderr, "status check failed: %v\n", err)
		}
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var pretty map[string]any
	if err := json.Unmarshal(body, &pretty); err == nil {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(pretty)
	} else {
		fmt.Println(string(body))
	}
}
