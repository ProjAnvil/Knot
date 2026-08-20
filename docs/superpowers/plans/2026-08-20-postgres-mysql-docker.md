# PostgreSQL/MySQL 完善与 Docker Compose 部署 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复后端多数据库缺陷、增加环境变量配置能力，并提供 Dockerfile + PG/MySQL 两套 Docker Compose 全栈部署。

**Architecture:** 单二进制部署模式：多阶段 Dockerfile 先用 bun 构建 Svelte 前端，再嵌入 Go 二进制（`embedfrontend` build tag），最终 alpine 非 root 运行。数据库差异完全通过 `KNOT_*` 环境变量在运行时注入，PG 与 MySQL 共用同一镜像；compose 服务定义通过 `extends` 从 base 文件复用。

**Tech Stack:** Go 1.25 + GORM + Fiber（backend/）、Svelte 5 + Vite + bun（frontend/）、Docker / Docker Compose v2。

**Spec:** `docs/superpowers/specs/2026-08-20-postgres-mysql-docker-design.md`

## Global Constraints

- 不执行任何 `git commit` / `git push` 等 git 变更操作（除非用户明确要求）；所有改动留在工作区。
- 后端改动向后兼容：不设置 `KNOT_*` 环境变量时行为与现状完全一致。
- 环境变量名固定为：`KNOT_DATABASE_TYPE`、`KNOT_SQLITE_PATH`、`KNOT_POSTGRES_URL`、`KNOT_MYSQL_URL`（纯运行时覆盖，不写回配置文件）。
- MySQL DSN 格式固定为：`user:pass@tcp(host:3306)/db?charset=utf8mb4&parseTime=True`（与 `backend/README.md` 一致）。
- compose 镜像 tag 固定为 `knot-app:latest`，project name 分别为 `knot-pg` / `knot-mysql`，端口 3000 / 3001。
- 容器内监听地址通过现有 `HOST=0.0.0.0` 环境变量机制注入（`backend/cmd/server/main.go` 已支持）。
- Go 测试放在与被测文件同包（`package config` / `package database`），遵循 `backend/internal/config/config_test.go` 现有风格：plain `testing`、`resetViper()`、手动保存/恢复 `HOME` 环境变量。
- 不引入任何新依赖。

---

### Task 1: 修复 database.go 的方言相关表探测 bug

**Files:**
- Modify: `backend/internal/database/database.go:60-91`
- Test: `backend/internal/database/database_test.go`

**Interfaces:**
- Consumes: `config.Config`（已有）、`models.Group/API/Parameter`（已有）。
- Produces: `InitDatabase(cfg *config.Config) (*gorm.DB, error)` 签名不变；仅内部探测逻辑改为跨方言的 `db.Migrator().HasTable()`。

- [ ] **Step 1: 写回归测试（重复初始化已有数据库不破坏数据）**

在 `backend/internal/database/database_test.go` 末尾追加：

```go
// TestGivenExistingDatabase_whenInitDatabaseAgain_thenDataPreserved tests re-initialization path
func TestGivenExistingDatabase_whenInitDatabaseAgain_thenDataPreserved(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "reinit_test.db")

	cfg := &config.Config{
		DatabaseType: "sqlite",
		SQLitePath:   dbPath,
	}

	// First init: create tables and insert a row
	db, err := InitDatabase(cfg)
	if err != nil {
		t.Fatalf("first InitDatabase failed: %v", err)
	}
	group := models.Group{Name: "preserved-group"}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("failed to insert group: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.Close()

	// Second init on the same file: tables exist, must not fail or lose data
	db2, err := InitDatabase(cfg)
	if err != nil {
		t.Fatalf("second InitDatabase failed: %v", err)
	}
	var count int64
	if err := db2.Model(&models.Group{}).Where("name = ?", "preserved-group").Count(&count).Error; err != nil {
		t.Fatalf("failed to count groups: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 group after re-init, got %d", count)
	}
	sqlDB2, _ := db2.DB()
	sqlDB2.Close()
}
```

- [ ] **Step 2: 运行测试确认通过（SQLite 下修复前后都应通过，作为回归保护）**

