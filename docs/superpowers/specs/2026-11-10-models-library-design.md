# Knot Models Library — Design

- Date: 2026-11-10
- Status: Final (builder check applied — 8 corrections folded into §7/§9/§10/§11)
- Target releases: `models/v1.0.0` (library) and `v0.7.0` (product)

## 1. Intent

Knot's data model (`Group`, `API`, `Parameter`, and the Knot JSON v1
interchange types) currently lives at `backend/internal/models` and
`backend/internal/services` — unreachable from outside the backend module.
Other ProjAnvil products (Go or otherwise, with their own frontends) should
be able to reuse the same data model without importing Knot's server.

Decisions (recommended by the author, not overridden by the product owner —
each is cheap to reverse before tagging):

1. **Publishing form: subdirectory Go module in this repo.** New module
   `github.com/ProjAnvil/knot/models` at `models/`, versioned with
   `models/vX.Y.Z` tags (Go-native subdirectory-module tagging). Model and
   product evolve atomically in one repo; no cross-repo sync tax.
2. **Dependency boundary: layered.** Root package `models` = pure types +
   constants + pure tree helpers, **zero third-party dependencies**.
   Subpackage `models/gormbind` = GORM helpers (`require gorm.io/gorm`).
   Consumers that never touch GORM never download it.
3. **Interoperability artifacts (this release): JSON Schema + TypeScript
   types.** The Knot JSON v1 wire format gets a published, committed JSON
   Schema; TS `.d.ts` types are generated from the Go types. Golden files
   are CI-locked against drift. (npm publishing is explicitly deferred —
   the `.d.ts` ships in-repo; publishing to npm is a later, separate
   decision.)
4. **Starting version: v1.0.0.** The model is small, stable, and already
   shipped in a production release (v0.6.0) with a versioned wire format.
   Semver applies from day one: breaking field changes bump major.

## 2. Goals

1. `go get github.com/ProjAnvil/knot/models` works from any Go module.
2. Backend consumes the library: `backend/internal/models` is deleted;
   all importers switch to the module (via `go.work` locally/CI, pinned
   version in `go.mod` for releases).
3. Wire-format types (`ParamNode`, Knot JSON v1 envelope) move into the
   library so external tools can produce/consume Knot JSON without the
   backend.
4. `models/schema/knot-v1.schema.json` — a JSON Schema (draft 2020-12)
   for the Knot JSON v1 document, generated from the Go types and
   committed; CI fails if regeneration drifts.
5. `models/ts/index.d.ts` — generated TypeScript declarations for the
   domain and wire types, committed; CI drift-locked the same way.
6. `models/gormbind` provides `MigrateModels()` and `AutoMigrate(db)`;
   `backend/internal/database` uses it (no behavior change).
7. Zero behavior change in the product: all existing backend tests,
   frontend tests, and the Playwright E2E suite stay green byte-for-byte
   on the wire.

## 3. Non-goals

- npm package publishing (deferred; in-repo `.d.ts` first).
- Generating client SDKs beyond types (no fetch wrappers, no language
  targets beyond TS).
- Multi-tenant/registry features (model distribution, licensing).
- OpenAPI spec of Knot's own REST API (separate future work).
- Renaming types or changing JSON field names — the library is a
  relocation, not a redesign.
- `mcp-server` adopting the module (it has no model dependency today).

## 4. Anti-goals

- The library must not depend on Fiber, viper, goldmark, or any backend
  package — dependency direction is strictly `backend → models`.
- No GORM import in the root `models` package (struct tags are inert
  strings and stay; the `gorm.io/gorm` import appears only in
  `gormbind`).
- No wire-format changes: Knot JSON v1 output before and after the
  refactor is byte-identical (pinned by golden tests).
- No database schema change.

## 5. Current state (verified)

- `backend/internal/models/`: `api.go` (41 LOC), `group.go` (44),
  `parameter.go` (43) + 1,420 LOC of tests. Imports: stdlib only
  (`encoding/json`, `time`). GORM appears as struct tags only.
- 11 backend files import `internal/models` (database, handlers×5,
  services×5).
- Wire-format types added in v0.6.0 live in services: `ParamNode` +
  tree helpers (`param_node.go`), Knot JSON envelope
  (`export_knot.go`), `ImportPlan` family (`import_knot.go`),
  `ImportFormat` consts (`import_detect.go`).
- Repo is already multi-module: `backend/go.mod`
  (`github.com/ProjAnvil/knot/backend`), `mcp-server/go.mod`. No
  `go.work` yet.
- `release.yml` triggers on tags matching `v*` — `models/v1.0.0` does
  NOT match (starts with `m`), so product release flow is unaffected.

## 6. Approaches considered

**A. Subdirectory module in this repo (chosen).** `models/` with its own
`go.mod`; versioned via `models/vX.Y.Z` tags; `go.work` joins
backend + models for local/CI development; `backend/go.mod` pins a real
version for tag builds (workspace overrides it locally, so there is no
tag-ordering chicken-and-egg). Trades: repo tags now serve two version
streams (distinguished by prefix); contributors must run `go work sync`.

