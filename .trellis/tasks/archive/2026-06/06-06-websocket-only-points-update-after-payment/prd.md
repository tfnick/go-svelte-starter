# WebSocket Only Points Update After Payment

## Goal

支付成功后，前端积分余额不再通过支付回调主动 HTTP 刷新，而是等待后端 WebSocket `points.balance` 通知更新。页面初始加载和用户显式刷新仍可读取当前积分。

## Requirements

- `Dashboard.svelte` 的支付成功处理只更新订单相关状态，不调用 `loadPoints()`。
- WebSocket 收到 `points.balance` 后继续更新 `pointsBalance`。
- WebSocket 断开不阻塞支付；断开期间支付成功后，积分展示等待重连通知或用户显式刷新。
- 更新前端 spec，明确支付后积分刷新不能依赖 HTTP 兜底。

## Acceptance

- 支付成功路径中不再出现 `Promise.all([loadOrders(), loadPoints()])` 或等价的支付后积分 HTTP 刷新。
- `npm test`、`npm run build` 和 `go test ./...` 通过。
