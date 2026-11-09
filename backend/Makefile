.PHONY: build build-all clean run test test-coverage deps fmt lint setup package package-all package-cli package-cli-all package-server package-server-all

# Build flags for size optimization
# -s: omit symbol table and debug info
# -w: omit DWARF symbol table
LDFLAGS := -s -w

# Build CLI binary for current platform
build:
	@echo "Building Knot CLI for current platform..."
	@mkdir -p bin
	@go build -ldflags="$(LDFLAGS)" -o bin/knot ./cmd/cli
	@echo ""
	@ls -lh bin/knot | awk '{print "✓ Binary: bin/knot (" $$5 ")"}'

# Build CLI binary for all platforms
build-all:
	@echo "Building Knot CLI for all platforms..."
	@mkdir -p bin
	@echo ""
	@echo "Building Linux AMD64..."
	@GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/knot-linux ./cmd/cli
	@ls -lh bin/knot-linux | awk '{print "✓ bin/knot-linux (" $$5 ")"}'
	@echo ""
	@echo "Building macOS AMD64..."
	@GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/knot-macos-amd64 ./cmd/cli
	@ls -lh bin/knot-macos-amd64 | awk '{print "✓ bin/knot-macos-amd64 (" $$5 ")"}'
	@echo ""
	@echo "Building macOS ARM64..."
	@GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o bin/knot-macos-arm64 ./cmd/cli
	@ls -lh bin/knot-macos-arm64 | awk '{print "✓ bin/knot-macos-arm64 (" $$5 ")"}'
	@echo ""
	@echo "Building Windows AMD64..."
	@GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/knot-windows.exe ./cmd/cli
	@ls -lh bin/knot-windows.exe | awk '{print "✓ bin/knot-windows.exe (" $$5 ")"}'
	@echo ""
	@echo "✅ All CLI binaries built successfully in bin/"

# Run server in development mode
run:
	@go run ./cmd/server/main.go

# Clean build artifacts
clean:
	@rm -rf bin/ internal/embedded/frontend_dist/
	@echo "✓ Cleaned build artifacts (bin/, internal/embedded/frontend_dist/)"

# Run tests
test:
	@go test -v ./...

# Run tests with race detector and coverage, then enforce the 85% gate on
# the export/import services (param_node, export_knot, export_openapi,
# export_postman, import_*). Exits non-zero if any gated file is below 85.0%.
test-coverage:
	@go test ./... -race -covermode=atomic -coverprofile=coverage.out
	@go tool cover -func=coverage.out
	@go tool cover -func=coverage.out | grep -E '(param_node|export_knot|export_openapi|export_postman|import_)' | awk '{gsub(/%/,"",$$NF); if ($$NF+0 < 85.0) { print "COVERAGE GATE FAIL:", $$1, $$NF"%"; f=1 }} END { exit f }'

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

# Setup development environment
setup:
	@go mod download
	@mkdir -p bin
	@echo "✓ Development environment ready"

# Package both CLI (with frontend) and server (current platform)
package: package-cli package-server
	@echo ""
	@echo "✅ Complete package built successfully!"
	@echo "   CLI: bin/knot (frontend embedded)"
	@echo "   Server: bin/knot-server (API only)"
	@echo ""
	@echo "To run: ./bin/knot start"

# Package both CLI and server for all platforms
package-all: package-cli-all package-server-all
	@echo ""
	@echo "✅ All packages built successfully!"
	@echo "   CLI binaries (with embedded frontend): bin/knot-*"
	@echo "   Server binaries (API only): bin/knot-server-*"

