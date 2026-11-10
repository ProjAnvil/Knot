# Sentia Integration Feedback Fixes — Design

- Date: 2026-11-10
- Status: Final (small-change scope; fixes verified against code this session)
- Target releases: `backend/v0.2.0` (engine minor) and `v0.8.0` (product minor). models module: NO release (no schema change).

## 1. Intent

Sentia (first embedded host) reported seven integration issues. Six are fixed
in this batch; the Group external-association question (#2, external_key vs
parentID) is explicitly DEFERRED by the product owner and out of scope.

All seven verified against code before this spec; the six in scope:

1. **PATCH /apis/:id ignores groupId** — `UpdateAPI` body has no GroupID
   field; an API can never move groups. (handlers/apis.go)
2. **Group.MarshalJSON drops `order`** — custom marshal struct omits it even
   though the field exists and `POST /groups/orders` writes it. Bug.
3. **Duplicate group name returns 500** — `CreateGroup`/`UpdateGroup` map
   unique violations to InternalError; should be 409. Bug.
4. **Mount idempotence is instance-level** — `if k.mounted { return }` makes
   mounting the same engine on a second Fiber app a silent no-op; host test
   setups lose routes. Design defect of the engine facade.
5. **No flat API list/search** — only `GET /apis/:id` and
   `GET /apis/group/:groupId`; hosts must fetch the full nested
   `with-apis` tree or query the DB directly.
6. **Export `format` value ambiguity** — request parameter is
   `"json"` but the produced document envelope says `"format":"knot"`. API is
   self-consistent; docs must state the distinction. Doc fix.

## 2. Goals / Non-goals

Goals: fix 1/2/3/4/6 above, add `GET /apis` (list + search) per 5, keep
every existing test green (additive wire changes only: new optional request
field, new response field, new route).

Non-goals: Group external_key/parentID (#2 — deferred by product owner);
pagination for other endpoints; auth; any DB schema change.

## 3. Fixes

### 3.1 UpdateAPI accepts groupId (1)

Body gains `GroupID *uint json:"groupId"`. When non-nil: load target group
(`First` by ID) — not found → 404 `"Group not found"`; found → set
`api.GroupID`. Zero value (`"groupId": 0`) treated as invalid input → 400
`"Invalid group ID"`. Tests: move happy path; 404 on missing group; 400 on
zero; absence of field leaves group unchanged.

### 3.2 Group JSON emits `order` (2)

`Group.MarshalJSON` anonymous struct gains `Order int json:"order"` between
Name and APIs. Additive to the wire (Knot's own frontend ignores extra
fields). Test: marshal a Group with Order 7 → contains `"order":7`.

### 3.3 Duplicate name → 409 (3)

CreateGroup and UpdateGroup: pre-check `Where("name = ?", name).First` →
exists → 409 `"Group name already exists"` (for Update: only when the name
actually changed and belongs to another row). Keep the unique index as the
concurrent-race backstop: on Create/Save error whose message matches
`UNIQUE|Duplicate|23505` (SQLite/MySQL/PG) → 409, else 500 as today. Tests:
create duplicate 409; rename to existing name 409; rename to own name OK;
unrelated DB error still 500 (injected via callback).

### 3.4 Mount is per-app (4)

Replace `mounted bool` with `mounted sync.Map` keyed by `*fiber.App`:
`Mount(app)` — if app already registered → no-op (same-app idempotence kept);
else register routes and mark. Add `sync.Mutex`-free implementation via
`sync.Map.LoadOrStore` guard. Tests: two different apps both get working
routes; same app twice still no duplicate registration (route-count check).

### 3.5 GET /apis — flat list with search (5)

`GET {prefix}/apis` handler `ListAPIs(db)`:
- Query params: `page` (default 1, ≥1), `pageSize` (default 50, clamp 1..100),
  `keyword` (optional; case-insensitive LIKE on name OR endpoint),
  `groupId` (optional exact), `method` (optional exact, upper-cased input).
- Order: group order → api order → id. Items are `models.API` WITHOUT
  preloading Parameters (list stays light; detail endpoint already exists).
- Response data: `{items: []API, total: int, page: int, pageSize: int}`.
- Invalid page/pageSize (non-numeric) → 400 `"Invalid pagination parameters"`.
- Route registered as `apis.Get("/", ListAPIs(db))` — no conflict with
  `/:id` (fiber matches literal `/` before param).
Tests: pagination math (total/page/pageSize), keyword matches name and
endpoint, groupId + method filters, combined filters, empty result set shape,
invalid page 400, default params. E2E: not required (host-facing; covered by
handler tests + backend suite).

### 3.6 Export docs clarify format values (6)

backend/README.md Export section: one sentence stating the request `format`
values (`""|html|json|openapi|postman`) vs the produced Knot JSON document's
envelope field `"format":"knot"` (v1 wire format). Same clarification in
models/README.md if it references the value.

## 4. Wire-compatibility statement

All wire changes are additive: new optional request field (3.1), new response
field (3.2), new route (3.5), changed status code only on error paths that
previously mis-served 500 (3.3). Knot's frontend, existing E2E, and CI smoke
are unaffected. Product tag `v0.8.0` (minor — new endpoint); engine library
tag `backend/v0.2.0` (Mount semantic fix + additive API).

## 5. Testing strategy

Per-fix unit/handler tests as listed in §3; full backend suite + coverage
gate (85% per-function, incl. new ListAPIs and changed handlers); models
suite green (3.2 is in models/group.go — covered there); frontend suite +
E2E 4/4 green twice (regression proof for 3.2's additive field through the
real product); CI embedded-host smoke untouched and green.

## 6. Execution sequence

spec (this doc) → plan → subagent audit → subagent-driven execution → final
review → verification → PR → CI → merge → `backend/v0.2.0` → `v0.8.0` via
gh (English notes; Full Changelog base `v0.7.1`).
