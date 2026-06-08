# Cross-Layer Thinking Guide

> 跨层功能先画数据流，再写代码。多数架构 bug 出现在边界，而不是单层内部。

---

## Current Layer Flow

后端核心流：

```text
HTTP -> routes -> usecase -> models -> db
```

前端参与时：

```text
Svelte -> frontend/src/api.js -> /api/* -> routes -> usecase -> models -> db
```

事件参与时：

```text
usecase transaction -> events.Publish -> domain-events queue -> subscriber -> models/db
```

其中，usecase 层负责业务逻辑的复用

---

## Map the Data

实现前先回答：

* 输入从哪里来？
* 每层的数据结构叫什么？
* 哪一层做校验？
* 哪一层做 DTO 映射？
* 哪一层决定错误码？
* 哪一层负责事务？
* 哪些字段是敏感字段？
* 失败时如何回滚或补偿？

示例格式：

```text
frontend form
  -> CreateOrderRequest
  -> usecase.CreateOrderCmd
  -> models.Order / models.OrderItem
  -> app DB transaction
  -> usecase.OrderCo
  -> routes.OrderResponse
  -> {success:true,data:{...}}
```

---

## Boundary Checklist

| Boundary | Check |
| --- | --- |
| frontend -> API | `frontend/src/api.js` 是否能 unwrap envelope？ |
| route -> usecase | 是否传 `fwcontext.InternalUsecaseContext(c)`？ |
| route -> response | 是否用 route-local DTO？ |
| usecase -> model | 是否只传 `ctx.Std()` 和业务参数？ |
| usecase -> db transaction | 是否用 `fwusecase.WithAppTx`？ |
| model -> db | 是否用 `ExecutorFor` / `DynamicExecutorFor`？ |
| usecase -> event | 是否只发布稳定 payload，不发布 model？ |
| API -> log | 是否只记录安全字段？ |

---

## Common Mistakes

### Hidden Format Assumption

Bad：前端假设 API 直接返回数组。
Good：前端通过 `request()` unwrap `{success:true,data}`。

### Leaked Storage Model

Bad：route 直接返回 `models.User`。
Good：usecase 返回 `UserCo`，route 映射 `UserResponse`。

### Wrong Transaction Layer

Bad：model 自己 `Begin/Commit`。
Good：usecase 使用 `fwusecase.WithAppTx`，model 使用 `db.ExecutorFor(ctx, "app")`。

### Split Error Logic

Bad：route 里解析 `err.Error()`。
Good：usecase 返回 `fwusecase.E(...)`，route 用 `httpresponse.InternalUsecaseError(...)`。

---

## When to Add Flow Notes

以下场景建议在 PRD 或 spec 中写清楚时序/流程：

* 变更跨三层以上。
* 涉及事务、事件、after-commit。
* 涉及前端 API envelope 或 DTO。
* 涉及 Open API 公开契约。
* 涉及 app/shared 两个数据库。
* 曾经因同类问题出过 bug。

---

## Checklist Before Implementation

* [ ] 画出完整数据流。
* [ ] 标明每层输入/输出类型。
* [ ] 标明 DTO 和 model 的边界。
* [ ] 标明错误码和 safe message 来源。
* [ ] 标明事务开始和结束位置。
* [ ] 标明日志字段和敏感字段。
* [ ] 明确哪些测试或 archguard 会验证边界。
