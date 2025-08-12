# Project variables
PROJECT_NAME := volcano-job-analyzer
VERSION ?= v1.0.0
GO_VERSION := 1.21

# Build variables
BUILD_DIR := build
BINARY_NAME := volcano-job-analyzer
PACKAGE := volcano-job-analyzer

# Docker variables
DOCKER_IMAGE := $(PROJECT_NAME):$(VERSION)
DOCKER_REGISTRY ?= your-registry.com

# Go build flags
LDFLAGS := -X main.version=$(VERSION) -X main.buildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
BUILD_FLAGS := -ldflags "$(LDFLAGS)"

# Default target
.PHONY: all
all: clean lint test build

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build          - Build the binary"
	@echo "  build-linux    - Build Linux binary"
	@echo "  build-darwin   - Build macOS binary"
	@echo "  build-windows  - Build Windows binary"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  lint           - Run linters"
	@echo "  format         - Format code"
	@echo "  clean          - Clean build artifacts"
	@echo "  install        - Install binary"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-push    - Push Docker image"
	@echo "  security-scan  - Run security scans"

# Build targets
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

.PHONY: build-linux
build-linux:
	@echo "Building $(BINARY_NAME) for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .

.PHONY: build-darwin
build-darwin:
	@echo "Building $(BINARY_NAME) for macOS..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .

.PHONY: build-windows
build-windows:
	@echo "Building $(BINARY_NAME) for Windows..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .

.PHONY: build-all
build-all: build-linux build-darwin build-windows

# Test targets
.PHONY: test
test:
	@echo "Running tests..."
	go test -v -race ./...

.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(BUILD_DIR)
	go test -v -race -coverprofile=$(BUILD_DIR)/coverage.out ./...
	go tool cover -html=$(BUILD_DIR)/coverage.out -o $(BUILD_DIR)/coverage.html
	@echo "Coverage report generated: $(BUILD_DIR)/coverage.html"

.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	go test -v -tags=integration ./tests/...

.PHONY: benchmark
benchmark:
	@echo "Running benchmarks..."
	go test -v -bench=. -benchmem ./...

# Code quality targets
.PHONY: lint
lint: check-tools
	@echo "Running linters..."
	golangci-lint run

.PHONY: format
format: check-tools
	@echo "Formatting code..."
	gofmt -s -w .
	goimports -w .

.PHONY: vet
vet:
	@echo "Running go vet..."
	go vet ./...

.PHONY: security-scan
security-scan: check-tools
	@echo "Running security scans..."
	gosec ./...
	govulncheck ./...

# Dependencies
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

.PHONY: deps-update
deps-update:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

# Install target
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME)..."
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/

# Docker targets
.PHONY: docker-build
docker-build:
	@echo "Building Docker image $(DOCKER_IMAGE)..."
	docker build -t $(DOCKER_IMAGE) .

.PHONY: docker-push
docker-push: docker-build
	@echo "Pushing Docker image $(DOCKER_IMAGE)..."
	docker tag $(DOCKER_IMAGE) $(DOCKER_REGISTRY)/$(DOCKER_IMAGE)
	docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE)

.PHONY: docker-run
docker-run: docker-build
	@echo "Running Docker container..."
	docker run --rm -it $(DOCKER_IMAGE) --help

# Clean target
.PHONY: clean
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	go clean

# Development targets
.PHONY: dev-setup
dev-setup: check-tools deps
	@echo "Setting up development environment..."
	pre-commit install

.PHONY: generate
generate:
	@echo "Running code generation..."
	go generate ./...

# Release targets
.PHONY: release-check
release-check:
	@echo "Checking release readiness..."
	@if [ -z "$(VERSION)" ]; then echo "VERSION is required"; exit 1; fi
	@if git diff --quiet; then echo "Working directory is clean"; else echo "Working directory has uncommitted changes"; exit 1; fi

.PHONY: release
release: release-check clean lint test build-all
	@echo "Creating release $(VERSION)..."
	@mkdir -p $(BUILD_DIR)/release
	@cp $(BUILD_DIR)/$(BINARY_NAME)-* $(BUILD_DIR)/release/
	@cd $(BUILD_DIR)/release && sha256sum * > checksums.txt
	@echo "Release artifacts created in $(BUILD_DIR)/release/"

