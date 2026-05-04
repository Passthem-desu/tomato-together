# Makefile for tomatogether/backend

# Binary output directory
BIN_DIR := bin

# Binary name
BINARY_NAME := tomatogether

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOMOD := $(GOCMD) mod

# Build flags
LDFLAGS := -ldflags "-s -w"

# Source files
MAIN_FILE := main.go

.PHONY: all build clean run test tidy migrate help

# Default target
all: build

## build: Build the binary to bin/ directory
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) $(MAIN_FILE)
	@echo "Build complete: $(BIN_DIR)/$(BINARY_NAME)"

## run: Build and run the binary
run: build
	@echo "Starting $(BINARY_NAME)..."
	./$(BIN_DIR)/$(BINARY_NAME)

## clean: Remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	rm -rf $(BIN_DIR)
	@echo "Clean complete"

## test: Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

## tidy: Tidy go modules
tidy:
	@echo "Tidying go modules..."
	$(GOMOD) tidy

## migrate: Run database migrations (placeholder)
migrate:
	@echo "Running migrations..."
	@echo "Note: Migrations are currently handled by the application on startup"

## dev: Build for development (with verbose output)
dev:
	@echo "Building for development..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -v -o $(BIN_DIR)/$(BINARY_NAME) $(MAIN_FILE)
	@echo "Dev build complete"

## help: Show this help message
help:
	@echo "TomatoTogether Backend Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-10s %s\n", $$1, $$2}' $(MAKEFILE_LIST)