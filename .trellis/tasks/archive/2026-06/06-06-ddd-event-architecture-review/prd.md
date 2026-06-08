# DDD Event Architecture Review

## Goal

从架构师视角审视当前 lite DDD event 方案，判断它是否已经形成清晰、可维护、低负担的事件基础设施，并产出可执行的优化建议。评审重点不是立即重写，而是把 framework 职责、业务职责、最终一致性/幂等保障、以及非 DDD 通用消息能力的边界讲清楚。

## What I Already Know

* 用户希望创建一个新任务，对现有 DDD event 总体设计做架构审视。
* 需要重点回答四类问题：
  * `framework` 应该做什么，业务侧职责是否清晰区分。
  * 业务开发人员现在要做什么，是否还能进一步降低接入负担。
  * 当前是否支持最终一致性，并为业务侧幂等消费提供技术保障。
  * 当前是否仍支持原来非 DDD 语义的基本消息发送。
* 当前 `api/framework/events` 是 queue-first durable-only DDD event facade。
* 当前 DDD event 发布流程是 `events.Publish(txCtx, event) -> domain_events -> domain_event_deliveries -> goqite queue: domain-events -> events.HandleMessage -> subscriber handler`。
* 当前 fan-out 是每个 `(topic, subscriber)` 独立 delivery row 和 queue message。
* 当前 `api/framework/queue` 仍提供通用队列能力，包括 `Send` / `SendJSON` / `NewRunner` / `NewJSONRunner` / `CreateJob`。
* 当前业务样例 `order.paid -> points.award_on_order_paid` 需要业务侧提供 payload struct、typed constructor、handler、registration function、subscriber transaction、业务幂等逻辑。
* 当前积分幂等由业务表 `point_transactions` 的 `UNIQUE(order_id, type)` 和 `INSERT OR IGNORE` 承担。
* 当前 delivery 表有 `UNIQUE(event_id, subscriber)`、`attempts`、`status`、`last_error`，但 handler 幂等仍需要业务侧设计。
* 当前 `events.HandleMessage` 按 `subscriber` 查找 handler；message 中带 `topic`，但 handler lookup 没有使用 `topic` 作为复合键。
* 当前 domain event runner 启动并发 limit 是 `1`，因此支持 fan-out 独立重试，但默认不保证多个 subscriber 物理并行执行。
* 当前仅有本地 DB 运行文件脏变更，不属于本任务。

## Requirements

* 评审当前 DDD event 方案的职责边界：
  * `api/framework/events` 的职责。
  * `api/framework/queue` 的职责。
  * `api/usecase/events` 与业务 usecase/model 的职责。
  * routes/models 不应承担的事件职责。
* 评审业务开发人员新增一个 DDD event 或 subscriber 时的实际步骤，并识别可降低负担的封装点。
* 评审最终一致性能力：
  * publisher transaction 与 event/delivery/queue message 的 commit/rollback 关系。
  * subscriber 成功/失败与 queue retry 的关系。
  * 多 subscriber 的隔离性。
* 评审幂等保障：
  * framework 已经保障什么。
  * 业务侧仍必须保障什么。
  * 是否需要新增 framework helper 来降低幂等实现成本。
* 评审非 DDD 通用消息能力：
  * 当前 `api/framework/queue` 是否足够支持普通消息/任务。
  * DDD event facade 与 generic queue API 是否需要更清晰命名或边界约束。
* 产出一份架构评审报告，列出：
  * 当前设计结论。
  * 已经满足的点。
  * 风险/缺口。
  * 推荐优化项，按优先级分组。
  * 若需要实现，建议拆分的小 PR 计划。

## Research References

* [`research/current-ddd-event-architecture-review.md`](research/current-ddd-event-architecture-review.md) - 基于当前代码和 spec 的 DDD event 架构评审，覆盖职责边界、业务接入负担、最终一致性/幂等、generic queue 支持。

## Feasible Directions

### Direction A: Review-only Report (Recommended first)

先把本任务定位为架构评审任务，只提交 PRD 和研究报告。优点是快速沉淀判断，不在未确认优先级时扩大代码改动；缺点是优化点不会立刻落地。

### Direction B: Review + Contract Hardening

