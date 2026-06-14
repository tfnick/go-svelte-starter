# Architecture Diagnosis: Decoupling, Anti-Corruption, Extensibility

## Goal

从架构角度诊断当前 Go + Svelte 项目是否存在可优化空间，重点关注解耦、防腐层、可扩展性，以及现有分层约束是否足以支撑后续业务增长。任务产出应优先是可执行的架构评估报告和改进路线，而不是立即做大规模重构。

## What I already know

* 项目是 single-repo：Go/Echo 后端 + Svelte/Vite 前端。
* 后端当前规范化分层为 `routes -> usecase -> models -> db`。
* `api/framework` 承载业务无关基础能力，`api/providers` 承载外部 provider adapter，`api/usecase/integrations/*` 已存在部分 port 边界。
* 已有 `api/framework/archguard` 架构守卫测试，用于约束跨层 import、DTO 边界、domain event 边界等。
* 前端位于 `frontend/`，是 Svelte + Vite SPA，生产构建由 Go embed 到可执行文件。
* 项目已有 Trellis specs 可作为诊断基线，尤其是 backend directory structure、quality、eventing、API contract、frontend embed 规范。

## Requirements

* 梳理当前架构的实际模块边界，包括后端 route/usecase/model/db/framework/provider/integration，以及前端 app/api/helper/component/page 边界。
* 诊断深度采用“深入架构审计”：系统扫描后端、前端、集成边界和架构守卫，最终给出可执行的分阶段重构路线图。
* 从解耦角度检查是否存在跨层依赖、重复映射、业务逻辑泄漏、共享 helper 放置不当、模型/DTO 混用等风险。
* 从防腐角度检查外部系统 provider、Open API、payment、OAuth、LLM、embedding、OSS、KB 等边界是否隔离了第三方 DTO、错误、凭证、策略和 provider-specific 概念。
* 从可扩展性角度检查新增业务模块、新增 provider、新增前端页面、新增事件订阅、新增数据库表/迁移时的扩展成本。
* 评估现有 archguard 测试是否覆盖关键架构规则，并识别值得新增的架构守卫。
* 输出按优先级排序的建议：quick wins、medium refactors、long-term architectural decisions。
* 对每条建议说明收益、风险、影响范围、是否需要迁移、建议验证方式。

## Acceptance Criteria

* [ ] 在任务目录下产出一份架构诊断报告，覆盖解耦、防腐、可扩展性三个主维度。
* [ ] 报告引用实际代码路径和现有 spec，不只给抽象原则。
* [ ] 报告区分已被现有架构良好覆盖的部分和仍存在风险的部分。
* [ ] 报告给出优先级排序，至少包含 P0/P1/P2 或 quick/medium/long-term 分类。
* [ ] 报告给出分阶段重构路线图，并明确哪些建议适合拆成后续 Trellis 任务。
* [ ] 报告包含是否建议修改 `.trellis/spec/` 或新增 archguard 测试的判断。
* [ ] 若发现需要代码重构，拆分为后续独立任务建议，不在本诊断任务中直接大改。

## Definition of Done

* 诊断报告写入 `.trellis/tasks/06-14-arch-diagnosis-decoupling-anticorruption-extensibility/research/` 或任务根目录下的明确文档。
* 相关发现能追溯到代码文件、测试文件、spec 文件或 README。
* 如果只产出文档，不要求运行完整测试；如果新增 archguard/spec/test，则需要运行对应质量检查。
* 最终明确下一步：无需改动、更新 spec、创建重构任务，或创建架构守卫任务。

## Out of Scope

* 本任务默认不直接进行大规模重构。
* 本任务默认不改变 API contract、数据库 schema、provider 实现或前端页面行为。
* 本任务默认不引入新的架构框架、DI 框架或代码生成系统。

## Decision (ADR-lite)

**Context**: 用户希望从架构角度分析当前项目是否可以优化，重点是解耦、防腐、可扩展性。这个问题如果只做快速体检，容易只发现表面 import 或目录问题，无法形成后续可执行路线。

**Decision**: 采用深入架构审计，而不是快速架构体检。审计会覆盖后端、前端、外部集成、数据库迁移、事件机制、Open API、provider adapter、archguard 测试和 `.trellis/spec/` 约束。

**Consequences**: 产出会更偏向架构报告和路线图，耗时高于 quick scan；本任务仍不直接做大规模重构，发现的问题将拆分成后续独立任务。

## Research References

* [`research/architecture-diagnosis.md`](research/architecture-diagnosis.md) - 深入架构审计报告，覆盖现有优势、风险点、优先级路线图和建议拆分的后续 Trellis 任务。

## Implementation Decision

本任务只交付架构诊断报告和后续任务建议，不在同一个任务内直接修改生产代码、`.trellis/spec/` 或新增 archguard 测试。报告中识别出的低风险落地点应拆分为独立 Trellis 任务执行，包括 frontend API boundary guard、support chat spec、bootstrap registration split、frontend API modularization、support chat usecase split。

## Technical Notes

* Relevant specs:
  * `.trellis/spec/backend/index.md`
  * `.trellis/spec/backend/directory-structure.md`
  * `.trellis/spec/backend/api-contracts.md`
  * `.trellis/spec/backend/eventing-guidelines.md`
  * `.trellis/spec/backend/quality-guidelines.md`
  * `.trellis/spec/frontend/index.md`
  * `.trellis/spec/frontend/svelte-vite-embed.md`
  * `.trellis/spec/guides/cross-layer-thinking-guide.md`
  * `.trellis/spec/guides/code-reuse-thinking-guide.md`
* Initial code areas to inspect:
  * `api/framework/archguard`
  * `api/routes`
  * `api/usecase`
  * `api/usecase/integrations`
  * `api/providers`
  * `api/models`
  * `api/db`
  * `frontend/src/api.js`
  * `frontend/src/helpers`
  * `frontend/src/components`
  * `frontend/src/pages`

## Open Questions

* None.
