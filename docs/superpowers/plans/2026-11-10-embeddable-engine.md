# Embeddable Knot Engine Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn the backend into an embeddable Go component (`knot.New(Config)` + `Mount` + direct service methods) a Fiber+GORM host (Sentia) initializes in-process with its own DB — shipped as `models/v1.1.0`, `backend/v0.1.0`, and product patch `v0.7.1`, with zero product behavior change.

**Architecture:** Approach A from the spec — in-place componentization: a new root package `knot` in the existing backend module exposes the facade; `SetupRoutes` gains a prefix-aware internal variant; hosts (CLI, cmd/server) keep viper/static-frontend/health and mount the engine; models v1.1 adds `SetTablePrefix` and relocates `ImportResult`.

**Tech Stack:** Go 1.25.5 workspace (backend/models/tools/mcp-server), Fiber v2, GORM + glebarez/sqlite (tests), GitHub Actions.

**Spec:** `docs/superpowers/specs/2026-11-10-embeddable-engine-design.md` (§7.5 corrections are binding)

## Global Constraints

- ALL code comments, commit messages, PR body, release notes in **English**; conventional commits.
- **Zero product behavior change**: binary behavior byte-identical; all existing backend tests, models tests, frontend tests (152), Playwright E2E (4 specs), and both coverage gates stay green.
- Engine facade file (`backend/knot.go` + its tests) falls under the backend 85% per-function coverage gate — real unit tests, not just smoke.
- Models v1.1 is purely additive (`SetTablePrefix`, `ImportResult` type, relocated types stay aliased in services) — default `TablePrefix` "" keeps existing deployments byte-identical.
- Engine ownership rules: never Close/mutate host DB; only DDL ever issued is `AutoMigrate` in `New()`; `Mount` idempotent.
- No new third-party dependencies.
- `backend/go.mod` `require github.com/ProjAnvil/knot/models v1.1.0` (bumped this branch; in-repo replace keeps everything green pre-tag).
- Work on branch `feat/embeddable-engine`.

## Review Focus

