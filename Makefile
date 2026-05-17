# Development and Build Tooling
.PHONY: all build test lint fmt clean help repo-hygiene k8s-verify k8s-acceptance k8s-acceptance-strict k8s-up k8s-down k8s-status k8s-oneclick deploy-event-acceptance multichain-e2e-acceptance multichain-e2e-acceptance-strict multichain-e2e-fork-acceptance multichain-e2e-fork-acceptance-strict deployed-real-event-acceptance check-policy-contract check-migration-manifest export-migration-kpi compare-migration-kpi compare-ticket-registry-health smoke-baseline-governance-scope compare-baseline-scope-smoke preflight-migration-baseline-update test-baseline-update-resolver compare-baseline-resolver-test check-migration-baseline update-migration-baseline check-migration-changelog-quality export-migration-owner-drift check test-short test-anvil learn benchmark-baseline benchmark-regression

# Force-clear stale GOROOT (homebrew Go self-detects; stale value breaks builds)
unexport GOROOT
# Ensure Go-installed tools are on PATH (computed at parse time with clean GOROOT)
GOPATH_BIN := $(shell GOROOT= go env GOPATH)/bin
export PATH := $(GOPATH_BIN):$(PATH)

# 变量
PROJECT_NAME := chainpulse
MODULE := chainpulse
BUILD_DIR := ./build
BIN_DIR := $(BUILD_DIR)/bin
COVERAGE_DIR := $(BUILD_DIR)/coverage
CMD_DIR := ./cmd

# 版本信息
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)"

# Go 命令
GO := go
GOFLAGS := -v
TESTFLAGS := -v -race -timeout 5m
LINT_BASE_REF ?=
LINT_CHANGED_FILES := $(shell if [ -n "$(LINT_BASE_REF)" ]; then git diff --name-only --diff-filter=ACMRTUXB "$(LINT_BASE_REF)"...HEAD -- '*.go' 2>/dev/null; else { git diff --name-only --diff-filter=ACMRTUXB -- '*.go' 2>/dev/null; git ls-files --others --exclude-standard -- '*.go' 2>/dev/null; }; fi | sort -u)
LINT_DIRS := $(shell if [ -n "$(LINT_CHANGED_FILES)" ]; then printf '%s\n' $(LINT_CHANGED_FILES) | xargs -n1 dirname | sort -u; fi)

# 默认目标
all: fmt lint test build

# ========== 构建 ==========

build: build-monolithic build-microservices

build-monolithic:
	@echo "Building monolithic binary..."
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BIN_DIR)/$(PROJECT_NAME) $(CMD_DIR)/monolithic/chainpulse

build-microservices:
	@echo "Building microservices..."
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BIN_DIR)/puller $(CMD_DIR)/microservices/puller
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BIN_DIR)/indexer $(CMD_DIR)/microservices/event-processor
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BIN_DIR)/query $(CMD_DIR)/microservices/api-service
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BIN_DIR)/gateway $(CMD_DIR)/microservices/api-gateway

build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(BIN_DIR)/linux
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BIN_DIR)/linux/$(PROJECT_NAME) $(CMD_DIR)/monolithic/chainpulse

# ========== 测试 ==========

test: test-unit test-integration test-e2e

test-unit:
	@echo "Running unit tests..."
	$(GO) test $(TESTFLAGS) -short ./pkg/...

test-integration:
	@echo "Running integration tests..."
	$(GO) test $(TESTFLAGS) ./test/integration/...

test-e2e:
	@echo "Running e2e tests..."
	$(GO) test $(TESTFLAGS) ./test/e2e/...

