# Build Output Bin Directory

## Goal

调整 Windows 打包脚本，让 `build.bat` 生成的可执行文件输出到 `bin/`，而不是当前的 `tmp/`，并把 `bin/` 加入 `.gitignore`，使构建产物和临时文件目录语义更清晰。

## What I Already Know

* `build.bat` 当前设置 `OUT_DIR=tmp`，最终产物为 `tmp\svelte-go-starter.exe`。
* `.gitignore` 当前忽略了 `tmp/`，但还没有忽略 `bin/`。
* `verify-build.bat` 当前从 `tmp\svelte-go-starter.exe` 读取构建产物，同时使用 `tmp\verify-empty` 作为验证运行目录。
* `README.md` 当前说明最终可执行文件写入 `tmp/`，并提到 executable parent lookup 对 `tmp/` 有帮助。
* `Makefile` 当前默认 `OUT_DIR ?= tmp`，但本次用户明确点名的是 `build.bat`。

## Assumptions

* 本任务的核心范围是 Windows 打包流程：`build.bat` 输出目录改为 `bin/`。
* 为保证 `build.bat` 后的验证流程可用，应同步更新 `verify-build.bat` 的 `SOURCE_EXE` 为 `bin\svelte-go-starter.exe`。
* `verify-build.bat` 的验证运行临时目录可以继续使用 `tmp\verify-empty`，因为它表达的是临时运行目录，不是构建产物目录。
* README 中与 Windows 打包输出目录相关的说明应同步更新为 `bin/`。
* `tmp/` 仍保留在 `.gitignore`，因为项目仍可能用它做临时目录。

## Open Questions

* None.

## Confirmed Decisions

* 本任务只调整 Windows 打包链路，不修改 `Makefile` 的默认 `OUT_DIR ?= tmp`。
* `verify-build.bat` 继续使用 `tmp\verify-empty` 作为临时验证运行目录。
* `bin/` 作为本地构建产物目录，需要同时加入 `.gitignore` 和 `.dockerignore`。

## Requirements

* 将 `build.bat` 的输出目录从 `tmp` 改为 `bin`。
* 将 `bin/` 加入 `.gitignore`。
* 保留 `tmp/` 的 ignore 规则。
* 同步更新依赖 `build.bat` 产物路径的验证脚本和文档说明。
* 不改变 Go 编译参数、前端构建流程或应用运行行为。

## Acceptance Criteria

* [x] `build.bat` 生成 `bin\svelte-go-starter.exe`。
* [x] `.gitignore` 包含 `bin/`，且仍包含 `tmp/`。
* [x] `verify-build.bat` 从 `bin\svelte-go-starter.exe` 验证构建产物。
* [x] README 中关于 Windows build 输出目录的说明改为 `bin/`。
* [x] `git diff --check` 通过。

## Definition of Done

* 脚本和文档更新完成。
* 可执行的轻量质量检查通过。
* 变更提交前与当前未完成任务保持清晰边界。

## Out of Scope

* 不改变 Dockerfile 构建输出。
* 不改变生产运行配置或数据库路径。
* 不清理已有 `tmp/` 内容。
* 默认不修改 `Makefile`，除非后续确认需要统一所有构建入口。

## Technical Notes

* Inspected files:
  * `build.bat`
  * `verify-build.bat`
  * `.gitignore`
  * `README.md`
  * `Makefile`
* Previous in-progress task has been committed and archived, so this task is now implemented independently.

## Verification

* `git diff --check` passed.
* `go test ./...` passed.
* `cd frontend && npm test` passed.
* `build.bat` passed and produced `bin\svelte-go-starter.exe`.
* `verify-build.bat` passed using `bin\svelte-go-starter.exe` and `tmp\verify-empty`.
* `docker build -t go-svelte-starter .` was attempted but Docker Desktop/daemon was not running: `open //./pipe/dockerDesktopLinuxEngine: The system cannot find the file specified`.
* Static path scan confirmed:
  * `build.bat` uses `OUT_DIR=bin`.
  * `verify-build.bat` uses `SOURCE_EXE=bin\%APP_NAME%`.
  * `.gitignore` keeps `tmp/` and adds `bin/`.
