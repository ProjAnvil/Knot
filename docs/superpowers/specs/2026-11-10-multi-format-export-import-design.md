# Multi-Format Export & Import — Design

- Date: 2026-11-10
- Status: Final (builder check applied)
- Target release: v0.6.0

## 1. Intent

Knot currently exports documentation to a single self-contained HTML file. That
serves human readers but blocks everything else: backups, migration between
instances, and interop with the API tooling ecosystem (Postman, Swagger UI,
code generators). This project makes export multi-format and adds a matching
import path, including ingestion of foreign formats, so users can move data in
and out of Knot losslessly.

Decisions already made with the product owner:

- New export formats: **Knot JSON v1** (native, lossless), **OpenAPI 3.1**,
  **Postman Collection v2.1**. HTML stays as-is. XML was considered and
  dropped.
- Import formats: **Knot JSON** (round-trip), **Postman Collection v2.0/v2.1**,
  **OpenAPI 3.x**.
- Group-name conflict on import: **merge** — APIs are appended into the
  existing same-name group.
- RPC-type APIs (which carry no HTTP method) in OpenAPI/Postman export:
  **round-trip via disguise** — exported as `POST` operations carrying an
  `x-knot-type: "rpc"` marker; import reads the marker back and restores
  `Type: "RPC"` with an empty method. Knot JSON stays fully lossless without
  disguise.
- README gains a bilingual (Chinese/English, matching the existing file
  style) section documenting import/export.

## 2. Goals

1. `POST /api/export` gains a `format` field (`html` | `json` | `openapi` |
   `postman`); default `html` keeps current clients working unchanged.
2. `POST /api/import` accepts an uploaded document, auto-detects its format,
   previews the effect (`dryRun`), and applies it inside one transaction.
3. Round-trip guarantee: Knot JSON export → import reproduces groups, APIs,
   parameter trees, and notes with no data loss.
4. Frontend: format picker in the export dialog; new import dialog with
   preview-then-confirm flow. Both fully internationalized (en + zh).
5. Test pyramid: Go unit + handler tests with coverage gating, vitest for new
   frontend code, and a new Playwright E2E suite covering the full
   create → export → wipe → import → verify loop.
6. README documents the new capabilities in a bilingual (zh/en) section
   matching the file's existing convention.
7. Release v0.6.0: PR-based merge to `main`, tag push, English release notes in
   a major-OSS template, published artifacts via the existing workflow, notes
   finalized with `gh`.

## 3. Non-goals

- XML export/import (explicitly dropped by the product owner).
- CLI subcommands (`knot export` / `knot import`) — the web UI is the only
  surface for now.
- Exporting i18n variants for non-HTML formats (JSON/OpenAPI/Postman are
  language-neutral; only HTML keeps its `locale` behavior).
- Postman fidelity beyond requests: auth, headers, pre-request scripts,
  tests, and environments are not imported or exported.
- Full JSON Schema support in OpenAPI import: a pragmatic subset
  (type/properties/required/items/description, local `$ref`) is enough for
  real-world documents; everything else degrades to a warning.
- Scheduled or incremental exports.

## 4. Anti-goals (failure even if it technically works)

- Any database schema migration. Import must reuse the existing
  `groups`/`apis`/`parameters` tables unchanged.
- Any breaking change to `POST /api/export` for callers that omit `format`.
- Any new authentication or permission surface.
- New heavyweight Go dependencies. Knot advertises "zero dependencies"
  deployment; stdlib `encoding/json` generation/parsing is the default.

## 5. Current state (verified in code)

- Backend: Go + Fiber + GORM. `POST /api/export` (`backend/internal/handlers/export.go`)
  takes `{apiIds: []uint}`, fetches APIs + parameters, and calls
  `services.GenerateHTML` (`backend/internal/services/export_service.go`) which
  renders a standalone HTML document (sidebar, parameter tables, example JSON,
  markdown notes via goldmark).
