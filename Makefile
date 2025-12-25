# Makefile for chainpulse Web3 Service

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=chainpulse-api
BINARY_UNIX=$(BINARY_NAME)_unix

# Docker parameters
DOCKER_IMAGE=chainpulse/api
DOCKER_REGISTRY=ghcr.io

# Anvil parameters
ANVIL_RPC_URL ?= http://localhost:8545
ANVIL_CHAIN_ID ?= 31337
ANVIL_PRIVATE_KEY ?= 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80

# Test parameters
TEST_PATH=./...
INTEGRATION_TEST_PATH=./test/anvil/integration/...

# Build the main API service
build:
	$(GOBUILD) -o bin/$(BINARY_NAME) -v cmd/api/main.go

# Build all services
build-all: build-api build-blockchain build-indexer build-event-processor

# Build individual services
build-api:
	$(GOBUILD) -o bin/chainpulse-api -v cmd/api/main.go

build-blockchain:
	$(GOBUILD) -o bin/chainpulse-blockchain -v cmd/blockchain/main.go

build-indexer:
	$(GOBUILD) -o bin/chainpulse-indexer -v cmd/indexer/main.go

build-event-processor:
	$(GOBUILD) -o bin/chainpulse-event-processor -v cmd/event-processor/main.go

# Run the application
run:
	$(GOBUILD) -o bin/$(BINARY_NAME) -v cmd/api/main.go
	bin/$(BINARY_NAME)

# Run standard tests
test:
	$(GOTEST) -v $(TEST_PATH)

# Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out $(TEST_PATH)
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Run tests with race detection
test-race:
	$(GOTEST) -v -race $(TEST_PATH)

# Anvil-specific targets
.PHONY: anvil-start
anvil-start:
	@echo "Starting Anvil local blockchain..."
	@anvil --host 0.0.0.0 --port 8545 --chain-id 31337 --block-time 1 || (echo "Please install Foundry first: curl -L https://foundry.paradigm.xyz | bash && foundryup" && exit 1)

.PHONY: anvil-test-deps
anvil-test-deps:
	@echo "Checking for Foundry installation..."
	@which anvil > /dev/null || (echo "Error: anvil not found. Please install Foundry first: curl -L https://foundry.paradigm.xyz | bash && foundryup" && exit 1)
	@which forge > /dev/null || (echo "Error: forge not found. Please install Foundry first: curl -L https://foundry.paradigm.xyz | bash && foundryup" && exit 1)
	@echo "Foundry tools are available."

.PHONY: anvil-deploy-contracts
anvil-deploy-contracts: anvil-test-deps
	@echo "Deploying test contracts to Anvil..."
	@cd test/anvil/contracts && forge build
	@echo "Test contracts built successfully."

.PHONY: anvil-run-contract-tests
anvil-run-contract-tests: anvil-test-deps
	@echo "Running contract tests with Anvil..."
	@cd test/anvil/contracts && forge test
	@echo "Contract tests completed."

.PHONY: anvil-run-integration-tests
anvil-run-integration-tests: 
	@echo "Running Anvil integration tests..."
	@ANVIL_RPC_URL=$(ANVIL_RPC_URL) ANVIL_CHAIN_ID=$(ANVIL_CHAIN_ID) ANVIL_PRIVATE_KEY=$(ANVIL_PRIVATE_KEY) $(GOTEST) -v $(INTEGRATION_TEST_PATH)
	@echo "Anvil integration tests completed."

.PHONY: anvil-run-all-tests
anvil-run-all-tests: anvil-test-deps anvil-run-contract-tests anvil-run-integration-tests
	@echo "All Anvil tests completed successfully."

