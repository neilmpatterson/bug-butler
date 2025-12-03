.PHONY: build clean install test run help

# Build the binary
build:
	@echo "Building bug-butler..."
	@mkdir -p bin
	@go build -o bin/bug-butler ./cmd/bug-butler
	@echo "✓ Build complete: bin/bug-butler"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@go clean
	@echo "✓ Clean complete"

# Install to system PATH
install: build
	@echo "Installing bug-butler to /usr/local/bin..."
	@sudo cp bin/bug-butler /usr/local/bin/
	@echo "✓ Installation complete"

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...
	@echo "✓ Tests complete"

# Run the tool with sample config
run: build
	@echo "Running bug-butler..."
	@./bin/bug-butler check

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "✓ Format complete"

# Lint code
lint:
	@echo "Linting code..."
	@golangci-lint run
	@echo "✓ Lint complete"

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	@go mod tidy
	@echo "✓ Tidy complete"

# Show help
help:
	@echo "Bug Butler - Makefile targets:"
	@echo "  make build    - Build the binary to bin/bug-butler"
	@echo "  make clean    - Remove build artifacts"
	@echo "  make install  - Install to /usr/local/bin (requires sudo)"
	@echo "  make test     - Run tests"
	@echo "  make run      - Build and run with config.yaml"
	@echo "  make fmt      - Format code"
	@echo "  make lint     - Run linter (requires golangci-lint)"
	@echo "  make tidy     - Tidy go.mod dependencies"
	@echo "  make help     - Show this help message"
