# Knot MCP 使用教程

中文文档 | **[English](./MCP_USAGE_GUIDE.md)**

Knot 提供了 Model Context Protocol (MCP) 服务器，让 AI 助手（如 Claude）能够直接访问和查询你的 API 文档。

## 目录

1. [什么是 MCP](#什么是-mcp)
2. [安装和构建](#安装和构建)
3. [使用方式一：项目级 MCP 配置](#使用方式一项目级-mcp-配置)
4. [使用方式二：全局 MCP 配置](#使用方式二全局-mcp-配置)
5. [工具权限配置](#工具权限配置)
6. [MCP 工具说明](#mcp-工具说明)
7. [使用示例](#使用示例)
8. [故障排除](#故障排除)

---

## 什么是 MCP

Model Context Protocol (MCP) 是一个标准协议，允许 AI 助手通过结构化的方式访问外部数据源和工具。Knot 的 MCP 服务器提供以下功能：

- 📋 列出所有 API 分组
- 🔍 搜索 API（支持模糊匹配）
- 📚 查看 API 分组详情
- 📄 列出分组内的所有 API
- 🔎 查看单个 API 的详细文档
- 📝 生成 API 请求/响应的 JSON 示例

---

## 安装和构建

### 1. 确保后端服务运行

MCP 服务器需要连接到 Knot 后端服务。

```bash
# 启动后端服务（默认端口 3000）
cd backend
./bin/knot-server

# 或使用 CLI 工具在后台启动
./bin/knot start
```

### 2. 构建 MCP 服务器

```bash
cd mcp-server
go mod download
make build
```

构建完成后，二进制文件位于 `mcp-server/bin/knot-mcp`。

### 3. 测试 MCP 服务器

```bash
cd mcp-server
./bin/knot-mcp
```

如果配置正确，服务器将启动并等待 stdio 输入。按 `Ctrl+C` 退出。

---

## 使用方式一：项目级 MCP 配置

项目级配置仅在当前项目目录下生效，适合团队协作和项目特定的 API 文档访问。

### 步骤 1：创建项目配置文件

在项目根目录创建 `.claude/config.json`：

```bash
mkdir -p .claude
```

### 步骤 2：配置 MCP 服务器

编辑 `.claude/config.json`，添加以下内容：

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

**重要配置说明：**

- `command`: MCP 服务器二进制文件的**绝对路径**
- `args`: 命令行参数（通常为空数组）
- `env.KNOT_BASE_URL`: Knot 后端服务地址（默认 `http://localhost:3000`）

### 步骤 3：配置工具权限（允许所有工具）

在同一个 `.claude/config.json` 文件中，添加 `allowedTools` 配置：

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

**权限说明：**

- `mcp__knot-mcp__*`: 通配符，允许所有 knot-mcp 提供的工具
- 无需用户逐个确认工具调用，提升交互效率

### 步骤 4：重启 Claude Code

保存配置后，重启 Claude Code 使配置生效：

1. 退出 Claude Code（`Ctrl+C` 或 `/exit`）
2. 重新启动 Claude Code
3. 配置会自动加载

### 步骤 5：验证 MCP 连接

在 Claude Code 中执行以下命令验证：

```
列出所有 API 分组
```

如果配置成功，Claude 会使用 MCP 工具返回所有 API 分组列表。

---

## 使用方式二：全局 MCP 配置

全局配置在所有项目中生效，适合个人开发者频繁访问同一套 API 文档。

### 步骤 1：定位全局配置文件

根据操作系统找到全局配置文件位置：

- **macOS/Linux**: `~/.config/claude-code/config.json`
- **Windows**: `%APPDATA%\claude-code\config.json`

### 步骤 2：创建或编辑全局配置

如果文件不存在，创建该文件：

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

### 步骤 3：配置全局 MCP 服务器

编辑全局配置文件，添加以下内容：

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

**注意事项：**

- 使用**绝对路径**指向 MCP 服务器二进制文件
- Windows 用户需使用反斜杠或双反斜杠：`C:\\path\\to\\knot-mcp.exe`
- 确保后端服务地址正确（如果使用非默认端口，修改 `KNOT_BASE_URL`）

### 步骤 4：配置全局工具权限（允许所有工具）

在同一个全局配置文件中，添加 `allowedTools` 配置：

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

**全局权限优势：**

- 在所有项目中都无需手动确认工具调用
- 提升跨项目使用 Knot 的效率
- 适合信任且频繁使用的 MCP 服务器

### 步骤 5：重启 Claude Code

保存配置后，重启 Claude Code 使配置生效。

### 步骤 6：验证全局配置

在任意项目目录下启动 Claude Code，执行：

```
查询所有 API 分组
```

如果配置成功，Claude 会使用全局 MCP 配置访问 Knot。

---

## 工具权限配置

### 权限级别说明

Knot MCP 提供以下工具，可以单独或批量授权：

| 工具名称 | 功能描述 |
|---------|---------|
| `mcp__knot-mcp__list_groups` | 列出所有 API 分组 |
| `mcp__knot-mcp__get_group` | 获取单个分组详情（支持模糊匹配）|
| `mcp__knot-mcp__list_apis_by_group` | 列出分组内所有 API |
| `mcp__knot-mcp__search_apis` | 搜索 API（支持名称/端点模糊匹配）|
| `mcp__knot-mcp__get_api` | 获取 API 详细文档 |
| `mcp__knot-mcp__get_api_json_example` | 生成 API 请求/响应 JSON 示例 |

### 配置选项

#### 选项 1：允许所有工具（推荐）

```json
{
  "allowedTools": [
    "mcp__knot-mcp__*"
  ]
}
```

**优点：**
- ✅ 无需逐个确认，使用流畅
- ✅ 支持所有功能，无限制
- ✅ 适合可信任的本地服务

#### 选项 2：允许特定工具

如果你只需要部分功能，可以明确列出所有需要的工具：

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

**完整工具列表：**

| 工具标识符 | 功能 |
|-----------|------|
| `mcp__knot-mcp__list_groups` | 列出所有 API 分组 |
| `mcp__knot-mcp__get_group` | 获取单个分组详情（支持模糊匹配） |
| `mcp__knot-mcp__list_apis_by_group` | 列出分组内所有 API |
| `mcp__knot-mcp__search_apis` | 搜索 API（支持名称/端点模糊匹配） |
| `mcp__knot-mcp__get_api` | 获取 API 详细文档 |
| `mcp__knot-mcp__get_api_json_example` | 生成 API 请求/响应 JSON 示例 |

**优点：**
- ✅ 精细控制权限
- ✅ 仅授权必要功能
- ✅ 适合生产环境或敏感数据

**示例（仅允许查询功能）：**

```json
{
  "allowedTools": [
    "mcp__knot-mcp__list_groups",
    "mcp__knot-mcp__search_apis",
    "mcp__knot-mcp__get_api"
  ]
}
```

#### 选项 3：不配置权限（每次确认）

不添加 `allowedTools` 配置，Claude 会在每次调用工具前请求用户确认。

**优点：**
- ✅ 最高安全性
- ❌ 交互繁琐，影响效率

### 完整配置示例（项目级 + 全部权限）

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

### 完整配置示例（全局 + 全部权限）

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
      "args": [],
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

## MCP 工具说明

### 1. list_groups - 列出所有分组

**功能：** 获取所有 API 分组列表

**参数：** 无

**返回示例：**

```json
[
  {
    "id": 1,
    "name": "IBG-SERVICE 名单系统联机",
    "createdAt": 1764732602
  },
  {
    "id": 2,
    "name": "ihybrid-forex子系统",
    "createdAt": 1764732603
  }
]
```

**使用示例：**

```
列出所有 API 分组
```

---

### 2. get_group - 获取分组详情

**功能：** 查询单个分组的详细信息（支持模糊匹配）

**参数：**

- `groupName` (string, required): 分组名称（支持部分匹配）

**返回示例：**

```json
{
  "id": 32,
  "name": "公共分类",
  "apiCount": 1
}
```

**使用示例：**

```
查询"公共"分组的详情
```

---

### 3. list_apis_by_group - 列出分组内所有 API

**功能：** 获取指定分组内的所有 API 列表

**参数：**

- `groupName` (string, required): 分组名称（支持模糊匹配）

**返回示例：**

```json
{
  "group": {
    "id": 33,
    "name": "个人会员服务"
  },
  "apis": [
    {
      "id": 305,
      "name": "升级中级钱包(个人实名认证)",
      "endpoint": "88200084-PERSONAL_REAL_NAME",
      "method": "",
      "type": "RPC"
    }
  ]
}
```

**使用示例：**

```
列出"个人会员服务"分组的所有 API
```

---

### 4. search_apis - 搜索 API

**功能：** 按名称或端点搜索 API（模糊匹配，最多返回 50 条）

**参数：**

- `query` (string, required): 搜索关键词

**返回示例：**

```json
{
  "count": 2,
  "apis": [
    {
      "id": 305,
      "name": "升级中级钱包(个人实名认证)",
      "endpoint": "88200084-PERSONAL_REAL_NAME",
      "method": "",
      "type": "RPC",
      "group": {
        "id": 33,
        "name": "个人会员服务"
      }
    }
  ]
}
```

**使用示例：**

```
搜索包含"钱包"的 API
```

---

### 5. get_api - 获取 API 详细文档

**功能：** 查看单个 API 的完整文档，包括请求/响应参数

**参数：**

- `apiId` (number, required): API ID（从搜索或列表中获取）

**返回示例：**

```json
{
  "id": 305,
  "name": "升级中级钱包(个人实名认证)",
  "endpoint": "88200084-PERSONAL_REAL_NAME",
  "method": "",
  "type": "RPC",
  "group": "个人会员服务",
  "requestParams": [
    {
      "name": "userId",
      "type": "string",
      "required": true,
      "description": "用户ID",
      "children": []
    }
  ],
  "responseParams": [
    {
      "name": "code",
      "type": "string",
      "required": true,
      "description": "响应码",
      "children": []
    }
  ]
}
```

**使用示例：**

```
查看 API ID 305 的详细文档
```

---

### 6. get_api_json_example - 生成 JSON 示例

**功能：** 根据 API 参数定义自动生成请求/响应 JSON 示例

**参数：**

- `apiId` (number, required): API ID

**返回示例：**

```json
{
  "apiName": "升级中级钱包(个人实名认证)",
  "endpoint": "88200084-PERSONAL_REAL_NAME",
  "method": "",
  "requestExample": {
    "userId": "string",
    "realName": "string"
  },
  "responseExample": {
    "code": "string",
    "message": "string",
    "data": {}
  }
}
```

**使用示例：**

```
生成 API ID 305 的 JSON 示例
```

---

## 使用示例

### 示例 1：查找特定业务的 API

**需求：** 找到所有与"充值"相关的 API

```
搜索"充值"相关的 API
```

Claude 会使用 `search_apis` 工具返回所有匹配的 API。

---

### 示例 2：查看分组下的所有 API

**需求：** 查看"密码设置"分组下有哪些 API

```
列出"密码设置"分组的所有 API
```

Claude 会使用 `list_apis_by_group` 工具返回该分组的 API 列表。

---

### 示例 3：查看 API 详细文档

**需求：** 了解"修改支付密码" API 的请求参数

```
搜索"修改支付密码"，然后查看详细文档
```

Claude 会先搜索 API，获取 ID，然后使用 `get_api` 工具返回完整文档。

---

### 示例 4：生成 API 调用示例

**需求：** 获取"会员充值" API 的 JSON 请求示例

```
搜索"会员充值"，并生成 JSON 示例
```

Claude 会使用 `search_apis` 和 `get_api_json_example` 工具返回示例代码。

---

### 示例 5：统计 API 数量

**需求：** 统计所有分组和 API 总数

```
统计总共有多少个 API 分组和多少个 API？
```

Claude 会使用 `list_groups` 和 `list_apis_by_group` 工具遍历所有分组并统计。

---

## 故障排除

### 问题 1：MCP 服务器无法启动

**症状：** Claude 提示无法连接到 MCP 服务器

**解决方法：**

1. 确认后端服务正在运行：
   ```bash
   curl http://localhost:3000/api/groups
   ```

2. 确认 MCP 服务器路径正确：
   ```bash
   ls -l /path/to/knot-mcp
   ```

3. 手动测试 MCP 服务器：
   ```bash
   cd mcp-server
   ./bin/knot-mcp
   ```

4. 检查配置文件路径和格式：
   ```bash
   cat .claude/config.json
   ```

---

### 问题 2：工具调用需要频繁确认

**症状：** 每次调用工具都需要手动确认

**解决方法：**

在配置文件中添加 `allowedTools` 配置：

```json
{
  "allowedTools": [
    "mcp__knot-mcp__*"
  ]
}
```

重启 Claude Code 使配置生效。

---

### 问题 3：KNOT_BASE_URL 环境变量无效

**症状：** MCP 服务器连接错误的后端地址

**解决方法：**

在配置文件中显式设置 `env.KNOT_BASE_URL`：

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

### 问题 4：Windows 路径配置错误

**症状：** Windows 用户无法启动 MCP 服务器

**解决方法：**

使用双反斜杠或正斜杠：

```json
{
  "command": "C:\\Users\\YourName\\Documents\\knot\\mcp-server\\bin\\knot-mcp.exe"
}
```

或：

```json
{
  "command": "C:/Users/YourName/Documents/knot/mcp-server/bin/knot-mcp.exe"
}
```

---

### 问题 5：配置文件不生效

**症状：** 修改配置后 Claude 仍使用旧配置

**解决方法：**

1. 确保配置文件 JSON 格式正确（使用 JSON 验证器）
2. 完全退出 Claude Code（不是切换项目）
3. 重新启动 Claude Code
4. 检查配置优先级：项目级配置 > 全局配置

---

## 高级配置

### 多环境配置

如果你有多个环境（开发/测试/生产），可以配置多个 MCP 服务器：

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

使用时指定服务器：

```
使用 knot-dev 服务器查询所有 API 分组
```

---

### 自定义端口配置

如果后端服务运行在非默认端口：

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

## 总结

### 项目级配置特点

- ✅ 配置文件在项目根目录 `.claude/config.json`
- ✅ 仅在当前项目生效
- ✅ 适合团队协作和项目特定配置
- ✅ Git 可忽略（添加到 `.gitignore`）

### 全局配置特点

- ✅ 配置文件在 `~/.config/claude-code/config.json`
- ✅ 在所有项目中生效
- ✅ 适合个人开发者频繁使用
- ✅ 无需重复配置

### 推荐配置

**个人开发者：** 使用全局配置 + 允许所有工具

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

**团队协作：** 使用项目级配置 + 允许所有工具

在项目根目录创建 `.claude/config.json`，内容同上。

---

## 相关资源

- **Knot 项目**: [GitHub Repository](https://github.com/ProjAnvil/knot)
- **MCP 协议规范**: [Model Context Protocol](https://modelcontextprotocol.io)
- **Claude Code 文档**: [Claude Code Guide](https://claude.ai/code)

---

## 贡献和反馈

如有问题或建议，欢迎提交 Issue 或 Pull Request！

---

**最后更新时间：** 2025-12-03
