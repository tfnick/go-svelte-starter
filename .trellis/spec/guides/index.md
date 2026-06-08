# Thinking Guides

> 这些 guide 用来在编码前扩展思考，帮助发现容易遗漏的跨层、复用和维护风险。

---

## Available Guides

| Guide | Use For | Trigger |
| --- | --- | --- |
| [Code Reuse Thinking Guide](./code-reuse-thinking-guide.md) | 识别重复模式，决定是否抽象 | 新增 helper、复制相似逻辑、批量改多个文件 |
| [Cross-Layer Thinking Guide](./cross-layer-thinking-guide.md) | 梳理跨层数据流和契约 | 变更涉及 route/usecase/model/db/frontend 中三层以上 |

---

## Quick Reference

### Cross-Layer Triggers

* API response shape 变化。
* DTO、usecase `Co`、model struct 之间需要映射。
* 数据同时穿过 frontend、route、usecase、model、DB。
* 错误码、日志、事务、事件会影响多个模块。
* 不确定逻辑应该放在哪一层。

读 [Cross-Layer Thinking Guide](./cross-layer-thinking-guide.md)。

### Code Reuse Triggers

* 发现同一模式出现三次以上。
* 正在新增 helper 或 framework capability。
* 正在修改常量、配置、字段名、error code、JSON key。
* 批量修改多个相似文件。
* 一处代码通过自动机制生成输出，另一处通过手写列表生成相同输出。

读 [Code Reuse Thinking Guide](./code-reuse-thinking-guide.md)。

---

## Pre-Modification Rule

改任何值之前先搜索：

```sh
rg "value_or_name_to_change"
```

这个规则尤其适用于：

* JSON field name
* API path
* error code
* event topic
* subscriber name
* dictionary type
* migration/table/column name
* frontend helper name

---

## Spec Writing Rule

`.trellis/spec/` 文档使用：

* 英文文件名、标题、章节标题。
* 中文正文说明。
* 英文代码、路径、函数名、字段名和 JSON key。
* 同一主题只在一个主 spec 展开，其他 spec 做引用。
