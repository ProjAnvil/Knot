# Sentia Feedback Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix six Sentia-reported integration issues (groupId move, group order in JSON, 409 on duplicate names, per-app Mount, flat GET /apis with search, export format docs) with additive-only wire changes. Ships as models v1.2.0 → backend v0.2.0 → product v0.8.0 (strict tag order; backend requires models v1.2.0).

**Architecture:** Five handler/facade-level fixes plus one docs fix; no schema change, no models release; product `v0.8.0` + engine `backend/v0.2.0`.

**Spec:** `docs/superpowers/specs/2026-11-10-sentia-feedback-fixes-design.md` (§3 semantics are binding, including exact status codes/messages)

**Tech Stack:** Go 1.25.5 workspace, Fiber v2, GORM, vitest/Playwright unchanged.

## Global Constraints

- ALL comments/commits in English; conventional commits.
- Wire changes additive-only; existing backend tests, frontend suite (152), E2E (4 specs, run twice), both coverage gates, and the CI embedded-host smoke stay green.
- Exact messages from spec §3 (e.g. 409 `"Group name already exists"`, 404 `"Group not found"`, 400 `"Invalid group ID"`, 400 `"Invalid pagination parameters"`).
- No DB schema change; no models release; no new dependencies.
- Branch `fix/sentia-integration-feedback`.

## Review Focus

1. **Fiber route precedence**: `apis.Get("/", ...)` must not shadow/be shadowed by `apis.Get("/:id", ...)` — test both resolve correctly (list at `/apis`, detail at `/apis/5`). Pinned in Task 4.
2. **409 detection across dialects**: pre-check plus unique-violation backstop (`UNIQUE|Duplicate|23505`) — injected-error test proves non-unique errors still 500. Pinned in Task 1.
3. **Same-app Mount idempotence preserved** while second-app mounting works — route-count assertion on both apps. Pinned in Task 3.
4. **`order` field placement** in Group JSON (Name→Order→APIs) — knot-json EXPORT is untouched (KnotExportGroup already has order); only the domain Group response changes. Pinned in Task 2.

---

### Task 1: UpdateAPI groupId + duplicate-name 409

**Files:**
- Modify: `backend/internal/handlers/apis.go` (UpdateAPI body + validation), `backend/internal/handlers/groups.go` (CreateGroup + UpdateGroup 409 paths)
- Test: `backend/internal/handlers/apis_test.go`, `backend/internal/handlers/groups_test.go`

**Interfaces (spec §3.1, §3.3):**
- UpdateAPI body: `GroupID *uint json:"groupId"`; nil → unchanged; non-nil: 0 → 400 `"Invalid group ID"`; missing group → 404 `"Group not found"`; valid → move.
- CreateGroup/UpdateGroup: pre-check name (Update: only when changed AND owned by another row) → 409 `"Group name already exists"`; Create/Save error matching `UNIQUE|Duplicate|23505` → 409; else 500 as today.

- [ ] **Step 1: Failing tests** — move happy/404/400/absence; create-dup 409; rename-to-existing 409; rename-to-self OK; injected non-unique error → 500.
- [ ] **Step 2: RED** verified.
- [ ] **Step 3: Implement; GREEN; full handler suites green; `make test-coverage` green.**
- [ ] **Step 4: Commit** `fix: allow moving APIs between groups and return 409 on duplicate group names`

### Task 2: Group JSON emits order

**Files:**
- Modify: `models/group.go` (MarshalJSON struct: `Order int json:"order"` after Name), `models/group_test.go`
- Test: marshal Order 7 contains `"order":7`; field order Name→Order→APIs asserted by key positions.
- MUST also update the now-false comments: `models/group_test.go` (~lines 60, 89-90 "does not include Order field") and `backend/internal/handlers/groups_test.go` (~line 225).
- MUST bump `backend/go.mod` require to `models v1.2.0` (hand-edit; in-repo replace keeps everything green).

