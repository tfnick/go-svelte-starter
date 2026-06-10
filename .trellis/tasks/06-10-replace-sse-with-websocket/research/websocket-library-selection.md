# WebSocket Library Selection

## Context

当前系统的实时能力通过自研 `realtime.Hub` 分发 JSON envelope，再由 route 层用 SSE (`EventSource` + `text/event-stream`) 输出到浏览器。迁移目标是移除 SSE 传输，改为 WebSocket。

## Options

### Option A: `github.com/coder/websocket` (Recommended)

Source: https://github.com/coder/websocket

该库定位为 minimal and idiomatic WebSocket library for Go，README 明确列出：

* first-class `context.Context` support
* zero dependencies
* JSON helpers in `wsjson`
* concurrent writes
* close handshake
* ping/pong API
* passes Autobahn testsuite

Fit for this repo:

* 当前项目偏好轻量依赖；该库 zero dependencies，符合现有 `go.mod` 风格。
* route 层已经以 request context 驱动取消，`context.Context` 支持能自然承接连接生命周期。
* 现有 realtime payload 是 JSON envelope，`wsjson` 可减少手写编码/写帧样板。
* 可以保留 `api/framework/realtime` 的 hub/subscription 模型，只替换传输层 writer。

Trade-offs:

* 相比 Gorilla，社区历史和使用规模较小。
* 需要新增依赖，并补齐 WebSocket close/heartbeat 测试。

### Option B: `github.com/gorilla/websocket`

Source: https://github.com/gorilla/websocket

Gorilla WebSocket README 表示该包提供完整、经过测试的 WebSocket protocol implementation，API stable，并通过 Autobahn server tests。

Fit for this repo:

* 成熟、广泛使用。
* 示例和资料多。

Trade-offs:

* API 风格相对底层，ping/pong、close、并发写入等需要更多手写约束。
* 当前项目没有 Gorilla 生态依赖，引入它不会复用既有栈。

### Option C: Hand-roll on `net/http`

Not recommended. Go 标准库没有完整 WebSocket server implementation，手写 upgrade/framing/masking/close/ping-pong 会引入不必要风险。

## Recommendation

使用 `github.com/coder/websocket`。

迁移重点不应放在重写业务分发，而是：

* 保留 `api/framework/realtime` 的 user/client subscription 模型。
* 新增 WebSocket route：验证当前用户，建立 subscription，循环写出已有 realtime envelope。
* 前端新增 `WebSocket` client helper，替换所有 `EventSource` 和 `*SSEURL` helper。
* 重命名业务文案、notification type、测试名和文档中的 SSE 概念。