- Models (`backend/internal/models/`): `Group {ID, Name unique, Order}`;
  `API {GroupID, Name, Endpoint, Method (GET/POST/PUT/DELETE/PATCH), Type
  ("HTTP"|"RPC"), Order, Note *string (markdown)}`;
  `Parameter {APIID, ParentID (tree), Name, Type (string|number|boolean|array|
  object), Description, Required, ParamType (request|response), Order}`.
- Frontend: Svelte 5 + Vite + Bun. `ExportDialog.svelte` selects groups/APIs,
  POSTs `/api/export`, downloads the blob as `api-docs-<ts>.html`. Vitest +
  happy-dom/jsdom + @testing-library/svelte already in place; **no Playwright**.
- Tests: Go per-package `_test.go` with in-memory SQLite + Fiber test apps
  (pattern in `handlers/export_test.go`); vitest unit tests in `src/lib`.
- Release: `.github/workflows/release.yml` triggers on `v*` tags, builds all
  binaries (`make package-all`, mcp-server), attaches 12 platform binaries +
  `checksums.txt`, and creates a non-draft release with a bilingual
  (Chinese/English) body via `softprops/action-gh-release@v1`.
- Config: viper-based (`port` default 3000, `sqlitePath` default user data
  dir). `KNOT_SQLITE_PATH` env binding already exists; **`KNOT_PORT` does
  not** — added in §9 for E2E isolation and containers.
- Dev: `frontend/vite.config.ts` proxies `/api` → `http://localhost:3000`.
- No CLI import/export subcommands exist; the MCP tools handler has no export
  tool. Both stay out of scope.

## 6. Approaches considered

**A. Format registry inside the services layer (chosen).** One export handler
parses the request and dispatches to a per-format generator function; one
import handler runs format detection, then a per-format parser, then a shared
apply step. Symmetric, unit-testable per format, zero new routes beyond
`/api/import`, zero new dependencies.

**B. One endpoint per format (`/api/export/json`, `/api/export/openapi`, …).**
Rejected: route sprawl, duplicated fetch/selection logic, and the UI would
need N callers. Nothing in the codebase wants per-format endpoints.

**C. Third-party codecs (kin-openapi, postman exporters).** Rejected as a
default: Knot's value proposition is a single static binary with minimal
dependencies. The OpenAPI subset we emit/consume (§7.3, §8.3) is small and
stable; hand-rolled `encoding/json` with golden tests is cheaper than carrying
a schema library. If full JSON-Schema support is ever needed, swapping the
OpenAPI codec for kin-openapi is localized to one file.

## 7. Export design

### 7.1 API contract

`POST /api/export` body:

```json
{ "apiIds": [1, 2, 3], "format": "json" }
```

- `format` ∈ `html` (default) | `json` | `openapi` | `postman`;
  anything else → 400 `"Unsupported export format"`.
- Response always carries `Content-Disposition: attachment` with a
  format-specific filename:

| format  | Content-Type       | Filename                    |
|---------|--------------------|-----------------------------|
| html    | text/html          | `api-docs.html` (unchanged, byte-compat) |
| json    | application/json   | `knot-export-<unixts>.json` |
| openapi | application/json   | `openapi-<unixts>.json`     |
| postman | application/json   | `postman-<unixts>.json`     |

The HTML row is the current behavior verbatim (the handler today sends a
static `filename="api-docs.html"`); only the three new formats carry a
server-side timestamp.

All formats share the existing fetch phase (APIs → groups → parameters →
`[]services.APIWithParams`), so selection semantics and ordering are identical
across formats.

### 7.2 Knot JSON v1 (native, lossless)

```json
{
  "format": "knot",
  "version": 1,
  "exportedAt": "2026-11-10T08:00:00Z",
  "groups": [
    {
      "name": "Users",
      "order": 0,
      "apis": [
        {
          "name": "Create user",
          "endpoint": "/users",
          "method": "POST",
          "type": "HTTP",
          "order": 0,
          "note": "markdown, or null",
          "parameters": {
            "request":  [ { "name": "...", "type": "object", "required": true,
                             "description": "...", "order": 0,
                             "children": [ "…recursive ParamNode…" ] } ],
            "response": [ "…ParamNode tree…" ]
          }
        }
      ]
    }
  ]
}
```