在报告基础上立即实现低风险硬化项，例如 subscriber 全局唯一检查或 `(topic, subscriber)` 复合 key、`HandleMessage` topic 校验、相关测试。优点是修掉最明确的契约缝隙；缺点是本任务从评审扩大为代码变更。

### Direction C: Review + Developer Ergonomics (Selected)

在报告基础上新增 typed payload helper、typed transactional handler adapter，并改造 `order.paid` 示例。优点是直接降低业务开发负担；缺点是 API 设计面更大，需要更仔细确认命名和长期演进。

## Decision (ADR-lite)

**Context**: 架构评审已经确认当前 DDD event 方案可用，但业务开发者新增 event/subscriber 时仍需要手写 JSON marshal/unmarshal、subscriber transaction adapter 等样板代码。

**Decision**: 本任务选择 Direction C：先保留架构评审报告，再实现一轮 developer ergonomics 优化。MVP 包含 typed payload helper、typed transactional handler adapter，并改造 `order.paid` 示例作为可复制模板。

**Consequences**: 这会扩大本任务到代码变更，但优化点集中在 framework helper 和现有示例，不改变 durable queue / delivery 核心语义。

## Acceptance Criteria

* [ ] 任务目录包含架构评审报告，覆盖用户列出的 4 个问题。
* [ ] 报告引用当前关键代码位置，而不是只做抽象评价。
* [ ] 报告明确区分 framework 职责和业务职责。
* [ ] 报告给出业务开发接入步骤，并提出至少 2 个降低业务负担的可选优化。
* [ ] 报告说明当前最终一致性链路是否成立，以及哪里仍依赖业务幂等。
* [ ] 报告说明当前是否支持非 DDD 基本消息发送，并给出是否需要保留/调整 API 的建议。
* [ ] 报告给出推荐方向和优先级。
* [ ] `api/framework/events` 提供 typed payload event helper，业务侧无需手写普通 payload marshal。
* [ ] `api/framework/events` 提供 typed transactional handler adapter，业务侧无需在每个 subscriber 中重复 decode + `WithAppTx` 样板。
* [ ] `order.paid` 积分 subscriber 改造为新 helper 的示例，并保持现有最终一致性/幂等语义。
* [ ] 新 helper 有框架层单测或用例层测试覆盖。

## Definition of Done

* 架构评审报告已写入任务目录。
* Developer ergonomics helper 已实现并被 `order.paid` 示例使用。
* 如产生新的团队约定，更新 `.trellis/spec/`。
* 若有文件变更，按 Trellis Phase 3.4 提交。

## Out of Scope

* 不在第一步直接实现完整重构。
* 不替换 goqite。
* 不引入外部 MQ。
* 不恢复旧 EventBus。
* 不把所有业务事件都改造成完整 DSL；本轮只做轻量 typed helper 和现有示例改造。

## Technical Notes

* Key files inspected:
  * `api/framework/events/events.go`
  * `api/framework/queue/queue.go`
  * `api/usecase/events/order_paid_points.go`
  * `api/usecase/events/durable_store.go`
  * `api/db/migrations/app/009_add_domain_event_delivery.sql`
  * `.trellis/spec/backend/eventing-guidelines.md`
* Current generic queue capability appears to live in `api/framework/queue`, while DDD semantics live in `api/framework/events`.
* Potential review topics already visible from code:
  * Handler registry uses `subscriber` as handler key instead of `(topic, subscriber)`.
  * Business developer must manually marshal/unmarshal payload JSON.
  * Business developer must manually wrap subscriber DB writes in `WithAppTx`.
  * Business idempotency is documented but not scaffolded by framework.
  * Generic queue capability exists but naming/usage boundaries may need clearer docs.

## Implementation Plan

1. 在 `api/framework/events` 增加 typed payload helper 和 typed transactional handler adapter。
2. 改造 `api/usecase/events/order_paid_points.go` 使用 helper，减少 JSON 和 transaction 样板。
3. 增加/更新测试，确保 fan-out、retry、transaction rollback、业务幂等语义不退化。
4. 更新 `.trellis/spec/backend/eventing-guidelines.md`，把新推荐模板写入规范。