- [ ] **Step 1: Failing test; RED.**
- [ ] **Step 2: Implement; models suite + `make test-coverage` green; backend suite green (handlers' group assertions unaffected or updated additively).**
- [ ] **Step 3: Commit** `fix: include order in Group JSON serialization`

### Task 3: Per-app Mount

**Files:**
- Modify: `backend/knot.go` (replace `mounted bool` with `sync.Map` of registered apps; `LoadOrStore` guard — store the key FIRST, then register routes, so exactly one goroutine registers per app under `-race`; rewrite Mount's doc comment to per-app semantics)
- Test: `backend/knot_test.go` — second app gets working routes (both apps serve GET …/groups 200); same app twice → no duplicate routes (app.Stack() count equal); concurrent Mount race (t.Parallel or goroutines) safe.

- [ ] **Step 1: Failing tests (second-app case fails today); RED.**
- [ ] **Step 2: Implement; GREEN; backend suite + coverage gate green.**
- [ ] **Step 3: Commit** `fix: make engine Mount per-app instead of per-instance`

### Task 4: GET /apis flat list + search

**Files:**
- Create: `backend/internal/handlers/apis_list.go` (ListAPIs handler), `backend/internal/handlers/apis_list_test.go`
- Modify: `backend/internal/handlers/routes.go` (`apis.Get("/", ListAPIs(db))` FIRST in the apis group)

**Interfaces (spec §3.5):** params page/pageSize/keyword/groupId/method; defaults 1/50, both clamp (page ≥1, pageSize 1..100); `LOWER(name) LIKE`/`LOWER(endpoint) LIKE` with pattern lowercased in Go, wildcards NOT escaped (comment why, mirroring mcp_tools); exact groupId (non-numeric → 400 `"Invalid group ID"`)/method; ORDER BY via JOIN on `(models.Group{}).TableName()` with `clause.OrderByColumn{Column: clause.Column{Table: <groupsTable>, Name: "order"}}` (reserved-word + TablePrefix safe) then api order then id; items WITHOUT Parameters preload; data `{items, total, page, pageSize}`; invalid page/pageSize → 400 `"Invalid pagination parameters"`.
- Tests MUST include a prefixed run: `models.SetTablePrefix("knot_")` subtest (defer restore) proving the JOIN/ORDER BY survives prefixes.

- [ ] **Step 1: Failing tests** — Review Focus 1 (list `/apis` + detail `/apis/5` both work), pagination math, keyword name+endpoint, groupId+method filters, combined, empty shape, invalid page 400, defaults.
- [ ] **Step 2: RED; implement; GREEN; suites + gate green.**
- [ ] **Step 3: Commit** `feat: add flat API list endpoint with search and pagination`

### Task 5: Docs + full verification

**Files:**
- Modify: `backend/README.md` (Export format-values clarification §3.6; route table + `GET /apis`; Mount per-app wording), `models/README.md` (Versioning note: v1.2.0 Group JSON emits `order`; WireFormat value clarification)

- [ ] **Step 1: Docs edit; verify claims.**
- [ ] **Step 2: Full gates**: backend `make test-coverage`; models `make test-coverage`; frontend `bunx vitest run && bun run check && bun run build`; E2E `bun run test:e2e` 4/4 TWICE; `go vet ./...` backend/models/tools; drift check clean.
- [ ] **Step 3: Commit** `docs: clarify export format values and document API list endpoint` (fixes from gates, if any, in separate fix commits).

---

## Post-merge runbook

1. CI green on main → `git tag models/v1.2.0` → push → external smoke (marshal Group{Order:7} → JSON contains "order":7) → `gh release create models/v1.2.0 --latest=false` (English notes).
2. `git tag backend/v0.2.0` (same merge commit; go.mod already requires models v1.2.0) → push → external smoke (two-app Mount + GET /apis list + group JSON order) → `gh release create backend/v0.2.0 --latest=false` (English notes: Mount per-app fix, groupId move, 409s, GET /apis).
3. `git tag v0.8.0` → release.yml → `gh release edit v0.8.0 --notes-file` (English minor notes; Full Changelog base `v0.7.1`; Highlights: Sentia-driven integration fixes; note Group external-association deferred).
