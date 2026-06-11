# Rename API Integrations Directory To Providers

## Goal

将 `api` 下当前用于第三方实现适配器的 `integrations` 目录重命名为 `providers`，让目录命名更贴近“外部 provider 实现层”，并保持后端测试、构建和架构边界一致。

## What I Already Know

* 当前明确存在顶层实现目录：
  * `api/integrations/llm/deepseek`
  * `api/integrations/oauth/github`
  * `api/integrations/oauth/google`
  * `api/integrations/oss/local`
  * `api/integrations/oss/s3compatible`
  * `api/integrations/payment/creem`
* 还存在两个同名但语义不同的目录：
  * `api/usecase/integrations/*`：usecase 层 ports，属于业务边界接口。
  * `api/framework/integrations/*`：framework 层 provider error / credentials 等通用能力。
* 本任务至少需要更新 Go import path、package references、架构测试、文档/spec 中的相关路径描述。

## Assumptions

* 推荐本任务只重命名顶层 `api/integrations` 为 `api/providers`。
* 暂不重命名 `api/usecase/integrations` 和 `api/framework/integrations`，因为它们分别表达 usecase ports 和 framework 通用能力，直接重命名会扩大边界语义变化。

## Open Questions

* None.

## Requirements

* 将顶层 `api/integrations` 目录重命名为 `api/providers`。
* 更新所有引用旧顶层 import path 的 Go 文件。
* 保持 provider 内部 package 名（如 `creem`、`deepseek`、`local`）不变，避免无意义包名 churn。
* 更新架构/规格文档中描述顶层 provider 实现目录的内容。
* 保证后端测试通过。

## Confirmed Decisions

* 仅重命名顶层 `api/integrations`。
* 保留 `api/usecase/integrations` 和 `api/framework/integrations`。
* 保留现有 HTTP webhook/callback 路径中的 `/api/integrations/...`，因为它们是对外 API surface，不是 Go provider 实现目录。

## Acceptance Criteria

* [x] `api/providers/...` 存在，原顶层 `api/integrations/...` 不再存在。
* [x] Go import path 不再引用 `github.com/tfnick/go-svelte-starter/api/integrations/...`。
* [x] `go test ./...` 通过。
* [x] 架构测试通过且目录边界仍清晰。
* [x] `.trellis/spec` 中相关目录规范已同步。

## Definition of Done

* 代码实现完成。
* 后端测试通过。
* 规格文档同步。
* 变更提交完成，任务归档前工作树干净。

## Out of Scope

* 不改变数据库 schema 或数据迁移。
* 不改变 provider 业务行为。
* 不重命名 provider 内部 package 名。
* 默认不重命名 `api/usecase/integrations` 和 `api/framework/integrations`，除非用户确认扩大范围。

## Technical Notes

* Inspected paths:
  * `api/integrations/*`
  * `api/usecase/integrations/*`
  * `api/framework/integrations/*`
* This is primarily a directory/import-path refactor, but it can touch many files and architecture specs.
* Residual `/api/integrations/...` matches are intentional HTTP route/callback URL references, not Go import paths or implementation directories.

## Verification

* `go test ./api/framework/archguard` passed.
* `go test ./...` passed.
* `git diff --check` passed.
* Path scan confirmed the top-level `api/integrations` directory no longer exists and provider implementation files now live under `api/providers`.