**B. Separate repository (`ProjAnvil/knot-models`).** Cleaner versioning
story, but every model change needs a cross-repo PR + release before the
product can consume it — exactly the drift the product owner wants to
avoid for a small team. Rejected for now; the module can be split out
later with its history intact if traffic justifies it.

**C. Publish nothing; consumers copy types or use `replace`.** Fails the
stated goal ("GitHub-referencable third-party library"). Rejected.

**D. Language-neutral IDL first (protobuf/JSON Schema as source of
truth, Go types generated).** Maximal interop, but it inverts an
existing, shipped Go model and adds a toolchain the team doesn't use.
The Go types stay the source of truth with generated Schema/TS as
projections (§8) — good enough for the stated consumers, and reversible.

## 7. Library design

### 7.1 Layout

```
models/                          # module github.com/ProjAnvil/knot/models
  go.mod                         # go 1.25; zero requires
  api.go group.go parameter.go   # moved verbatim from backend/internal/models
  constants.go                   # method/type/paramType enums + Is* helpers
  wire.go                        # Knot JSON v1 envelope types (moved from services)
  param_node.go                  # ParamNode + pure tree helpers (moved)
  wire_test.go param_node_test.go api_test.go ...   # tests move with code
  gormbind/
    gormbind.go                  # MigrateModels() []any; AutoMigrate(db *gorm.DB) error
    gormbind_test.go             # in-memory sqlite migration test (test-only dep)
  schema/
    knot-v1.schema.json          # committed golden (generated)
  ts/
    index.d.ts                   # committed golden (generated)
  README.md                      # usage, semver policy, artifact pointers
tools/                           # separate module: generation tooling
  go.mod                         # deps: invopop/jsonschema, gzuidhof/tygo
  cmd/gen-schema/main.go         # writes models/schema/knot-v1.schema.json
  cmd/gen-ts/main.go             # writes models/ts/index.d.ts
go.work                          # use (./backend ./models ./tools ./mcp-server)
```

`go.work` must `use` `./mcp-server` too: `release.yml` runs `make
build-all` inside `mcp-server/`, and Go fails in a module that sits under
an ancestor `go.work` that doesn't list it.

Notes:

- `constants.go`: `HTTPMethods` (GET/POST/PUT/DELETE/PATCH),
  `APITypes` (HTTP/RPC), `ParamTypes` (request/response), `ParamValueTypes`
  (string/number/boolean/array/object) + `IsValidMethod(string) bool`
  style helpers. Backend importers may adopt them opportunistically;
  nothing in backend is required to change beyond import paths.
- `wire.go` holds `KnotExportDocument/Group/API/Params` (renamed export
  side stays as-is: field names and JSON tags unchanged) plus
  `WireFormat = "knot"` and `WireVersion = 1` constants; services keep
  their builder/parser functions but now reference `models.KnotExportDocument`
  etc.
- `ImportPlan`/`ImportGroup`/`ImportAPI`/`ImportFormat` also move to the
  library (`wire.go`/`import_plan.go`): external tools building import
  pipelines want them, and they reference only `ParamNode`. The
  detect/parse/apply logic stays in backend services.
- `BuildParameterTree` (services, HTML-export-specific shape) stays in
  services; `TreeFromParams`/`ParamNodesFromJSONValue`/`CountNodes` move
  with `ParamNode` (pure, zero deps).
- Test placement: the GORM-tag behavioral tests currently in
  `group_test.go` (they import gorm + glebarez/sqlite) land in
  `gormbind/` as `gormbind_test.go`; only stdlib-only marshal/type tests
  move into the root package. This is what keeps the root `go.mod` at
  zero requires.
- Coverage attribution: functions in the separate `models` module are
  invisible to `backend`'s profile without `-coverpkg`; the ≥85% gate
  for moved code (pattern `param_node|wire|import_plan|api.go|group.go|
  parameter.go|constants`) runs in the new `models` CI job with
  `-coverpkg=./...` scoped to the module — `backend/Makefile` keeps its
  existing pattern unchanged.
- Round-trip golden tests (`BuildKnotDocument` ↔ `ParseKnotJSON`) call
  backend functions and stay in backend as integration proof; the
  library carries the type-level marshal tests.
- `gormbind` supplies `MigrateModels()` (the model list, same order as
  today) and `AutoMigrate(db)`; `backend/internal/database/database.go`
  keeps its existing conditional HasTable/migrator logic and only sources
  the model list from `gormbind` — no behavior change.

### 7.2 Backend refactor

- Delete `backend/internal/models`; all 11 importers change to
  `import models "github.com/ProjAnvil/knot/models"` (alias keeps the
  diff mechanical: `models.Group` unchanged at call sites).
- `backend/internal/database` migrates via `gormbind.AutoMigrate(db)`
  (identical model list/order as today).
- `backend/go.mod` gains `require github.com/ProjAnvil/knot/models v1.0.0`
  (+ `go.work` makes local + CI use the workspace copy; tag builds get
  the published module — both paths compile the same code).
- services' wire/parser files drop their local type definitions and use
  the library's; only import lines and type re-locations change.

### 7.3 Versioning