test-anvil: ## Run Anvil-based integration tests (requires foundryup)
	@echo "Running Anvil integration tests..."
	@which anvil > /dev/null 2>&1 || { echo "Installing Foundry..."; curl -L https://foundry.paradigm.xyz | bash 2>/dev/null && $(GOPATH_BIN)/foundryup; }
	$(GO) test $(TESTFLAGS) -tags=e2e -run TestAnvil ./test/e2e/...

learn: ## Start Anvil + deploy EventEmitter + emit 9 test events
	bash scripts/learn-chainpulse.sh up

learn-debug: ## learn + launch delve with learning breakpoints
	bash scripts/learn-chainpulse.sh debug

test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	$(GO) test -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	$(GO) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report: $(COVERAGE_DIR)/coverage.html"

test-race:
	@echo "Running tests with race detector..."
	$(GO) test -race ./pkg/...

test-bench:
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./...

# ========== 代码质量 ==========

check: vet fmt-check
	@echo "All checks passed."

test-short:
	@echo "Running short tests (no integration/e2e)..."
	$(GO) test -short -count=1 ./pkg/... ./cmd/...

lint:
	@echo "Running linter..."
	@LINTER_BIN="$$(GOROOT= $(GO) env GOBIN)"; \
	if [ -n "$$LINTER_BIN" ]; then \
		LINTER_BIN="$$LINTER_BIN/golangci-lint"; \
	else \
		LINTER_BIN="$$(GOROOT= $(GO) env GOPATH)/bin/golangci-lint"; \
	fi; \
	GOROOT= $(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8 2>/dev/null; true; \
	if [ ! -x "$$LINTER_BIN" ]; then \
		echo "golangci-lint v1.64.8 is not installed at $$LINTER_BIN"; \
		exit 127; \
	fi; \
	LINT_ARGS="--tests=false"; \
	if [ -n "$(LINT_BASE_REF)" ]; then \
		LINT_ARGS="$$LINT_ARGS --new-from-rev=$(LINT_BASE_REF)"; \
	elif git rev-parse HEAD~1 >/dev/null 2>&1; then \
		LINT_ARGS="$$LINT_ARGS --new-from-rev=HEAD~1"; \
	fi; \
	GOCACHE=$${GOCACHE:-/tmp/chainpulse-go-build-cache} "$$LINTER_BIN" run $$LINT_ARGS ./...

lint-fix:
	@echo "Running linter with auto-fix..."
	@LINTER_BIN="$$(GOROOT= $(GO) env GOBIN)"; \
	if [ -n "$$LINTER_BIN" ]; then \
		LINTER_BIN="$$LINTER_BIN/golangci-lint"; \
	else \
		LINTER_BIN="$$(GOROOT= $(GO) env GOPATH)/bin/golangci-lint"; \
	fi; \
	if [ ! -x "$$LINTER_BIN" ]; then \
		echo "golangci-lint v1.64.8 is not installed at $$LINTER_BIN"; \
		exit 127; \
	fi; \
	GOCACHE=$${GOCACHE:-/tmp/chainpulse-go-build-cache} "$$LINTER_BIN" run --tests=false --fix $(LINT_DIRS)

vet:
	@echo "Running go vet..."
	$(GO) vet ./...

staticcheck:
	@echo "Running staticcheck..."
	@GOROOT= $(GO) install honnef.co/go/tools/cmd/staticcheck@latest 2>/dev/null; true
	staticcheck ./...

fmt:
	@echo "Formatting code..."
	@GOROOT= $(GO) install mvdan.cc/gofumpt@latest 2>/dev/null; true
	gofumpt -w .
	$(GO) mod tidy

fmt-check:
	@echo "Checking code formatting..."
	@GOROOT= $(GO) install mvdan.cc/gofumpt@latest 2>/dev/null; true
	@if [ -n "$$(gofumpt -l .)" ]; then \
		echo "Code is not formatted. Run 'make fmt' to fix."; \
		gofumpt -l .; \
		exit 1; \
	fi

# ========== 依赖管理 ==========

deps:
	@echo "Downloading dependencies..."
	$(GO) mod download

deps-update:
	@echo "Updating dependencies..."
	$(GO) get -u ./...
	$(GO) mod tidy

deps-vendor:
	@echo "Vendoring dependencies..."
	$(GO) mod vendor

# ========== 代码生成 ==========

generate:
	@echo "Running code generation..."
	# Note: mockgen requires Go toolchain matching the project (go1.25+).
	# Install: go install go.uber.org/mock/mockgen@latest
	# Then remove manual mocks from test files before running.
	$(GO) generate ./...

proto:
	@echo "Generating protobuf code..."
	@which protoc-gen-go > /dev/null || $(GO) install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@which protoc-gen-go-grpc > /dev/null || $(GO) install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	protoc --go_out=. --go-grpc_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative pkg/plugins/api/proto/*.proto

wire:
	@echo "Running wire..."
	@GOROOT= $(GO) install github.com/google/wire/cmd/wire@latest 2>/dev/null; true
	wire ./...

# ========== 安全扫描 ==========

security:
	@echo "Running security scan..."
	@GOROOT= $(GO) install github.com/securego/gosec/v2/cmd/gosec@latest 2>/dev/null; true
	gosec -fmt sarif -out $(BUILD_DIR)/gosec.sarif ./...
	gosec ./...

govulncheck:
	@echo "Running Go vulnerability scan..."
	@GOROOT= $(GO) install golang.org/x/vuln/cmd/govulncheck@latest 2>/dev/null; true
	govulncheck ./...

bench:
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem -benchtime=1x -count=1 ./pkg/...

benchstat:
	@echo "Running benchmarks with benchstat..."
	@GOROOT= $(GO) install golang.org/x/perf/cmd/benchstat@latest 2>/dev/null; true
	$(GO) test -bench=. -benchmem -benchtime=1x -count=5 -timeout 300s ./pkg/core/... 2>&1 | tee /tmp/bench-output.txt
	benchstat /tmp/bench-output.txt

BENCHMARK_DIR := $(BUILD_DIR)/benchmark
BENCHMARK_BASELINE := $(BENCHMARK_DIR)/baseline.txt

benchmark-baseline:
	@echo "Saving benchmark baseline..."
	@mkdir -p $(BENCHMARK_DIR)
	$(GO) test -bench=. -benchmem -benchtime=1x -count=5 -timeout 300s ./pkg/core/... 2>&1 | tee $(BENCHMARK_BASELINE)
	@echo "Baseline saved to $(BENCHMARK_BASELINE)"

benchmark-regression:
	@echo "Running benchmark regression check..."
	@if [ ! -f "$(BENCHMARK_BASELINE)" ]; then \
		echo "No baseline found. Run 'make benchmark-baseline' first to create one."; \
		exit 1; \
	fi
	@GOROOT= $(GO) install golang.org/x/perf/cmd/benchstat@latest 2>/dev/null; true
	@mkdir -p $(BENCHMARK_DIR)
	$(GO) test -bench=. -benchmem -benchtime=1x -count=5 -timeout 300s ./pkg/core/... 2>&1 | tee /tmp/bench-current.txt
	@echo ""
	@echo "=== Benchmark Comparison (baseline vs current) ==="
	@-benchstat $(BENCHMARK_BASELINE) /tmp/bench-current.txt 2>/dev/null | tee /tmp/bench-delta.txt
	@echo ""
	@regression=0; \
	while IFS= read -r line; do \
		if echo "$$line" | grep -qE '[0-9]+\\.[0-9]+%'; then \
			pct=$$(echo "$$line" | grep -oE '[0-9]+\\.[0-9]+%' | head -1 | tr -d '%'); \
			if [ -n "$$pct" ] && (( $$(echo "$$pct > 15" | bc -l 2>/dev/null || echo 0) )); then \
				plus=$$(echo "$$line" | grep -oE '\+[0-9]+' | head -1); \
				if [ -n "$$plus" ]; then \
					echo "REGRESSION DETECTED: $$line"; \
					regression=1; \
				fi; \
			fi; \
		fi; \
	done < /tmp/bench-delta.txt; \
	if [ "$$regression" -eq 1 ]; then \
		echo "Benchmark regression check FAILED - performance degradation detected."; \
		exit 1; \
	else \
		echo "Benchmark regression check PASSED - no significant degradation detected."; \
	fi

pprof-cpu:
	@echo "Collecting 30s CPU profile from http://localhost:8080..."
	curl -s -o /tmp/cpu.prof "http://localhost:8080/debug/pprof/profile?seconds=30"
	@echo "Saved to /tmp/cpu.prof. View with: go tool pprof /tmp/cpu.prof"

pprof-heap:
	@echo "Collecting heap profile from http://localhost:8080..."
	curl -s -o /tmp/heap.prof "http://localhost:8080/debug/pprof/heap"
	@echo "Saved to /tmp/heap.prof. View with: go tool pprof /tmp/heap.prof"

# ========== 本地开发 ==========

run-monolithic:
	@echo "Running monolithic mode..."
	DEPLOYMENT_MODE=monolithic $(GO) run $(CMD_DIR)/monolithic/chainpulse

run-puller:
	@echo "Running puller service..."
	DEPLOYMENT_MODE=microservice $(GO) run $(CMD_DIR)/microservices/puller

run-indexer:
	@echo "Running indexer service..."
	DEPLOYMENT_MODE=microservice $(GO) run $(CMD_DIR)/microservices/event-processor

dev-setup:
	@echo "Setting up development environment..."
	@which air > /dev/null || $(GO) install github.com/air-verse/air@latest
	@echo "Installing tools..."
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8
	$(GO) install honnef.co/go/tools/cmd/staticcheck@latest
	$(GO) install mvdan.cc/gofumpt@latest
	$(GO) install github.com/securego/gosec/v2/cmd/gosec@latest

# ========== Claude CLI 工具 ==========

claude-profile:
	@echo "应用 Claude CLI 插件 Profile..."
	@PROFILE=$${PROFILE:-go-backend}; \
	./scripts/claude-plugin-profile.sh apply $$PROFILE

claude-audit:
	@echo "Claude CLI Token 审计..."
	@PROFILE=$${PROFILE:-go-backend} ./scripts/claude-plugin-stats.sh

docker-up:
	@echo "Starting Docker services..."
	docker-compose -f docker/docker-compose.dev.yml up -d

docker-down:
	@echo "Stopping Docker services..."
	docker-compose -f docker/docker-compose.dev.yml down

docker-logs:
	docker-compose -f docker/docker-compose.dev.yml logs -f

# ========== CI/CD ==========

ci: check test-short lint k8s-verify

ci-full: check test-unit lint security k8s-verify
	@echo "Full CI pipeline completed."

# ========== Database Migration ==========

migrate-up:
	@echo "Running database migrations (up)..."
	$(GO) run ./cmd/migrate -db "$${DATABASE_URL}" -path migrations up

migrate-down:
	@echo "Rolling back last migration..."
	$(GO) run ./cmd/migrate -db "$${DATABASE_URL}" -path migrations down

migrate-version:
	@echo "Checking migration version..."
	$(GO) run ./cmd/migrate -db "$${DATABASE_URL}" -path migrations version

repo-hygiene:
	@echo "Checking repository file/structure hygiene..."
	./scripts/check-file-organization.sh

k8s-verify:
	@echo "Verifying Kubernetes deployment capability (static)..."
	MODE=static ./scripts/verify-k8s-deployment-capability.sh

k8s-acceptance:
	@echo "Running Kubernetes deployment acceptance (auto)..."
	MODE=auto ./scripts/verify-k8s-deployment-capability.sh

k8s-acceptance-strict:
	@echo "Running Kubernetes deployment acceptance (strict dry-run)..."
	STRICT_CLUSTER_DRY_RUN=1 MODE=auto ./scripts/verify-k8s-deployment-capability.sh

k8s-up:
	@echo "One-click K8s deploy up (default overlay: microservice)..."
	./scripts/run-k8s-deploy.sh up

k8s-down:
	@echo "One-click K8s deploy down (default overlay: microservice)..."
	./scripts/run-k8s-deploy.sh down

k8s-status:
	@echo "Showing K8s runtime status..."
	./scripts/run-k8s-deploy.sh status

k8s-oneclick:
	@echo "Running one-click K8s deploy + acceptance + status..."
	./scripts/run-k8s-deploy.sh all

deploy-event-acceptance:
	@echo "Running deploy -> real event -> API/H5 acceptance..."
	bash scripts/run-deploy-event-acceptance.sh all

multichain-e2e-acceptance:
	@echo "Running multi-chain E2E acceptance (auto)..."
	MODE=auto bash scripts/multi-chain-e2e.sh

multichain-e2e-acceptance-strict:
	@echo "Running multi-chain E2E acceptance (strict, require Solana)..."
	MODE=strict bash scripts/multi-chain-e2e.sh

multichain-e2e-fork-acceptance:
	@echo "Running multi-chain E2E acceptance in EVM fork mode..."
	EVM_FORK_MODE=1 MODE=auto bash scripts/multi-chain-e2e.sh

multichain-e2e-fork-acceptance-strict:
	@echo "Running strict multi-chain E2E acceptance in EVM fork mode..."
	EVM_FORK_MODE=1 MODE=strict bash scripts/multi-chain-e2e.sh

deployed-real-event-acceptance:
	@echo "Running deployed real on-chain event acceptance..."
	bash scripts/run-deployed-real-event-acceptance.sh

check-policy-contract:
	@echo "Checking policy metric/tag contract..."
	./scripts/check-policy-metric-contract.sh

check-migration-manifest:
	@echo "Checking migration manifest deadlines..."
	./scripts/check-migration-manifest.sh

export-migration-kpi:
	@echo "Exporting migration governance KPI snapshot..."
	./scripts/export-migration-governance-kpi.sh

compare-migration-kpi:
	@echo "Comparing migration governance KPI against baseline..."
	./scripts/compare-migration-governance-kpi.sh

compare-ticket-registry-health:
	@echo "Comparing ticket registry health against baseline..."
	./scripts/compare-ticket-registry-health.sh

smoke-baseline-governance-scope:
	@echo "Running baseline governance scope smoke tests..."
	./scripts/smoke-baseline-governance-scope.sh

compare-baseline-scope-smoke:
	@echo "Comparing baseline scope smoke metrics against baseline..."
	./scripts/compare-baseline-scope-smoke.sh

preflight-migration-baseline-update:
	@echo "Running baseline update preflight..."
	./scripts/preflight-migration-baseline-update.sh

test-baseline-update-resolver:
	@echo "Running baseline update resolver tests..."
	./scripts/test-baseline-update-resolver.sh

compare-baseline-resolver-test:
	@echo "Comparing baseline resolver tests against baseline..."
	./scripts/compare-baseline-resolver-test.sh

check-migration-baseline:
	@echo "Checking migration KPI baseline governance..."
	./scripts/check-migration-baseline-governance.sh

check-migration-changelog-quality:
	@echo "Checking migration KPI changelog entry quality..."
	./scripts/check-migration-changelog-quality.sh

export-migration-owner-drift:
	@echo "Exporting migration owner drift report..."
	./scripts/export-migration-owner-drift-report.sh

update-migration-baseline:
	@echo "Updating migration KPI baseline (guarded)..."
	./scripts/update-migration-governance-baseline.sh

cd: build test-coverage

# ========== 文档 ==========

doc:
	@echo "Generating documentation..."
	@which godoc > /dev/null || $(GO) install golang.org/x/tools/cmd/godoc@latest
	godoc -http=:6060 &
	@echo "Documentation server started at http://localhost:6060"

# ========== 清理 ==========

clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -rf vendor/
	$(GO) clean -cache

# ========== 帮助 ==========

help:
	@echo "ChainPulse Makefile Targets"
	@echo ""
	@echo "Build:"
	@echo "  build                - Build all binaries"
	@echo "  build-monolithic     - Build monolithic binary only"
	@echo "  build-microservices  - Build microservice binaries"
	@echo "  build-linux          - Build for Linux (amd64)"
	@echo ""
	@echo "Test:"
	@echo "  test                 - Run all tests"
	@echo "  test-unit            - Run unit tests only"
	@echo "  test-short           - Run short tests (no integration/e2e)"
	@echo "  test-integration     - Run integration tests"
	@echo "  test-e2e             - Run end-to-end tests"
	@echo "  test-anvil           - Run Anvil integration tests (requires foundryup)"
	@echo "  test-coverage        - Run tests with coverage report"
	@echo "  test-race            - Run tests with race detector"
	@echo "  test-bench           - Run benchmarks"
	@echo ""
	@echo "Code Quality:"
	@echo "  check                - Run vet + fmt-check (fast pre-push gate)"
	@echo "  lint                 - Run golangci-lint"
	@echo "  lint-fix             - Run linter with auto-fix"
	@echo "  vet                  - Run go vet"
	@echo "  staticcheck          - Run staticcheck"
	@echo "  fmt                  - Format code with gofumpt"
	@echo "  fmt-check            - Check code formatting"
	@echo ""
	@echo "Development:"
	@echo "  run-monolithic       - Run monolithic mode"
	@echo "  run-puller           - Run puller service"
	@echo "  run-indexer          - Run indexer service"
	@echo "  dev-setup            - Install development tools"
	@echo "  docker-up            - Start Docker services"
	@echo "  docker-down          - Stop Docker services"
	@echo "  docker-logs          - View Docker logs"
	@echo ""
	@echo "Code Generation:"
	@echo "  generate             - Run go generate"
	@echo "  proto                - Generate protobuf code"
	@echo "  wire                 - Run wire dependency injection"
	@echo ""
	@echo "Other:"
	@echo "  deps                 - Download dependencies"
	@echo "  deps-update          - Update dependencies"
	@echo "  security             - Run security scan"
	@echo "  ci                   - Run CI checks"
	@echo "  repo-hygiene         - Run repository hygiene checks"
	@echo "  k8s-verify           - Verify k8s capability (static checks)"
	@echo "  k8s-acceptance       - Run k8s acceptance (auto + dry-run when available)"
	@echo "  k8s-acceptance-strict - Run k8s acceptance (require kubectl dry-run)"
	@echo "  k8s-up               - One-click K8s deploy up (overlay via OVERLAY=...)"
	@echo "  k8s-down             - One-click K8s deploy down (overlay via OVERLAY=...)"
	@echo "  k8s-status           - Show K8s pods/services/deployments"
	@echo "  k8s-oneclick         - Run K8s up + acceptance + status"
	@echo "  deploy-event-acceptance - Deploy, inject a real event, then run API/H5 acceptance"
	@echo ""
	@echo "Claude CLI:"
	@echo "  claude-profile       - Apply Claude CLI plugin Profile (PROFILE=go-backend)"
	@echo "  claude-audit         - Show Claude CLI plugin token audit"
	@echo "  multichain-e2e-acceptance - Run multi-chain E2E acceptance (multi-EVM + optional Solana)"
	@echo "  multichain-e2e-acceptance-strict - Run strict multi-chain E2E acceptance (require Solana)"
	@echo "  multichain-e2e-fork-acceptance - Run multi-chain E2E with EVM fork mode"
	@echo "  multichain-e2e-fork-acceptance-strict - Run strict multi-chain E2E with EVM fork mode"
	@echo "  deployed-real-event-acceptance - Inject a real chain event and verify deployed visibility"
	@echo "  clean                - Clean build artifacts"
	@echo "  learn                - Start Anvil + deploy contract + emit events"
	@echo "  learn-debug          - learn + launch delve with learning breakpoints"
	@echo "  help                 - Show this help"
