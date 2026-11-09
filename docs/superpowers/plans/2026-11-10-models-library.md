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
- Test names follow existing conventions; run Go via `cd backend && go test ./...` etc. (workspace mode); never commit a `replace` for the models module in `backend/go.mod`.
- Coverage: backend Makefile keeps its gate but its pattern must stop matching moved-out functions (T2); the models module gets its own `make test-coverage` gate (T3/T5).
- Work on branch `feat/models-library`.

## Review Focus

1. **`go mod tidy` under `go.work` with an unpublished version** — `backend/go.mod` requires `models v1.0.0` before the tag exists; workspace builds/tests must succeed without tidy fetching it. If tidy fails pre-tag, the dev flow is: edit `go.mod` by hand (require + go.sum entries added at the §10.5 bootstrap after tagging), and T1 must verify `go build ./... && go test ./...` both work under the workspace. Pinned in Task 1.
2. **Backend Makefile gate false-trip after the move** — once `param_node`/`export_knot` functions leave the backend module, the old grep matches fewer/no lines and the v0.6.0 "no functions matched" guard fires. Pinned in Task 2.
3. **`group_test.go`'s GORM-tag tests** must land in `gormbind_test.go` (they import gorm + sqlite), keeping the root package's moved tests stdlib-only. Pinned in Task 3.
4. **Generator pinning** — `tools/go.mod` must pin exact generator versions; regenerated goldens are committed and CI-diffed. Pinned in Task 4.
5. **`--latest=false` on the library release** so it never steals the product's Latest badge (release runbook step, not a code task; recorded in Task 6's README runbook section).

---

### Task 1: Models module skeleton + domain types move

**Files:**
- Create: `models/go.mod`, `models/api.go`, `models/group.go`, `models/parameter.go`, `models/constants.go`, `models/api_test.go`, `models/group_test.go` (stdlib-only subset), `models/parameter_test.go` (stdlib-only subset), `models/Makefile` (minimal `test` target for now)
- Create: `go.work`
- Delete: `backend/internal/models/` (whole dir)
- Modify: all 11 backend importers (`backend/internal/database/database.go`, handlers ×4: `groups.go` `apis.go` `export.go` `mcp_tools.go`, services ×6: `export_service.go` `export_knot.go` `export_openapi.go` `export_postman.go` `param_node.go` `parameter_service.go` `import_apply.go` — verify with grep), `backend/go.mod` (+`require github.com/ProjAnvil/knot/models v1.0.0`)

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

- `go.work`: `go 1.25` + `use (./backend ./models ./tools ./mcp-server)` — create `tools/go.mod` as a placeholder module now (`module github.com/ProjAnvil/knot/tools`) so go.work is valid; generators come in Task 4.

