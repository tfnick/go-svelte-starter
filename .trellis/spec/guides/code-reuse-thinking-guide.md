# Code Reuse Thinking Guide

> 写新代码前先停一下：现有代码里是否已经有同类模式、helper 或约定？

---

## The Problem

重复代码最常见的后果不是“多几行”，而是不一致：

* bug 修复只改了一处。
* 同一个 API envelope 被手写成多个变体。
* DTO mapper、dictionary lookup、error mapping 分散在不同层。
* 常量、topic、route path、JSON key 出现拼写漂移。

---

## Search First

优先用 `rg`：

```sh
rg "function_or_type_name"
rg "json_field_name"
rg "error_code"
rg "event.topic"
```

如果要新增 helper，先搜索：

```sh
rg "helper|mapper|response|executor|lookup|context" api
```

---

## Questions Before Adding Code

| Question | Preferred action |
| --- | --- |
| 是否已有类似函数？ | 复用或扩展它 |
| 是否已有 framework capability？ | 放到现有 capability 下 |
| 是否只是某个 route 的 DTO？ | 保持在 route 文件，不抽成共享类型 |
| 是否是跨业务通用基础设施？ | 放到 `api/framework/<capability>` |
| 是否是 usecase 应用流程？ | 放到 `api/usecase` |
| 是否是 SQL/data helper？ | 放到 `api/models` 或 `api/framework/data/*` |

---

## When to Abstract

适合抽象：

* 同一逻辑已经出现三次以上。
* 逻辑复杂，复制会导致行为分叉。
* 多个模块需要同一 framework-level 能力。
* 已有 archguard 或 spec 可以清楚定义边界。
* usecase 层是稳定的业务抽象。

不适合抽象：

* 只是两个字段一样的 route DTO。
* 抽象比重复更难读。
* 不同 API surface 有不同稳定契约，例如内部 `/api/*` 与 `/open-api/v1/*`。
* 业务语义不同，只是代码长得像。

---

## Project Examples

* 内部 API envelope 已抽到 `api/framework/http/response`，route 不再手写 `success/data/error` map。
* Open API error envelope 与内部 API 分开，避免公开契约被内部页面需求牵连。
* ID/name 翻译抽到 `api/framework/data/namelookup`，业务注册在 `api/usecase/translate`。
* Domain event 只暴露 `api/framework/events` durable facade，业务不直接使用 raw queue 或 raw EventBus。
* 事务 wrapper 是 `fwusecase.WithAppTx`，usecase 不重复手写 `ctx.WithStd(...)`。

---

## Gotcha: Asymmetric Output Mechanisms

如果两个机制必须生成同一类输出，一个是自动扫描，另一个是手写列表，结构变化时手写列表最容易漏。

常见例子：

* `index.md` 手写 spec 文件列表，而目录里文件已经增删。
* frontend enum label 和 options 分开维护。
* route path 在后端和前端 helper 中分别手写。
* migration 文件自动执行，但文档中的 migration 清单手写过期。

检查：

```sh
rg --files .trellis/spec
rg "/api/" frontend/src api
rg "order.created|subscriber" api
```

---

## Checklist Before Commit

* [ ] 搜索过现有模式。
* [ ] 没有把同一规则写进多个 spec。
* [ ] 新 helper 放在正确层级。
* [ ] route-local DTO 没有被过度抽象。
* [ ] 常量、路径、error code、topic 已全局搜索。
* [ ] 相关测试或 archguard 覆盖了边界。
