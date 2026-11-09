# Knot Models Library Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract Knot's data model into a publishable Go module (`github.com/ProjAnvil/knot/models`, tagged `models/v1.0.0`) with layered dependency boundaries, generated JSON Schema + TypeScript artifacts, and CI drift locks — with zero behavior change to the product.

**Architecture:** New root-level `models/` Go module holds domain types, the Knot JSON v1 wire types, and pure helpers (stdlib-only root package); `models/gormbind` subpackage carries GORM helpers; `tools/` module hosts schema/TS generators whose deps never touch the library; `go.work` (backend, models, tools, mcp-server) wires local + CI development; backend imports the library and deletes its internal copy.

**Tech Stack:** Go 1.25 multi-module workspace, GORM (tags only in root / import only in gormbind), invopop/jsonschema + gzuidhof/tygo (tools module), GitHub Actions.

**Spec:** `docs/superpowers/specs/2026-11-10-models-library-design.md`

## Global Constraints

- ALL code comments, commit messages, PR body, release notes in **English**; conventional commits.
- **Zero behavior change**: Knot JSON v1 output byte-identical; DB migration model list/order identical; all existing backend tests, frontend tests (152), and Playwright E2E (4 specs) green.
- Root `models` package imports **stdlib only** (GORM stays as inert struct tags); `gorm.io/gorm` is imported only in `models/gormbind` (module `go.mod` will list gorm + test-only sqlite — consumers of the root package don't download them, thanks to module graph pruning).
- No JSON field-name or type renames — relocation, not redesign.
- `go.work` must `use` `./backend ./models ./tools ./mcp-server` (release.yml builds mcp-server under the workspace).
- Test names follow existing conventions; run Go via `cd backend && go test ./...` etc. (workspace mode). RULING (Task 1, empirically verified): backend/go.mod carries a PERMANENT dev-time `replace github.com/ProjAnvil/knot/models => ../models` — an unpublished require is unresolvable in module graph resolution even under go.work once any proxy dep exists; the replace is consumer-inert and keeps Docker/CI/tag builds working.
- Coverage: backend Makefile keeps its gate but its pattern must stop matching moved-out functions (T2); the models module gets its own `make test-coverage` gate (T3/T5).
- Work on branch `feat/models-library`.

## Review Focus

1. **`go mod tidy` under `go.work` with an unpublished version** — `backend/go.mod` requires `models v1.0.0` before the tag exists; workspace builds/tests must succeed without tidy fetching it. Dev flow: hand-edit `go.mod` (require only; go.sum entries added at the §10.5 bootstrap after tagging). T1 verifies workspace builds/tests. Note: `go.work` must declare `go 1.25.5` (the max module go directive — `1.25` fails), and root-level `go build ./...` does NOT work in a multi-module workspace — use explicit module patterns (`./backend/... ./models/...`) or per-module `cd`. Pinned in Task 1.
2. **Backend Makefile gate after the move** — `param_node` matches zero lines once param_node.go is deleted, but `export_knot`/`export_openapi`/`export_postman`/`import_*`/handlers still match, so the guard never fires; the pattern stays UNCHANGED except dropping the dead `param_node|` token. Do not un-gate live code. Pinned in Task 2.
3. **Test split scope** — `setupTestDB` (defined in `group_test.go`, importing gorm+sqlite) is called by ~26 test functions across all THREE test files, not just group_test.go. Every `setupTestDB`-calling test is removed from the root package in T1 and re-homed in T3's `gormbind_test.go`; only stdlib-only tests (marshal/type behavior) stay in the root. Pinned in Tasks 1 and 3.
4. **Generator pinning** — `tools/go.mod` pins exact generator versions; goldens committed and CI-diffed. Pinned in Task 4.
5. **`--latest=false` on the library release** so it never steals the product's Latest badge (runbook step; recorded in Task 6's README runbook section).

---

### Task 1: Models module skeleton + domain types move