- Library tags: `models/v1.0.0` (semver from day one). Product tags stay
  `vX.Y.Z`.
- `models/README.md` documents: semver policy (breaking = major),
  artifact refresh policy (schema + ts regenerate in the same commit as
  type changes; CI enforces), and how to consume
  (`go get`, schema URL, `ts/index.d.ts` path).

## 8. Generated artifacts

- **JSON Schema** (`models/schema/knot-v1.schema.json`): draft 2020-12,
  generated from the wire types via `invopop/jsonschema` reflection in
  `tools/cmd/gen-schema`. Root = `KnotExportDocument`; `$id` =
  `https://github.com/ProjAnvil/knot/models/schema/knot-v1.schema.json`;
  `const` pins `format: "knot"`, `version: 1`. Any language can validate
  or codegen from it.
- **TypeScript** (`models/ts/index.d.ts`): generated via
  `gzuidhof/tygo` from the root package (domain + wire types). External
  frontends consume types only — no runtime, no fetch layer.
- **Drift lock:** CI regenerates both (`go run ./tools/cmd/gen-schema`,
  `go run ./tools/cmd/gen-ts` via the workspace) and fails on
  `git diff --exit-code -- models/schema models/ts`.
- Generators live in `tools/` (own module) so their deps never touch the
  library or backend go.mod graphs.

## 9. Testing strategy

- Library: the 1,420 LOC of existing model tests move with the code;
  new tests for constants helpers; `gormbind` migration test
  (test-only `glebarez/sqlite`).
- Backend: full suite + coverage gate unchanged (`make test-coverage`);
  the mechanical import swap must not move coverage numbers materially
  on new-file rules (gate pattern extended to `models/` functions that
  moved — the moved code keeps its existing coverage through moved
  tests).
- Wire compatibility: existing Knot JSON round-trip golden tests now
  live in the library; backend export tests (byte assertions on
  `format:"knot"` documents) stay as integration proof.
- E2E: the 4 Playwright specs re-run unchanged (the strongest
  no-behavior-change gate).
- External consumption smoke: CI job creates a scratch module in a temp
  dir with `replace github.com/ProjAnvil/knot/models => <repo>/models`,
  imports `models` + `gormbind`, compiles. Post-tag, a manual
  `go get github.com/ProjAnvil/knot/models@v1.0.0` smoke runs outside
  the repo and is recorded in the release checklist.

## 10. Release plan

1. Branch `feat/models-library`; PR (English, conventional commits).
2. CI green (existing 3 jobs + new `models` job: build, test,
   regenerate drift check, GOWORK=off consumption smoke).
3. Squash-merge → push `models/v1.0.0` → verify external
   `go get github.com/ProjAnvil/knot/models@v1.0.0` smoke outside the
   repo.
4. `gh release create models/v1.0.0 --latest=false` (must NOT steal the
   product's Latest badge) with short English notes (library usage +
   artifact links).
5. Bootstrap go.sum for image builds (no workspace inside Docker):
   `cd backend && GOWORK=off go mod tidy` against the now-published tag
   → fixup commit on main → verify `GOWORK=off go build ./...` passes.
6. Push `v0.7.0` **on the fixup commit** → existing `release.yml` builds
   binaries; `gh release edit v0.7.0 --notes-file` with the English
   template (Highlights: model library extraction, schema + TS
   artifacts, no product behavior change). Full Changelog base bumps to
   `v0.6.0`.

## 11. Risks & mitigations

| Risk | Mitigation |
|------|------------|
| Docker image builds have NO workspace (Dockerfile copies only `backend/`), so `go.mod`+`go.sum` must resolve the published tag | §10.5 bootstrap: `GOWORK=off go mod tidy` fixup commit after `models/v1.0.0` exists; `v0.7.0` tags that commit; CI models job runs a `GOWORK=off` replace-drift check |
| Tag ordering: `backend` tag build resolves `models v1.0.0` before it exists | Workspace satisfies it locally/in CI; product tag is pushed after `models/v1.0.0` exists |
| `release.yml` builds `mcp-server/` under the new root `go.work` | `go.work` uses `./mcp-server` too (verified failure mode if omitted) |
| Schema/TS generators drift silently | CI regeneration diff gate |
| Moved tests lose coverage attribution | `-coverpkg` in the models CI job gate |
| tygo output churn across versions | `tools/go.mod` pins exact generator versions; regenerate = explicit commit |
| Library release steals product's "Latest" badge | `gh release create models/v1.0.0 --latest=false` |
| Double maintenance of `BuildParameterTree` vs `TreeFromParams` | Pre-existing duplication, unchanged scope; noted for future dedupe |
| Root `LICENSE` has a "[Your Name]" MIT placeholder | Pre-existing repo state; out of scope here, flagged for the product owner |

## 12. Execution sequence

Same pipeline as v0.6.0: this spec → builder-check subagent audit →
fixes → plan (writing-plans) → subagent audit → subagent-driven
execution (fresh implementer + reviewer per task) → final whole-branch
review → verification (backend, frontend, E2E, smoke) → PR → merge →
`models/v1.0.0` → `v0.7.0` via `gh`.
