# Knot backend — the embeddable engine

[![Go Reference](https://pkg.go.dev/badge/github.com/ProjAnvil/knot/backend.svg)](https://pkg.go.dev/github.com/ProjAnvil/knot/backend)

Knot's backend as an embeddable Go component. A host application brings its
own [Fiber](https://gofiber.io) app and its own [GORM](https://gorm.io)
database connection, and delegates one concern to the engine: API-documentation
management — groups, APIs and parameter trees over REST, plus multi-format
export (HTML, Knot JSON, OpenAPI, Postman) and import (Knot JSON, Postman
Collection v2.x, OpenAPI 3.x).

The whole integration is three calls:

```go
engine, err := knot.New(knot.Config{DB: db}) // validates + migrates
engine.Mount(app)                            // routes under /api
payload, filename, err := engine.Export(ids, "html", "en")
```

Knot's own `knot-server` and `knot` CLI are thin hosts over this engine
(`cmd/server` and `cmd/cli` in this module): what they do with Fiber, GORM and
the process, an embedding application can do too.

> **Version status:** the module is **v0** — the engine API is allowed to
> change between releases until the first hosts have stabilized it. Pin the
> version in your `go.mod` and expect possible breakage until v1.

## Installation

```sh
go get github.com/ProjAnvil/knot/backend@v0.1.1
```

```go
import knot "github.com/ProjAnvil/knot/backend"
```

The package is named `knot` but lives at the module root (the import path
ends in `backend`), so the explicit alias keeps references reading
`knot.New` / `knot.Config`. Requires Go 1.25.5+ and any GORM driver you
choose (SQLite, PostgreSQL, MySQL — the host opens the connection, not the
engine).

The engine depends on the
[`models`](https://github.com/ProjAnvil/knot/blob/main/models/README.md)
module (v1.1+) for its data types; for example `Knot.Import` returns
`*models.ImportResult` from there.

## Example: Knot inside a host application

A complete host ("Sentia") that owns its database, mounts the engine's routes
at `/knot/api`, imports a Knot JSON backup directly, and exports HTML:

```go
package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	knot "github.com/ProjAnvil/knot/backend"
	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func main() {
	// Sentia owns its database and its HTTP app; Knot is a guest in both.
	db, err := gorm.Open(sqlite.Open("sentia.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// New validates the config, installs the table prefix and runs the
	// engine's only DDL — AutoMigrate of knot_groups / knot_apis /
	// knot_parameters — leaving Sentia's own tables alone.
	engine, err := knot.New(knot.Config{
		DB:          db,
		APIPrefix:   "/knot/api", // "" means the default "/api"
		TablePrefix: "knot_",     // "" keeps the bare table names
	})
	if err != nil {
		log.Fatal(err)
	}
	defer engine.Close() // no-op: the DB connection belongs to Sentia

	// Mount registers every engine route under /knot/api on Sentia's own
	// Fiber app. It is idempotent per app — mounting defensively is safe.
	app := fiber.New()
	engine.Mount(app)

	// Service methods are direct calls; no HTTP round-trip needed.
	data, err := os.ReadFile("backup.knot.json")
	if err != nil {
		log.Fatal(err)
	}

	preview, err := engine.Import(data, true) // dry run: resolve only
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("dry run: %d group(s) created, %d API(s) imported\n",
		preview.GroupsCreated, preview.APIsImported)

	if _, err := engine.Import(data, false); err != nil {
		if errors.Is(err, knot.ErrImportParse) {
			log.Fatal("backup.knot.json is not a parseable Knot document")
		}
		log.Fatal(err)
	}

	// The prefixed tables are ordinary tables in Sentia's database.
	var ids []uint
	db.Table("knot_apis").Pluck("id", &ids)

	payload, filename, err := engine.Export(ids, "html", "en")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s: %d bytes\n", filename, len(payload))

	// Sentia serves the mounted routes next to its own.
	log.Fatal(app.Listen(":3000"))
}
```

After `New` + `Mount`: the three `knot_*` tables live in Sentia's database
next to its own schema, the engine's routes answer under `/knot/api`, and
Sentia's tables, routes and listener stay entirely under its control.

## Host contract

### Config

```go
type Config struct {
	DB          *gorm.DB // required — the host's own connection
	APIPrefix   string   // "" → "/api" (DefaultAPIPrefix)
	TablePrefix string   // "" → bare table names (see Table prefix)
}
```

### New

```go
func New(cfg Config) (*Knot, error)
```

- `DB` must be non-nil; anything else fails before touching the database.
- A non-empty `TablePrefix` is installed via `models.SetTablePrefix`
  **before** migration. An empty one leaves any prefix the host already set
  untouched — `New` never clobbers it.
- Runs the canonical `gormbind.AutoMigrate` — the only DDL the engine ever
  executes (see [Ownership rules](#ownership-rules)).
- Every failure is wrapped with the failing step
  (e.g. `knot: auto-migrate failed: ...`).

### Mount

```go
func (k *Knot) Mount(app *fiber.App)
```

Registers all engine routes on the host's app under `Config.APIPrefix`
(default `/api`). Mount is **idempotent per app**: mounting the same app
twice is a no-op, so hosts can mount defensively, while mounting additional
apps registers the full route set on each. It registers nothing else —
static file serving, health checks and middleware stay with the host (the
standalone server registers its own `/api/health` before mounting).

Mounted routes:

```
GET    {prefix}/groups                        list groups
GET    {prefix}/groups/with-apis              groups incl. their APIs
POST   {prefix}/groups                        create group
POST   {prefix}/groups/orders                 reorder groups
PATCH  {prefix}/groups/:id                    update group
DELETE {prefix}/groups/:id                    delete group
GET    {prefix}/apis                          list APIs (paginated, searchable)
GET    {prefix}/apis/:id                      get one API
GET    {prefix}/apis/group/:groupId           APIs of one group
POST   {prefix}/apis                          create API
PATCH  {prefix}/apis/:id                      update API
PATCH  {prefix}/apis/:id/note                 update API note
POST   {prefix}/apis/orders                   reorder APIs
POST   {prefix}/apis/:id/duplicate            duplicate API
DELETE {prefix}/apis/:id                      delete API
PUT    {prefix}/apis/:id/parameters           replace parameter tree
POST   {prefix}/apis/:id/parameters/from-json replace parameters from JSON
POST   {prefix}/export                        export (HTML/JSON/OpenAPI/Postman)
POST   {prefix}/import                        import (auto-detected format)
POST   {prefix}/mcp-tools                     MCP tools endpoint
```

Responses use the Knot envelope — `{"success":true,"data":...}` on success,
`{"success":false,"error":"..."}` on failure.

`GET {prefix}/apis` is a flat, paginated listing for search UIs, ordered by
the parent group's position, then API order. Query parameters: `page`
(default 1), `pageSize` (default 50, clamped to 1..100), `keyword`
(case-insensitive substring match on API name or endpoint), `groupId`
(exact) and `method` (exact, upper-cased before matching). The data payload
is `{"items":[...],"total":N,"page":N,"pageSize":N}`; non-numeric `page`,
`pageSize` or `groupId` values fail with a 400.

### Export

```go
func (k *Knot) Export(apiIDs []uint, format string, locale string) ([]byte, string, error)
```

Renders the selected APIs and returns the payload bytes plus the download
filename, byte-identical to the HTTP export handler (both are stamped with
the current time at second granularity). `format` `""` and `"html"` produce
the standalone HTML document; `"json"`, `"openapi"` and `"postman"` produce
the respective JSON documents; anything else fails with
`ErrUnsupportedFormat`. `locale` selects the HTML document language and
defaults to `"zh"` when empty. An empty `apiIDs` fails with `ErrNoAPIIDs`.

Two different "format" notions are at play: the `format` argument above is
a **request** value — `""`, `html`, `json`, `openapi` or `postman` — while
the Knot JSON document produced by `json` carries `"format":"knot"` in its
envelope (`models.WireFormat`, the v1 wire-format identifier). `"knot"` is
not a valid request value.

### Import

```go
func (k *Knot) Import(data []byte, dryRun bool) (*models.ImportResult, error)
```

Detects the document's format (Knot JSON, Postman Collection v2.x,
OpenAPI 3.x), parses it, and applies the whole plan in one transaction. With
`dryRun=true` the identical resolution logic runs but nothing is written; the
returned counts are what a real run would produce. Imported groups whose name
matches an existing group merge into it, otherwise new groups are created.
The result reports `format`, `groupsCreated`, `groupsMerged`, `apisImported`,
`parametersImported` and `warnings` (field names as JSON-encoded by the
import endpoint).

### Close

```go
func (k *Knot) Close() error
```

A documented no-op. The engine holds no resources of its own — the database
connection belongs to the host — so `Close` exists only to be deferred
symmetrically with `New`.

## Errors

The service methods report failures as sentinel errors, listed here exactly
as defined. Match them with `errors.Is`; wrapped detail is preserved, so
`err.Error()` stays log-worthy while the mapping stays stable. The suggested
HTTP analog mirrors the standalone server's handlers:

| Sentinel | Meaning | HTTP analog |
| --- | --- | --- |
| `ErrUnsupportedFormat` | export format or import document not recognized | 415 |
| `ErrNoAPIIDs` | `Export` called with an empty `apiIDs` | 400 |
| `ErrExportFailed` | database or encoding failure during export | 500 |
| `ErrImportParse` | a detected import document failed to parse | 400 |
| `ErrImportApply` | failure while persisting an import plan | 500 |

## Ownership rules

- **The database connection belongs to the host.** The engine never closes
  it — `Close` is a no-op. Open, share and close the `*gorm.DB` like any
  other part of the host.
- **The engine's only DDL is the `AutoMigrate` inside `New`.** It creates or
  extends the (optionally prefixed) `groups`, `apis` and `parameters` tables
  and nothing else — no drops, no renames, no other schema objects.
- **One engine instance per process.** The table prefix is process-wide
  state (`models.TablePrefix`); a second engine instance in the same process
  would silently share it. `Mount` is idempotent per app by design —
  remounting the same app is a no-op, and each additional app gets its own
  route registration.
- **`SetTablePrefix` runs once, before the first database use.**
  `Config.TablePrefix` does this for you (installed before migration, only
  when non-empty). Hosts that call `models.SetTablePrefix` themselves should
  leave `Config.TablePrefix` empty — GORM caches parsed table names per
  `*gorm.DB`, so a later change has no effect on an open database, and
  existing tables are never renamed.

## Table prefix

Knot's table names are generic — `groups`, `apis`, `parameters` — which is
fine for a dedicated database but a collision risk in a database shared with
the host's own schema. `Config.TablePrefix` namespaces them:

| `TablePrefix` | Physical tables |
| --- | --- |
| `""` (default) | `groups`, `apis`, `parameters` |
| `"knot_"` | `knot_groups`, `knot_apis`, `knot_parameters` |

- The default empty prefix is a compatibility guarantee: it produces exactly
  the standalone product's physical schema, so pointing the engine at an
  existing Knot database (or the
  [models](https://github.com/ProjAnvil/knot/blob/main/models/README.md)
  module's v1.0 layout) just works.
- The prefix is process-global — set it through `Config` before migration,
  once (see [Ownership rules](#ownership-rules)).
- The host's own tables are untouched; the embedded-host smoke test in CI
  (`backend` job) asserts exactly this coexistence.

## Versioning

This module is the [`backend/`](https://github.com/ProjAnvil/knot/tree/main/backend)
directory of [ProjAnvil/knot](https://github.com/ProjAnvil/knot), versioned
independently with `backend/v*` git tags (the same scheme as the `models`
module). The current line is **v0**: the engine API may change between
releases until host feedback stabilizes it; v1 brings full semver
guarantees.

## Standalone binaries

`cmd/server` (`knot-server`) and `cmd/cli` (`knot`) in this module are
reference hosts: they open the database from the configuration file, register
their own health endpoint and static frontend, then `New` + `Mount` the
engine. Build them with `make build` from this directory, or see the
[repository README](https://github.com/ProjAnvil/knot) for the full
development workflow.

## License

MIT — same as the repository
([LICENSE](https://github.com/ProjAnvil/knot/blob/main/LICENSE)).
