# Variable configuration and management

## Goal

Add a `Variable` management menu so administrators can configure variables that are reused as global parameters or as key logic-control conditions. The feature should make variable values visible and editable through the authenticated app UI, while keeping backend validation and persistence explicit enough for later business logic to consume safely.

## What I already know

* User requested a new task for variable configuration and management.
* The new menu label should be `Variable`.
* Variables must support two business purposes:
  * global parameters
  * key logic control conditions
* Existing app navigation is driven by `frontend/src/router.js`, rendered in `frontend/src/App.svelte`, and displayed through `frontend/src/components/AppSidebar.svelte`.
* The current `Parameter` menu and page provide a nearby CRUD-style management pattern for admin configuration.
* Backend protected JSON APIs are registered in `index.go` under the authenticated `/api` group.
* App data uses embedded SQLite migrations under `api/db/migrations/app/`.

## Assumptions (temporary)

* Variables are application-level configuration records stored in the app database, not user-specific preferences.
* MVP variables should be managed through authenticated protected APIs, matching other admin-like pages.
* Variable values should be typed enough for future logic to consume without ad hoc string parsing.
* Sensitive secrets are out of scope for Variable; secret-like values should stay in credential-specific storage such as the existing integration credential flow.

## Open Questions

* None.

## Requirements

* Add a new authenticated `Variable` menu item and route.
* Provide a Variable page to list, create, edit, and enable/disable variables.
* Implement the MVP as a typed variable registry.
* Store each variable with `key`, `name`, `purpose`, `value_type`, `value_json`, `enabled`, `description`, `created_at`, and `updated_at`.
* Support `purpose` values for global parameters and logic-control conditions.
* Support `value_type` values `string`, `number`, `boolean`, and `json`.
* Persist variables in the app database through a dedicated migration.
* Expose protected backend API endpoints for variable management.
* Validate stable variable keys so business logic can reference them safely.
* Validate values according to their declared type.
* Keep API responses safe and structured for frontend consumption.

## Acceptance Criteria

* [ ] `Variable` appears in the authenticated sidebar and routes to a working management page.
* [ ] A user can create a variable with key, label/name, purpose, type, value, enabled state, and description.
* [ ] A user can edit a variable without changing unrelated records.
* [ ] A user can enable or disable a variable.
* [ ] Backend rejects duplicate variable keys.
* [ ] Backend rejects invalid keys or values that do not match the declared type.
* [ ] Frontend API tests cover the new variable endpoints.
* [ ] Backend model/usecase/route tests cover create, update, list, validation, and enable/disable behavior.

## Definition of Done

* Tests added/updated where appropriate.
* Lint, type-check, and relevant Go/frontend tests pass.
* Docs/spec notes updated if this task introduces a reusable configuration pattern.
* Existing uncommitted Parameter task changes are not reverted or mixed into this task accidentally.

## Out of Scope (explicit)

* No runtime wiring into specific business decisions until a consuming flow is named.
* No encrypted secret management for variables.
* No audit history/versioning for variable changes in the MVP.
* No import/export UI unless later requested.

## Technical Approach

Use a dedicated `variables` table in the app database and expose a small protected CRUD-style API. The backend owns normalization and validation for keys, purposes, types, JSON encoding, duplicate conflicts, and enable/disable behavior. The frontend follows the existing Svelte admin page pattern: add the route/menu entry, add API helpers, and create a compact management page with list/edit/new flows.

## Decision (ADR-lite)

**Context**: Variable can evolve toward global parameters, feature switches, or rule-engine inputs, but the current task only needs configuration and management.

**Decision**: Use Approach A, a typed variable registry with stable keys and typed JSON-backed values.

**Consequences**: The MVP stays small and flexible while preserving a stable contract for later business logic. Complex rule expressions, operators, targets, priority, and evaluation semantics remain out of scope until a concrete consuming workflow needs them.

## Technical Notes

* Likely backend files:
  * `api/db/migrations/app/012_add_variables.sql`
  * `api/models/variable.go`
  * `api/usecase/variable.go`
  * `api/routes/variable.go`
  * `index.go`
* Likely frontend files:
  * `frontend/src/router.js`
  * `frontend/src/App.svelte`
  * `frontend/src/api.js`
  * `frontend/src/pages/Variables.svelte`
* Existing nearby references inspected:
  * `frontend/src/pages/Parameters.svelte`
  * `api/routes/parameter.go`
  * `api/usecase/parameter.go`
  * `api/models/integration.go`
* Relevant specs to read before implementation:
  * `.trellis/spec/backend/index.md`
  * `.trellis/spec/backend/database-guidelines.md`
  * `.trellis/spec/backend/api-contracts.md`
  * `.trellis/spec/frontend/svelte-vite-embed.md`

## Expansion Sweep

### Future evolution

* Variables may later become references used by scheduled tasks, payment/LLM behavior, feature switches, or rules engines.
* A stable key and typed value contract is worth preserving now.

### Related scenarios

* `Parameter` manages integration channel settings; `Variable` should feel consistent but be domain-neutral.
* Existing router/API test patterns should be extended rather than replaced.

### Failure and edge cases

* Duplicate keys, invalid value/type combinations, disabled variables, and attempts to store secret-like data need explicit handling.
* Business logic consumers should eventually define safe defaults when a variable is missing or disabled.

## Feasible MVP Approaches

### Approach A: Typed variable registry (recommended)

Variables have `key`, `name`, `purpose`, `value_type`, `value_json`, `enabled`, `description`, and timestamps. Supported types are `string`, `number`, `boolean`, and `json`.

Selected by user.

Pros:
* Small and flexible.
* Good fit for global parameters and simple control conditions.
* Easy to validate, test, and extend.

Cons:
* Does not model complex rule expressions yet.

### Approach B: Rule-oriented variables

Variables include condition-specific metadata such as operator, target entity, and evaluation priority.

Pros:
* Closer to future rule-engine use cases.

Cons:
* Higher product/design commitment before a concrete consuming workflow exists.
* More risk of building unused condition machinery.
