# Embeddable Knot Engine — Design

- Date: 2026-11-10
- Status: Draft (pending builder check)
- Target releases: `models/v1.1.0` (library minor) and `v0.7.1` (product patch)

## 1. Intent

Turn the Knot backend into an embeddable Go component — the moral equivalent
of a Spring Boot component in Go's library-first idiom — so a host application
(Sentia: Go + Fiber + GORM, reusing its own `*gorm.DB`) can import it,
configure it, and initialize it in-process instead of running Knot as a
separate service.

Product-owner decisions already made:

- **Access form: embedded component, not hosted multi-tenant.** The host owns
  the process, the DB connection, auth, and the HTTP server. Knot is a
  mountable engine.
- **Host facts (confirmed): Sentia uses Fiber and GORM and passes its own
  `*gorm.DB` instance.** The DSN bootstrapping channel is cut (YAGNI);
  `Config.DB` is the only way to supply persistence.
- **Same database, table prefix:** Knot's tables (`groups`, `apis`,
  `parameters`) are too generic to drop into a host DB unprotected — a
  configurable `TablePrefix` (e.g. `knot_`) is part of the contract.
- **Versioning: the product ships this as a PATCH (`v0.7.1`)** — no product
  behavior change; the binary is the same engine plus host wiring. The models
  library takes its own MINOR (`models/v1.1.0`) for the new prefix capability.
  The two version streams are independent by design.

## 2. Goals

1. `import` + `New(Config)` + `Mount` is the entire host contract:

```go
k, err := knot.New(knot.Config{
    DB:          sentiaDB,     // required: host's *gorm.DB
    APIPrefix:   "/knot/api",  // default "/api"
    TablePrefix: "knot_",      // default "" (existing deployments unchanged)
})
k.Mount(sentiaFiberApp)         // registers all API routes under the prefix
defer k.Close()                 // closes ONLY self-created resources
```

2. Service methods callable directly, bypassing HTTP: `Export`, `Import`
   (mirroring the POST /export / POST /import semantics incl. dry-run).
3. Existing binary behavior byte-identical: CLI and `cmd/server` become thin
   hosts over the same engine; all existing tests, E2E (4 specs), and CI stay
   green.
4. models v1.1: injectable table prefix (`TableName()` driven by a package
  `TablePrefix` variable + `SetTablePrefix`), `gormbind` unchanged in shape.
5. CI gains a "host B" consumption smoke: a scratch module with its own
   Fiber app + GORM sqlite DB that mounts the engine under a prefix with a
   table prefix, exercises one endpoint and one direct service call — the
   Sentia scenario, locked forever.

## 3. Non-goals

- HTTP-facing changes (no `/api/v1`, no auth middleware — the HOST owns auth;
  the previous public-API discussion is superseded by the component approach).
- Hosted/multi-tenant anything.
- DSN/self-managed DB bootstrapping (cut; revisit if a non-GORM host appears).
- `http.Handler` adapter (Sentia is Fiber; add later for non-Fiber hosts).
- Changing any route path, response shape, or wire format.

## 4. Anti-goals

- The engine must never `Close()` or mutate the host's DB connection; the only
  DDL it ever issues is `AutoMigrate` inside `New()`.
- No global mutable config in the engine beyond the documented models table
  prefix (single engine instance per process is the documented contract).
- The embedded frontend and viper configuration stay OUT of the engine — they
  are host concerns (CLI host keeps them).
- No new third-party dependencies.

## 5. Current state (verified this session)

- Module layout after v0.7.0: `backend` (module `github.com/ProjAnvil/knot/backend`),
  `models` (published), `tools`, `mcp-server`; `go.work` joins all four;
  backend/go.mod carries a permanent `replace => ../models`.
- `handlers.SetupRoutes(app *fiber.App, db *gorm.DB)` registers everything
  under `/api` — no prefix parameter, no middleware injection.
- fiber app creation + viper `LoadConfig` + static frontend mounting live in
  `internal/cli/serve.go` and `cmd/server/main.go` (already host-shaped).
- Models' `TableName()` return fixed strings; `gormbind.MigrateModels()` is
  the canonical migration list.
- CI: backend / models / frontend / e2e jobs; models job already has a
  GOWORK=off consumption smoke pattern to extend.
- All product behavior currently verified by 4 Playwright E2E specs + handler
  test suites with an 85% coverage gate.

## 6. Approaches considered

**A. In-place componentization of the backend module (chosen).** The backend
module stays `github.com/ProjAnvil/knot/backend`; a new root package in
`backend/` exposes the engine facade (`New`/`Mount`/service methods). Hosts
import it as `import knot "github.com/ProjAnvil/knot/backend"`. Zero module
surgery, `internal/` keeps hiding implementation, CI/release wiring untouched.
Cost: the import path says "backend" while the package says "knot" (alias is
one line for hosts; a v2 rename can follow if it ever grates).

**B. New top-level `engine` module extracting everything from backend.**
Cleanest naming (`github.com/ProjAnvil/knot/engine`), but it re-homes ~40
files, rewrites every import, re-plumbs the replace directives, and risks the
release path for zero functional gain now. Rejected; A reaches the same
contract with a fraction of the churn.

**C. Keep the server binary as the only integration surface (previous plan's
self-host HTTP form).** Superseded by the product owner's component direction:
Sentia embeds, so no second process, no network hop, host-owned auth.

## 7. Engine design

