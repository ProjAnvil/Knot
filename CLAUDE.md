# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Knot is an API documentation management system with Model Context Protocol (MCP) support. It features a **Svelte 5 frontend** and **Go backend**, supports multiple databases (SQLite, PostgreSQL, MySQL), and includes standalone CLI tools for both backend and MCP server management.

The project is structured as **three independent components**:
- **frontend/**: Svelte 5 application (manages its own dependencies with bun/npm)
- **backend/**: Go backend server
- **mcp-server/**: Go MCP server for AI integration

## Essential Commands

### Frontend Development (Svelte 5 + Vite)
```bash
cd frontend
bun install              # Install dependencies
bun dev                  # Start dev server on port 5173 with hot reload
bun run build            # Build for production to dist/
```

### Backend Development (Go + Chi + GORM)
```bash
cd backend
go mod download          # Download Go dependencies
make build               # Build binary to bin/knot-server
./bin/knot-server        # Run server on port 3000

# Development with auto-reload (using air)
air

# Build for all platforms
make build-all           # Outputs to dist/
make package-all         # Create distribution packages
```

### MCP Server Development (Go)
```bash
cd mcp-server
go mod download          # Download Go dependencies
make build               # Build binary to bin/knot-mcp
./bin/knot-mcp          # Run MCP server

# Build for all platforms
make build-all           # Outputs to dist/
```

### CLI Tool (knot)
The backend includes a CLI tool for server management:
```bash
cd backend
./bin/knot setup         # Initialize config at ~/.knot/config.json
./bin/knot start         # Start server in background
./bin/knot stop          # Stop running server
./bin/knot status        # Check server status and PID
```

## Architecture

### Database Layer
- **ORM**: GORM (Go ORM) with multi-database support
- **Schema**: Defined in `backend/internal/models/`
  - `Group`: API groupings
  - `API`: API endpoints (references groups)
  - `Parameter`: Request/response parameters with nested support (self-referencing for object/array types)
- **Migrations**: Auto-migration via GORM's `AutoMigrate` on startup
- **Database Support**: SQLite, PostgreSQL, MySQL with runtime switching

### Configuration System
- **Location**: `~/.knot/config.json` (Linux/macOS) or `%LOCALAPPDATA%/knot/config.json` (Windows)
- **Management**: `backend/internal/config/` handles all configuration and PID file operations
- **Database Path**: User config directory (`~/.knot/knot.db` by default for SQLite)
- **PID Management**: `~/.knot/knot.pid` tracks running server process

### Frontend Structure (Standalone)
- **Framework**: Svelte 5 + Vite
- **Location**: `frontend/` directory (independent from backend)
- **Package Management**: Has its own `package.json`, `bun.lock`, `node_modules`
- **Build Output**: `frontend/dist/` (copied to `backend/public/` during deployment)
- **UI Components**: Svelte components in `frontend/src/lib/components/`
- **Routing**: Client-side routing with Svelte
- **i18n**: svelte-i18n with messages in `frontend/messages/` directory
- **Development**: Runs independently on port 5173, proxies API calls to backend

### Backend Structure (Go)
- **Framework**: Chi router (lightweight, idiomatic Go HTTP router)
- **Entry Point**: `backend/cmd/server/main.go`
- **API Routes**: Defined in `backend/internal/routes/`
  - `api_routes.go`: API endpoint CRUD operations
  - `group_routes.go`: API group management
  - `export_routes.go`: Data export functionality
- **Database Access**: GORM queries in handlers (`backend/internal/handlers/`)
- **Static File Serving**: Serves Svelte frontend from `backend/public/` using `http.FileServer`
- **Middleware**: CORS, logging, error handling in `backend/internal/middleware/`

### MCP Server (Go)
- **Location**: `mcp-server/` directory (independent component)
- **Entry Point**: `mcp-server/main.go`
- **Purpose**: Provides read-only API documentation access to AI assistants
- **Communication**: Connects to Go backend API via HTTP
- **Protocol**: Implements Model Context Protocol using stdio transport

## Key Implementation Details

### Database Initialization
The backend initializes the database on startup:
- Uses GORM's `AutoMigrate` to create/update tables automatically
- Database connection configured via `backend/internal/config/config.go`
- Supports SQLite, PostgreSQL, and MySQL with runtime switching based on config

### CLI Tool & Process Management
The CLI tool (`backend/cmd/cli/`) manages server lifecycle:
- `setup`: Creates default configuration file
- `start`: Starts server in background, writes PID to `~/.knot/knot.pid`
- `stop`: Sends SIGTERM to process, removes PID file
- `status`: Checks if PID process exists and is running

### Parameter Schema
Parameters support nested structures through self-referencing:
- `ParentID` field creates hierarchical relationships
- Used for object and array types with nested properties
- `ParamType`: "request" or "response"
- `Type`: "string", "number", "boolean", "array", "object"

### Static File Serving
The backend serves static files using Go's standard library:
- Uses `http.FileServer` to serve from `backend/public/`
- Implements SPA fallback routing for client-side routes
- Returns `index.html` for non-API routes
- Proper MIME type detection via `http.DetectContentType`

## Common Development Patterns

### Database Queries
Use GORM in handlers (`backend/internal/handlers/`):
```go
import (
    "github.com/yourusername/knot/internal/models"
    "gorm.io/gorm"
)

// Query with preloading relations
var groups []models.Group
db.Preload("APIs").Find(&groups)
```

### API Route Handlers
Routes are defined using Chi router in `backend/internal/routes/`:
```go
import (
    "github.com/go-chi/chi/v5"
    "net/http"
)

func SetupRoutes(r *chi.Mux, db *gorm.DB) {
    r.Get("/api/groups", handlers.GetGroups(db))
    r.Post("/api/groups", handlers.CreateGroup(db))
}
```

### Configuration Management
Configuration is handled in `backend/internal/config/`:
- `LoadConfig()`: Read configuration from file
- `SaveConfig()`: Write configuration to file
- `InitConfig()`: Create default configuration

## Testing New Features

1. **Frontend Development**:
   ```bash
   cd frontend
   bun dev              # Start at http://localhost:5173
   ```

2. **Backend Development**:
   ```bash
   cd backend
   make build           # Build binary
   ./bin/knot-server    # Start at http://localhost:3000
   ```

3. **Full Stack Development**:
   - Run backend on port 3000
   - Run frontend dev server (proxies API to backend)
   - Access via http://localhost:5173

4. **Production Build**:
   ```bash
   # Build frontend
   cd frontend && bun run build

   # Copy frontend to backend
   cp -r frontend/dist/* backend/public/

   # Build backend
   cd backend && make build

   # Run
   ./bin/knot-server
   ```

5. **Database**: Check `~/.knot/knot.db` for data (SQLite default)

## Important Files

### Backend (Go)
- `backend/cmd/server/main.go`: Server entry point
- `backend/cmd/cli/main.go`: CLI tool entry point
- `backend/internal/models/`: Database models (Group, API, Parameter)
- `backend/internal/handlers/`: HTTP request handlers
- `backend/internal/routes/`: Route definitions
- `backend/internal/config/`: Configuration management
- `backend/internal/middleware/`: HTTP middleware
- `backend/Makefile`: Build commands

### Frontend (Svelte 5)
- `frontend/src/App.svelte`: Root component
- `frontend/src/lib/components/`: Reusable components
- `frontend/src/lib/`: Utilities and stores
- `frontend/package.json`: Frontend dependencies (independent)
- `frontend/vite.config.ts`: Vite build configuration

### MCP Server (Go)
- `mcp-server/main.go`: MCP server entry point
- `mcp-server/Makefile`: Build commands
