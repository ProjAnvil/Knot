# models

[![Go Reference](https://pkg.go.dev/badge/github.com/ProjAnvil/knot/models.svg)](https://pkg.go.dev/github.com/ProjAnvil/knot/models)

Knot's data model as an importable Go module: the domain types behind the
product (`Group`, `API`, `Parameter`), the **Knot JSON v1** wire-format types
used by export/import, and the validation constants that define what a legal
document is.

## Installation

```sh
go get github.com/ProjAnvil/knot/models
```

The module is the [`models/`](https://github.com/ProjAnvil/knot/tree/main/models)
directory of the [ProjAnvil/knot](https://github.com/ProjAnvil/knot) repository.
It is versioned with `models/vX.Y.Z` git tags (e.g. `models/v1.0.0`), and
`go get` resolves releases from those tags.

## Versioning

The module follows [semantic versioning](https://semver.org/) via the
repository's `models/v*` tags — the current release is **v1.2.0**. Breaking
changes to any exported type or to the wire format bump the major version
(per Go module conventions, a v2 would move the import path to
`github.com/ProjAnvil/knot/models/v2`); backwards-compatible additions are
minor releases; fixes are patches.

v1.2.0 is an additive wire change: the `Group` domain type's JSON now emits
`order` (its `MarshalJSON` previously omitted the field). Consumers see one
new field wherever a `Group` is marshaled; nothing is renamed or removed.

## Packages

| Package | Import path | Brings in |
| --- | --- | --- |
| Root | `github.com/ProjAnvil/knot/models` | Standard library only |
| GORM bindings | `github.com/ProjAnvil/knot/models/gormbind` | [GORM](https://gorm.io) |

The **root package** defines:

- the domain types `Group`, `API`, and `Parameter`, plus `ParamNode` (the
  nested parameter tree projected from flat `Parameter` rows) and the
  import-preview types (`ImportPlan`, `ImportGroup`, `ImportAPI`,
  `ImportResult`);
- the Knot JSON v1 wire-format types — `KnotExportDocument` with its
  `KnotExportGroup` / `KnotExportAPI` / `KnotExportParams` children, and the
  envelope constants `WireFormat` (`"knot"`) and `WireVersion` (`1`).
  `WireFormat` is the value written into a Knot JSON document's envelope
  `format` field to identify the v1 wire format — it is not the backend
  export request's `format` parameter (whose values are `""`, `html`,
  `json`, `openapi` and `postman`);
- the validation constants `HTTPMethods`, `APITypes`, `ParamTypes`, and
  `ParamValueTypes`, with their `IsValidMethod` / `IsValidAPIType` /
  `IsValidParamType` / `IsValidParamValueType` predicates.

The root package imports only the Go standard library.

**`gormbind`** maps those types onto database tables: pass your `*gorm.DB` to
`gormbind.AutoMigrate(db)`, or use `gormbind.MigrateModels()` with your own
migration flow, to create/migrate the `groups`, `apis`, and `parameters`
tables. Importing this subpackage pulls in GORM.

### Table name prefix (v1.1)

Embedders that share one database with other software can namespace Knot's
tables with `models.SetTablePrefix` (default `""`, i.e. the v1.0 bare names):

```go
models.SetTablePrefix("knot_")   // before the first DB use: knot_groups, knot_apis, knot_parameters
err := gormbind.AutoMigrate(db)
```

The prefix is process-global: set it once before the first database use and
run a single engine instance per process. GORM resolves table names when a
model schema is first parsed and caches them per `*gorm.DB`, so changing the
prefix afterwards has no effect on an open database, and existing tables are
never renamed.

### Example

```go
import (
	"github.com/ProjAnvil/knot/models"
	"github.com/ProjAnvil/knot/models/gormbind"
)

g := models.Group{Name: "user-service"} // domain type
ok := models.IsValidMethod("GET")       // validation predicate

err := gormbind.AutoMigrate(db) // db *gorm.DB; migrates groups/apis/parameters
```

## Changes in v1.1

Backwards-compatible additions over v1.0 (see [Versioning](#versioning)):

- `SetTablePrefix` / `TablePrefix`: prefix all model tables
  (`knot_groups`/`knot_apis`/`knot_parameters` style) so the models can be
  embedded in a shared database. The default empty prefix preserves the v1.0
  physical schema.
- `ImportResult` relocated verbatim from the Knot backend's services package
  (same fields, same JSON tags: `format`/`groupsCreated`/`groupsMerged`/
  `apisImported`/`parametersImported`/`warnings`) so import tooling can
  consume the preview/confirm payload without importing the backend.
- The embeddable engine module
  ([`backend`](https://github.com/ProjAnvil/knot/blob/main/backend/README.md))
  consumes both additions through this module: `knot.Config.TablePrefix`
  installs the prefix via `SetTablePrefix`, and `knot.Knot.Import` returns
  `*models.ImportResult` (the backend's internal services package aliases it
  as `type ImportResult = models.ImportResult`) — so engine hosts get these
  types through the backend import alone.

## Wire-format artifacts
Two generated artifacts ship in the module so non-Go consumers can speak the
same Knot JSON v1 interchange format:

- [`schema/knot-v1.schema.json`](schema/knot-v1.schema.json) — a
  [JSON Schema draft 2020-12](https://json-schema.org/draft/2020-12) document
  describing the Knot JSON v1 format. Any language can use it to validate
  Knot exports or to generate types.
- [`ts/index.d.ts`](ts/index.d.ts) — TypeScript declarations for the root
  package, for frontends and other TS tooling.

Both files are **generated — do not edit them by hand.** They are derived from
the Go types; regenerate them from the repository root with:

```sh
go run ./tools/cmd/gen-schema   # rewrites models/schema/knot-v1.schema.json
go run ./tools/cmd/gen-ts       # rewrites models/ts/index.d.ts
```

CI drift-locks them: the [`models` job](https://github.com/ProjAnvil/knot/blob/main/.github/workflows/ci.yml)
regenerates both files and fails the build if the committed copies differ, so
a Go type change must always be committed together with its regenerated
artifacts. How the generators work is documented in
[`tools/README.md`](https://github.com/ProjAnvil/knot/blob/main/tools/README.md).

## For maintainers: releasing

Tag the release and publish the GitHub release **without** taking the Latest
badge:

```sh
git tag models/vX.Y.Z && git push origin models/vX.Y.Z
gh release create models/vX.Y.Z --latest=false
```

`--latest=false` matters: the repository's Latest badge must stay on the Knot
product release, not on a library release.

## License

MIT — same as the repository
([LICENSE](https://github.com/ProjAnvil/knot/blob/main/LICENSE)).