# Tool checks
.PHONY: check-tools
check-tools:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not found. Install it with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; exit 1; }
	@command -v goimports >/dev/null 2>&1 || { echo "goimports not found. Install it with: go install golang.org/x/tools/cmd/goimports@latest"; exit 1; }
	@command -v gosec >/dev/null 2>&1 || { echo "gosec not found. Install it with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; exit 1; }
	@command -v govulncheck >/dev/null 2>&1 || { echo "govulncheck not found. Install it with: go install golang.org/x/vuln/cmd/govulncheck@latest"; exit 1; }

# Examples targets
.PHONY: run-examples
run-examples: build
	@echo "Running example analyses..."
	@echo "=== PyTorch Job Example ==="
	./$(BUILD_DIR)/$(BINARY_NAME) analyze examples/pytorch-job.yaml
	@echo ""
	@echo "=== TensorFlow Job Example ==="
	./$(BUILD_DIR)/$(BINARY_NAME) analyze examples/tensorflow-job.yaml
	@echo ""
	@echo "=== Spark Job Example ==="
	./$(BUILD_DIR)/$(BINARY_NAME) analyze examples/spark-job.yaml

.PHONY: validate-examples
validate-examples: build
	@echo "Validating all example jobs..."
	@for file in examples/*.yaml examples/*/*.yaml; do \
		if [ -f "$$file" ]; then \
			echo "Validating $$file..."; \
			./$(BUILD_DIR)/$(BINARY_NAME) validate "$$file" || exit 1; \
		fi \
	done
	@echo "All examples validated successfully!"

.PHONY: demo
demo: build
	@echo "Running interactive demo..."
	@echo "1. Analyzing PyTorch distributed training job..."
	./$(BUILD_DIR)/$(BINARY_NAME) analyze examples/pytorch-job.yaml --output table
	@echo ""
	@echo "2. Analyzing with JSON output..."
	./$(BUILD_DIR)/$(BINARY_NAME) analyze examples/tensorflow-job.yaml --output json
	@echo ""
	@echo "3. Validating job specification..."
	./$(BUILD_DIR)/$(BINARY_NAME) validate examples/complex-jobs/multi-framework-training.yaml

.PHONY: demo-json
demo-json: build
	@echo "Generating JSON outputs for all examples..."
	@mkdir -p $(BUILD_DIR)/demo-outputs
	@for file in examples/*.yaml; do \
		if [ -f "$$file" ]; then \
			output_name=$$(basename "$$file" .yaml); \
			echo "Analyzing $$file -> $(BUILD_DIR)/demo-outputs/$$output_name.json"; \
			./$(BUILD_DIR)/$(BINARY_NAME) analyze "$$file" --output json > "$(BUILD_DIR)/demo-outputs/$$output_name.json"; \
		fi \
	done
	@echo "Demo outputs generated in $(BUILD_DIR)/demo-outputs/"

# Documentation targets
.PHONY: docs
docs:
	@echo "Generating documentation..."
	@mkdir -p docs/api
	godoc -http=:6060 &
	@echo "Documentation server started at http://localhost:6060"
	@echo "Press Ctrl+C to stop"

.PHONY: docs-generate
docs-generate:
	@echo "Generating static documentation..."
	@mkdir -p docs/generated
	go doc -all ./... > docs/generated/api-docs.txt
	@echo "API documentation generated in docs/generated/"

# Local development targets
.PHONY: dev-run
dev-run: build
	@echo "Running analyzer in development mode..."
	./$(BUILD_DIR)/$(BINARY_NAME) --log-level debug analyze examples/pytorch-job.yaml

.PHONY: dev-watch
dev-watch:
	@echo "Starting development watcher..."
	@command -v air >/dev/null 2>&1 || { echo "air not found. Install it with: go install github.com/air-verse/air@latest"; exit 1; }
	air

.PHONY: local-cluster-test
local-cluster-test:
	@echo "Testing with local Kind cluster..."
	@command -v kind >/dev/null 2>&1 || { echo "kind not found. Please install kind first"; exit 1; }
	@if ! kind get clusters | grep -q "volcano-analyzer-test"; then \
		echo "Creating test cluster..."; \
		kind create cluster --name volcano-analyzer-test; \
	fi
	@echo "Installing Volcano..."
	kubectl apply -f https://raw.githubusercontent.com/volcano-sh/volcano/master/installer/volcano-development.yaml --context kind-volcano-analyzer-test
	@echo "Waiting for Volcano to be ready..."
	kubectl wait --for=condition=ready pod -l app=volcano-controller -n volcano-system --timeout=300s --context kind-volcano-analyzer-test
	@echo "Local cluster ready for testing!"

.PHONY: cleanup-cluster
cleanup-cluster:
	@echo "Cleaning up test cluster..."
	@if kind get clusters | grep -q "volcano-analyzer-test"; then \
		kind delete cluster --name volcano-analyzer-test; \
		echo "Test cluster deleted"; \
	else \
		echo "No test cluster found"; \
	fi

# Performance targets
.PHONY: profile-cpu
profile-cpu: build
	@echo "Running CPU profiling..."
	@mkdir -p $(BUILD_DIR)/profiles
	go test -cpuprofile=$(BUILD_DIR)/profiles/cpu.prof -bench=. ./...
	@echo "CPU profile saved to $(BUILD_DIR)/profiles/cpu.prof"
	@echo "View with: go tool pprof $(BUILD_DIR)/profiles/cpu.prof"

.PHONY: profile-mem
profile-mem: build
	@echo "Running memory profiling..."
	@mkdir -p $(BUILD_DIR)/profiles
	go test -memprofile=$(BUILD_DIR)/profiles/mem.prof -bench=. ./...
	@echo "Memory profile saved to $(BUILD_DIR)/profiles/mem.prof"
	@echo "View with: go tool pprof $(BUILD_DIR)/profiles/mem.prof"

# Utility targets
.PHONY: mod-graph
mod-graph:
	@echo "Generating module dependency graph..."
	go mod graph | grep volcano-job-analyzer

.PHONY: licenses
licenses:
	@echo "Checking licenses of dependencies..."
	@command -v go-licenses >/dev/null 2>&1 || { echo "go-licenses not found. Install it with: go install github.com/google/go-licenses@latest"; exit 1; }
	go-licenses report ./...

.PHONY: size-analysis
size-analysis: build
	@echo "Analyzing binary size..."
	@ls -lh $(BUILD_DIR)/$(BINARY_NAME)
	@file $(BUILD_DIR)/$(BINARY_NAME)
	@echo "Binary size analysis:"
	@go tool nm -size $(BUILD_DIR)/$(BINARY_NAME) | head -20

# Quick development commands
.PHONY: quick-test
quick-test:
	@echo "Running quick tests (no race detection)..."
	go test ./...

.PHONY: quick-build
quick-build:
	@echo "Quick build (no optimization)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) .

.PHONY: run-pytorch
run-pytorch: quick-build
	./$(BUILD_DIR)/$(BINARY_NAME) analyze examples/pytorch-job.yaml

.PHONY: run-tensorflow
run-tensorflow: quick-build
	./$(BUILD_DIR)/$(BINARY_NAME) analyze examples/tensorflow-job.yaml

.PHONY: run-complex
run-complex: quick-build
	./$(BUILD_DIR)/$(BINARY_NAME) analyze examples/complex-jobs/multi-framework-training.yaml

# Container development
.PHONY: docker-dev
docker-dev:
	@echo "Starting development container..."
	docker-compose up dev

.PHONY: docker-test
docker-test:
	@echo "Running tests in container..."
	docker-compose run --rm dev make test

# Git hooks
.PHONY: install-hooks
install-hooks:
	@echo "Installing git hooks..."
	@mkdir -p .git/hooks
	@echo '#!/bin/bash\nmake lint' > .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "Git hooks installed"

# Show project info
.PHONY: info
info:
	@echo "Project: $(PROJECT_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Go Version: $(GO_VERSION)"
	@echo "Build Dir: $(BUILD_DIR)"
	@echo "Binary Name: $(BINARY_NAME)"
	@echo "Docker Image: $(DOCKER_IMAGE)"
	@echo ""
	@echo "Available examples:"
	@find examples -name "*.yaml" -type f | sort
	@echo ""
	@echo "Project structure:"
	@find . -type d -name ".*" -prune -o -type d -print | head -20

# Complete clean
.PHONY: deep-clean
deep-clean: clean cleanup-cluster
	@echo "Deep cleaning..."
	@rm -rf vendor/
	@rm -rf .cache/
	@rm -rf docs/generated/
	@go clean -cache
	@go clean -modcache
	@echo "Deep clean completed"

# Development shortcuts
.PHONY: dev
dev: deps format lint test build

.PHONY: ci
ci: deps lint test build-all docker-build

.PHONY: fast
fast: quick-build run-pytorch

# Help with examples
.PHONY: help-examples
help-examples:
	@echo "Example Commands:"
	@echo "  make run-pytorch           - Quick analyze PyTorch job"
	@echo "  make run-tensorflow        - Quick analyze TensorFlow job"
	@echo "  make run-complex           - Analyze complex multi-framework job"
	@echo "  make demo                  - Run interactive demo"
	@echo "  make validate-examples     - Validate all example files"
	@echo "  make demo-json             - Generate JSON outputs for all examples"
	@echo ""
	@echo "Development Commands:"
	@echo "  make dev                   - Full development cycle"
	@echo "  make fast                  - Quick build and test"
	@echo "  make dev-watch            - Start file watcher for development"
	@echo "  make local-cluster-test   - Set up local Kind cluster with Volcano"