Run: `cd backend && go test ./internal/database/ -run TestGivenExistingDatabase_whenInitDatabaseAgain_thenDataPreserved -v`
Expected: PASS（此测试是回归保护；真正的 PG/MySQL 路径由 Task 5 的 docker 端到端验证覆盖）

- [ ] **Step 3: 修改 `backend/internal/database/database.go`**

把第 60-65 行：

```go
	// Check if tables exist before running migration
	// This prevents migration issues with existing data
	var tableExists bool
	db.Raw("SELECT count(*) > 0 FROM sqlite_master WHERE type='table' AND name='groups'").Scan(&tableExists)
```

替换为：

```go
	// Check if tables exist before running migration
	// This prevents migration issues with existing data
	// Use Migrator.HasTable which works across SQLite/PostgreSQL/MySQL
	migrator := db.Migrator()
	tableExists := migrator.HasTable(&models.Group{})
```

同时把 else 分支第 73 行的 `migrator := db.Migrator()` 删除（变量已在上面声明）。

- [ ] **Step 4: 运行 database 包全部测试**

Run: `cd backend && go test ./internal/database/ -v`
Expected: 全部 PASS

---

### Task 2: config 包增加 KNOT_* 环境变量覆盖

**Files:**
- Modify: `backend/internal/config/config.go`（`LoadConfig()`，约 114-119 行附近）
- Test: `backend/internal/config/config_test.go`

**Interfaces:**
- Consumes: 无（独立改动）。
- Produces: `LoadConfig() (*Config, error)` 签名不变；新增行为：以下环境变量非空时覆盖对应字段 —— `KNOT_DATABASE_TYPE`→`DatabaseType`、`KNOT_SQLITE_PATH`→`SQLitePath`、`KNOT_POSTGRES_URL`→`PostgresURL`、`KNOT_MYSQL_URL`→`MySQLURL`。Task 4 的 compose 文件依赖这些变量名。

- [ ] **Step 1: 写失败测试**

在 `backend/internal/config/config_test.go` 末尾追加（沿用现有 HOME 覆盖风格）：

```go
// TestGivenEnvOverrides_whenLoadConfig_thenEnvValuesApplied tests KNOT_* env var overrides
func TestGivenEnvOverrides_whenLoadConfig_thenEnvValuesApplied(t *testing.T) {
	resetViper()

	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
		os.Unsetenv("KNOT_DATABASE_TYPE")
		os.Unsetenv("KNOT_POSTGRES_URL")
		os.Unsetenv("KNOT_SQLITE_PATH")
	}()

	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	os.Setenv("KNOT_DATABASE_TYPE", "postgres")
	os.Setenv("KNOT_POSTGRES_URL", "postgres://u:p@db:5432/knot")
	os.Setenv("KNOT_SQLITE_PATH", "/custom/knot.db")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.DatabaseType != "postgres" {
		t.Errorf("expected DatabaseType 'postgres', got '%s'", cfg.DatabaseType)
	}
	if cfg.PostgresURL != "postgres://u:p@db:5432/knot" {
		t.Errorf("expected PostgresURL from env, got '%s'", cfg.PostgresURL)
	}
	if cfg.SQLitePath != "/custom/knot.db" {
		t.Errorf("expected SQLitePath from env, got '%s'", cfg.SQLitePath)
	}
}

// TestGivenNoEnvOverrides_whenLoadConfig_thenDefaultsUnchanged tests backward compatibility
func TestGivenNoEnvOverrides_whenLoadConfig_thenDefaultsUnchanged(t *testing.T) {
	resetViper()

	originalHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("HOME", originalHome)
	}()

	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.DatabaseType != "sqlite" {
		t.Errorf("expected default DatabaseType 'sqlite', got '%s'", cfg.DatabaseType)
	}
	if cfg.PostgresURL != "" {
		t.Errorf("expected empty PostgresURL, got '%s'", cfg.PostgresURL)
	}
}
```

- [ ] **Step 2: 运行测试确认第一个失败**

Run: `cd backend && go test ./internal/config/ -run 'TestGivenEnvOverrides|TestGivenNoEnvOverrides' -v`
Expected: `TestGivenEnvOverrides_whenLoadConfig_thenEnvValuesApplied` FAIL（env 值未被应用）；`TestGivenNoEnvOverrides_whenLoadConfig_thenDefaultsUnchanged` PASS

