# Rename project from htmx-hyperscript-starter to go-svelte-starter

## Summary
Replace all occurrences of `htmx-hyperscript-starter` with `go-svelte-starter` across the entire codebase, including Go import paths, module declaration, docs, and task files.

## Scope
- `go.mod`: module path
- All `.go` files under `api/`: import paths
- `.trellis/spec/`: references in spec docs
- `.trellis/tasks/`: references in task prd/info files
- Any other files containing the old name

## Verification
- `go build ./...` succeeds
- No remaining references to `htmx-hyperscript-starter`
