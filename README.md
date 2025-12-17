# Knot

<div align="center">

A modern, lightweight API documentation management system with AI assistant integration.

[Features](#features) • [Quick Start](#quick-start) • [Documentation](#documentation) • [Development](#development) • [Contributing](#contributing)

**[中文文档](./doc/README_zh.md)** | English

</div>

---

## Overview

Knot is a comprehensive API documentation platform that helps teams organize, document, and share their API specifications. Built with Go and Svelte 5, it offers a fast, intuitive interface with native AI assistant support through Model Context Protocol (MCP).

### Key Features

- 📚 **Organized API Management** - Group and categorize API endpoints with hierarchical structure
- 🔍 **Fuzzy Search** - Quickly find APIs across all groups with intelligent search
- 📝 **Rich Documentation** - Document APIs with markdown, request/response schemas, and examples
- 🎨 **Syntax Highlighting** - Beautiful JSON syntax highlighting with dark mode support
- 🔄 **Drag & Drop Interface** - Intuitive API reordering and organization
- 🌐 **Multilingual** - Built-in support for English and Chinese
- 🗄️ **Flexible Database** - Choose between SQLite, PostgreSQL, or MySQL
- 🤖 **AI Integration** - Native MCP server for Claude and other AI assistants
- ⚡ **High Performance** - Go-powered backend with minimal resource usage
- 🚀 **Zero Dependencies** - Single binary deployment with embedded frontend

## Quick Start

### Installation

Download the latest release for your platform:

```bash
# macOS (Apple Silicon)
curl -LO https://github.com/ProjAnvil/knot/releases/latest/download/knot-macos-arm64
chmod +x knot-macos-arm64
sudo mv knot-macos-arm64 /usr/local/bin/knot

# macOS (Intel)
curl -LO https://github.com/ProjAnvil/knot/releases/latest/download/knot-macos-amd64
chmod +x knot-macos-amd64
sudo mv knot-macos-amd64 /usr/local/bin/knot

# Linux (AMD64)
curl -LO https://github.com/ProjAnvil/knot/releases/latest/download/knot-linux
chmod +x knot-linux
sudo mv knot-linux /usr/local/bin/knot

# Windows (AMD64)
# Download knot-windows.exe from releases page
```

### Usage

```bash
# Initialize configuration
knot setup

# Start the server (runs in background)
knot start

# Check server status
knot status

# Stop the server
knot stop

# View configuration
knot config

# Get help
knot help
```

The web interface will be available at [http://localhost:3000](http://localhost:3000)

## Documentation

- [English Documentation](./README.md) (This file)
- [中文文档](./doc/README_zh.md)
- [MCP Server Setup](./mcp-server/README.md)
- [MCP Usage Guide](./doc/MCP_USAGE_GUIDE.md)
- [Development Guide](./CLAUDE.md)

## Configuration

Knot stores its configuration at:
- **Linux/macOS**: `~/.knot/config.json`
- **Windows**: `%LOCALAPPDATA%\knot\config.json`

Example configuration:

```json
{
  "databaseType": "sqlite",
  "sqlitePath": "/Users/username/.knot/knot.db",
  "port": 3000,
  "host": "localhost",
  "enableLogging": false
}
```

### Database Options

| Database | Use Case | Configuration |
|----------|----------|---------------|
| **SQLite** (default) | Personal use, development | `sqlitePath: "/path/to/knot.db"` |
| **PostgreSQL** | Production, teams | `postgresUrl: "postgresql://..."` |
| **MySQL** | Enterprise | `mysqlUrl: "user:pass@tcp(...)/"` |

## Development

Knot consists of three independent components:

### Prerequisites

- **Go** 1.21 or later
- **Bun** or npm (for frontend)
- **Make** (optional, for build commands)

### Project Structure

```
knot/
├── frontend/          # Svelte 5 web application
│   ├── src/
│   │   ├── lib/      # Reusable components
│   │   └── messages/ # i18n translations
│   └── package.json
├── backend/           # Go API server
│   ├── cmd/          # Entry points (CLI & server)
│   ├── internal/     # Core logic
│   └── Makefile
├── mcp-server/        # MCP server for AI integration
│   ├── main.go
│   └── Makefile
└── doc/              # Documentation
```

### Frontend Development

```bash
cd frontend

# Install dependencies
bun install

# Start dev server with hot reload (port 5173)
bun dev

# Build for production
bun run build
```

The frontend runs independently and proxies API requests to the backend during development.

### Backend Development

```bash
cd backend

# Install Go dependencies
go mod download

# Run in development mode
make run

# Build CLI binary
make build

# Build for all platforms
make build-all

# Package with embedded frontend
make package
```

Available commands:
- `make run` - Run server in development mode
- `make build` - Build CLI binary for current platform
- `make build-all` - Build for all platforms (Linux, macOS, Windows)
- `make package` - Build complete package with embedded frontend
- `make clean` - Clean build artifacts

### MCP Server Development

```bash
cd mcp-server

# Install dependencies
go mod download

# Build MCP server
make build

# Build for all platforms
make build-all
```

### Running Tests

```bash
# Backend tests
cd backend
go test ./...

# Frontend tests (if available)
cd frontend
bun test
```

## MCP Integration

Knot includes a Model Context Protocol server that enables AI assistants like Claude to query your API documentation naturally.

### Features

- List all API groups
- Search APIs by name or endpoint
- Get detailed API documentation
- Generate JSON request/response examples
- Fuzzy matching on group and API names

### Setup

1. Build the MCP server:
```bash
cd mcp-server
make build
```

2. Configure Claude Desktop to use Knot MCP server. See [MCP Usage Guide](./doc/MCP_USAGE_GUIDE.md) for detailed instructions.

3. Start querying your APIs:
```
"Show me all APIs in the user-service group"
"Find APIs related to authentication"
"Generate an example request for the login API"
```

## Architecture

### Technology Stack

**Frontend:**
- Svelte 5 (Latest reactivity model)
- TypeScript
- Vite (Build tool)
- Tailwind CSS
- shadcn-svelte (UI components)
- svelte-i18n (Internationalization)

**Backend:**
- Go 1.21+
- Chi (HTTP router)
- GORM (ORM with multi-database support)
- Cobra (CLI framework)
- Viper (Configuration management)

**MCP Server:**
- Go with MCP SDK
- Stdio transport
- RESTful API integration

### Database Schema

```
groups
  ├── id (primary key)
  ├── name
  └── apis (has many)

apis
  ├── id (primary key)
  ├── group_id (foreign key)
  ├── name
  ├── endpoint
  ├── method (GET/POST/etc)
  ├── type (HTTP/RPC)
  ├── note (markdown)
  └── parameters (has many)

parameters
  ├── id (primary key)
  ├── api_id (foreign key)
  ├── parent_id (self-referencing for nested)
  ├── name
  ├── type (string/number/boolean/array/object)
  ├── param_type (request/response)
  ├── required
  └── description
```

## Building from Source

### Build Complete Package

```bash
# Clone the repository
git clone https://github.com/ProjAnvil/knot.git
cd knot

# Build frontend
cd frontend
bun install
bun run build
cd ..

# Build backend with embedded frontend
cd backend
make package

# Build MCP server
cd ../mcp-server
make build
```

Binaries will be in:
- Backend CLI: `backend/bin/knot`
- Backend server: `backend/bin/knot-server`
- MCP server: `mcp-server/bin/knot-mcp`

### Cross-Platform Builds

```bash
# Build for all platforms
cd backend
make package-all

cd ../mcp-server
make build-all
```

This creates binaries for:
- Linux (AMD64)
- macOS (AMD64 and ARM64)
- Windows (AMD64)

## Contributing

We welcome contributions! Here's how you can help:

### Reporting Issues

- Use the [issue tracker](https://github.com/ProjAnvil/knot/issues)
- Include detailed steps to reproduce
- Provide system information (OS, Go version, etc.)

### Pull Requests

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests and ensure code quality
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to your fork (`git push origin feature/amazing-feature`)
7. Open a Pull Request

### Development Guidelines

- Follow Go best practices and conventions
- Write tests for new features
- Update documentation for user-facing changes
- Keep commits atomic and well-described
- Ensure all tests pass before submitting PR

## Roadmap

- [ ] OpenAPI/Swagger import/export
- [ ] API versioning support
- [ ] Team collaboration features
- [ ] API testing interface
- [ ] GraphQL support
- [ ] Docker deployment
- [ ] Cloud hosting option
- [ ] Plugin system

## License

MIT License - see [LICENSE](./LICENSE) for details.

## Author

**Howe Chen**
- Email: yuhao.howe.chen@gmail.com
- GitHub: [@ProjAnvil](https://github.com/ProjAnvil)

## Links

- **Repository**: https://github.com/ProjAnvil/knot
- **Issues**: https://github.com/ProjAnvil/knot/issues
- **Releases**: https://github.com/ProjAnvil/knot/releases
- **NPM Package**: https://www.npmjs.com/package/@ProjAnvil/knot

## Acknowledgments

- [Model Context Protocol](https://modelcontextprotocol.io) for AI integration standard
- [Svelte](https://svelte.dev/) for the amazing frontend framework
- [GORM](https://gorm.io/) for the powerful ORM
- [shadcn-svelte](https://www.shadcn-svelte.com/) for beautiful UI components

---

<div align="center">

Made with ❤️ by the Knot team

**[⬆ Back to Top](#knot)**

</div>