Rules:

- Parameter trees are **nested** (`children`), not flat parent-ID lists, so a
  file is portable across instances. No DB IDs are exported.
- `exportedAt` is RFC 3339 UTC. `version: 1` gates future schema evolution
  (unknown fields ignored on read; bump version on breaking changes).
- Groups with zero selected APIs are omitted; the group's `order` is its
  current DB order (used only as a hint on import).

### 7.3 OpenAPI 3.1 mapping

```json
{
  "openapi": "3.1.0",
  "info": { "title": "Knot Export", "version": "1.0.0" },
  "tags": [ { "name": "Users" } ],
  "paths": {
    "/users": {
      "post": {
        "operationId": "createUser",
        "summary": "Create user",
        "description": "<note markdown>",
        "tags": ["Users"],
        "requestBody": { "required": true, "content": { "application/json": { "schema": { … } } } },
        "responses": { "200": { "description": "OK", "content": { "application/json": { "schema": { … } } } } }
      }
    }
  }
}
```

- `operationId` = slugified API name (lowercase, non-alphanumerics → `-`).
- ParamNode tree → JSON Schema: `string|number|boolean` map directly;
  `object` → `{type: "object", properties: {…}, required: [names of required
  children]}`; `array` → `{type: "array", items: <child schema or generic>}`;
  `description` is carried on every node.
- Response tree → `responses["200"]`; request tree → `requestBody`
  (`required: true` if any top-level request param is required).
- RPC APIs (`Type == "RPC"`, possibly no HTTP method) are exported as `post`
  operations carrying `"x-knot-type": "rpc"`. OpenAPI consumers that ignore
  vendor extensions see a valid document; Knot's own import (§8.3) reads the
  marker back and restores `Type: "RPC"` with an empty method, making the
  export→import loop lossless for the classification.
- Collision policy (same endpoint + method selected twice): operations merge —
  first occurrence wins for `summary`/`operationId`; subsequent notes are
  appended to `description` separated by `\n\n---\n\n`. Deterministic in input
  (selection) order.
- Path keys with template variables pass through unchanged (`/users/{id}`).

### 7.4 Postman Collection v2.1 mapping

```json
{
  "info": {
    "name": "Knot Export",
    "_postman_id": "<generated uuid v4>",
    "description": "Exported from Knot at <RFC3339>",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "item": [
    {
      "name": "Users",
      "item": [
        {
          "name": "Create user",
          "request": {
            "method": "POST",
            "header": [],
            "url": { "raw": "/users" },
            "description": "<note markdown>",
            "body": { "mode": "raw", "raw": "<example JSON from request params>",
                      "options": { "raw": { "language": "json" } } }
          },
          "response": [ { "name": "Example 200", "body": "<example JSON from response params>" } ]
        }
      ]
    }
  ]
}
```

- Example bodies reuse the existing `services.GenerateExampleJSON` tree
  walker; absent params → omit `body`/`response`.
- RPC APIs are exported with `request.method: "POST"` plus a top-level
  custom property `"x-knot-type": "rpc"` on the item. Postman ignores
  unknown properties; Knot's import restores `Type: "RPC"` (§8.3).
- Groups with zero selected APIs are omitted.

## 8. Import design

### 8.1 API contract

`POST /api/import`, `multipart/form-data`:

- `file` (required): the document. Server-side size cap 10 MB (Fiber
  `BodyLimit` raised accordingly in route setup).
- `dryRun` (optional, `"true"`): parse + analyze, write nothing.

Response (`response.Success` envelope, same shape for dry-run and real run):