# Run the complete Anvil test environment
.PHONY: anvil-full-test
anvil-full-test:
	@echo "Starting full Anvil test environment..."
	@docker-compose -f test/anvil/config/docker-compose.test.yml up -d --build
	@sleep 10  # Wait for services to start
	@echo "Running integration tests against Anvil environment..."
	@ANVIL_RPC_URL=http://localhost:8545 $(GOTEST) -v $(INTEGRATION_TEST_PATH)
	@echo "Stopping Anvil test environment..."
	@docker-compose -f test/anvil/config/docker-compose.test.yml down

# Docker-related targets
docker-build:
	docker build -t $(DOCKER_IMAGE) -f build/Dockerfile .

docker-build-service:
	docker build --build-arg SERVICE=$(service) -t $(DOCKER_IMAGE)-$(service) -f build/Dockerfile .

docker-compose-up:
	cd build && docker-compose up -d

docker-compose-down:
	cd build && docker-compose down

docker-push:
	docker push $(DOCKER_IMAGE)

# Run all services
.PHONY: run-all-services
run-all-services: build-all
	@echo "Starting all services..."
	@./bin/chainpulse-api &
	@sleep 2
	@./bin/chainpulse-blockchain &
	@sleep 2
	@./bin/chainpulse-indexer &
	@sleep 2
	@./bin/chainpulse-event-processor &
	@echo "All services started. Use 'pkill -f chainpulse-' to stop all services."

# Run tests for all modules
.PHONY: test-modules
test-modules: test test-blockchain test-indexer test-event-processor

.PHONY: test-api
test-api:
	$(GOTEST) -v ./cmd/api/...

.PHONY: test-blockchain
test-blockchain:
	$(GOTEST) -v ./pkg/blockchain/...

.PHONY: test-indexer
test-indexer:
	$(GOTEST) -v ./pkg/service/...

.PHONY: test-event-processor
test-event-processor:
	$(GOTEST) -v ./pkg/service/event_processor.go

# Run all tests (standard + Anvil)
test-all: test anvil-run-all-tests
	@echo "All tests completed successfully."

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f bin/$(BINARY_NAME)
	rm -f coverage.out
	rm -f coverage.html

# Install dependencies
deps:
	$(GOGET) -v ./...

# Generate code (if needed)
generate:
	$(GOCMD) generate ./...

# Run linter (if golangci-lint is installed)
lint:
	@if command -v golangci-lint >/dev/null 2>&1 ; then \
		golangci-lint run ; \
	else \
		echo "golangci-lint not found. Install with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$GOPATH/bin v1.54.2" ; \
	fi

# Run all checks (lint + tests)
check: lint test

# Install development tools
dev-tools:
	@echo "Installing development tools..."
	$(GOGET) -u github.com/golangci/golangci-lint/cmd/golangci-lint
	@echo "Development tools installed."

# Help target
help:
	@echo "chainpulse Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make build                        Build the project"
	@echo "  make run                          Run the application"
	@echo "  make test                         Run standard tests"
	@echo "  make test-coverage               Generate test coverage report"
	@echo "  make test-race                   Run tests with race detection"
	@echo "  make anvil-start                 Start Anvil local blockchain"
	@echo "  make anvil-test-deps             Check for Anvil dependencies"
	@echo "  make anvil-deploy-contracts      Deploy test contracts to Anvil"
	@echo "  make anvil-run-contract-tests    Run contract tests with Anvil"
	@echo "  make anvil-run-integration-tests Run Go integration tests with Anvil"
	@echo "  make anvil-run-all-tests         Run all Anvil tests"
	@echo "  make anvil-full-test             Run full Anvil test environment with Docker"
	@echo "  make test-all                    Run all tests (standard + Anvil)"
	@echo "  make docker-build                Build Docker image"
	@echo "  make clean                       Clean build artifacts"
	@echo "  make deps                        Install dependencies"
	@echo "  make generate                    Generate code"
	@echo "  make lint                        Run linter"
	@echo "  make check                       Run all checks (lint + tests)"
	@echo "  make dev-tools                   Install development tools"
	@echo "  make help                        Show this help message"