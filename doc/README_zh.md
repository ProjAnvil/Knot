# Knot

<div align="center">

现代化、轻量级的 API 文档管理系统，支持 AI 助手集成

[功能特性](#功能特性) • [快速开始](#快速开始) • [文档](#文档) • [开发指南](#开发指南) • [贡献指南](#贡献指南)

中文文档 | **[English](../README.md)**

</div>

---

## 概述

Knot 是一个全面的 API 文档管理平台，帮助团队组织、记录和分享他们的 API 规范。使用 Go 和 Svelte 5 构建，提供快速、直观的界面，并通过模型上下文协议（MCP）原生支持 AI 助手。

### 核心特性

- 📚 **API 组织管理** - 使用层级结构对 API 端点进行分组和分类
- 🔍 **模糊搜索** - 通过智能搜索快速查找所有分组中的 API
- 📝 **丰富的文档** - 使用 Markdown、请求/响应模式和示例记录 API
- 🎨 **语法高亮** - 优美的 JSON 语法高亮，支持深色模式
- 🔄 **拖放界面** - 直观的 API 重新排序和组织方式
- 🌐 **多语言支持** - 内置英文和中文支持
- 🗄️ **灵活的数据库** - 可选择 SQLite、PostgreSQL 或 MySQL
- 🤖 **AI 集成** - 为 Claude 等 AI 助手提供原生 MCP 服务器
- ⚡ **高性能** - Go 驱动的后端，资源占用极少
- 🚀 **零依赖** - 单二进制部署，内嵌前端

## 快速开始

### 安装

下载适合您平台的最新版本：

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
# 从 releases 页面下载 knot-windows.exe
```

### 使用方法

```bash
# 初始化配置
knot setup

# 启动服务器（后台运行）
knot start

# 查看服务器状态
knot status

# 停止服务器
knot stop

# 查看配置
knot config

# 获取帮助
knot help
```

Web 界面将在 [http://localhost:3000](http://localhost:3000) 可用

## 文档

- [中文文档](./README_zh.md)（本文件）
- [English Documentation](../README.md)
- [MCP 服务器配置](../mcp-server/README.md)
- [MCP 使用指南](./MCP_USAGE_GUIDE_zh.md)
- [开发指南](../CLAUDE.md)

## 配置

Knot 的配置文件存储位置：
- **Linux/macOS**: `~/.knot/config.json`
- **Windows**: `%LOCALAPPDATA%\knot\config.json`

配置示例：

```json
{
  "databaseType": "sqlite",
  "sqlitePath": "/Users/username/.knot/knot.db",
  "port": 3000,
  "host": "localhost",
  "enableLogging": false
}
```

### 数据库选项

| 数据库 | 使用场景 | 配置方式 |
|--------|---------|---------|
| **SQLite**（默认） | 个人使用、开发 | `sqlitePath: "/path/to/knot.db"` |
| **PostgreSQL** | 生产环境、团队 | `postgresUrl: "postgresql://..."` |
| **MySQL** | 企业级 | `mysqlUrl: "user:pass@tcp(...)/"` |

## 开发指南

Knot 由三个独立组件组成：

### 环境要求

- **Go** 1.21 或更高版本
- **Bun** 或 npm（用于前端）
- **Make**（可选，用于构建命令）

### 项目结构

```
knot/
├── frontend/          # Svelte 5 Web 应用
│   ├── src/
│   │   ├── lib/      # 可复用组件
│   │   └── messages/ # i18n 翻译
│   └── package.json
├── backend/           # Go API 服务器
│   ├── cmd/          # 入口点（CLI 和服务器）
│   ├── internal/     # 核心逻辑
│   └── Makefile
├── mcp-server/        # AI 集成的 MCP 服务器
│   ├── main.go
│   └── Makefile
└── doc/              # 文档
```

### 前端开发

```bash
cd frontend

# 安装依赖
bun install

# 启动开发服务器（热重载，端口 5173）
bun dev

# 构建生产版本
bun run build
```

前端独立运行，在开发期间将 API 请求代理到后端。

### 后端开发

```bash
cd backend

# 安装 Go 依赖
go mod download

# 以开发模式运行
make run

# 构建 CLI 二进制文件
make build

# 为所有平台构建
make build-all

# 打包（包含嵌入式前端）
make package
```

可用命令：
- `make run` - 以开发模式运行服务器
- `make build` - 为当前平台构建 CLI 二进制文件
- `make build-all` - 为所有平台构建（Linux、macOS、Windows）
- `make package` - 构建包含嵌入式前端的完整包
- `make clean` - 清理构建产物

### MCP 服务器开发

```bash
cd mcp-server

# 安装依赖
go mod download

# 构建 MCP 服务器
make build

# 为所有平台构建
make build-all
```

### 运行测试

```bash
# 后端测试
cd backend
go test ./...

# 前端测试（如果有）
cd frontend
bun test
```

## MCP 集成

Knot 包含一个模型上下文协议服务器，使 Claude 等 AI 助手能够自然地查询您的 API 文档。

### 功能特性

- 列出所有 API 分组
- 按名称或端点搜索 API
- 获取详细的 API 文档
- 生成 JSON 请求/响应示例
- 对分组和 API 名称进行模糊匹配

### 配置步骤

1. 构建 MCP 服务器：
```bash
cd mcp-server
make build
```

2. 配置 Claude Desktop 使用 Knot MCP 服务器。详细说明请参见 [MCP 使用指南](./MCP_USAGE_GUIDE.md)。

3. 开始查询您的 API：
```
"显示 user-service 分组中的所有 API"
"查找与身份验证相关的 API"
"为登录 API 生成示例请求"
```

## 架构

### 技术栈

**前端：**
- Svelte 5（最新的响应式模型）
- TypeScript
- Vite（构建工具）
- Tailwind CSS
- shadcn-svelte（UI 组件）
- svelte-i18n（国际化）

**后端：**
- Go 1.21+
- Chi（HTTP 路由器）
- GORM（支持多数据库的 ORM）
- Cobra（CLI 框架）
- Viper（配置管理）

**MCP 服务器：**
- Go 与 MCP SDK
- Stdio 传输
- RESTful API 集成

### 数据库模式

```
groups（分组）
  ├── id（主键）
  ├── name（名称）
  └── apis（包含多个 API）

apis（API）
  ├── id（主键）
  ├── group_id（外键）
  ├── name（名称）
  ├── endpoint（端点）
  ├── method（GET/POST 等）
  ├── type（HTTP/RPC）
  ├── note（Markdown 备注）
  └── parameters（包含多个参数）

parameters（参数）
  ├── id（主键）
  ├── api_id（外键）
  ├── parent_id（自引用，用于嵌套）
  ├── name（名称）
  ├── type（string/number/boolean/array/object）
  ├── param_type（request/response）
  ├── required（是否必需）
  └── description（描述）
```

## 从源码构建

### 构建完整包

```bash
# 克隆仓库
git clone https://github.com/ProjAnvil/knot.git
cd knot

# 构建前端
cd frontend
bun install
bun run build
cd ..

# 构建包含嵌入式前端的后端
cd backend
make package

# 构建 MCP 服务器
cd ../mcp-server
make build
```

二进制文件位置：
- 后端 CLI：`backend/bin/knot`
- 后端服务器：`backend/bin/knot-server`
- MCP 服务器：`mcp-server/bin/knot-mcp`

### 跨平台构建

```bash
# 为所有平台构建
cd backend
make package-all

cd ../mcp-server
make build-all
```

这将为以下平台创建二进制文件：
- Linux（AMD64）
- macOS（AMD64 和 ARM64）
- Windows（AMD64）

## 贡献指南

我们欢迎贡献！以下是您可以提供帮助的方式：

### 报告问题

- 使用 [问题跟踪器](https://github.com/ProjAnvil/knot/issues)
- 包含详细的重现步骤
- 提供系统信息（操作系统、Go 版本等）

### 拉取请求

1. Fork 仓库
2. 创建功能分支（`git checkout -b feature/amazing-feature`）
3. 进行更改
4. 运行测试并确保代码质量
5. 提交更改（`git commit -m 'Add amazing feature'`）
6. 推送到您的 fork（`git push origin feature/amazing-feature`）
7. 打开拉取请求

### 开发指南

- 遵循 Go 最佳实践和约定
- 为新功能编写测试
- 为面向用户的更改更新文档
- 保持提交原子化且描述清晰
- 在提交 PR 之前确保所有测试通过

## 路线图

- [ ] OpenAPI/Swagger 导入/导出
- [ ] API 版本控制支持
- [ ] 团队协作功能
- [ ] API 测试界面
- [ ] GraphQL 支持
- [ ] Docker 部署
- [ ] 云托管选项
- [ ] 插件系统

## 许可证

MIT 许可证 - 详见 [LICENSE](../LICENSE)

## 作者

**Howe Chen**
- 邮箱：yuhao.howe.chen@gmail.com
- GitHub：[@ProjAnvil](https://github.com/ProjAnvil)

## 链接

- **仓库**：https://github.com/ProjAnvil/knot
- **问题**：https://github.com/ProjAnvil/knot/issues
- **发布**：https://github.com/ProjAnvil/knot/releases
- **NPM 包**：https://www.npmjs.com/package/@ProjAnvil/knot

## 致谢

- [Model Context Protocol](https://modelcontextprotocol.io) 提供的 AI 集成标准
- [Svelte](https://svelte.dev/) 出色的前端框架
- [GORM](https://gorm.io/) 强大的 ORM
- [shadcn-svelte](https://www.shadcn-svelte.com/) 精美的 UI 组件

---

<div align="center">

由 Knot 团队用 ❤️ 制作

**[⬆ 回到顶部](#knot)**

</div>