# Package CLI with embedded frontend (current platform)
package-cli:
	@echo "📦 Building CLI with embedded frontend for current platform..."
	@echo ""
	@echo "Step 1/3: Building frontend..."
	@cd ../frontend && bun run build
	@echo ""
	@echo "Step 2/3: Copying frontend to internal/embedded/frontend_dist..."
	@rm -rf internal/embedded/frontend_dist
	@mkdir -p internal/embedded/frontend_dist
	@cp -r ../frontend/dist/* internal/embedded/frontend_dist/
	@echo "✓ Frontend copied for embedding"
	@echo ""
	@echo "Step 3/3: Building CLI binary (with embedded frontend)..."
	@mkdir -p bin
	@go build -tags embedfrontend -ldflags="$(LDFLAGS)" -o bin/knot ./cmd/cli
	@echo ""
	@ls -lh bin/knot | awk '{print "   Size: " $$5}'
	@echo "✅ CLI package built successfully!"
	@echo "   bin/knot (frontend embedded in binary)"

# Package CLI with embedded frontend (all platforms)
package-cli-all:
	@echo "📦 Building CLI with embedded frontend for all platforms..."
	@echo ""
	@echo "Step 1/3: Building frontend..."
	@cd ../frontend && bun run build
	@echo ""
	@echo "Step 2/3: Copying frontend to internal/embedded/frontend_dist..."
	@rm -rf internal/embedded/frontend_dist
	@mkdir -p internal/embedded/frontend_dist
	@cp -r ../frontend/dist/* internal/embedded/frontend_dist/
	@echo "✓ Frontend copied for embedding"
	@echo ""
	@echo "Step 3/3: Building CLI for all platforms..."
	@mkdir -p bin
	@echo ""
	@echo "Building Linux AMD64..."
	@GOOS=linux GOARCH=amd64 go build -tags embedfrontend -ldflags="$(LDFLAGS)" -o bin/knot-linux ./cmd/cli
	@ls -lh bin/knot-linux | awk '{print "✓ bin/knot-linux (" $$5 ")"}'
	@echo ""
	@echo "Building macOS AMD64..."
	@GOOS=darwin GOARCH=amd64 go build -tags embedfrontend -ldflags="$(LDFLAGS)" -o bin/knot-macos-amd64 ./cmd/cli
	@ls -lh bin/knot-macos-amd64 | awk '{print "✓ bin/knot-macos-amd64 (" $$5 ")"}'
	@echo ""
	@echo "Building macOS ARM64..."
	@GOOS=darwin GOARCH=arm64 go build -tags embedfrontend -ldflags="$(LDFLAGS)" -o bin/knot-macos-arm64 ./cmd/cli
	@ls -lh bin/knot-macos-arm64 | awk '{print "✓ bin/knot-macos-arm64 (" $$5 ")"}'
	@echo ""
	@echo "Building Windows AMD64..."
	@GOOS=windows GOARCH=amd64 go build -tags embedfrontend -ldflags="$(LDFLAGS)" -o bin/knot-windows.exe ./cmd/cli
	@ls -lh bin/knot-windows.exe | awk '{print "✓ bin/knot-windows.exe (" $$5 ")"}'
	@echo ""
	@echo "✅ All CLI packages built successfully in bin/"

# Package server only (current platform)
package-server:
	@echo "📦 Building server binary for current platform..."
	@mkdir -p bin
	@go build -ldflags="$(LDFLAGS)" -o bin/knot-server ./cmd/server
	@echo ""
	@ls -lh bin/knot-server | awk '{print "   Size: " $$5}'
	@echo "✅ Server package built successfully!"
	@echo "   bin/knot-server (API only, no frontend)"

# Package server only (all platforms)
package-server-all:
	@echo "📦 Building server binary for all platforms..."
	@mkdir -p bin
	@echo ""
	@echo "Building Linux AMD64..."
	@GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/knot-server-linux ./cmd/server
	@ls -lh bin/knot-server-linux | awk '{print "✓ bin/knot-server-linux (" $$5 ")"}'
	@echo ""
	@echo "Building macOS AMD64..."
	@GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/knot-server-macos-amd64 ./cmd/server
	@ls -lh bin/knot-server-macos-amd64 | awk '{print "✓ bin/knot-server-macos-amd64 (" $$5 ")"}'
	@echo ""
	@echo "Building macOS ARM64..."
	@GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o bin/knot-server-macos-arm64 ./cmd/server
	@ls -lh bin/knot-server-macos-arm64 | awk '{print "✓ bin/knot-server-macos-arm64 (" $$5 ")"}'
	@echo ""
	@echo "Building Windows AMD64..."
	@GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o bin/knot-server-windows.exe ./cmd/server
	@ls -lh bin/knot-server-windows.exe | awk '{print "✓ bin/knot-server-windows.exe (" $$5 ")"}'
	@echo ""
	@echo "✅ All server packages built successfully in bin/"