### 7.1 Facade (new `backend/knot.go`, package `knot`)

```go
package knot // import knot "github.com/ProjAnvil/knot/backend"

type Config struct {
    DB          *gorm.DB // required; validated non-nil
    APIPrefix   string   // default "/api"
    TablePrefix string   // default "" — passed to models.SetTablePrefix BEFORE any DB call
}

type Knot struct { /* db, prefix, mounted state — unexported fields */ }

func New(cfg Config) (*Knot, error)
// validates DB non-nil; sets models table prefix (once); runs gormbind.AutoMigrate(db);
// error wraps migration failures with context.

func (k *Knot) Mount(app *fiber.App)
// idempotent (second call no-ops); registers ALL existing API routes under
// cfg.APIPrefix via the shared handler-registration function; host may wrap
// the fiber app with its own auth middleware before/after mounting.

func (k *Knot) Export(apiIDs []uint, format string, locale string) ([]byte, string, error)
// mirrors POST /api/export semantics incl. filename; format validated by the
// same code path as the handler.

func (k *Knot) Import(data []byte, format ImportFormat, dryRun bool) (*models.ImportResult, error)
// detect + parse + ApplyImport — mirrors POST /api/import minus multipart.

func (k *Knot) Close() error
// no-op today (engine owns nothing); exists for host lifecycle symmetry and
// future resources. NEVER touches the host DB.
```

### 7.2 Internal restructuring (minimal)

- `handlers.SetupRoutes` gains an internal variant
  `registerRoutes(app *fiber.App, db *gorm.DB, prefix string)`; the public
  `SetupRoutes` delegates with `"/api"` (existing callers unchanged).
- Export/Import handlers' bodies split into service-callable functions when
  trivially extractable; where the handler is already thin over services
  (import) the facade calls services directly. No duplication.
- serve.go / cmd/server/main.go: construct `knot.New` with their DB and
  `"/api"` prefix, then `Mount`, then mount static frontend from the HOST
  side (code moves from SetupRoutes-adjacent into the two hosts if it isn't
  already).

### 7.3 models v1.1 — table prefix

- `models.SetTablePrefix(p string)` sets a package-level variable; each
  `TableName()` returns `prefix + "groups"` etc. Default `""` = existing
  deployments byte-identical.
- Contract documented: set once before first DB use (the engine's `New` does
  it from `Config.TablePrefix`); single engine instance per process.
- `gormbind` needs no shape change (migrates the same models, now prefixed).
- Tests: prefix round-trip (migrate with `knot_` prefix on in-memory sqlite,
  CRUD, verify physical table names), default-empty regression.

### 7.4 Versioning

- Product `v0.7.1` (patch): binary behavior identical; release notes state
  the engine library capability.
- `models/v1.1.0` (minor): new `SetTablePrefix` capability; no breaking
  change (additive).
- Engine facade rides the backend module (unversioned library surface inside
  the product module — documented as such; external library consumers are
  expected to take the models module, not the engine, until real hosts ask
  for a stable versioned engine module).

## 8. Testing strategy

- models: prefix tests (§7.3) in the models module; existing 98.6% gate.
- backend: facade unit tests (`New` validation, `Mount` idempotence + prefix,
  `Export`/`Import` parity with handler outputs on the same fixture).
- Existing handler tests + E2E unchanged and green (host A proof).
- CI "host B" smoke (extends the models-job pattern): scratch module with
  replace → backend module, own fiber app + glebarez sqlite DB, `knot.New`
  with `APIPrefix:"/knot/api"`, `TablePrefix:"knot_"`, mount, hit
  `GET /knot/api/groups` (expect 200 envelope), direct `Import` dry-run call
  (expect counts), assert `knot_groups` table exists physically.
  GOWORK=off, mirroring the existing smoke.

## 9. Release plan

1. Branch `feat/embeddable-engine`; PR (English, conventional commits).
2. CI green (4 existing jobs; models job extended or a smoke step added to
   backend job — plan decides based on which module the smoke imports).
3. Squash-merge → tag `models/v1.1.0` → external `go get` smoke for the
   models minor.
4. `gh release create models/v1.1.0 --latest=false` (English notes).
5. Tag `v0.7.1` → release.yml binaries → `gh release edit v0.7.1 --notes-file`
   (English patch template: engine capability, no behavior change; Full
   Changelog base `v0.7.0`).

## 10. Risks & mitigations

| Risk | Mitigation |
|------|------------|
| Global table-prefix state surprises concurrent users | Documented single-instance contract; `New` sets it once from config; default "" keeps existing deployments untouched |
| `TableName()` prefix change breaks existing DBs | Default "" is the no-op path; migration of existing installs is NOT attempted (documented: prefix is for new embeds) |
| Facade drifts from handler behavior | Parity tests assert facade output == handler output on shared fixtures |
| Mount double-registration panics fiber | Idempotent Mount guard + test |
| mcp-tools route (stateful?) mounted in hosts | Same route set mounts everywhere; host can choose not to expose by prefix/middleware — documented |
| Import path `.../backend` with package `knot` | One-line alias documented in README; recorded as accepted cost of approach A |

## 11. Execution sequence

spec → builder-check subagent audit → fixes → plan (writing-plans) →
subagent audit → subagent-driven execution → final whole-branch review →
verification (all gates + host-B smoke) → PR → merge → `models/v1.1.0` →
`v0.7.1` via gh.
