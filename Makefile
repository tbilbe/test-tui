.PHONY: build test lint fmt clean run release-local

# Build the binary
build:
	go build -o seven-test-tui ./cmd/main.go

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run linter
lint:
	go vet ./...
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "Code is not formatted. Run 'make fmt'"; \
		gofmt -d .; \
		exit 1; \
	fi

# Format code
fmt:
	go fmt ./...

# Clean build artifacts
clean:
	rm -f seven-test-tui
	rm -rf dist/
	rm -f coverage.out coverage.html
	rm -f *.log

# Run the application
run: build
	./seven-test-tui

# Build for all platforms (local release testing)
release-local:
	@mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/seven-test-tui-linux-amd64 ./cmd/main.go
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o dist/seven-test-tui-linux-arm64 ./cmd/main.go
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o dist/seven-test-tui-darwin-amd64 ./cmd/main.go
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o dist/seven-test-tui-darwin-arm64 ./cmd/main.go
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/seven-test-tui-windows-amd64.exe ./cmd/main.go
	@echo "Binaries built in dist/"

# Show help
help:
	@echo "Available targets:"
	@echo "  build          - Build the binary"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  lint           - Run linter and format check"
	@echo "  fmt            - Format code"
	@echo "  clean          - Remove build artifacts"
	@echo "  run            - Build and run the application"
	@echo "  release-local  - Build binaries for all platforms"