```json
{
  "success": true,
  "data": {
    "format": "knot-json",
    "groupsCreated": 1,
    "groupsMerged": 1,
    "apisImported": 5,
    "parametersImported": 23,
    "warnings": ["Postman item 'X': body is not valid JSON, skipped params"]
  }
}
```

Errors: 400 for a malformed file of a known format (with a specific message);
415 `"Unsupported file format"` for anything unrecognizable (including
Swagger 2.0 documents — message names the supported list).

### 8.2 Format detection

Attempt `json.Unmarshal` into `map[string]json.RawMessage`:

1. root has `"format": "knot"` → **knot-json**
2. root has `info.schema` containing `schema.getpostman.com` → **postman**
   (v2.0 and v2.1 URL prefixes both accepted)
3. root has `"openapi": "3.x…"` → **openapi**
4. root has `"swagger"` → 415 with "OpenAPI 3.x required" hint

Not valid JSON → 415. The detection order is unambiguous because the three
signatures are disjoint.

### 8.3 Format → `ImportPlan` mappings

All parsers produce the same intermediate:

```go
type ImportPlan struct { Groups []ImportGroup }
type ImportGroup struct { Name string; APIs []ImportAPI }
type ImportAPI struct { Name, Endpoint, Method, Type, Note string; HasNote bool;
                        Request, Response []ParamNode }
```

**Knot JSON:** direct inverse of §7.2. `version != 1` → 400. Unknown fields
ignored. `method` upper-cased; `type` validated against `HTTP|RPC`
(default `HTTP` when absent).

**Postman:** walk `item[]` recursively. If an item carries the custom
`"x-knot-type": "rpc"` property, the imported API gets `Type: "RPC"` and an
empty `method` (round-trip of §7.4); otherwise `Type: "HTTP"` with the
item's method.

- Folder (has `item`) → group named `"<parent>/<child>"` (top-level folders
  use their own name; nesting flattens with `/` separators).
- Top-level lone request → group named after `info.name` (collection name).
- Request → API: `name`, `request.method`, endpoint = `request.url.raw` (or
  the string form), note = `request.description` (`string` or
  `{content: …}` variant). `{{variable}}` placeholders pass through verbatim.
- `request.body.mode == "raw"` and body parses as a JSON object → convert to
  request ParamNodes via the existing JSON→parameters service logic
  (reuse of `parameter_service`; types inferred: string/number/boolean,
  nested objects/arrays). Parse failure → warning, params skipped.

**OpenAPI 3.x:** for each `path` × method (`get|post|put|delete|patch|head|
options|trace`):

- API name = `summary` || `operationId` || `"<METHOD> <path>"`;
  endpoint = path; method upper-cased; note = `description`.
- An operation carrying `"x-knot-type": "rpc"` imports as `Type: "RPC"`
  with an empty `method` (round-trip of §7.3); all other operations
  import as `Type: "HTTP"`.
- Group = first `tags[]` entry, else `"Imported"`.
- `requestBody.content["application/json"].schema` → ParamNode tree (inverse
  of §7.3). Local `$ref: "#/components/schemas/X"` resolved against the
  document; cycles guarded by a visited set; depth cap 32; unresolved or
  external refs → warning + subtree skipped.
- `responses`: prefer `"200"`, then `"2XX"`, then `default`, first with an
  `application/json` schema → response ParamNodes.
- `components.schemas` entries not reachable from any path are **not**
  imported (they have no endpoint).
- `servers` is ignored — Knot stores bare paths.

### 8.4 Apply step (merge semantics, transaction)

`ApplyImport(db *gorm.DB, plan *ImportPlan, dryRun bool) (*ImportResult, error)`:

- Whole import runs in one `db.Transaction`; any failure rolls back
  everything (all-or-nothing).
- For each `ImportGroup`, look up `groups.name` **inside the transaction**:
  - exists → **merge**: append APIs; each imported API's `order` continues
    after the current max `order` in that group.
  - missing → create; `order` continues after the current max group order.
- Parameter trees are flattened to rows with `ParentID` links and per-API
  sequential `order`, reusing the tree the parsers built.