- [ ] **Step 3: 修改 `backend/internal/config/config.go`**

在 `LoadConfig()` 中 `viper.Unmarshal(&config)` 之后、`return &config, nil` 之前插入：

```go
	// Environment variable overrides (useful for container deployments).
	// These take precedence over the config file and are not persisted.
	if v := os.Getenv("KNOT_DATABASE_TYPE"); v != "" {
		config.DatabaseType = v
	}
	if v := os.Getenv("KNOT_SQLITE_PATH"); v != "" {
		config.SQLitePath = v
	}
	if v := os.Getenv("KNOT_POSTGRES_URL"); v != "" {
		config.PostgresURL = v
	}
	if v := os.Getenv("KNOT_MYSQL_URL"); v != "" {
		config.MySQLURL = v
	}
```

（`os` 包已在该文件 import 中，无需新增 import。）

- [ ] **Step 4: 运行 config 包全部测试**

Run: `cd backend && go test ./internal/config/ -v`
Expected: 全部 PASS

- [ ] **Step 5: 运行后端全部测试**

Run: `cd backend && go test ./...`
Expected: 全部 PASS

---

### Task 3: 根目录 Dockerfile 与 .dockerignore

**Files:**
- Create: `Dockerfile`（仓库根）
- Create: `.dockerignore`（仓库根）

**Interfaces:**
- Consumes: `frontend/package.json`（build 脚本为 `vite build`）、`frontend/bun.lock`、`backend/` Go 源码、`backend/internal/embedded/`（`embedfrontend` tag 嵌入 `internal/embedded/frontend_dist/`）。
- Produces: 镜像 `knot-app:latest`，入口 `/knot-server`，监听 3000，数据目录 `/home/knot/.knot`。Task 4 的 compose 文件依赖此镜像与目录。

- [ ] **Step 1: 创建 `Dockerfile`（仓库根）**

```dockerfile
# syntax=docker/dockerfile:1

# Stage 1: build frontend
FROM oven/bun:1 AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/bun.lock ./
RUN bun install --frozen-lockfile
COPY frontend/ ./
RUN bun run build

# Stage 2: build backend with embedded frontend
FROM golang:1.25-alpine AS backend
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
# Overwrite any stale prebuilt assets with the fresh frontend build
RUN rm -rf internal/embedded/frontend_dist
COPY --from=frontend /app/frontend/dist ./internal/embedded/frontend_dist
RUN CGO_ENABLED=0 go build -tags embedfrontend -ldflags="-s -w" -o /knot-server ./cmd/server

# Stage 3: runtime
FROM alpine:3
RUN adduser -D -h /home/knot knot && mkdir -p /home/knot/.knot && chown -R knot:knot /home/knot
COPY --from=backend /knot-server /knot-server
USER knot
EXPOSE 3000
ENTRYPOINT ["/knot-server"]
```

- [ ] **Step 2: 创建 `.dockerignore`（仓库根）**

```
.git
.github
.vscode
.claude
docs
doc
mcp-server
**/node_modules
frontend/dist
backend/bin
backend/dist
**/.DS_Store
```

- [ ] **Step 3: 验证镜像构建**

Run: `docker build -t knot-app:latest .`
Expected: 构建成功；最后输出 `naming to docker.io/library/knot-app:latest`

- [ ] **Step 4: 冒烟验证镜像可启动（SQLite 默认模式）**

Run:
```bash
docker run -d --name knot-smoke -p 3100:3000 -e HOST=0.0.0.0 knot-app:latest
sleep 3
curl -s http://localhost:3100/api/health
```
Expected: 返回 JSON，`"status":"ok"`，`database.type` 为 `"sqlite"`

清理：
```bash
docker rm -f knot-smoke
```

---

### Task 4: 三个 docker-compose 文件

**Files:**
- Create: `docker-compose.base.yml`（仓库根）
- Create: `docker-compose.pg.yml`（仓库根）
- Create: `docker-compose.mysql.yml`（仓库根）

**Interfaces:**
- Consumes: Task 3 的镜像构建定义（`build.context`/`dockerfile`）；Task 2 的 `KNOT_DATABASE_TYPE`/`KNOT_POSTGRES_URL`/`KNOT_MYSQL_URL` 环境变量；现有 `HOST` 环境变量机制。
- Produces: `docker compose -f docker-compose.pg.yml up -d`（:3000）与 `docker compose -f docker-compose.mysql.yml up -d`（:3001）两个可运行全栈。

