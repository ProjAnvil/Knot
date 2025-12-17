# Knot MCP Server (Go)

Model Context Protocol (MCP) server for Knot - provides read-only access to API documentation through Claude Desktop and other MCP clients.

This is a **pure Go implementation** of the MCP server, functionally equivalent to the Node.js version but with better performance and easier deployment (single binary, no Node.js runtime required).

## Features

- **Read-only API exploration**: Browse API groups, endpoints, and parameters
- **Fuzzy search**: Find APIs by partial name or endpoint path
- **Structured data**: Hierarchical view of groups → APIs → parameters
- **JSON examples**: Auto-generate example request/response payloads
- **Native Go**: Single binary, no runtime dependencies
- **MCP Protocol**: Compatible with Claude Desktop, Cline, and other MCP clients

## Installation

### Option 1: Build from Source

```bash
# Build the binary
go build -o knot-mcp main.go

# Run the server
./knot-mcp
```

### Option 2: Install via Go

```bash
go install github.com/ProjAnvil/knot/mcp-server@latest
```

## Configuration

The server connects to the Knot backend via HTTP. Configure the backend URL using an environment variable:

```bash
export KNOT_BASE_URL=http://localhost:3000
```

Default: `http://localhost:3000`

## Usage with Claude Desktop

Add this configuration to your Claude Desktop config file:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`

**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "knot": {
      "command": "/path/to/knot-mcp",
      "env": {
        "KNOT_BASE_URL": "http://localhost:3000"
      }
    }
  }
}
```

## Available Tools

### 1. `list_groups`
List all API groups in the database.

**Usage**: "Show me all API groups"

### 2. `get_group`
Get details about a specific group (supports fuzzy matching).

**Arguments**:
- `groupName` (string): Full or partial group name

**Usage**: "Get information about the user group"

### 3. `list_apis_by_group`
List all APIs within a specific group.

**Arguments**:
- `groupName` (string): Full or partial group name

**Usage**: "Show me all APIs in the authentication group"

### 4. `get_api`
Get comprehensive details about a specific API.

**Arguments**:
- `apiId` (number): The unique API ID

**Usage**: "Get details for API ID 123"

### 5. `search_apis`
Search for APIs by name or endpoint path.

**Arguments**:
- `query` (string): Search term

**Usage**: "Search for APIs containing 'login'"

### 6. `get_api_json_example`
Generate example JSON payloads for an API.

**Arguments**:
- `apiId` (number): The unique API ID

**Usage**: "Show me JSON examples for API ID 456"

## Available Resources

### `knot://groups`
Returns a JSON list of all API groups. Can be used by MCP clients to discover available groups.

## Available Prompts

### `explore-api-group`
Interactive prompt to explore APIs in a specific group.

**Arguments**:
- `groupName`: Name of the group to explore

### `find-api`
Interactive prompt to search for APIs.

**Arguments**:
- `query`: Search term

## Architecture

```
┌─────────────────┐
│  MCP Client     │  (Claude Desktop, Cline, etc.)
│  (STDIO)        │
└────────┬────────┘
         │ MCP Protocol
         │
┌────────▼────────┐
│  knot-mcp│  (This Go server)
│  (Go Binary)    │
└────────┬────────┘
         │ HTTP POST
         │
┌────────▼────────┐
│  Knot           │  /api/mcp-tools endpoint
│  Backend (Go)   │
└─────────────────┘
```

## Development

```bash
# Install dependencies
go mod download

# Run in development mode
go run main.go

# Build
go build -o knot-mcp main.go

# Test with environment variable
KNOT_BASE_URL=http://localhost:3000 go run main.go
```

## Differences from Node.js Version

This Go implementation is functionally identical to the Node.js version but offers:

- **No runtime dependencies**: Single binary, no Node.js required
- **Better performance**: Native compiled binary
- **Lower memory footprint**: More efficient resource usage
- **Cross-platform**: Compile once for any platform
- **Easier deployment**: Just copy the binary

## License

MIT

## Author

Howe Chen <yuhao.howe.chen@gmail.com>