**Files:**
- Create: `models/go.mod`, `models/api.go`, `models/group.go`, `models/parameter.go`, `models/constants.go`, `models/constants_test.go`, `models/api_test.go` (stdlib-only subset), `models/group_test.go` (stdlib-only subset), `models/parameter_test.go` (stdlib-only subset), `models/Makefile` (minimal `test` target for now)
- Create: `go.work` (declares `go 1.25.5`), `tools/go.mod` (placeholder `module github.com/ProjAnvil/knot/tools`)
- Delete: `backend/internal/models/` (whole dir)
- Modify: every backend file importing `internal/models` — non-test: `database/database.go`, handlers `apis.go` `export.go` `groups.go` `mcp_tools.go`, services `export_openapi.go` `export_postman.go` `export_service.go` `import_apply.go` `param_node.go` `parameter_service.go` — PLUS ~14 `_test.go` files (find the full set with `grep -rln 'internal/models' backend --include='*.go'`); `backend/go.mod` (+`require github.com/ProjAnvil/knot/models v1.0.0`, hand-edited — no tidy pre-tag)

**Interfaces:**
- Produces: module `github.com/ProjAnvil/knot/models`, package `models`, exporting exactly what `backend/internal/models` exported (`API`, `Group`, `Parameter` + methods) PLUS `constants.go`:

```go
// constants.go
var HTTPMethods = []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
var APITypes = []string{"HTTP", "RPC"}
var ParamTypes = []string{"request", "response"}
var ParamValueTypes = []string{"string", "number", "boolean", "array", "object"}
func IsValidMethod(m string) bool   // membership, case-sensitive
func IsValidAPIType(t string) bool
func IsValidParamType(t string) bool
func IsValidParamValueType(t string) bool
```

- `go.work`: `go 1.25.5` + `use (./backend ./models ./tools ./mcp-server)` — `tools/go.mod` placeholder now; generators come in Task 4. Generate via `go work init ./backend ./models ./tools ./mcp-server` to get a valid go directive, or hand-write with 1.25.5.