- [ ] **Step 1: 创建 `docker-compose.base.yml`**

```yaml
# Shared app service definition. Not meant to be used directly;
# extend it from docker-compose.pg.yml or docker-compose.mysql.yml.
services:
  app:
    build:
      context: .
      dockerfile: Dockerfile
    image: knot-app:latest
    environment:
      HOST: 0.0.0.0
      # SQLite standalone mode: remove the KNOT_* variables in the extending
      # file and the app falls back to SQLite persisted in the knot-data volume.
    volumes:
      - knot-data:/home/knot/.knot
    healthcheck:
      test: ["CMD", "wget", "-q", "-O", "/dev/null", "http://localhost:3000/api/health"]
      interval: 10s
      timeout: 3s
      retries: 5
      start_period: 10s
    restart: unless-stopped

volumes:
  knot-data:
```

- [ ] **Step 2: 创建 `docker-compose.pg.yml`**

```yaml
name: knot-pg

services:
  app:
    extends:
      file: docker-compose.base.yml
      service: app
    environment:
      KNOT_DATABASE_TYPE: postgres
      KNOT_POSTGRES_URL: postgres://${POSTGRES_USER:-knot}:${POSTGRES_PASSWORD:-knot}@postgres:5432/${POSTGRES_DB:-knot}?sslmode=disable
    ports:
      - "3000:3000"
    depends_on:
      postgres:
        condition: service_healthy

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: ${POSTGRES_USER:-knot}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-knot}
      POSTGRES_DB: ${POSTGRES_DB:-knot}
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U $${POSTGRES_USER} -d $${POSTGRES_DB}"]
      interval: 5s
      timeout: 3s
      retries: 10
    restart: unless-stopped

volumes:
  knot-data:
  pgdata:
```

- [ ] **Step 3: 创建 `docker-compose.mysql.yml`**

```yaml
name: knot-mysql

services:
  app:
    extends:
      file: docker-compose.base.yml
      service: app
    environment:
      KNOT_DATABASE_TYPE: mysql
      KNOT_MYSQL_URL: ${MYSQL_USER:-knot}:${MYSQL_PASSWORD:-knot}@tcp(mysql:3306)/${MYSQL_DATABASE:-knot}?charset=utf8mb4&parseTime=True
    ports:
      - "3001:3000"
    depends_on:
      mysql:
        condition: service_healthy

  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD:-root}
      MYSQL_DATABASE: ${MYSQL_DATABASE:-knot}
      MYSQL_USER: ${MYSQL_USER:-knot}
      MYSQL_PASSWORD: ${MYSQL_PASSWORD:-knot}
    volumes:
      - mysqldata:/var/lib/mysql
    healthcheck:
      test: ["CMD-SHELL", "mysqladmin ping -h localhost -u root -p$${MYSQL_ROOT_PASSWORD}"]
      interval: 5s
      timeout: 3s
      retries: 10
      start_period: 30s
    restart: unless-stopped

volumes:
  knot-data:
  mysqldata:
```

- [ ] **Step 4: 校验 compose 配置语法**

Run:
```bash
docker compose -f docker-compose.pg.yml config -q
docker compose -f docker-compose.mysql.yml config -q
```
Expected: 两条命令均无输出、退出码 0

---

### Task 5: Docker 端到端验证（PG 与 MySQL 各一遍）

**Files:**
- 无新增/修改（纯验证任务；如发现缺陷则修复对应文件并重跑）

**Interfaces:**
- Consumes: Task 3 的镜像、Task 4 的两个 compose 文件。
- Produces: 两套栈的验证结论。

- [ ] **Step 1: 启动 PG 栈**

Run: `docker compose -f docker-compose.pg.yml up -d --build`
Expected: `knot-pg-postgres-1` 与 `knot-pg-app-1` 启动；`docker compose -f docker-compose.pg.yml ps` 显示两者 healthy

- [ ] **Step 2: 验证 PG 健康检查与建表**

Run: `curl -s http://localhost:3000/api/health`
Expected: `"status":"ok"` 且 `"database":{"status":"ok","type":"postgres"}`

