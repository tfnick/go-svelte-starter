# Refresh and Deduplicate Trellis Specs

## Goal

全量整理 `.trellis/spec/`，移除主题相同或职责重叠的重复 spec 文件，并把保留下来的每一份 spec 更新为与当前代码实现一致的项目规范。目标是让后续 AI 或开发者在开始编码前读到的是“当前真实架构约定”，而不是历史任务累积下来的局部记录。

## What I Already Know

* 当前 spec 分为 `backend`、`frontend`、`guides` 三组。
* 后端现有 spec 包括目录结构、数据库、错误处理、质量、日志、事件、路由处理、API 契约、Open API 契约。
* 前端现有 spec 只有 Svelte/Vite/Go embed 一份主文档。
* 近期代码已完成若干架构标准化：日志文件存储与 API surface 标记、DTO 边界、EventBus 双场景、内部 API envelope。
* 这次任务的主体应是 spec 文档整理，不应修改业务代码来迎合文档。

## Requirements

* 盘点 `.trellis/spec/` 下所有 spec 文件，识别主题重复、职责重叠、内容过期、乱码或模板残留。
* 移除主题相同的重复 spec 文件，或把重复内容合并到更合适的保留文件中。
* 保留文件必须根据当前代码实现重写或更新，包含真实路径、真实 helper、真实约定、真实测试命令。
* 保留文件之间要有清晰职责边界，避免同一条规则分散在多个文件中重复维护。
* 更新 `index.md`，确保索引只指向存在且仍有独立价值的 spec 文件。
* 不要引入与代码现状不一致的理想化规范；如果发现代码缺口，只在 spec 中标记为当前约束或后续建议，不把它写成已实现事实。
* 保持 spec 面向“后续功能开发加速”的用途：每份文档应能指导编码决策，而不是只记录背景。
* 最终 `.trellis/spec/` 文档的标题、章节标题、文件名使用英文，便于检索、索引和工具识别；正文说明尽量使用中文描述，代码示例、路径、类型名、函数名、字段名等仍保持英文。

## Acceptance Criteria

* [ ] `.trellis/spec/` 下不存在主题相同、职责重复的 spec 文件。
* [ ] 每份保留 spec 都能对应到当前代码中的实际模块、helper、测试或命令。
* [ ] `backend/index.md`、`frontend/index.md`、`guides/index.md` 均与实际文件列表一致。
* [ ] 删除或修复明显乱码、模板占位语、过期描述。
* [ ] spec 文档满足“英文标题 + 中文正文说明 + 英文代码/标识符”的语言约定。
* [ ] 完成后运行至少一次 spec 文件列表检查和关键文档引用检查，确保没有索引指向已删除文件。
* [ ] 不提交 `data/app.db-shm`、`data/app.db-wal` 这类运行时文件。

## Definition of Done

* PRD 已确认并启动任务。
* spec 文件完成去重、更新和索引同步。
* 运行必要检查，例如 `rg --files .trellis/spec`、索引引用检查、以及必要的项目测试或轻量验证。
* 修改被提交为一个清晰的 docs/spec commit。

## Technical Approach

1. 建立 spec 清单：按目录、标题、主题、引用关系列出现有文档。
2. 对照代码现状：重点检查 `api/framework`、`api/routes`、`api/usecase`、`api/models`、`api/db`、`frontend/src`、`frontend/vite.config.js`。
3. 合并重复主题：
   * API envelope、DTO、Open API 边界优先收敛在 `api-contracts.md` / `open-api-guidelines.md`。
   * 路由处理流程保留在 `route-handler-guidelines.md`，只引用 API 契约，不重复完整 envelope 规则。
   * 事务和数据库访问优先收敛在 `database-guidelines.md`，质量文档只保留高层规则。
   * EventBus 只保留在 `eventing-guidelines.md`。
   * 日志只保留在 `logging-guidelines.md`。
4. 更新索引和交叉引用。
5. 做引用和状态检查。

## Decision (ADR-lite)

**Context**: 当前 spec 是多个架构优化任务逐步补充出来的，容易出现重复、局部过期和上下文不一致。

**Decision**: 本任务按“代码事实优先、spec 职责单一、索引可导航”的原则全量刷新 spec。删除或合并重复主题，而不是保留多个相似文档。

**Consequences**: 文档会更短、更聚焦，但实现时需要谨慎确认代码现状，避免把尚未实现的设计写成规范事实。

## Out of Scope

* 不重构业务代码。
* 不新增架构功能。
* 不归档或处理其他历史 `in_progress` 任务。
* 不修改 `.trellis/workflow.md` 或 Trellis 脚本行为。

## Technical Notes

* 当前 spec 文件通过 `rg --files .trellis/spec` 初步盘点。
* 当前代码重点目录包括 `api/framework`、`api/routes`、`api/usecase`、`api/models`、`api/db`、`frontend/src`。
* 工作区已有未提交运行时文件：`data/app.db-shm`、`data/app.db-wal`，本任务不得纳入提交。
* 实现决策：初步盘点后没有发现需要硬删除的“同主题独立文件”；重复主要发生在文档内容层面。因此保留现有文件集合，按单一职责重写内容，并把重复展开的规则收敛到主 spec。

## Open Questions

* None.