- [ ] **Step 1: Write failing tests** for `constants.go` (`TestIsValidMethod` etc. — new file `models/constants_test.go`).
- [ ] **Step 2: Verify RED** (`cd models && go test ./...` — package doesn't exist yet).
- [ ] **Step 3: Implement the move**: `git mv backend/internal/models/*.go models/`; strip from the three test files EVERY test function that calls `setupTestDB` (~26 across api/group/parameter_test.go — `setupTestDB` itself and its gorm/sqlite imports go too; these tests are re-homed in Task 3) keeping only stdlib-only tests; fix ALL importers (incl. `_test.go`) to `import models "github.com/ProjAnvil/knot/models"`; hand-add the require to `backend/go.mod`; write `go.work`.
- [ ] **Step 4: Verify GREEN (Review Focus 1)**: `go build ./backend/... ./models/... && go test ./backend/... ./models/...` (from repo root, explicit module patterns — bare `./...` fails in a workspace); `cd mcp-server && go build ./...` still builds (go.work uses it). `cd backend && make test-coverage` — gate still green (param_node etc. still in services until Task 2).
- [ ] **Step 5: Commit** `refactor: extract domain models into publishable models module`

### Task 2: Wire-format types move

**Files:**
- Create: `models/param_node.go` (+ test), `models/wire.go` (+ test), `models/import_plan.go`
- Delete: `backend/internal/services/param_node.go` ENTIRELY (it contains only `ParamNode`, the private helper `paramNodeFromValue`, and the four functions — all of which move); type definition blocks removed from `services/export_knot.go`, `services/import_knot.go`, `services/import_detect.go` (their functions stay)
- Modify: every services file referencing the moved types; `backend/Makefile` (gate pattern: drop ONLY the now-dead `param_node|` token); services test files (move type-level tests, keep round-trip tests in backend)

**Interfaces:**
- Moves verbatim (names, fields, JSON tags unchanged): `ParamNode`, `paramNodeFromValue` (private), `TreeFromParams`, `ParamNodesFromJSONValue`, `CountNodes`; `KnotExportDocument/Group/API/Params` + new consts `WireFormat = "knot"`, `WireVersion = 1` (services literals `"knot"`/`1` switch to the consts — output unchanged); `ImportPlan`, `ImportGroup`, `ImportAPI`, `ImportFormat` + its three consts.
- Services files keep every function (`BuildKnotDocument`, `ParseKnotJSON`, `DetectImportFormat`, `ApplyImport`, …) now referencing `models.<Type>`.
- `backend/Makefile` gate: pattern stays otherwise UNCHANGED — `export_knot`/`export_openapi`/`export_postman`/`import_*`/handlers still gate live code; only the `param_node|` token dies with the deleted file. Confirm the "no functions matched" guard does not fire (it won't — remaining tokens still match).
- [ ] **Step 1: Move with `git mv` semantics** (param_node.go moves whole; the others lose only their type blocks), update references; no go.mod change (same module already required).
- [ ] **Step 2: Verify** (Review Focus 2): `go build ./backend/... ./models/... && go test ./backend/... ./models/...`; `cd backend && make test-coverage` green with the one-token pattern fix; wire-format byte check: existing `TestBuildKnotDocument` golden assertions pass unmodified.
- [ ] **Step 3: Commit** `refactor: move wire-format and param tree types into models module`

### Task 3: gormbind subpackage + database wiring

**Files:**
- Create: `models/gormbind/gormbind.go`, `models/gormbind/gormbind_test.go`, `models/Makefile` (`test`, `test-coverage` with `-coverpkg=./...` + 85% awk gate over the module's functions)
- Modify: `backend/internal/database/database.go` (model list sourced from `gormbind.MigrateModels()`; conditional logic unchanged); `models/go.mod` (gorm require + glebarez/sqlite test dep)

**Interfaces:**

```go
// gormbind
func MigrateModels() []any                    // [&models.Group{}, &models.API{}, &models.Parameter{}] — same order as database.go today
func AutoMigrate(db *gorm.DB) error           // db.AutoMigrate(MigrateModels()...)
```

- `gormbind_test.go`: re-homes the ~26 `setupTestDB`-calling behavioral tests rescued in Task 1 from ALL THREE model test files (api/group/parameter: GORM tags, unique-name constraint, cascade, index, query ops) — importing `gormbind`'s own sqlite-backed `setupTestDB` — plus new `TestMigrateModels` (model list order) and `TestAutoMigrate` (in-memory sqlite, succeeds, idempotent double-run).
- [ ] **Step 1: Failing tests** (`cd models && go test ./gormbind/...` — package missing).
- [ ] **Step 2: RED verified.** Implement; wire `database.go`; `models/go.mod` gains `gorm.io/gorm` + test-only `github.com/glebarez/sqlite`.
- [ ] **Step 3: Verify GREEN**: models suite + `make test-coverage` (models gate green, ≥85% on gated functions); backend full suite green; `cd backend && go build ./...` under GOWORK=off still resolves? NO — v1.0.0 unpublished (expected; workspace only). Record in report.
- [ ] **Step 4: Commit** `feat: add gormbind subpackage and wire database migration`

### Task 4: Schema + TypeScript generators and goldens

**Files:**
- Create: `tools/cmd/gen-schema/main.go`, `tools/cmd/gen-ts/main.go`, `models/schema/knot-v1.schema.json` (generated), `models/ts/index.d.ts` (generated), `tools/README.md` (how to regen)
- Modify: `tools/go.mod` (pin `github.com/invopop/jsonschema` + `github.com/gzuidhof/tygo` at exact versions)

**Interfaces:**
- `gen-schema`: reflect over `models.KnotExportDocument` via invopop/jsonschema (draft 2020-12), `$id` = `https://github.com/ProjAnvil/knot/models/schema/knot-v1.schema.json`, then post-process: pin `properties.format.const = "knot"`, `properties.version.const = 1`. Deterministic output (sorted keys) → written to `models/schema/knot-v1.schema.json`.
- `gen-ts`: tygo config (YAML `packages:` with `path` + `output_path`, NOT `outPath`) targeting package `models` root only → `models/ts/index.d.ts`; the wrapper prepends the "Code generated by tools/cmd/gen-ts. DO NOT EDIT." header itself.
- Both run from repo root: `go run ./tools/cmd/gen-schema` / `go run ./tools/cmd/gen-ts` (workspace resolves models).

- [ ] **Step 1: Implement both generators; generate goldens; commit goldens.**
- [ ] **Step 2: Verify (Review Focus 4)**: regenerate twice → `git diff --exit-code -- models/schema models/ts` clean (determinism). Schema correctness is pinned by construction — the schema is reflection over the same types the round-trip tests exercise — plus the drift lock; do NOT add a JSON-Schema-validator dependency for this.
- [ ] **Step 3: Commit** `feat: add JSON Schema and TypeScript generators with committed goldens`

### Task 5: CI — models job + drift + consumption smoke

**Files:**
- Modify: `.github/workflows/ci.yml`

**Interfaces:**
- New `models` job (runs with the other three): checkout; setup-go (`go-version-file: models/go.mod`; note GOTOOLCHAIN=auto may download the go.work toolchain version — acceptable); `cd models && go build ./... && go test ./...`; `make test-coverage` in `models/` (gate); drift lock FROM REPO ROOT (not models/): `go run ./tools/cmd/gen-schema && go run ./tools/cmd/gen-ts && git diff --exit-code -- models/schema models/ts`; consumption smoke: `SMOKE=$(mktemp -d) && cd $SMOKE && go mod init smoke && go mod edit -replace github.com/ProjAnvil/knot/models=$GITHUB_WORKSPACE/models`, write `main.go` importing `models` (Group literal + `models.IsValidMethod("GET")`) and `models/gormbind` (`len(gormbind.MigrateModels()) == 3`), then `GOWORK=off go mod tidy && GOWORK=off go build ./...` (tidy is REQUIRED — a replace without a require yields "no required module provides package"); proves the module graph is consumable without the workspace.
- Existing `backend` job stays workspace-based; `GOWORK=off go build ./...` in backend WORKS now (committed replace resolves ../models) — the ci.yml comment should state the replace is what makes module-mode builds possible pre-tag.
- [ ] **Step 1: Write ci.yml changes; YAML-validate.**
- [ ] **Step 2: Verify locally**: replicate the models job steps in a shell script run (all pass, incl. smoke with temp dir + replace + tidy + GOWORK=off).
- [ ] **Step 3: Commit** `ci: add models module job with drift lock and consumption smoke`

### Task 6: Documentation

**Files:**
- Create: `models/README.md` (library usage: go get, semver policy, artifact pointers, regen instructions, `--latest=false` release runbook note)
- Modify: `README.md` + `doc/README_zh.md` (short bilingual section under a "Library / 模型库" heading pointing to `models/`)

- [ ] **Step 1: Write; verify links/paths exist.**
- [ ] **Step 2: Commit** `docs: document the models library`

### Task 7: Full no-behavior-change verification

**Files:** none (verification only; fixes allowed if gates fail — via a fix commit)

- [ ] **Step 1:** `cd backend && make test-coverage` green (updated gate).
- [ ] **Step 2:** `cd models && make test-coverage` green.
- [ ] **Step 3:** `cd frontend && bunx vitest run && bun run check && bun run build` green.
- [ ] **Step 4:** `cd frontend && bun run test:e2e` 4/4 green (twice).
- [ ] **Step 5:** `go vet ./...` in backend, models, tools; `gofmt -l` on new files only clean.
- [ ] **Step 6:** Commit any fixes; else record "no fixes needed" in the task report.

---

## Post-merge release runbook (executed by controller after PR merge)

1. CI green on main.
2. `git tag models/v1.0.0 && git push origin models/v1.0.0`.
3. External smoke OUTSIDE the repo: temp module, `go get github.com/ProjAnvil/knot/models@v1.0.0`, compile Group + gormbind usage (records the real publish works).
4. `gh release create models/v1.0.0 --latest=false --title "models v1.0.0" --notes-file <short English notes>`.
5. (MOOT per Task-1 ruling: the committed replace resolves the sibling locally; directory replaces need no go.sum entries. Only if the replace is ever dropped does this bootstrap revive.)
6. `git tag v0.7.0` → push → `release.yml` builds; `gh release edit v0.7.0 --notes-file <English template, Full Changelog base v0.6.0>`.
7. Verify: `gh release view v0.7.0 --json assets` (13 assets), Latest badge on v0.7.0 not the models release.
