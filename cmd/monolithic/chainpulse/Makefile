.PHONY: build build-linux build-macos build-windows clean run test help

# Variables
BINARY_NAME=chainpulse
VERSION=$(shell git describe --tags --always 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Default target
help:
	@echo "ChainPulse Monolithic Executable - Build Targets"
	@echo ""
	@echo "Available targets:"
	@echo "  build              Build executable for current platform"
	@echo "  build-linux        Build executable for Linux (amd64)"
	@echo "  build-macos        Build executable for macOS (amd64)"
	@echo "  build-windows      Build executable for Windows (amd64)"
	@echo "  build-all          Build for all platforms"
	@echo "  clean              Remove built executables"
	@echo "  run                Build and run the executable"
	@echo "  test               Run tests"
	@echo "  help               Show this help message"
	@echo ""
	@echo "Environment variables:"
	@echo "  VERSION            Version string (default: git describe or 'dev')"
	@echo "  BUILD_TIME         Build timestamp (default: current time)"

# Build for current platform
build:
	@echo "Building $(BINARY_NAME) for current platform..."
	@go build $(LDFLAGS) -o $(BINARY_NAME) .
	@echo "Build complete: $(BINARY_NAME)"

# Build for Linux
build-linux:
	@echo "Building $(BINARY_NAME) for Linux (amd64)..."
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux .
	@echo "Build complete: $(BINARY_NAME)-linux"

# Build for macOS
build-macos:
	@echo "Building $(BINARY_NAME) for macOS (amd64)..."
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-macos .
	@echo "Build complete: $(BINARY_NAME)-macos"

# Build for Windows
build-windows:
	@echo "Building $(BINARY_NAME) for Windows (amd64)..."
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME).exe .
	@echo "Build complete: $(BINARY_NAME).exe"

# Build for all platforms
build-all: build-linux build-macos build-windows
	@echo "Build complete for all platforms"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(BINARY_NAME) $(BINARY_NAME)-linux $(BINARY_NAME)-macos $(BINARY_NAME).exe
	@echo "Clean complete"

# Build and run
run: build
	@echo "Running $(BINARY_NAME)..."
	@./$(BINARY_NAME)

# Run tests
test:
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Test complete. Coverage report: coverage.html"

# Build with debug symbols
build-debug:
	@echo "Building $(BINARY_NAME) with debug symbols..."
	@go build -gcflags="all=-N -l" $(LDFLAGS) -o $(BINARY_NAME)-debug .
	@echo "Build complete: $(BINARY_NAME)-debug"

# Build with optimizations
build-release:
	@echo "Building $(BINARY_NAME) with optimizations..."
	@go build -ldflags="-s -w $(LDFLAGS)" -o $(BINARY_NAME)-release .
	@echo "Build complete: $(BINARY_NAME)-release"

# Check code quality
lint:
	@echo "Running linter..."
	@golangci-lint run ./...

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Run static analysis
vet:
	@echo "Running go vet..."
	@go vet ./...

# Show version
version:
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"

# Show binary info
info: build
	@echo "Binary Information:"
	@file $(BINARY_NAME)
	@ls -lh $(BINARY_NAME)
	@echo ""
	@echo "Dependencies:"
	@go list -m all | wc -l
	@echo "modules"
