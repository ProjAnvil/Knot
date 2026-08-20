# Knot PostgreSQL/MySQL 支持完善与 Docker Compose 部署设计

日期：2026-08-20
状态：已批准

## 1. 背景与目标

Knot 是一个 API 文档管理系统（Svelte 5 前端 + Go 后端）。本设计覆盖两件事：

1. 完善后端对 PostgreSQL / MySQL 的支持（修复现存缺陷、补齐容器化所需的配置能力）。
2. 提供 Docker Compose 部署：一份多阶段 Dockerfile 构建嵌入前端的单二进制镜像，两套 compose 文件分别编排 PostgreSQL 和 MySQL 全栈。

## 2. 调研结论（现状）

对代码库的实际核查结果：

**已完成的部分**

- `backend/go.mod` 已包含 `gorm.io/driver/postgres` 和 `gorm.io/driver/mysql`。
- `backend/internal/database/database.go` 已按 `databaseType` 切换 sqlite/postgres/mysql 三种 dialector。
- `backend/internal/config/config.go` 的 `Config` 已有 `postgresUrl`、`mysqlUrl` 字段，`backend/README.md` 有配置文档。
- 模型层（Group/API/Parameter，位于 `backend/internal/models/`）全部使用 GORM 标准类型，无方言相关 SQL，PG/MySQL 兼容。
- 前端 API 调用使用相对路径 `/api`（`frontend/src/lib/api.ts`），前端嵌入后端二进制后同源服务，容器内无需处理跨域。
- 前端嵌入机制：`backend/internal/embedded/`，通过 `go build -tags embedfrontend` 把 `internal/embedded/frontend_dist/` 嵌入二进制；`backend/Makefile` 的 `package-cli` 已有完整构建流程（bun 构建前端 → 拷贝 dist → 带 tag 编译）。
- 全仓库无任何 Dockerfile / docker-compose 文件。

**存在的缺陷**

1. **方言相关 bug**：`database.go:63` 用 `SELECT count(*) > 0 FROM sqlite_master WHERE ...` 探测表是否存在，这是 SQLite 专属。在 PG/MySQL 上该查询报错（错误被静默吞掉），`tableExists` 恒为 false，导致每次启动都跑全量 `AutoMigrate`，对已有数据可能触发 ALTER，语义错误。
2. **配置不支持环境变量**：只有 `PORT`/`HOST` 在 `cmd/server/main.go` 中有环境变量覆盖，数据库配置只能走 `~/.knot/config.json` 文件，不适合容器场景。
3. **无真实 PG/MySQL 验证**：现有测试全部基于 SQLite in-memory。

## 3. 设计

### 3.1 后端改动（两处，均为小改）

**3.1a 修复方言相关的表探测 bug** — `backend/internal/database/database.go`

把第 60-65 行的 `sqlite_master` Raw 查询替换为 GORM 跨方言的 `db.Migrator().HasTable(&models.Group{})`。修复后三种数据库走同一正确逻辑：表不存在时全量 `AutoMigrate`；已存在时按现有 else 分支逐表 `HasTable`/`CreateTable`，不再对已有数据做全量迁移。

**3.1b 数据库配置支持环境变量** — `backend/internal/config/config.go` 的 `LoadConfig()`

在 `viper.Unmarshal` 之后追加环境变量覆盖（与 `main.go` 已有的 `PORT`/`HOST` 覆盖同一模式，放进 config 包使 CLI 和 server 都受益）：

| 环境变量 | 覆盖字段 |
|---|---|
| `KNOT_DATABASE_TYPE` | `databaseType` |
| `KNOT_SQLITE_PATH` | `sqlitePath` |
| `KNOT_POSTGRES_URL` | `postgresUrl` |
| `KNOT_MYSQL_URL` | `mysqlUrl` |

- 环境变量优先级高于配置文件；不设置时行为与现在完全一致（向后兼容）。
- 纯运行时覆盖，不写回配置文件。

### 3.2 Dockerfile（仓库根目录，多阶段）

构建上下文为仓库根（需同时访问 `frontend/` 与 `backend/`）。

- **Stage 1（`oven/bun:1`）**：复制 `frontend/`，执行 `bun install --frozen-lockfile && bun run build`，产出 `dist/`。
- **Stage 2（`golang:1.25-alpine`）**：复制 `backend/`，把 Stage 1 的 `dist/` 拷入 `internal/embedded/frontend_dist/`，执行 `CGO_ENABLED=0 go build -tags embedfrontend -ldflags="-s -w" -o /knot-server ./cmd/server`。SQLite 驱动为纯 Go（glebarez/modernc），无 CGO 依赖，可静态编译。
- **Stage 3（`alpine:3`）**：创建非 root 用户 `knot`，复制二进制，`USER knot`，`EXPOSE 3000`，入口 `/knot-server`。数据目录为该用户 home 下的 `~/.knot`（即 `/home/knot/.knot`），由 compose 挂卷持久化（SQLite 数据文件与日志均在此目录）。

配套在仓库根新增 `.dockerignore`，排除 `node_modules`、`dist`、`bin`、`.git` 等，缩小构建上下文。

