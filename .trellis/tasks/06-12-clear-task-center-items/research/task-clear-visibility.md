# Task Clear Visibility

## Question

用户点击 `Tasks` 面板“清空”时，应该复用任务执行状态，还是新增独立的可见性/归档字段？

## Current Repo Constraints

* `async_tasks.status` 当前只表达执行生命周期：`queued`、`processing`、`completed`、`failed`。
* worker 通过 `UpdateAsyncTaskStatus` 更新执行结果；下载逻辑依赖 `completed` 和 `orders_excel_export` 判断。
* `GET /api/user/tasks` 是当前用户任务中心列表；用户希望“清空”后不再显示当前已经进入终态的任务。
* 用户已确认清空只处理 `completed` / `failed`，不处理 `queued` / `processing`。

## Options

### Option A: Reuse `status = cleared`

* Pros: 字段少。
* Cons: 会污染执行生命周期，worker 处理 queued/processing 的任务时可能覆盖 cleared 状态；下载和失败处理需要额外分支；后续审计会混淆“任务是否成功”和“用户是否隐藏”。

### Option B: Add `cleared_at`

* Pros: 执行状态和展示可见性分离；列表默认过滤 `cleared_at = ''`；worker 仍可正常推进状态；任务历史和排查记录保留。
* Cons: 需要新增迁移、模型方法和过滤条件。

## Recommendation

Use Option B. `clear` is a user-facing visibility action, not a task lifecycle transition. A separate `cleared_at` field preserves the current status contract and makes future history/admin tooling easier. The update statement should constrain `status IN ('completed','failed')` so in-flight work remains visible.