- [ ] **Step 3: 验证 PG CRUD 与重启持久化**

Run:
```bash
curl -s -X POST http://localhost:3000/api/groups -H 'Content-Type: application/json' -d '{"name":"e2e-pg"}'
docker compose -f docker-compose.pg.yml restart app
sleep 5
curl -s http://localhost:3000/api/groups
```
Expected: 最后一次输出中包含 `e2e-pg`（同时验证修复后的迁移路径对已有数据无害）

- [ ] **Step 4: 验证 PG 栈前端页面**

Run: `curl -s http://localhost:3000/ | head -5`
Expected: 返回 `index.html`（含 `<!doctype html` 或 `<html`）

- [ ] **Step 5: 停掉 PG 栈（保留卷，避免端口占用）**

Run: `docker compose -f docker-compose.pg.yml down`
Expected: 容器与网络移除，卷保留（不加 `-v`，数据卷留作演示；如用户要求再清理）

- [ ] **Step 6: MySQL 栈重复 Step 1-5**

Run: `docker compose -f docker-compose.mysql.yml up -d --build`
注意点：
- 确认 build 阶段命中缓存（frontend/backend stage 显示 `CACHED`，秒级完成）。
- health 期望 `"type":"mysql"`，端口为 3001，group 名用 `e2e-mysql`。
- MySQL 首次初始化较慢，必要时增加等待/重试。
结束后：`docker compose -f docker-compose.mysql.yml down`

---

### Task 6: 文档更新

**Files:**
- Modify: `README.md`（仓库根，新增/更新部署章节）
- Modify: `backend/README.md`（环境变量配置说明）

**Interfaces:**
- Consumes: Task 2 的环境变量名、Task 4 的 compose 文件名与端口。
- Produces: 用户可照做的部署文档。

- [ ] **Step 1: 根 `README.md` 增加 Docker 部署章节**

内容要点（插入到合适的部署/安装章节，沿用文件现有语言风格——若为英文则写英文）：

```markdown
## Docker Deployment

The frontend is embedded into the backend binary at image build time, so a
single container serves the full stack. Two Compose stacks are provided —
PostgreSQL and MySQL — sharing one image via a common base file:

```bash
# PostgreSQL stack on http://localhost:3000
docker compose -f docker-compose.pg.yml up -d --build

# MySQL stack on http://localhost:3001
docker compose -f docker-compose.mysql.yml up -d --build
```

Database credentials default to `knot`/`knot` and can be overridden through a
`.env` file (`POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`,
`MYSQL_USER`, `MYSQL_PASSWORD`, `MYSQL_DATABASE`, `MYSQL_ROOT_PASSWORD`).

For a standalone SQLite container, remove the `KNOT_*` environment variables
from the compose file; data persists in the `knot-data` volume.
```

（若根 README 为中文结构则相应写中文；先读文件确认。）

- [ ] **Step 2: `backend/README.md` 配置章节补充环境变量**

在其数据库配置章节（约 109-140 行附近）之后追加：

```markdown
#### Environment variable overrides

All database settings can be overridden at runtime via environment variables
(useful for containers). These take precedence over `config.json` and are not
persisted:

| Variable | Config field |
|---|---|
| `KNOT_DATABASE_TYPE` | `databaseType` |
| `KNOT_SQLITE_PATH` | `sqlitePath` |
| `KNOT_POSTGRES_URL` | `postgresUrl` |
| `KNOT_MYSQL_URL` | `mysqlUrl` |
```

- [ ] **Step 3: 校验文档与实现一致**

Run: `grep -n "KNOT_" backend/README.md README.md docker-compose.pg.yml docker-compose.mysql.yml`
Expected: 变量名四处一致，无拼写分歧

---

## 完成后检查清单

- [ ] `cd backend && go test ./...` 全部通过
- [ ] `docker build -t knot-app:latest .` 成功
- [ ] `docker compose -f docker-compose.pg.yml config -q` 与 mysql 版本均无错
- [ ] PG 栈端到端验证通过（health/CRUD/重启持久化/前端页面）
- [ ] MySQL 栈端到端验证通过（同上）
- [ ] 文档变量名与实现一致
- [ ] 所有改动未提交，留在工作区（不做 git commit）