### 3.3 Compose 文件（3 个文件，extends 复用）

数据库差异全部通过环境变量在运行时注入，因此 PG 与 MySQL 共用同一个镜像（tag `knot-app:latest`）。两个 compose 文件 build 相同的 context + Dockerfile + image tag，BuildKit 层缓存保证所有构建 stage 只执行一次；compose 服务定义通过 `extends` 复用。

**`docker-compose.base.yml`** — 只定义共用的 `app` 服务：

- `build` 指向根 Dockerfile，`image: knot-app:latest`
- `HOST=0.0.0.0`（容器内必须监听全部网卡，现有代码已支持该环境变量）
- 命名卷 `knot-data` 挂到 `/home/knot/.knot`
- healthcheck 打 `/api/health`（alpine 镜像自带 busybox wget）

**`docker-compose.pg.yml`** — PostgreSQL 栈：

- 顶层 `name: knot-pg`
- `app` 通过 `extends: {file: docker-compose.base.yml, service: app}` 复用，追加：
  - `KNOT_DATABASE_TYPE=postgres`
  - `KNOT_POSTGRES_URL=postgres://knot:knot@postgres:5432/knot?sslmode=disable`
  - `depends_on: postgres: condition: service_healthy`
  - 端口映射 `3000:3000`
- `postgres` 服务：`postgres:16-alpine`，`POSTGRES_USER=knot` / `POSTGRES_PASSWORD=knot` / `POSTGRES_DB=knot`（可用 `.env` 覆盖），命名卷 `pgdata`，healthcheck 用 `pg_isready`。不暴露端口到宿主机（仅容器网络内访问）。

**`docker-compose.mysql.yml`** — MySQL 栈，结构完全对称：

- 顶层 `name: knot-mysql`
- `app` extends base，追加：
  - `KNOT_DATABASE_TYPE=mysql`
  - `KNOT_MYSQL_URL=knot:knot@tcp(mysql:3306)/knot?charset=utf8mb4&parseTime=True`（README 中已有的 DSN 格式）
  - `depends_on: mysql: condition: service_healthy`
  - 端口映射 `3001:3000`
- `mysql` 服务：`mysql:8.0`，`MYSQL_ROOT_PASSWORD` / `MYSQL_DATABASE=knot` / `MYSQL_USER=knot` / `MYSQL_PASSWORD=knot`（可用 `.env` 覆盖），命名卷 `mysqldata`，healthcheck 用 `mysqladmin ping`。

两个栈的 project name 与端口刻意错开（`knot-pg`/`knot-mysql`、3000/3001），可同时运行互不影响，便于对比验证。

使用方式：

```bash
docker compose -f docker-compose.pg.yml up -d --build     # PG 版，访问 :3000
docker compose -f docker-compose.mysql.yml up -d --build  # MySQL 版，访问 :3001
```

SQLite 单机模式（不启动外部数据库）：在任一 compose 文件中去掉 `KNOT_*` 环境变量即可，compose 文件内附注释说明。

### 3.4 验证与测试

**后端**

- `go test ./...` 全绿。
- config 包新增环境变量覆盖的单测（设置/不设置 env 两种路径）。
- database 包现有测试不受影响（`HasTable` 对 SQLite 同样适用）。

**Docker 端到端（PG 与 MySQL 各走一遍）**

1. `docker compose -f docker-compose.<db>.yml up -d --build`
2. `curl localhost:<port>/api/health`，确认 `database.type` 为对应数据库且 status 为 ok。
3. 调 API 创建一个 group（`POST /api/groups`）。
4. `docker compose -f ... restart app`，再次查询确认数据仍在 —— 同时验证持久化和修复后的迁移路径不破坏已有数据。
5. 浏览器打开前端页面确认可用。
6. 构建第二个栈时确认 BuildKit 命中缓存（构建秒级完成）。

## 4. 明确不做（YAGNI）

- 不做前后端分容器、不引入 nginx；沿用"单二进制嵌入前端"部署模式。
- 不动 mcp-server（stdio 协议，不适合容器化，与本任务无关）。
- 不做 SQLite → PG/MySQL 数据迁移工具。
- 不发布镜像到 registry，只提供本地构建。
- 数据库密码仅用 compose `.env` 管理，不引入 secrets 编排。
- MySQL 固定用 `mysql:8.0` 官方镜像，不评估 MariaDB 等替代品。

## 5. 涉及文件清单

新增：

- `Dockerfile`（仓库根）
- `.dockerignore`（仓库根）
- `docker-compose.base.yml`（仓库根）
- `docker-compose.pg.yml`（仓库根）
- `docker-compose.mysql.yml`（仓库根）

修改：

- `backend/internal/database/database.go`（修复 sqlite_master 探测）
- `backend/internal/config/config.go`（环境变量覆盖）
- `backend/internal/config/config_test.go`（新增环境变量覆盖测试）

文档：

- `backend/README.md` 与根 `README.md` 补充 Docker 部署说明（如已有部署章节则就地更新）。
