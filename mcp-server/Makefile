.PHONY: build clean run test build-all

# Build flags for size optimization
# -s: omit symbol table and debug info
# -w: omit DWARF symbol table
LDFLAGS := -s -w

# Build for current platform
build:
	@echo "Building knot-mcp for current platform..."
	@go build -ldflags="$(LDFLAGS)" -o bin/knot-mcp main.go
	@ls -lh bin/knot-mcp | awk '{print "✓ Binary: bin/knot-mcp (" $$5 ")"}'

# Build for all platforms
build-all:
	@echo "Building for all platforms..."
	@mkdir -p bin

	@echo "Building Linux AMD64..."
	@GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/knot-mcp-linux main.go
	@ls -lh bin/knot-mcp-linux | awk '{print "✓ Linux AMD64: bin/knot-mcp-linux (" $$5 ")"}'

	@echo "Building macOS AMD64..."
	@GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/knot-mcp-macos main.go
	@ls -lh bin/knot-mcp-macos | awk '{print "✓ macOS AMD64: bin/knot-mcp-macos (" $$5 ")"}'

	@echo "Building macOS ARM64..."
	@GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o bin/knot-mcp-macos-arm64 main.go
	@ls -lh bin/knot-mcp-macos-arm64 | awk '{print "✓ macOS ARM64: bin/knot-mcp-macos-arm64 (" $$5 ")"}'

	@echo "Building Windows AMD64..."
	@GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/knot-mcp-windows.exe main.go
	@ls -lh bin/knot-mcp-windows.exe | awk '{print "✓ Windows AMD64: bin/knot-mcp-windows.exe (" $$5 ")"}'

	@echo ""
	@echo "✅ All platform binaries built successfully in bin/"

# Run server in development mode
run:
	@echo "Starting MCP server..."
	@go run main.go

# Clean build artifacts
clean:
	@rm -rf bin/ bin/
	@echo "✓ Cleaned build artifacts"

# Run tests
test:
	@go test -v ./...

# Install dependencies
deps:
	@go mod download
	@go mod tidy

# Format code
fmt:
	@go fmt ./...

# Run linter
lint:
	@golangci-lint run

# Install the binary to $GOPATH/bin
install:
	@echo "Installing knot-mcp to GOPATH..."
	@go install
	@echo "✓ Installed to $(shell go env GOPATH)/bin/knot-mcp"
