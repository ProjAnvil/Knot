# Knot MCP Setup Guide

English | **[中文版](./MCP_USAGE_GUIDE_zh.md)**
**[中文版](./MCP_USAGE_GUIDE.md)** | English

Knot provides a Model Context Protocol (MCP) server that enables AI assistants (like Claude) to directly access and query your API documentation.

## Table of Contents

1. [What is MCP](#what-is-mcp)
2. [Installation and Build](#installation-and-build)
3. [Option 1: Project-Level MCP Configuration](#option-1-project-level-mcp-configuration)
4. [Option 2: Global MCP Configuration](#option-2-global-mcp-configuration)
5. [Tool Permissions Configuration](#tool-permissions-configuration)
6. [MCP Tools Reference](#mcp-tools-reference)
7. [Usage Examples](#usage-examples)
8. [Troubleshooting](#troubleshooting)

---

## What is MCP

Model Context Protocol (MCP) is a standard protocol that allows AI assistants to access external data sources and tools in a structured way. Knot's MCP server provides the following capabilities:

- 📋 List all API groups
- 🔍 Search APIs (with fuzzy matching)
- 📚 View API group details
- 📄 List all APIs in a group
- 🔎 View detailed API documentation
- 📝 Generate JSON examples for API requests/responses

---

## Installation and Build

### 1. Ensure Backend Service is Running

The MCP server needs to connect to the Knot backend service.

```bash
# Start backend service (default port 3000)
cd backend
./bin/knot-server

# Or use CLI tool to start in background
./bin/knot start
```

### 2. Build MCP Server

```bash
cd mcp-server
go mod download
make build
```

After building, the binary will be at `mcp-server/bin/knot-mcp`.

### 3. Test MCP Server

```bash
cd mcp-server
./bin/knot-mcp
```

If configured correctly, the server will start and wait for stdio input. Press `Ctrl+C` to exit.

---

## Option 1: Project-Level MCP Configuration

Project-level configuration only applies to the current project directory. Ideal for team collaboration and project-specific API documentation access.

### Step 1: Create Project Configuration File

Create `.claude/config.json` in your project root:

```bash
mkdir -p .claude
```

### Step 2: Configure MCP Server

Edit `.claude/config.json` and add the following:

```json
{
  "mcpServers": {
    "knot-mcp": {
      "command": "/absolute/path/to/mcp-server/bin/knot-mcp",
      "env": {
        "KNOT_BASE_URL": "http://localhost:3000"
      }
    }
  }
}
```

**Important Configuration Details:**

- `command`: **Absolute path** to the MCP server binary
- `env.KNOT_BASE_URL`: Knot backend service address (default `http://localhost:3000`)

### Step 3: Configure Tool Permissions (Allow All Tools)

In the same `.claude/config.json` file, add `allowedTools` configuration:

```json
{
  "mcpServers": {
    "knot-mcp": {
      "command": "/absolute/path/to/mcp-server/bin/knot-mcp",
      "env": {
        "KNOT_BASE_URL": "http://localhost:3000"
      }
    }
  },
  "allowedTools": [
    "mcp__knot-mcp__*"
  ]
}
```

**Permission Details:**

- `mcp__knot-mcp__*`: Wildcard to allow all knot-mcp tools
- No need for user confirmation on each tool call, improving interaction efficiency

### Step 4: Restart Claude Code

After saving the configuration, restart Claude Code to apply changes:

1. Exit Claude Code (`Ctrl+C` or `/exit`)
2. Restart Claude Code
3. Configuration will be loaded automatically

### Step 5: Verify MCP Connection

In Claude Code, execute the following to verify:

```
List all API groups
```

If configured successfully, Claude will use the MCP tool to return all API groups.

---

## Option 2: Global MCP Configuration

Global configuration applies to all projects. Ideal for individual developers who frequently access the same API documentation.

### Step 1: Locate Global Configuration File

Find the global configuration file location based on your OS:

- **macOS/Linux**: `~/.config/claude-code/config.json`
- **Windows**: `%APPDATA%\claude-code\config.json`

### Step 2: Create or Edit Global Configuration

If the file doesn't exist, create it:

```bash
# macOS/Linux
mkdir -p ~/.config/claude-code
touch ~/.config/claude-code/config.json
```

```powershell
# Windows (PowerShell)
New-Item -ItemType Directory -Force -Path "$env:APPDATA\claude-code"
New-Item -ItemType File -Force -Path "$env:APPDATA\claude-code\config.json"
```

### Step 3: Configure Global MCP Server

Edit the global configuration file and add:

```json
{
  "mcpServers": {
    "knot-mcp": {
      "command": "/absolute/path/to/mcp-server/bin/knot-mcp",
      "env": {
        "KNOT_BASE_URL": "http://localhost:3000"
      }
    }
  }
}
```

**Important Notes:**

- Use **absolute path** to the MCP server binary
- Windows users should use backslashes or double backslashes: `C:\\path\\to\\knot-mcp.exe`
- Ensure the backend service address is correct (modify `KNOT_BASE_URL` if using non-default port)

### Step 4: Configure Global Tool Permissions (Allow All Tools)

In the same global configuration file, add `allowedTools`:

```json
{
  "mcpServers": {
    "knot-mcp": {
      "command": "/absolute/path/to/mcp-server/bin/knot-mcp",
      "env": {
        "KNOT_BASE_URL": "http://localhost:3000"
      }
    }
  },
  "allowedTools": [
    "mcp__knot-mcp__*"
  ]
}
```

**Global Permission Benefits:**

- No manual confirmation needed for tool calls across all projects
- Improves efficiency when using Knot across multiple projects
- Suitable for trusted and frequently used MCP servers

### Step 5: Restart Claude Code

After saving the configuration, restart Claude Code to apply changes.

### Step 6: Verify Global Configuration

Start Claude Code in any project directory and execute:

```
Query all API groups
```

If configured successfully, Claude will use the global MCP configuration to access Knot.

---

## Tool Permissions Configuration

### Permission Levels

Knot MCP provides the following tools that can be authorized individually or in bulk:

| Tool Name | Description |
|-----------|-------------|
| `mcp__knot-mcp__list_groups` | List all API groups |
| `mcp__knot-mcp__get_group` | Get single group details (fuzzy match supported) |
| `mcp__knot-mcp__list_apis_by_group` | List all APIs in a group |
| `mcp__knot-mcp__search_apis` | Search APIs (name/endpoint fuzzy match) |
| `mcp__knot-mcp__get_api` | Get detailed API documentation |
| `mcp__knot-mcp__get_api_json_example` | Generate API request/response JSON examples |

### Configuration Options

#### Option 1: Allow All Tools (Recommended)

```json
{
  "allowedTools": [
    "mcp__knot-mcp__*"
  ]
}
```

**Advantages:**
- ✅ No confirmation needed, smooth experience
- ✅ Full functionality, no restrictions
- ✅ Suitable for trusted local services

#### Option 2: Allow Specific Tools

If you only need certain features, explicitly list the required tools:

```json
{
  "allowedTools": [
    "mcp__knot-mcp__list_groups",
    "mcp__knot-mcp__get_group",
    "mcp__knot-mcp__list_apis_by_group",
    "mcp__knot-mcp__search_apis",
    "mcp__knot-mcp__get_api",
    "mcp__knot-mcp__get_api_json_example"
  ]
}
```

**Advantages:**
- ✅ Fine-grained permission control
- ✅ Only authorize necessary features
- ✅ Suitable for production or sensitive data

**Example (Query-only permissions):**

```json
{
  "allowedTools": [
    "mcp__knot-mcp__list_groups",
    "mcp__knot-mcp__search_apis",
    "mcp__knot-mcp__get_api"
  ]
}
```

#### Option 3: No Permission Configuration (Confirm Each Time)

Don't add `allowedTools` configuration. Claude will request user confirmation before each tool call.

**Advantages:**
- ✅ Maximum security
- ❌ Tedious interaction, affects efficiency

### Complete Configuration Example (Project-Level + Full Permissions)

```json
{
  "mcpServers": {
    "knot-mcp": {
      "command": "/absolute/path/to/mcp-server/bin/knot-mcp",
      "env": {
        "KNOT_BASE_URL": "http://localhost:3000"
      }
    }
  },
  "allowedTools": [
    "mcp__knot-mcp__*"
  ]
}
```

### Complete Configuration Example (Global + Full Permissions)

**macOS/Linux** (`~/.config/claude-code/config.json`):

```json
{
  "mcpServers": {
    "knot-mcp": {
      "command": "/absolute/path/to/knot-mcp",
      "env": {
        "KNOT_BASE_URL": "http://localhost:3000"
      }
    }
  },
  "allowedTools": [
    "mcp__knot-mcp__*"
  ]
}
```

**Windows** (`%APPDATA%\claude-code\config.json`):

```json
{
  "mcpServers": {
    "knot-mcp": {
      "command": "C:\\Users\\YourName\\Documents\\knot\\mcp-server\\bin\\knot-mcp.exe",
      "env": {
        "KNOT_BASE_URL": "http://localhost:3000"
      }
    }
  },
  "allowedTools": [
    "mcp__knot-mcp__*"
  ]
}
```

---

## MCP Tools Reference

### 1. list_groups - List All Groups

**Function:** Get all API group listings

**Parameters:** None

**Return Example:**

```json
[
  {
    "id": 1,
    "name": "User Service",
    "createdAt": 1764732602
  },
  {
    "id": 2,
    "name": "Payment Service",
    "createdAt": 1764732603
  }
]
```

**Usage Example:**

```
List all API groups
```

---

### 2. get_group - Get Group Details

**Function:** Query single group's detailed information (fuzzy match supported)

**Parameters:**

- `groupName` (string, required): Group name (partial match supported)

**Return Example:**

```json
{
  "id": 32,
  "name": "Common Category",
  "apiCount": 1
}
```

**Usage Example:**

```
Query details for "common" group
```

---

### 3. list_apis_by_group - List APIs in Group

**Function:** Get all API listings in specified group

**Parameters:**

- `groupName` (string, required): Group name (fuzzy match supported)

**Return Example:**

```json
{
  "group": {
    "id": 33,
    "name": "User Service"
  },
  "apis": [
    {
      "id": 305,
      "name": "User Registration",
      "endpoint": "/api/user/register",
      "method": "POST",
      "type": "HTTP"
    }
  ]
}
```

**Usage Example:**

```
List all APIs in "User Service" group
```

---

### 4. search_apis - Search APIs

**Function:** Search APIs by name or endpoint (fuzzy match, max 50 results)

**Parameters:**

- `query` (string, required): Search keyword

**Return Example:**

```json
{
  "count": 2,
  "apis": [
    {
      "id": 305,
      "name": "User Registration",
      "endpoint": "/api/user/register",
      "method": "POST",
      "type": "HTTP",
      "group": {
        "id": 33,
        "name": "User Service"
      }
    }
  ]
}
```

**Usage Example:**

```
Search APIs containing "register"
```

---

### 5. get_api - Get Detailed API Documentation

**Function:** View complete documentation for a single API, including request/response parameters

**Parameters:**

- `apiId` (number, required): API ID (obtained from search or list)

**Return Example:**

```json
{
  "id": 305,
  "name": "User Registration",
  "endpoint": "/api/user/register",
  "method": "POST",
  "type": "HTTP",
  "group": "User Service",
  "requestParams": [
    {
      "name": "email",
      "type": "string",
      "required": true,
      "description": "User email address",
      "children": []
    }
  ],
  "responseParams": [
    {
      "name": "code",
      "type": "number",
      "required": true,
      "description": "Response code",
      "children": []
    }
  ]
}
```

**Usage Example:**

```
View detailed documentation for API ID 305
```

---

### 6. get_api_json_example - Generate JSON Examples

**Function:** Automatically generate request/response JSON examples based on API parameter definitions

**Parameters:**

- `apiId` (number, required): API ID

**Return Example:**

```json
{
  "apiName": "User Registration",
  "endpoint": "/api/user/register",
  "method": "POST",
  "requestExample": {
    "email": "string",
    "password": "string"
  },
  "responseExample": {
    "code": 0,
    "message": "string",
    "data": {}
  }
}
```

**Usage Example:**

```
Generate JSON example for API ID 305
```

---

## Usage Examples

### Example 1: Find APIs for Specific Business Logic

**Need:** Find all APIs related to "payment"

```
Search for APIs related to "payment"
```

Claude will use the `search_apis` tool to return all matching APIs.

---

### Example 2: View All APIs in a Group

**Need:** See what APIs are in the "Authentication" group

```
List all APIs in the "Authentication" group
```

Claude will use the `list_apis_by_group` tool to return the API list for that group.

---

### Example 3: View Detailed API Documentation

**Need:** Understand the request parameters for the "Login" API

```
Search for "Login" and view detailed documentation
```

Claude will first search for the API, get its ID, then use the `get_api` tool to return complete documentation.

---

### Example 4: Generate API Call Example

**Need:** Get JSON request example for "User Registration" API

```
Search for "User Registration" and generate JSON example
```

Claude will use `search_apis` and `get_api_json_example` tools to return example code.

---

### Example 5: Count API Statistics

**Need:** Count total number of groups and APIs

```
Count how many API groups and APIs are there in total?
```

Claude will use `list_groups` and `list_apis_by_group` tools to iterate through all groups and count.

---

## Troubleshooting

### Issue 1: MCP Server Won't Start

**Symptom:** Claude reports unable to connect to MCP server

**Solution:**

1. Confirm backend service is running:
   ```bash
   curl http://localhost:3000/api/groups
   ```

2. Confirm MCP server path is correct:
   ```bash
   ls -l /path/to/knot-mcp
   ```

3. Manually test MCP server:
   ```bash
   cd mcp-server
   ./bin/knot-mcp
   ```

4. Check configuration file path and format:
   ```bash
   cat .claude/config.json
   ```

---

### Issue 2: Tool Calls Require Frequent Confirmation

**Symptom:** Each tool call requires manual confirmation

**Solution:**

Add `allowedTools` configuration to your config file:

```json
{
  "allowedTools": [
    "mcp__knot-mcp__*"
  ]
}
```

Restart Claude Code to apply configuration.

---

### Issue 3: KNOT_BASE_URL Environment Variable Not Working

**Symptom:** MCP server connects to wrong backend address

**Solution:**

Explicitly set `env.KNOT_BASE_URL` in configuration file:

```json
{
  "mcpServers": {
    "knot-mcp": {
      "env": {
        "KNOT_BASE_URL": "http://localhost:3000"
      }
    }
  }
}
```

---

### Issue 4: Windows Path Configuration Error

**Symptom:** Windows users unable to start MCP server

**Solution:**

Use double backslashes or forward slashes:

```json
{
  "command": "C:\\Users\\YourName\\Documents\\knot\\mcp-server\\bin\\knot-mcp.exe"
}
```

Or:

```json
{
  "command": "C:/Users/YourName/Documents/knot/mcp-server/bin/knot-mcp.exe"
}
```

---

### Issue 5: Configuration File Not Taking Effect

**Symptom:** Claude still uses old configuration after modification

**Solution:**

1. Ensure configuration file JSON format is correct (use JSON validator)
2. Completely exit Claude Code (not just switch projects)
3. Restart Claude Code
4. Check configuration priority: Project-level config > Global config

---

## Advanced Configuration

### Multi-Environment Configuration

If you have multiple environments (dev/test/prod), you can configure multiple MCP servers:

```json
{
  "mcpServers": {
    "knot-dev": {
      "command": "/path/to/knot-mcp",
      "env": {
        "KNOT_BASE_URL": "http://localhost:3000"
      }
    },
    "knot-prod": {
      "command": "/path/to/knot-mcp",
      "env": {
        "KNOT_BASE_URL": "https://knot.production.com"
      }
    }
  },
  "allowedTools": [
    "mcp__knot-dev__*",
    "mcp__knot-prod__*"
  ]
}
```

Usage with specific server:

```
Using knot-dev server, query all API groups
```

---

### Custom Port Configuration

If backend service runs on non-default port:

```json
{
  "mcpServers": {
    "knot-mcp": {
      "command": "/path/to/knot-mcp",
      "env": {
        "KNOT_BASE_URL": "http://localhost:8080"
      }
    }
  }
}
```

---

## Summary

### Project-Level Configuration Features

- ✅ Configuration file at project root `.claude/config.json`
- ✅ Only applies to current project
- ✅ Suitable for team collaboration and project-specific configuration
- ✅ Can be ignored in Git (add to `.gitignore`)

### Global Configuration Features

- ✅ Configuration file at `~/.config/claude-code/config.json`
- ✅ Applies to all projects
- ✅ Suitable for individual developers with frequent use
- ✅ No need to configure repeatedly

### Recommended Configuration

**Individual Developers:** Use global configuration + allow all tools

```json
{
  "mcpServers": {
    "knot-mcp": {
      "command": "/absolute/path/to/knot-mcp",
      "env": {
        "KNOT_BASE_URL": "http://localhost:3000"
      }
    }
  },
  "allowedTools": [
    "mcp__knot-mcp__*"
  ]
}
```

**Team Collaboration:** Use project-level configuration + allow all tools

Create `.claude/config.json` in project root with the same content as above.

---

## Related Resources

- **Knot Project**: [GitHub Repository](https://github.com/ProjAnvil/knot)
- **MCP Protocol Specification**: [Model Context Protocol](https://modelcontextprotocol.io)
- **Claude Code Documentation**: [Claude Code Guide](https://claude.ai/code)

---

## Contributing and Feedback

If you have questions or suggestions, feel free to submit an Issue or Pull Request!

---

**Last Updated:** 2025-12-17