- No API-level deduplication: importing the same file twice duplicates APIs
  inside merged groups. The preview makes this visible before commit, and
  duplication is trivially reversible (delete group/API). Documented.
- `dryRun` runs the same resolution logic without writes and reports the
  counts the real run would produce.

## 9. Backend layout

New files (existing files unchanged unless noted):

```
backend/internal/services/
  param_node.go          // ParamNode DTO + tree⇄models.Parameter converters
  export_knot.go         // BuildKnotDocument / marshal
  export_openapi.go      // BuildOpenAPIDocument
  export_postman.go      // BuildPostmanCollection
  import_detect.go       // DetectImportFormat
  import_knot.go         // parse Knot JSON → ImportPlan
  import_postman.go      // parse Postman → ImportPlan
  import_openapi.go      // parse OpenAPI → ImportPlan
  import_apply.go        // ApplyImport (transaction, merge, counts)
backend/internal/handlers/
  import.go              // ImportDocument handler (multipart, dryRun)
  export.go              // extended: format switch (filename/content-type table)
  routes.go              // + api.Post("/import", ImportDocument(db)); BodyLimit 10MB
backend/internal/config/
  config.go              // + KNOT_PORT env binding (KNOT_SQLITE_PATH already exists)
```

`export_service.go` keeps HTML generation untouched. Each new file gets a
matching `_test.go`.

## 10. Frontend design

- `src/lib/types.ts`: `ExportFormat = 'html'|'json'|'openapi'|'postman'`,
  `ImportPreview` mirroring §8.1 `data`.
- `src/lib/api.ts`: `exportApis(apiIds, format)` → `{blob, filename}` (filename
  parsed from `Content-Disposition`), `importDocument(file, dryRun)` →
  multipart POST.
- `ExportDialog.svelte`: format picker — four radio cards (icon + label +
  one-line description) above the selection tree; button label neutral
  ("Export"); download filename from the response header instead of the
  hardcoded `.html`.
- New `ImportDialog.svelte`: trigger button (Upload icon) beside Export in the
  sidebar. Flow: pick file (`accept=".json,application/json"`) → automatic
  `dryRun` import → preview panel (detected format badge, groups
  created/merged, API/param counts, warnings list) → Confirm → real import →
  success toast → parent refreshes groups. Error states: 415 → "unsupported
  format" message; 400 → server message verbatim.
- i18n: full `export.format.*`, `import.*` keys in
  `frontend/src/lib/i18n/locales/en.json` and `zh-CN.json`.
- Vitest: `api.test.ts` additions (fetch mock incl. multipart FormData,
  Content-Disposition parsing); `ImportDialog` render + flow test with
  @testing-library/svelte.

## 11. Playwright E2E (new infrastructure)

- `frontend/playwright.config.ts`, `frontend/e2e/` specs + `e2e/fixtures/`,
  `@playwright/test` devDependency, Chromium installed via
  `bunx playwright install chromium`.
- `webServer`: one shell script (`e2e/run-env.sh`) that builds the frontend,
  builds the Go CLI binary with the embedded-frontend build tag
  (`-tags embedfrontend`, same artifact shape as `make package`), and runs it
  in the foreground (`knot __serve`, see `backend/internal/cli/serve.go`)
  against `KNOT_PORT=3999` and `KNOT_SQLITE_PATH=$TMPDIR/knot-e2e.db` (env
  binding from §9). Global setup deletes any stale DB so every run starts
  empty. Frontend is served by the Go binary's embedded static handler — E2E
  tests the real shipped artifact, not a dev-server proxy.
- Specs:
  1. `export-import-roundtrip.spec.ts` — create group + API + nested params +
     note via UI → export JSON (Playwright download event; assert envelope
     `format/version/groups`) → delete the group via UI → import the file →
     assert tree, params, note are back.
  2. `import-postman.spec.ts` — `setInputFiles` a fixture collection →
     preview counts → confirm → assert groups/APIs in sidebar and API view.
  3. `import-openapi.spec.ts` — fixture minimal petstore-style document →
     assert names/params imported.
  4. `export-formats.spec.ts` — export OpenAPI and Postman via UI and assert
     structural keys of the downloaded JSON; HTML export still downloads and
     contains an `api-section`.
