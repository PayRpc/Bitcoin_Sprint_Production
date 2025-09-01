# Bitcoin Sprint Makefile
# Cross-platform build system for Go + Rust + C++ integration

# Detect OS
ifeq ($(OS),Windows_NT)
    DETECTED_OS := Windows
    EXE_EXT := .exe
    LIB_EXT := .dll
    LIB_PREFIX := 
    CC := clang
    CXX := clang++
    CARGO_TARGET := x86_64-pc-windows-msvc
    GO_BUILD_FLAGS := -ldflags "-extldflags '-static'"
else
    DETECTED_OS := $(shell uname -s)
    EXE_EXT := 
    LIB_EXT := .so
    LIB_PREFIX := lib
    CC := clang
    CXX := clang++
    CARGO_TARGET := x86_64-unknown-linux-gnu
    GO_BUILD_FLAGS := -ldflags "-extldflags '-static'"
endif

# Build Configuration
RUST_DIR := secure/rust
RUST_LIB := $(RUST_DIR)/target/release/$(LIB_PREFIX)securebuffer$(LIB_EXT)
GO_MAIN := cmd/sprintd
BINARY_NAME := bitcoin-sprint$(EXE_EXT)
BUILD_DIR := build
CGO_CFLAGS := -I$(RUST_DIR)/include
CGO_LDFLAGS := -L$(RUST_DIR)/target/release -lsecurebuffer

# Version Information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go Build Flags
GO_LDFLAGS := -X main.Version=$(VERSION) -X main.Commit=$(COMMIT)

.PHONY: all clean rust go test install deps check help

# Default target
all: check rust go

# Help target
help:
	@echo "Bitcoin Sprint Build System"
	@echo "=========================="
	@echo ""
	@echo "Targets:"
	@echo "  all       - Build everything (default)"
	@echo "  rust      - Build Rust SecureBuffer library"
	@echo "  go        - Build Go application"
	@echo "  clean     - Clean all build artifacts"
	@echo "  test      - Run all tests"
	@echo "  install   - Install dependencies"
	@echo "  check     - Check build environment"
	@echo "  demo      - Build demo version"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION   - Version string (default: git tag)"
	@echo "  COMMIT    - Commit hash (default: git rev-parse)"
	@echo "  CC        - C compiler (default: clang)"
	@echo "  CXX       - C++ compiler (default: clang++)"
	@echo ""
	@echo "Examples:"
	@echo "  make all VERSION=1.0.0"
	@echo "  make go CC=gcc"
	@echo "  make clean && make all"

# Check build environment
check:
	@echo "Checking build environment..."
	@echo "OS: $(DETECTED_OS)"
	@command -v rustc >/dev/null 2>&1 || { echo "ERROR: Rust not found! Install from https://rustup.rs/"; exit 1; }
	@command -v cargo >/dev/null 2>&1 || { echo "ERROR: Cargo not found!"; exit 1; }
	@command -v go >/dev/null 2>&1 || { echo "ERROR: Go not found! Install from https://golang.org/"; exit 1; }
	@command -v $(CC) >/dev/null 2>&1 || { echo "ERROR: C compiler ($(CC)) not found!"; exit 1; }
	@rustc --version
	@cargo --version
	@go version
	@$(CC) --version
	@echo "Build environment ready!"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@cd $(RUST_DIR) && cargo fetch
	@cd $(GO_MAIN) && go mod download
	@echo "Dependencies installed!"

# Build Rust SecureBuffer library
rust:
	@echo "Building Rust SecureBuffer library..."
	@cd $(RUST_DIR) && cargo build --release --target $(CARGO_TARGET)
ifeq ($(DETECTED_OS),Windows)
	@echo "Windows artifacts:"
	@ls -la $(RUST_DIR)/target/release/ | grep -E "\.(dll|lib|pdb)$$" || true
else
	@echo "Unix artifacts:"
	@ls -la $(RUST_DIR)/target/release/ | grep -E "\.(so|a)$$" || true
endif
	@echo "Rust build completed!"

# Build Go application
go: rust
	@echo "Building Go application..."
	@mkdir -p $(BUILD_DIR)
	@cd $(GO_MAIN) && \
		CGO_ENABLED=1 \
		CGO_CFLAGS="$(CGO_CFLAGS)" \
		CGO_LDFLAGS="$(CGO_LDFLAGS)" \
		CC=$(CC) \
		go build \
		-ldflags "$(GO_LDFLAGS)" \
		-o ../../$(BUILD_DIR)/$(BINARY_NAME) \
		.
	@echo "Go build completed: $(BUILD_DIR)/$(BINARY_NAME)"

# Build demo version
demo: rust
	@echo "Building demo version..."
	@mkdir -p $(BUILD_DIR)
	@cd $(GO_MAIN) && \
		CGO_ENABLED=1 \
		CGO_CFLAGS="$(CGO_CFLAGS)" \
		CGO_LDFLAGS="$(CGO_LDFLAGS)" \
		CC=$(CC) \
		go build \
		-tags=demo \
		-ldflags "$(GO_LDFLAGS) -X main.Version=$(VERSION)-demo -X main.Commit=demo" \
		-o ../../$(BUILD_DIR)/bitcoin-sprint-demo$(EXE_EXT) \
		.
	@echo "Demo build completed: $(BUILD_DIR)/bitcoin-sprint-demo$(EXE_EXT)"

# Run tests
test: rust
	@echo "Running tests..."
	@cd $(RUST_DIR) && cargo test
	@cd $(GO_MAIN) && \
		CGO_ENABLED=1 \
		CGO_CFLAGS="$(CGO_CFLAGS)" \
		CGO_LDFLAGS="$(CGO_LDFLAGS)" \
		CC=$(CC) \
		go test -v ./...
	@echo "All tests passed!"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@cd $(RUST_DIR) && cargo clean
	@cd $(GO_MAIN) && go clean
	@rm -f bitcoin-sprint$(EXE_EXT) bitcoin-sprint-demo$(EXE_EXT)
	@echo "Clean completed!"

# Install to system
install: all
	@echo "Installing Bitcoin Sprint..."
ifeq ($(DETECTED_OS),Windows)
	@copy $(BUILD_DIR)\$(BINARY_NAME) %USERPROFILE%\AppData\Local\Microsoft\WindowsApps\
else
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
endif
	@echo "Bitcoin Sprint installed!"

# Development shortcuts
dev: clean all
	@echo "Development build complete!"

release: VERSION := $(shell git describe --tags --exact-match 2>/dev/null || echo "v0.0.0")
release: clean all
	@echo "Release build complete: $(VERSION)"
