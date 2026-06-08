# Go Project Bitbucket Build Analysis

## Goal

Analyze whether the current Go + Svelte project can still be packaged after moving the repository to Bitbucket, and record the build risks and required migration steps.

## What I Already Know

* The project builds a Svelte/Vite frontend into `frontend/dist`, then embeds that directory into the Go binary with `//go:embed`.
* The Go module path is currently `github.com/tfnick/go-svelte-starter`.
* Internal Go imports use the current module path throughout the backend.
* The root `Dockerfile`, `Makefile`, and `build.bat` all build from local source and do not call GitHub-specific APIs.
* The current Git remote still points to GitHub: `git@github.com:tfnick/go-svelte-starter.git`.
* The current working tree has unrelated uncommitted marketing-site changes that currently break Go compilation.

## Requirements

* Verify the current project can build locally before any repository migration.
* Identify whether Bitbucket migration itself changes the packaging result.
* Separate the low-risk path of changing only the Git remote from the higher-risk path of changing the Go module path.
* Record concrete build commands and results.
* Record recommended next steps for a Bitbucket migration.

## Acceptance Criteria

* [x] Frontend production build result is captured.
* [x] Go test result is captured, including the current worktree failure.
* [x] Go production binary build result is captured, including the current worktree failure.
* [x] Migration risks are documented.
* [x] A clear yes/no conclusion is documented.

## Definition of Done

* Analysis is persisted under this task.
* No application source changes are required for the analysis itself.
* The final answer summarizes the outcome and points to the analysis artifact.

## Out of Scope

* Actually migrating the remote repository to Bitbucket.
* Renaming the Go module path to a Bitbucket import path.
* Creating Bitbucket Pipelines configuration.
* Changing application source code.

## Technical Notes

* See `research/bitbucket-build-analysis.md` for command output summary and migration conclusions.
* Relevant files inspected: `go.mod`, `static.go`, `Dockerfile`, `Makefile`, `build.bat`, `verify-build.bat`, `.gitignore`, `.dockerignore`, `README.md`, `frontend/package.json`, and `api/framework/archguard/layer_boundary_test.go`.