- [ ] **Step 1: Write failing tests** for `constants.go` (`TestIsValidMethod` etc. — move-ready new file `models/constants_test.go`).
- [ ] **Step 2: Verify RED** (`cd models && go test ./...` — package doesn't exist yet).
- [ ] **Step 3: Implement the move**: `git mv backend/internal/models/*.go models/` (keep history); split test files — GORM-importing tests (`group_test.go`'s gorm parts) parked into `models/gormbind_parked_test.go` temporarily OR simply moved in Task 3 (cleanest: delete the gorm-dependent test funcs from root test files now, re-home them in Task 3's `gormbind_test.go` — they are identified by their imports; keep every stdlib-only test). Then fix all backend importers to `import models "github.com/ProjAnvil/knot/models"`, add the require to `backend/go.mod` by hand, write `go.work`.
- [ ] **Step 4: Verify GREEN (Review Focus 1)**: from repo root: `go build ./... && go test ./backend/... ./models/...` all pass under the workspace; `cd mcp-server && go build ./...` still builds (go.work uses it). `cd backend && make test-coverage` — gate still green (nothing moved out of its pattern yet — param_node etc. still in services until Task 2).
- [ ] **Step 5: Commit** `refactor: extract domain models into publishable models module`

### Task 2: Wire-format types move

**Files:**
- Create: `models/param_node.go` (+ test), `models/wire.go` (+ test), `models/import_plan.go`
- Delete: type definitions from `backend/internal/services/param_node.go`, `export_knot.go`, `import_knot.go`, `import_detect.go` (functions stay)
- Modify: every services file referencing the moved types; `backend/Makefile` (gate pattern); their test files (move type-level tests, keep round-trip tests in backend)

**Interfaces:**
- Moves verbatim (names, fields, JSON tags unchanged): `ParamNode`, `TreeFromParams`, `ParamNodesFromJSONValue`, `CountNodes`; `KnotExportDocument/Group/API/Params` + new consts `WireFormat = "knot"`, `WireVersion = 1` (services literals `"knot"`/`1` switch to the consts — output unchanged); `ImportPlan`, `ImportGroup`, `ImportAPI`, `ImportFormat` + its three consts.
- Services files keep every function (`BuildKnotDocument`, `ParseKnotJSON`, `DetectImportFormat`, `ApplyImport`, …) now referencing `models.<Type>`.
- `backend/Makefile` gate: drop `param_node|export_knot|export_openapi|export_postman` from the function-name pattern only if those functions fully moved (they did NOT — builders stay in services); the pattern matches `go tool cover -func` output **file paths**: after the move, `param_node.go`/`export_knot.go` still exist in services with functions — the moved items are types + 3 helpers. Reconcile: remove the 3 moved helper names' file from matching by updating the pattern to keep gating what remains (`import_detect|import_knot|import_postman|import_openapi|import_apply|export_service|handlers/...` — final list = every remaining non-test services/handlers file touched by this project series) and confirm the "no functions matched" guard does not fire.
- [ ] **Step 1: Move with `git mv` semantics** (copy file → trim to functions → delete originals' type blocks), update references, hand-add nothing to go.mod (same module).
- [ ] **Step 2: Verify** (Review Focus 2): `go build ./... && go test ./...` (root, workspace); `cd backend && make test-coverage` green with updated pattern; wire-format byte check: existing `TestBuildKnotDocument` golden assertions pass unmodified.
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

- `gormbind_test.go`: the GORM-tag behavioral tests rescued from `group_test.go` (unique-name constraint, cascade, index) + `TestMigrateModels` (in-memory sqlite, AutoMigrate succeeds, tables exist) + `TestAutoMigrate` idempotent double-run.
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
- `gen-ts`: tygo config targeting package `models` (root only, not gormbind) → `models/ts/index.d.ts`. Header comment "Code generated by tools/cmd/gen-ts. DO NOT EDIT."
- Both run from repo root: `go run ./tools/cmd/gen-schema` / `go run ./tools/cmd/gen-ts` (workspace resolves models).

- [ ] **Step 1: Implement both generators; generate goldens; commit goldens.**
- [ ] **Step 2: Verify (Review Focus 4)**: regenerate twice → `git diff --exit-code` clean (determinism); schema validates a real exported Knot JSON document — write `tools/cmd/gen-schema/schema_smoke_test.go`? No — put the validation test in `models/wire_test.go`-adjacent backend-independent spot: `models/schema_test.go` loading the committed schema with invopop… invopop is tools-only. Instead: backend test `export_knot_test.go` gains a subtest `givenKnotDocument_whenValidatedAgainstSchema_thenValid` using samjholt… no — keep it simple: `tools/cmd/gen-schema` gains a `--check` mode that validates `services` output? Simplest honest check: a tools-module test that loads the golden schema and a fixed sample document via `github.com/santhosh-tekuri/jsonschema` — that's another dep. **Decision:** schema correctness is pinned by (a) generation is reflection from the same types the round-trip tests exercise, and (b) `git diff` drift lock. A validator test is nice-to-have; skip extra deps. Document this trade-off in the report.
- [ ] **Step 3: Commit** `feat: add JSON Schema and TypeScript generators with committed goldens`

### Task 5: CI — models job + drift + consumption smoke

**Files:**
- Modify: `.github/workflows/ci.yml`

**Interfaces:**
- New `models` job (runs with the other three): checkout; setup-go (`go-version-file: models/go.mod`); `go build ./... && go test ./...` from `models/`; `make test-coverage` in `models/` (gate); `go run ./tools/cmd/gen-schema && go run ./tools/cmd/gen-ts && git diff --exit-code -- models/schema models/ts` (drift lock); consumption smoke: create `$(mktemp -d)/smoke`, `go mod init smoke`, `go mod edit -replace github.com/ProjAnvil/knot/models=$GITHUB_WORKSPACE/models`, write `main.go` importing `models` (a Group literal + `models.IsValidMethod("GET")`) and `models/gormbind` (`_ "github.com/ProjAnvil/knot/models/gormbind"` or MigrateModels length check), `GOWORK=off go build ./...` — proves the module graph is consumable without the workspace (Review Focus for the Docker path).
- Existing `backend` job gains one step AFTER tests: `GOWORK=off go build ./...` is expected to FAIL pre-tag — do NOT add it; instead the job stays workspace-based, and the release runbook (§10.5) does the bootstrap. Add a comment in ci.yml noting this.
- [ ] **Step 1: Write ci.yml changes; YAML-validate.**
- [ ] **Step 2: Verify locally**: replicate the models job steps in a shell script run (all pass, incl. smoke with temp dir + replace + GOWORK=off).
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
5. Bootstrap go.sum: `cd backend && GOWORK=off go mod tidy` (resolves the published tag) → commit fixup `chore: pin models v1.0.0 in backend go.sum` → verify `GOWORK=off go build ./...`.
6. `git tag v0.7.0` ON THE FIXUP COMMIT → push → `release.yml` builds; `gh release edit v0.7.0 --notes-file <English template, Full Changelog base v0.6.0>`.
7. Verify: `gh release view v0.7.0 --json assets` (13 assets), Latest badge on v0.7.0 not the models release.
