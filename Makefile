.PHONY: build test clean run-example install

# Build the project
build:
	go build ./...

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	go clean
	rm -rf coverage.out coverage.html
	rm -rf uploads/
	rm -rf build/

# Install dependencies
install:
	go mod download
	go mod tidy

# Run the example server
run-example:
	@echo "Starting GoKit example server..."
	@echo "Visit http://localhost:8080 for the demo"
	@echo "API endpoints:"
	@echo "  POST /api/register - User registration with validation"
	@echo "  POST /api/upload-avatar - File upload demo"
	@echo "  GET  /api/greeting - i18n demo"
	@echo ""
	go run examples/main.go

# Build the example
build-example:
	go build -o build/example examples/main.go

# Run linter
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...

# Generate documentation
docs:
	godoc -http=:6060

# Create uploads directory
setup:
	mkdir -p uploads
	mkdir -p build

# All-in-one setup and run
dev: setup install run-example

# Help
help:
	@echo "Available targets:"
	@echo "  build         - Build the project"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  clean         - Clean build artifacts"
	@echo "  install       - Install dependencies"
	@echo "  run-example   - Run the example server"
	@echo "  build-example - Build the example binary"
	@echo "  lint          - Run linter"
	@echo "  fmt           - Format code"
	@echo "  docs          - Start documentation server"
	@echo "  setup         - Create necessary directories"
	@echo "  dev           - Setup, install, and run example"
	@echo "  help          - Show this help" 