1. **Table-prefix leakage**: any code path that touches a physical table name without going through `TableName()` (verified none exist today — the gate is that T1's prefix round-trip test migrates, CRUDs, and physically checks `knot_groups` exists while bare `groups` does not). Pinned in Task 1.
2. **Facade/handler parity**: facade `Export`/`Import` must return byte-identical payloads/counts to the HTTP handlers on shared fixtures — parity tests, not just green paths. Pinned in Task 2.
3. **Mount idempotence + prefix correctness**: double `Mount` must not double-register; non-default prefix must isolate from host routes. Pinned in Task 2.
4. **Host-B smoke reality**: the CI smoke must use BOTH replaces (backend + models) — a single replace fails with unresolvable models. Pinned in Task 4.
5. **Default-prefix regression**: CLI host keeps `"/api"` and the frontend's hardcoded `API_BASE='/api'` keeps working — E2E is the proof; do not touch route defaults. Pinned in Task 3.

---

### Task 1: models v1.1 — table prefix + ImportResult relocation

**Files:**
- Modify: `models/group.go`, `models/api.go`, `models/parameter.go` (TableName prefix-driven), `models/import_plan.go` (add `ImportResult` with the exact JSON tags from `backend/internal/services/import_apply.go`), `models/README.md` (v1.1 notes)
- Create: `models/prefix.go` (`TablePrefix` var + `SetTablePrefix(p string)` + doc comment: set once before first DB use), `models/prefix_test.go`
- Modify: `backend/internal/services/import_apply.go` (`ImportResult` becomes alias `type ImportResult = models.ImportResult` — zero call-site churn)
- Test: `models/gormbind/gormbind_test.go` (+ prefix round-trip subtest)

**Interfaces:**
- Produces (consumed by Tasks 2-4 and models v1.1 consumers):
  - `func SetTablePrefix(p string)`; `TableName()` returns `TablePrefix + "groups"|"apis"|"parameters"`.
  - `models.ImportResult` struct with JSON tags `format/groupsCreated/groupsMerged/apisImported/parametersImported/warnings` (verbatim from services).
  - services keeps compiling via the alias; handler import tests untouched.

- [ ] **Step 1: Failing tests**: `TestSetTablePrefix` (default "" → "groups"; after SetTablePrefix("knot_") → "knot_groups"; restore "" via defer), `TestImportResultJSONTags` (marshal matches the v1.0 handler response byte-shape), gormbind prefix subtest (migrate with prefix → `knot_groups`/`knot_apis`/`knot_parameters` physically exist, bare tables absent, full CRUD round-trip on prefixed DB).
- [ ] **Step 2: RED verified** (`cd models && go test ./...` fails as expected).
- [ ] **Step 3: Implement** prefix.go + TableName changes + ImportResult + services alias. Prefix round-trip green; full models suite green; `make test-coverage` in models green (gate ≥85%).
- [ ] **Step 4: Backend still green**: `go test ./backend/...` (alias keeps everything compiling); `cd backend && make test-coverage` green.
- [ ] **Step 5: Commit** `feat: add table prefix support and relocate ImportResult to models`

### Task 2: Engine facade

**Files:**
- Create: `backend/knot.go` (package `knot`), `backend/knot_test.go`, `backend/go.mod` (require models v1.1.0)
- Modify: `backend/internal/handlers/routes.go` (`registerRoutes(app, db, prefix)` internal variant; `SetupRoutes` delegates with `"/api"`)

**Interfaces:**
```go
package knot // import knot "github.com/ProjAnvil/knot/backend"

type Config struct { DB *gorm.DB; APIPrefix string; TablePrefix string }
type Knot struct{ /* unexported: db, prefix, mounted */ }
func New(cfg Config) (*Knot, error)      // validate DB non-nil; SetTablePrefix (only if non-empty — do not clobber a host-set prefix); gormbind.AutoMigrate; wrap errors
func (k *Knot) Mount(app *fiber.App)     // idempotent guard; registerRoutes under cfg.APIPrefix (default "/api")
func (k *Knot) Export(apiIDs []uint, format string, locale string) ([]byte, string, error)
// byte-identical to handler output incl. Content-Type-appropriate payload and filename; format validated identically ("", "html" default)
func (k *Knot) Import(data []byte, dryRun bool) (*models.ImportResult, error)
// DetectImportFormat → parse → ApplyImport; error messages match handler mapping (unsupported → sentinel the host can map to 415)
func (k *Knot) Close() error             // no-op, documented
```
- Export implementation: extract the fetch+build block from `handlers/export.go` into an internal function both the handler and facade call (single source; no duplication). Import: facade calls `services.DetectImportFormat` + parser + `ApplyImport` directly (handler stays as is — multipart parsing is HTTP-only).

- [ ] **Step 1: Failing tests** (`backend/knot_test.go`): `TestNew` (nil DB → error; valid sqlite DB → migrate ran: tables exist), `TestMountPrefixAndIdempotence` (fiber app; routes reachable under custom prefix `/knot/api/groups` 200; double Mount → same route set, no panic/duplicates), `TestExportParity` (seeded fixture → facade Export bytes == handler POST response body for html/json/openapi/postman; filenames equal), `TestImportParity` (knot-json fixture → facade dryRun counts == handler dryRun response; real import persists).
- [ ] **Step 2: RED verified** (`go test ./backend/ ./backend/internal/...` — knot package undefined).
- [ ] **Step 3: Implement** facade + routes.go variant. All Step-1 tests green; full backend suite green; `make test-coverage` green incl. gate on knot.go functions.
- [ ] **Step 4: Commit** `feat: add embeddable engine facade with mountable routes and service methods`

### Task 3: Hosts rework

**Files:**
- Modify: `backend/internal/cli/serve.go`, `backend/cmd/server/main.go` (construct `knot.New(Config{DB: db, APIPrefix: "/api"})` + `Mount(app)` instead of direct `SetupRoutes`; static frontend + `/api/health` stay exactly where they are), `backend/internal/handlers/routes.go` (`SetupRoutes` retained as thin delegate — mcp-server or tests may call it)
- Test: existing suites are the proof; no new tests beyond an app-level smoke if trivial

- [ ] **Step 1: Rework hosts**; `go build ./... && go test ./backend/...` green.
- [ ] **Step 2: E2E proof (Review Focus 5)**: `cd frontend && bun run test:e2e` 4/4 green (frontend served by CLI host binary with default prefix).
- [ ] **Step 3: Commit** `refactor: mount engine from cli and server hosts`

### Task 4: CI host-B smoke (Sentia scenario)

**Files:**
- Modify: `.github/workflows/ci.yml` (backend job: new step after test-coverage)

**Interfaces:**
- Smoke script (inline in the step): `SMOKE=$(mktemp -d)`; `go mod init sentia-smoke`; TWO replaces — `github.com/ProjAnvil/knot/backend=$GITHUB_WORKSPACE/backend` AND `github.com/ProjAnvil/knot/models=$GITHUB_WORKSPACE/models` (backend's own replace is inert downstream); `main.go`: own fiber app + glebarez sqlite DB with a host table `sentia_users` (proves coexistence), `knot.New(Config{DB, APIPrefix:"/knot/api", TablePrefix:"knot_"})`, Mount, `app.Test` GET `/knot/api/groups` (expect 200 envelope), direct `Import` dry-run on a knot-json fixture (assert counts), physical check `knot_groups` exists && `groups` does not; assert host table untouched; `GOWORK=off go mod tidy && go build && go test ./...` (smoke as a Go test or main with os.Exit codes — follow the models-job smoke pattern).

- [ ] **Step 1: Replicate locally** (temp dir, local paths, GOWORK=off) — all assertions pass.
- [ ] **Step 2: Add ci.yml step**; YAML-validate.
- [ ] **Step 3: Commit** `ci: add embedded-host consumption smoke for the engine`

### Task 5: Documentation

**Files:**
- Create: `backend/README.md` — the engine's face: host contract (Config/New/Mount/service methods), ownership rules, table-prefix contract, `go get github.com/ProjAnvil/knot/backend@v0.1.0` + import alias, Sentia-style example
- Modify: `models/README.md` (v1.1: SetTablePrefix, ImportResult), `README.md` + `doc/README_zh.md` (short bilingual "Embeddable engine / 可嵌入引擎" section)

- [ ] **Step 1: Write; verify every path/claim against the code.**
- [ ] **Step 2: Commit** `docs: document the embeddable engine and models v1.1`

### Task 6: Full no-behavior-change verification

- [ ] **Step 1:** `cd backend && make test-coverage` (gate incl. knot.go) — green.
- [ ] **Step 2:** `cd models && make test-coverage` — green.
- [ ] **Step 3:** `cd frontend && bunx vitest run && bun run check && bun run build` — green.
- [ ] **Step 4:** `cd frontend && bun run test:e2e` — 4/4, twice.
- [ ] **Step 5:** `go vet ./...` in backend/models/tools; gofmt clean on changed files; drift check (`go run ./tools/cmd/gen-* && git diff --exit-code -- models/schema models/ts`) — NOTE: models v1.1 adds `ImportResult` to the root package, so the TS golden and possibly the schema CHANGE legitimately — regenerate and commit as part of this task with a dedicated commit `chore: regenerate schema and TS goldens for models v1.1`.
- [ ] **Step 6:** Fix anything broken (fix commit) or record "no fixes needed".

---

## Post-merge release runbook (controller)

1. CI green on main.
2. `git tag models/v1.1.0 && git push origin models/v1.1.0` → external smoke: temp module, `go get github.com/ProjAnvil/knot/models@v1.1.0`, exercise `SetTablePrefix` + `ImportResult`.
3. `gh release create models/v1.1.0 --latest=false` (English notes).
4. `cd backend && GOWORK=off go mod tidy` (require models v1.1.0 now resolvable) → fixup commit if go.sum changes → `git tag backend/v0.1.0` on it → push → external smoke: temp module, `go get github.com/ProjAnvil/knot/backend@v0.1.0` + `import knot "..."` + New/Mount compile-run.
5. `gh release create backend/v0.1.0 --latest=false` (English notes: v0 engine, host contract).
6. `git tag v0.7.1` → push → release.yml builds → `gh release edit v0.7.1 --notes-file` (English PATCH template: embeddable engine capability, no product behavior change; Full Changelog base `v0.7.0`).
7. Verify: 13 assets on v0.7.1, Latest badge on v0.7.1.