- `package.json`: `"test:e2e": "playwright test"`.

## 12. Coverage & CI

- Backend `Makefile`: `test-coverage` target —
  `go test ./... -race -covermode=atomic -coverprofile=coverage.out` then
  `go tool cover -func`. Gate: new `services`/`handlers` code ≥ 85% statement
  coverage; overall package coverage reported in the PR.
- Frontend: `bun run test:coverage` must stay green; new modules covered.
- New `.github/workflows/ci.yml` on `push(main)` + `pull_request`:
  1. Go: build + `make test-coverage` (Go 1.25, ubuntu-latest)
  2. Frontend: `bun install`, `bun run check`, `bun run test`, `bun run build`
  3. Playwright: build server binary, run `bun run test:e2e`, upload traces
     on failure
  This is standard gating for a serious OSS project and protects the release
  path (release.yml can additionally `needs` nothing — it stays tag-driven).

## 13. Release plan

1. Branch `feat/multi-format-export-import`; PR to `main`; squash-merge with a
   conventional-commit title (English). The PR includes the bilingual README
   section (§2 goal 6).
2. `release.yml` body template replaced with an English major-OSS layout:
   `## Highlights`, `## What's Changed` (grouped: Features / Tests / CI &
   Release), `## New Contributors` (if any), `## Full Changelog` (diff link),
   `## Downloads` (asset table incl. checksums note). Asset list unchanged.
3. CI on `main` green → tag `v0.6.0` → push → workflow builds and publishes.
4. Verify: `gh release view v0.6.0 --json assets` (12 platform binaries +
   `checksums.txt`), then `gh release edit v0.6.0 --notes-file <rendered
   template>` if the auto-generated commit list needs enriching.
5. All repo-facing text in English: code comments, commit messages, PR body,
   release notes (per product owner requirement).

## 14. Risks & mitigations

| Risk | Mitigation |
|------|------------|
| `groups.name` unique constraint race between lookup and insert | Lookup + insert inside one transaction; SQLite serializes writers, PG/MySQL rely on tx isolation + unique error → retry once with merge path |
| OpenAPI `$ref` cycles / depth bombs | Visited set per resolution + depth cap 32 → warning, subtree skipped |
| Oversized uploads | BodyLimit 10 MB; handler validates before parse |
| HTML export regression | Format default `html`; existing handler tests keep passing byte-for-byte behavior checks |
| Playwright flakiness in CI | Traces saved on failure; specs use `expect.poll`/auto-waiting; no artificial sleeps |
| Bilingual release body replaced | Intentional: product owner mandates English-only notes |

## 15. Execution sequence (as directed by product owner)

1. This spec → **subagent audit** (builder check) → apply fixes.
2. Plan (`superpowers:writing-plans`) → **subagent audit** → apply fixes.
3. **Subagent execution** (`superpowers:subagent-driven-development`, TDD).
4. `superpowers:verification-before-completion`: full suite green locally
   (Go + coverage, vitest + coverage, Playwright), CI green on the PR.
5. Merge → tag `v0.6.0` → release via workflow + `gh` (English notes).

## 16. Open items settled by the author (marked as calls, not agreements)

- `operationId` slug rule and the OpenAPI collision-merge policy (§7.3).
- Postman nested-folder flattening with `/` separators (§8.3).
- Loose top-level Postman requests land in a group named after the collection
  (§8.3).
- OpenAPI unreachable `components.schemas` are not imported (§8.3).
- No API-level dedupe on import; preview is the safety net (§8.4).
- Env binding `KNOT_PORT` / `KNOT_SQLITE_PATH` added for E2E and container
  users (§9) — smallest change that makes isolated runs possible.
