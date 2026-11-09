# Multi-Format Export & Import Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend Knot's export beyond HTML (Knot JSON v1, OpenAPI 3.1, Postman v2.1) and add auto-detected import (Knot JSON, Postman, OpenAPI 3.x) with merge-on-conflict and preview-before-commit, fully tested (unit + coverage + Playwright E2E), released as v0.6.0.

**Architecture:** A format registry inside `backend/internal/services`: one shared fetch phase feeds per-format export builders; import runs format detection → per-format parser producing a common `ImportPlan` → one transactional apply step with merge semantics. Frontend gains a format picker in `ExportDialog` and a new preview-first `ImportDialog`. Playwright E2E drives the real embedded binary.

**Tech Stack:** Go 1.25 (Fiber v2, GORM, stdlib `encoding/json`), Svelte 5 + Vite + Bun, vitest + happy-dom, @playwright/test (new), GitHub Actions.

**Spec:** `docs/superpowers/specs/2026-11-10-multi-format-export-import-design.md`

## Global Constraints

- ALL code comments, commit messages, PR body, and release notes in **English**.
- **No new Go module dependencies** — stdlib only (`encoding/json`, `crypto/rand`, `mime`).
- **No DB schema changes/migrations** — reuse `groups`/`apis`/`parameters` tables.
- `POST /api/export` with no `format` field must behave exactly as today (HTML, `api-docs-<unixts>.html`).
- Fiber `BodyLimit` = 10 MB wherever the app is created (`internal/cli/serve.go:50`, `cmd/server/main.go:38`, handler test apps).
- Import group conflict: **merge** (append APIs to same-name group; new API `order` continues after that group's max; new groups continue after max group `order`).
- RPC round-trip in OpenAPI/Postman: export as `POST` + `x-knot-type: "rpc"`; import restores `Type: "RPC"`, `Method: ""`.
- Coverage gate: **≥ 85% statements on every new `services`/`handlers` file**, measured by `make test-coverage` (backend).
- Conventional Commits (`feat:`, `feat(frontend):`, `test:`, `ci:`, `docs:`); one commit per task.
- Test naming follows the existing `givenX_whenY_thenZ` convention in `_test.go`.
- Work on branch `feat/multi-format-export-import` (already created; spec already committed).

## Review Focus

Failure modes the spec implies but task tests must pin (each pinned in its owning task):

1. **UTF-8 BOM before JSON** in an uploaded import file (Windows editors) → detection must strip BOM before parsing, not return 415. Pinned in Task 6.
2. **Postman `request.url` as plain string** (v2.0 exports) alongside the v2.1 object form → both accepted. Pinned in Task 7.
3. **OpenAPI `$ref` cycle** (`A → B → A`) → warning + subtree skipped, parser terminates. Pinned in Task 8.
4. **Missing `format` field on export** → HTML bytes identical to current behavior (backward compat). Pinned in Task 5.
5. **Two selected APIs sharing endpoint+method in OpenAPI export** → deterministic merge (first summary wins, notes joined by `\n\n---\n\n`). Pinned in Task 3.

---

### Task 1: ParamNode DTO and tree converters

**Files:**
- Create: `backend/internal/services/param_node.go`
- Test: `backend/internal/services/param_node_test.go`

**Interfaces:**
- Consumes: `models.Parameter`, `models.APIWithParams` (existing).
- Produces (used by Tasks 2–9):

```go
// param_node.go
type ParamNode struct {
    Name        string      `json:"name"`
    Type        string      `json:"type"`              // string|number|boolean|array|object
    Required    bool        `json:"required"`
    Description *string     `json:"description"`
    Order       int         `json:"order"`
    Children    []ParamNode `json:"children"`
}
// Flat ordered parameter rows (as fetched for one API + paramType) → nested tree.
func TreeFromParams(params []models.Parameter) []ParamNode
// Best-effort JSON value → ParamNodes (type inference: string/number/boolean/array/object).
// Used by the Postman importer for raw JSON bodies.
func ParamNodesFromJSONValue(v interface{}) []ParamNode
// Total node count of a tree (for ImportResult.ParametersImported).
func CountNodes(nodes []ParamNode) int
```

- [ ] **Step 1: Write failing tests** — `TestTreeFromParams` (flat rows with `ParentID` links → nested `Children`, order preserved), `TestParamNodesFromJSONValue` (nested `map[string]interface{}` from `json.Unmarshal` → typed nodes: string→string, float64→number, bool→boolean, []any→array, map→object with children), `TestCountNodes`.
- [ ] **Step 2: Run** `cd backend && go test ./internal/services/ -run 'TestTreeFromParams|TestParamNodesFromJSONValue|TestCountNodes' -v` → FAIL (undefined).
- [ ] **Step 3: Implement `param_node.go`.** For `TreeFromParams`, mirror the parent-map walk in `export_service.go` `BuildParameterTree` but emit `ParamNode`. `ParamNodesFromJSONValue` sorts map keys for deterministic order.
- [ ] **Step 4: Run** same command → PASS.
- [ ] **Step 5: Commit** `git add backend/internal/services/param_node.go backend/internal/services/param_node_test.go && git commit -m "feat: add ParamNode DTO and tree converters for portable parameter data"`

### Task 2: Knot JSON v1 export document

**Files:**
- Create: `backend/internal/services/export_knot.go`
- Test: `backend/internal/services/export_knot_test.go`

**Interfaces:**
- Consumes: `APIWithParams`, `ParamNode`/`TreeFromParams` (Task 1).
- Produces (used by Task 5; parser in Task 6 reads the same JSON shape):

```go
// export_knot.go
type KnotExportDocument struct { Format string `json:"format"`; Version int `json:"version"`; ExportedAt string `json:"exportedAt"`; Groups []KnotExportGroup `json:"groups"` }
type KnotExportGroup struct { Name string `json:"name"`; Order int `json:"order"`; APIs []KnotExportAPI `json:"apis"` }
type KnotExportAPI struct { Name string `json:"name"`; Endpoint string `json:"endpoint"`; Method string `json:"method"`; Type string `json:"type"`; Order int `json:"order"`; Note *string `json:"note"`; Parameters KnotExportParams `json:"parameters"` }
type KnotExportParams struct { Request []ParamNode `json:"request"`; Response []ParamNode `json:"response"` }
func BuildKnotDocument(apis []APIWithParams, now time.Time) *KnotExportDocument
```

- [ ] **Step 1: Write failing test** `TestBuildKnotDocument`: two APIs in one group + one in another, nested request/response params, one nil note → assert `Format=="knot"`, `Version==1`, `ExportedAt==now.UTC().Format(time.RFC3339)`, group names/orders, `parameters.request[0].children` nesting, `method` upper-cased, empty groups omitted.
- [ ] **Step 2: Run** `go test ./internal/services/ -run TestBuildKnotDocument -v` → FAIL.
- [ ] **Step 3: Implement.** Group `APIWithParams` by `API.GroupID` (insertion order, as `GenerateHTML` does); per API build `TreeFromParams` for each paramType.
- [ ] **Step 4: Run** → PASS. Also run `go test ./internal/services/` (no regressions).
- [ ] **Step 5: Commit** `feat: add Knot JSON v1 export document builder`

### Task 3: OpenAPI 3.1 export

**Files:**
- Create: `backend/internal/services/export_openapi.go`
- Test: `backend/internal/services/export_openapi_test.go`

**Interfaces:**
- Consumes: Task 1–2 types.
- Produces:

```go
// export_openapi.go — typed structs with json tags; unknown-key paths held as map[string]*OpenAPIPathItem
type OpenAPIDocument struct { OpenAPI string `json:"openapi"`; Info OpenAPIInfo `json:"info"`; Tags []OpenAPITag `json:"tags,omitempty"`; Paths map[string]*OpenAPIPathItem `json:"paths"` }
func BuildOpenAPIDocument(apis []APIWithParams) *OpenAPIDocument
// plus OpenAPIInfo{Title,Version}, OpenAPITag{Name}, OpenAPIPathItem, OpenAPIOperation
// (Summary, Description, OperationID, Tags, XKnotType `json:"x-knot-type,omitempty"`,
//  RequestBody *OpenAPIRequestBody, Responses map[string]*OpenAPIResponse)
// ParamNode tree → JSON Schema map: func NodesToJSONSchema(nodes []ParamNode, objectRequired bool) map[string]interface{}
```

- [ ] **Step 1: Write failing tests:**
  - `TestNodesToJSONSchema`: object with required+optional children → `{type: object, properties, required: [only required names]}`; array node → `{type: array, items}`; leaf types map directly; description carried.
  - `TestBuildOpenAPIDocument`: GET `/users/{id}` with note → `paths["/users/{id}"]["get"]` with `summary`, `description` = note, `tags=["Users"]`, `operationId` slug (`"Create User!"` → `create-user`), requestBody/responses `200` schemas.
  - `TestBuildOpenAPIDocument_RPC`: RPC API (empty method) → keyed as `post` with `x-knot-type: "rpc"`.
  - `TestBuildOpenAPIDocument_Collision` (Review Focus 5): two APIs same endpoint+method → one operation; first summary wins; descriptions joined `\n\n---\n\n` in input order.
- [ ] **Step 2: Run** `go test ./internal/services/ -run 'TestNodesToJSONSchema|TestBuildOpenAPIDocument' -v` → FAIL.
- [ ] **Step 3: Implement.** RPC (or empty-method) APIs → verb `post` + `XKnotType:"rpc"`. Response tree → `responses["200"]`; request tree → requestBody `required: true` iff any root node required.
- [ ] **Step 4: Run** → PASS.
- [ ] **Step 5: Commit** `feat: add OpenAPI 3.1 export builder`

### Task 4: Postman Collection v2.1 export

**Files:**
- Create: `backend/internal/services/export_postman.go`
- Test: `backend/internal/services/export_postman_test.go`

**Interfaces:**
- Consumes: Task 1 types, `GenerateExampleJSON` (existing in `export_service.go`).
- Produces:

```go
// export_postman.go
type PostmanCollection struct { Info PostmanInfo `json:"info"`; Item []PostmanItem `json:"item"` }
type PostmanInfo struct { Name string `json:"name"`; PostmanID string `json:"_postman_id"`; Description string `json:"description,omitempty"`; Schema string `json:"schema"` }
type PostmanItem struct { Name string `json:"name"`; XKnotType string `json:"x-knot-type,omitempty"`; Item []PostmanItem `json:"item,omitempty"`; Request *PostmanRequest `json:"request,omitempty"` }
func BuildPostmanCollection(apis []APIWithParams, now time.Time) *PostmanCollection
func NewUUIDv4() string  // crypto/rand, no dependency
```

- [ ] **Step 1: Write failing test** `TestBuildPostmanCollection`: group→folder items; request item has `method` (RPC → `"POST"` + `x-knot-type:"rpc"`), `url.raw` = endpoint, `description` = note, `body.mode=="raw"` with `GenerateExampleJSON` output when request params exist (omitted when absent), `response[0].body` from response params; `info.schema` is the v2.1 URL; `_postman_id` matches UUID format. Plus `TestNewUUIDv4` (unique across 100 calls).
- [ ] **Step 2: Run** `go test ./internal/services/ -run 'TestBuildPostmanCollection|TestNewUUIDv4' -v` → FAIL.
- [ ] **Step 3: Implement.** Reuse `GenerateExampleJSON` on the built param trees; marshal examples with `json.MarshalIndent(..., "", "  ")`.
- [ ] **Step 4: Run** → PASS.
- [ ] **Step 5: Commit** `feat: add Postman Collection v2.1 export builder`

### Task 5: Export handler format switch

**Files:**
- Modify: `backend/internal/handlers/export.go`
- Test: modify `backend/internal/handlers/export_test.go`

**Interfaces:**
- Consumes: `BuildKnotDocument`, `BuildOpenAPIDocument`, `BuildPostmanCollection`, existing `GenerateHTML`.
- Produces: `POST /api/export` body `{apiIds []uint, format string}`; responses per spec §7.1 table (Content-Type + `Content-Disposition: attachment; filename=...` built with `mime.FormatMediaType`). Unknown format → `response.BadRequest(c, "Unsupported export format")`.

- [ ] **Step 1: Write failing tests** in `export_test.go` (reuse `setupTestDB`/`setupExportTestApp`, extend the app setup only if routes change — they don't):
  - `givenNoFormatField_whenExport_thenHTMLIdenticalBehavior`: assert status 200, `Content-Type: text/html; charset=utf-8`, `Content-Disposition` contains `api-docs-` and `.html` (Review Focus 4).
  - `givenFormatJSON_whenExport_thenKnotEnvelope`: body parses as JSON with `format:"knot"`, `version:1`; filename `knot-export-*.json`.
  - `givenFormatOpenAPI_whenExport_thenOpenAPIEnvelope`: `openapi:"3.1.0"`; filename `openapi-*.json`.
  - `givenFormatPostman_whenExport_thenPostmanEnvelope`: `info.schema` contains `schema.getpostman.com`; filename `postman-*.json`.
  - `givenUnknownFormat_whenExport_then400`.
- [ ] **Step 2: Run** `go test ./internal/handlers/ -run TestExportAPIs -v` → new subtests FAIL.
- [ ] **Step 3: Implement** the switch in `ExportAPIs`; keep the fetch phase untouched; `time.Now()` passed to builders.
- [ ] **Step 4: Run** full `go test ./internal/handlers/` → PASS.
- [ ] **Step 5: Commit** `feat: support multi-format export in export endpoint`

### Task 6: Import detection + Knot JSON parser

**Files:**
- Create: `backend/internal/services/import_detect.go`, `backend/internal/services/import_knot.go`
- Test: `backend/internal/services/import_detect_test.go`, `backend/internal/services/import_knot_test.go`

**Interfaces:**
- Consumes: `ParamNode` (Task 1), `KnotExportDocument` shape (Task 2).
- Produces (used by Tasks 7–10):

```go
// import_detect.go
type ImportFormat string
const ( ImportFormatKnot ImportFormat = "knot-json"; ImportFormatPostman ImportFormat = "postman"; ImportFormatOpenAPI ImportFormat = "openapi" )
// Strips UTF-8 BOM, unmarshals, applies spec §8.2 signature order.
func DetectImportFormat(data []byte) (ImportFormat, error) // error → 415-style message strings

// import_knot.go — plan types live here (shared by all parsers)
type ImportPlan struct { Groups []ImportGroup; Warnings []string }
type ImportGroup struct { Name string; APIs []ImportAPI }
type ImportAPI struct { Name, Endpoint, Method, Type string; Note *string; Request, Response []ParamNode }
func ParseKnotJSON(data []byte) (*ImportPlan, error)
```

- [ ] **Step 1: Write failing tests:**
  - `TestDetectImportFormat`: knot / postman v2.1 / postman v2.0 schema URL / openapi 3.0.x / openapi 3.1 / swagger 2.0 (error mentions "OpenAPI 3") / not JSON (error) / **BOM-prefixed knot file detected** (Review Focus 1) / empty input (error).
  - `TestParseKnotJSON_RoundTrip`: build fixture via `BuildKnotDocument` from in-memory `APIWithParams`, marshal, parse → `ImportPlan` groups/APIs/param trees/notes equal (methods upper-cased, `type` defaults `"HTTP"`, `version != 1` → error).
- [ ] **Step 2: Run** `go test ./internal/services/ -run 'TestDetectImportFormat|TestParseKnotJSON' -v` → FAIL.
- [ ] **Step 3: Implement.** Detection per spec §8.2 (disjoint signatures: root `format=="knot"`, `info.schema` contains `schema.getpostman.com`, root `openapi` major `"3"`).
- [ ] **Step 4: Run** → PASS.
- [ ] **Step 5: Commit** `feat: add import format detection and Knot JSON parser`

### Task 7: Postman parser

**Files:**
- Create: `backend/internal/services/import_postman.go`
- Test: `backend/internal/services/import_postman_test.go`

**Interfaces:**
- Consumes: `ImportPlan`, `ParamNodesFromJSONValue` (Task 1), `DetectImportFormat` types.
- Produces: `func ParsePostmanCollection(data []byte) (*ImportPlan, error)`

- [ ] **Step 1: Write failing tests** with inline JSON fixtures:
  - folder + nested subfolder → groups `"Folder"` and `"Folder/Sub"`; top-level lone request → group named `info.name`; `request.url` as object (`.raw`) and **as plain string** (Review Focus 2); `description` as string and as `{content: ...}` object; `x-knot-type:"rpc"` item → `Type:"RPC"`, `Method:""`; raw JSON `body` → request ParamNodes; body invalid JSON → `Warnings` non-empty, params empty; method upper-cased.
- [ ] **Step 2: Run** `go test ./internal/services/ -run TestParsePostmanCollection -v` → FAIL.
- [ ] **Step 3: Implement** recursive item walk per spec §8.3.
- [ ] **Step 4: Run** → PASS.
- [ ] **Step 5: Commit** `feat: add Postman collection importer`

### Task 8: OpenAPI 3.x parser

**Files:**
- Create: `backend/internal/services/import_openapi.go`
- Test: `backend/internal/services/import_openapi_test.go`

**Interfaces:**
- Consumes: `ImportPlan`; `NodesToJSONSchema` inverse lives here: `func JSONSchemaToNodes(schema map[string]interface{}, root map[string]interface{}, visited map[string]bool, depth int) []ParamNode` (unexported helpers fine).
- Produces: `func ParseOpenAPIDocument(data []byte) (*ImportPlan, error)`

- [ ] **Step 1: Write failing tests** with inline fixtures:
  - paths×methods iteration (get+post on same path → 2 APIs); name fallback chain `summary`→`operationId`→`"METHOD /path"`; first tag → group, no tag → `"Imported"`; requestBody schema → request nodes; `responses["200"]` preferred over `default`; local `$ref: "#/components/schemas/X"` resolved; **cyclic `$ref` terminates with warning + subtree skipped** (Review Focus 3); depth > 32 → warning; external `http://...` ref → warning; `x-knot-type:"rpc"` → `Type:"RPC"`, `Method:""`; `components.schemas` unreachable entries not imported.
- [ ] **Step 2: Run** `go test ./internal/services/ -run TestParseOpenAPIDocument -v` → FAIL.
- [ ] **Step 3: Implement.** Parse into `map[string]interface{}`; guarded resolver with visited-set + depth cap 32 (spec §8.3).
- [ ] **Step 4: Run** → PASS.
- [ ] **Step 5: Commit** `feat: add OpenAPI 3.x importer`

### Task 9: Transactional apply with merge semantics

**Files:**
- Create: `backend/internal/services/import_apply.go`
- Test: `backend/internal/services/import_apply_test.go`

**Interfaces:**
- Consumes: `ImportPlan` (Task 6), `models.*`, `*gorm.DB` (test DB setup pattern from `export_service_test.go` / `parameter_service_test.go`).
- Produces (used by Task 10):

```go
type ImportResult struct {
    Format             string   `json:"format"`
    GroupsCreated      int      `json:"groupsCreated"`
    GroupsMerged       int      `json:"groupsMerged"`
    APIsImported       int      `json:"apisImported"`
    ParametersImported int      `json:"parametersImported"`
    Warnings           []string `json:"warnings"`
}
func ApplyImport(db *gorm.DB, format ImportFormat, plan *ImportPlan, dryRun bool) (*ImportResult, error)
```

- [ ] **Step 1: Write failing tests:**
  - fresh DB: groups created with order 0..n, APIs + nested params (ParentID chain) correct, counts exact.
  - existing same-name group: `GroupsMerged==1`, APIs appended, API order continues after existing max, group order untouched.
  - `dryRun=true`: identical result counts, zero rows written.
  - parse warnings surfaced into `Warnings`.
  - failure mid-import (make one API row invalid, e.g. empty name → DB error) → transaction rolled back, DB unchanged.
- [ ] **Step 2: Run** `go test ./internal/services/ -run TestApplyImport -v` → FAIL.
- [ ] **Step 3: Implement** per spec §8.4: single `db.Transaction`; recursive parameter insert (create parent row, use returned ID for children); group lookup inside the tx.
- [ ] **Step 4: Run** → PASS.
- [ ] **Step 5: Commit** `feat: add transactional import apply with merge semantics`

### Task 10: Import endpoint + wiring

**Files:**
- Create: `backend/internal/handlers/import.go`
- Modify: `backend/internal/handlers/routes.go` (add `api.Post("/import", ImportDocument(db))`), `backend/internal/cli/serve.go:50` and `cmd/server/main.go:38` (`BodyLimit: 10 * 1024 * 1024`)
- Test: `backend/internal/handlers/import_test.go`

**Interfaces:**
- Consumes: `DetectImportFormat`, `ParseKnotJSON`/`ParsePostmanCollection`/`ParseOpenAPIDocument`, `ApplyImport`, `response` helpers.
- Produces: `func ImportDocument(db *gorm.DB) fiber.Handler` — multipart `file` (required) + `dryRun`; success → `response.Success(c, result)`; unknown/malformed → `response.Error(c, fiber.StatusUnsupportedMediaType, ...)` / `response.BadRequest`; missing file → `BadRequest("No file provided")`.

- [ ] **Step 1: Write failing tests** (new `setupImportTestApp` mirroring `setupExportTestApp`, `BodyLimit: 10MB`): dryRun returns counts and writes nothing; real import persists (query DB); missing `file` → 400; garbage bytes → 415; swagger 2.0 → 415 with "OpenAPI 3" in message; oversized file (> 10 MB) → 413.
- [ ] **Step 2: Run** `go test ./internal/handlers/ -run TestImportDocument -v` → FAIL.
- [ ] **Step 3: Implement** handler + route + BodyLimit bumps.
- [ ] **Step 4: Run** full `go test ./...` in `backend/` → PASS.
- [ ] **Step 5: Commit** `feat: add POST /api/import endpoint with dry-run preview`

### Task 11: KNOT_PORT env override

**Files:**
- Modify: `backend/internal/config/config.go` (the existing "Environment variable overrides" block)
- Test: `backend/internal/config/config_test.go`

**Interfaces:**
- Consumes: existing `KNOT_SQLITE_PATH` override pattern (adjacent code).
- Produces: `KNOT_PORT` env → `config.Port` (needed by Task 15's E2E runner).

- [ ] **Step 1: Write failing test** `TestLoadConfig_PortEnvOverride` (set env, assert `config.Port`), mirroring the existing sqlite-path test.
- [ ] **Step 2: Run** `go test ./internal/config/ -run TestLoadConfig_PortEnvOverride -v` → FAIL.
- [ ] **Step 3: Implement** one `strconv.Atoi` + assignment in the override block.
- [ ] **Step 4: Run** `go test ./internal/config/` → PASS.
- [ ] **Step 5: Commit** `feat: support KNOT_PORT environment override`

### Task 12: Frontend types + API client

**Files:**
- Modify: `frontend/src/lib/types.ts`, `frontend/src/lib/api.ts`
- Test: modify `frontend/src/lib/api.test.ts`

**Interfaces:**
- Consumes: existing `ApiResult` wrapper style.
- Produces:

```ts
// types.ts
export type ExportFormat = 'html' | 'json' | 'openapi' | 'postman'
export interface ImportPreview { format: string; groupsCreated: number; groupsMerged: number;
  apisImported: number; parametersImported: number; warnings: string[] }
// api.ts
export async function exportApis(apiIds: number[], format: ExportFormat): Promise<{ blob: Blob; filename: string } | { error: string }>
export async function importDocument(file: File, dryRun: boolean): Promise<ApiResult<ImportPreview>>
```

- [ ] **Step 1: Write failing vitest tests** (mock `fetch`): `exportApis` posts `{apiIds, format}`, parses filename from `Content-Disposition` via regex `filename="([^"]+)"`, returns blob; non-OK → error object. `importDocument` posts `FormData` with `file` + `dryRun`, returns parsed `ApiResult`.
- [ ] **Step 2: Run** `cd frontend && bun run test -- src/lib/api.test.ts` → FAIL.
- [ ] **Step 3: Implement** both functions following the existing try/catch style in `api.ts`.
- [ ] **Step 4: Run** `bun run test` → PASS.
- [ ] **Step 5: Commit** `feat(frontend): add export/import API client functions`

### Task 13: ExportDialog format picker

**Files:**
- Modify: `frontend/src/lib/components/dialogs/ExportDialog.svelte`
- Modify: `frontend/src/lib/i18n/locales/en.json`, `frontend/src/lib/i18n/locales/zh-CN.json` (add `export.format.html/json/openapi/postman` + `export.formatLabel`; adjust `export.exportHtml` usage to a generic `export.confirm`)

**Interfaces:**
- Consumes: `ExportFormat`, `exportApis` (Task 12).
- Produces: `ExportDialog` accepts prop-free usage as today; emits nothing new (parent `Sidebar`/page unchanged).

- [ ] **Step 1: Write failing vitest component test** `frontend/src/lib/components/dialogs/ExportDialog.test.ts`: renders four format radios (i18n labels via test locale init pattern from existing component tests if present, else set `init({fallbackLocale:'en', locale:'en'})` before mount); default selection `html`; selecting `json` and submitting calls `fetch` with `"format":"json"` and triggers download with server filename. If component-test harness for dialogs proves impractical with `Dialog.svelte` portal behavior, test the extracted pure helper `formatExtension(format)` + the fetch payload, and note the UI click-path is covered by Task 15 E2E — do not skip the fetch-assertion part.
- [ ] **Step 2: Run** `bun run test -- ExportDialog` → FAIL.
- [ ] **Step 3: Implement** radio-card group above the selection tree; replace hardcoded `.html` download name with server-provided filename; neutral button label.
- [ ] **Step 4: Run** `bun run test && bun run check` → PASS.
- [ ] **Step 5: Commit** `feat(frontend): add export format picker to export dialog`

### Task 14: ImportDialog + sidebar entry

**Files:**
- Create: `frontend/src/lib/components/dialogs/ImportDialog.svelte`, `frontend/src/lib/components/dialogs/ImportDialog.test.ts`
- Modify: `frontend/src/lib/components/Sidebar.svelte` (Upload icon button beside the Export trigger), `en.json`, `zh-CN.json` (`import.*` keys: title, selectFile, detectedFormat, preview labels, confirm, success, error, unsupportedFormat)

**Interfaces:**
- Consumes: `importDocument` (Task 12), `Dialog`/`Button` UI components, `toast`, `$_`.
- Produces: `ImportDialog` with callback prop `onImported: () => void` (parent refreshes groups).

- [ ] **Step 1: Write failing vitest tests**: file input sets state; mock `importDocument` dryRun response renders counts + warnings; confirm calls non-dryRun `importDocument` then `onImported`; 415 error renders unsupported-format message.
- [ ] **Step 2: Run** `bun run test -- ImportDialog` → FAIL.
- [ ] **Step 3: Implement** dialog flow (pick → auto dryRun → preview panel → confirm → toast + `onImported()`); add Sidebar trigger button.
- [ ] **Step 4: Run** `bun run test && bun run check` → PASS.
- [ ] **Step 5: Commit** `feat(frontend): add import dialog with preview-before-confirm flow`

### Task 15: Playwright infrastructure + round-trip E2E

**Files:**
- Create: `frontend/playwright.config.ts`, `frontend/e2e/run-env.sh` (executable), `frontend/e2e/export-import-roundtrip.spec.ts`, `frontend/e2e/fixtures/.gitkeep`
- Modify: `frontend/package.json` (`@playwright/test` devDep; `"test:e2e": "playwright test"`)

**Interfaces:**
- Consumes: Tasks 10–11 (endpoint + `KNOT_PORT`/`KNOT_SQLITE_PATH`), Tasks 13–14 (UI).
- Produces: E2E harness reused by Task 16. `run-env.sh`: `bun run build` → verify the embedded-frontend build expectation in `backend/internal/handlers/static_embedded.go` (embed path — check the `embedfrontend` tag usage in `backend/Makefile` `package` target and mirror it) → `go build -tags embedfrontend -o e2e/bin/knot-e2e ./cmd/cli` → `KNOT_PORT=3999 KNOT_SQLITE_PATH=$E2E_DB_DIR/knot-e2e.db exec ./e2e/bin/knot-e2e __serve`. `playwright.config.ts`: `webServer.command='bash e2e/run-env.sh'`, `url='http://localhost:3999'`, `reuseExistingServer: !process.env.CI`, `use.baseURL`, downloads acceptance.

- [ ] **Step 1: Spec first** — `export-import-roundtrip.spec.ts`: create group "E2E Group" via UI → create API `POST /users` with note → add nested request params via the parameters editor → export dialog select group + format `json` → capture `page.waitForEvent('download')`, save to temp, `JSON.parse` assert `format/version/groups[0].apis[0]` → delete group via UI → import the file through ImportDialog → preview shows `groupsCreated: 1` → confirm → sidebar shows group again; open API view, assert note + param tree.
- [ ] **Step 2: Run** `bunx playwright install chromium && bun run test:e2e` → FAIL (no harness).
- [ ] **Step 3: Implement harness + make spec pass.** Global setup wipes the DB dir; `expect.poll` for async UI; no artificial sleeps.
- [ ] **Step 4: Run** `bun run test:e2e` → PASS (locally, twice in a row to prove idempotent setup).
- [ ] **Step 5: Commit** `test: add Playwright E2E harness and export/import round-trip spec`

### Task 16: Import + multi-format E2E specs

**Files:**
- Create: `frontend/e2e/fixtures/postman-collection.json` (folder + nested folder + rpc-marked item + raw JSON body), `frontend/e2e/fixtures/openapi-petstore-min.json` (paths get/post, `$ref` to components, tag), `frontend/e2e/import-postman.spec.ts`, `frontend/e2e/import-openapi.spec.ts`, `frontend/e2e/export-formats.spec.ts`

**Interfaces:**
- Consumes: Task 15 harness; ImportDialog/ExportDialog.

- [ ] **Step 1: Write the three specs** — `import-postman`: setInputFiles fixture → preview counts (2 groups: `Folder`, `Folder/Sub` or collection group per fixture) → confirm → assert sidebar entries + one API's params came from raw body. `import-openapi`: fixture → confirm → assert API names/params from `$ref` schema; rpc operation (include one `x-knot-type` op in fixture) shows as RPC type. `export-formats`: export `openapi` → downloaded JSON has `openapi:"3.1.0"` + expected path; export `postman` → `info.schema` URL; export `html` → content contains `api-section`.
- [ ] **Step 2: Run** `bun run test:e2e` → FAIL (fixtures/assertions).
- [ ] **Step 3: Fix fixtures/selectors until green.** Use stable roles/labels (add `data-testid` to the two dialogs where text is localized — localized labels make `getByText` locale-dependent).
- [ ] **Step 4: Run** `bun run test:e2e` full suite → PASS.
- [ ] **Step 5: Commit** `test: add Postman/OpenAPI import and multi-format export E2E specs`

### Task 17: Coverage target + CI workflow

**Files:**
- Modify: `backend/Makefile` (add `test-coverage` to `.PHONY` and target: `go test ./... -race -covermode=atomic -coverprofile=coverage.out` + `go tool cover -func`)
- Create: `.github/workflows/ci.yml`

**Interfaces:**
- Consumes: everything above.
- Produces: CI on `push` (main) + `pull_request`: job `backend` (Go 1.25 via `go-version-file: backend/go.mod`, `make test-coverage`, upload coverage artifact); job `frontend` (`oven-sh/setup-bun`, `bun install`, `bun run check`, `bun run test`, `bun run build`); job `e2e` (needs backend+frontend build: install chromium, build embedded binary, `bun run test:e2e`, upload `frontend/test-results/` on failure).

- [ ] **Step 1: Write ci.yml** following `release.yml` conventions (actions/checkout@v4, setup-go@v5, setup-bun@v1).
- [ ] **Step 2: Verify locally** — `cd backend && make test-coverage` passes and reports the new files ≥ 85%; `bun run check && bun run test && bun run build` in `frontend/` pass; `bun run test:e2e` passes. Fix any gaps found (this is the coverage gate moment).
- [ ] **Step 3: Validate workflow syntax** — `gh workflow list` after push, or `actionlint` if available; at minimum `python3 -c "import yaml,sys;yaml.safe_load(open('.github/workflows/ci.yml'))"`.
- [ ] **Step 4: Commit** `ci: add test and E2E workflow with coverage reporting`

### Task 18: README (bilingual)

**Files:**
- Modify: `README.md`

**Interfaces:** none.

- [ ] **Step 1:** Add a bilingual (中文 / English, matching the file's existing heading style) subsection under Features/Usage documenting: export formats (HTML/JSON/OpenAPI/Postman), import (Knot JSON/Postman/OpenAPI) with preview-then-confirm, and the merge-on-conflict behavior.
- [ ] **Step 2: Verify** markdown renders (visual pass of the diff).
- [ ] **Step 3: Commit** `docs: document multi-format export and import in README`

### Task 19: Release template (in-PR part)

**Files:**
- Modify: `.github/workflows/release.yml` (replace the bilingual `body:` with the English major-OSS template: `## Highlights`, `## What's Changed` (Features / Tests / CI & Release groups), `## Full Changelog` (`https://github.com/ProjAnvil/knot/compare/v0.5.0...${{ github.ref_name }}`), `## Downloads` asset table + checksums note). Asset list and build steps unchanged.

**Interfaces:** none (tag mechanics stay identical).

- [ ] **Step 1:** Edit `body:` per above; keep `draft: false`, `prerelease: false`.
- [ ] **Step 2: Verify** YAML parses (`python3 -c "import yaml;yaml.safe_load(open('.github/workflows/release.yml'))"`).
- [ ] **Step 3: Commit** `ci: use English release notes template in release workflow`

---

## Post-merge release runbook (not code tasks; executed after PR merge)

1. Confirm CI green on `main`.
2. `git tag v0.6.0 && git push origin v0.6.0` → `release.yml` builds 12 binaries + `checksums.txt` and creates the release with the Task 19 template.
3. Wait for workflow: `gh run watch` (tag run).
4. Enrich notes if needed: render `docs/superpowers/plans/release-notes-v0.6.0.md` (English, per template) → `gh release edit v0.6.0 --notes-file ...`.
5. Verify: `gh release view v0.6.0 --json assets,body` — 13 assets, English body.
