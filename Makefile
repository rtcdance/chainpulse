# Development and Build Tooling
.PHONY: all build test lint fmt clean help check-policy-contract check-migration-manifest export-migration-kpi compare-migration-kpi compare-ticket-registry-health smoke-baseline-governance-scope compare-baseline-scope-smoke preflight-migration-baseline-update test-baseline-update-resolver compare-baseline-resolver-test check-migration-baseline update-migration-baseline check-migration-changelog-quality export-migration-owner-drift

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
TESTFLAGS := -v -race
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
	$(GO) test $(TESTFLAGS) -tags=integration ./test/integration/...

test-e2e:
	@echo "Running e2e tests..."
	$(GO) test $(TESTFLAGS) -tags=e2e ./test/e2e/...

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

lint:
	@echo "Running linter..."
	@GOROOT= $(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8 2>/dev/null; true
	GOCACHE=$${GOCACHE:-/tmp/chainpulse-go-build-cache} golangci-lint run --tests=false $(LINT_DIRS)

lint-fix:
	@echo "Running linter with auto-fix..."
	GOCACHE=$${GOCACHE:-/tmp/chainpulse-go-build-cache} golangci-lint run --tests=false --fix $(LINT_DIRS)

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

docker-up:
	@echo "Starting Docker services..."
	docker-compose -f docker/docker-compose.dev.yml up -d

docker-down:
	@echo "Stopping Docker services..."
	docker-compose -f docker/docker-compose.dev.yml down

docker-logs:
	docker-compose -f docker/docker-compose.dev.yml logs -f

# ========== CI/CD ==========

ci: fmt-check lint vet test-unit

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
	@echo "  test-integration     - Run integration tests"
	@echo "  test-e2e             - Run end-to-end tests"
	@echo "  test-coverage        - Run tests with coverage report"
	@echo "  test-race            - Run tests with race detector"
	@echo "  test-bench           - Run benchmarks"
	@echo ""
	@echo "Code Quality:"
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
	@echo "  clean                - Clean build artifacts"
	@echo "  help                 - Show this help"
