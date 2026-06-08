# Fix Payment Callback Receipt Hardening

## Goal

根据 `06-06-external-integration-anticorruption-layer` 的设计要求，修复 Creem payment slice 中 callback receipt 的三个边界问题：并发重复回调幂等、签名失败 receipt 的 header hash、以及 provider webhook ACK 的外部响应形态。

## Requirements

- 暂时忽略旧 `/api/orders/:id/pay` 直接支付接口问题。
- 修复 callback receipt 创建的并发竞态：重复 webhook 在唯一约束冲突时应返回 existing receipt，不应返回内部错误或重复入队。
- 签名失败/验证失败 callback 也要持久化 raw payload ciphertext、payload hash、safe snapshot，并记录 signature/header hash。
- Creem webhook route 返回 provider-facing 的极简 ACK，不再暴露 receipt id、provider event id、event type 等内部信息。

## Acceptance Criteria

- [ ] 并发或冲突场景下 `CreateIntegrationCallbackReceipt` 能返回 duplicate existing receipt。
- [ ] invalid signature route/usecase 测试能证明 receipt 写入了 `headers_hash`。
- [ ] callback route 成功 ACK 不包含内部 receipt/provider metadata。
- [ ] `go test ./...` 通过。
- [ ] `cd frontend && npm test` 通过。

## Out of Scope

- 不处理旧 `/api/orders/:id/pay` 直接支付接口。
- 不新增支付管理 UI。
- 不改变 hosted checkout 创建流程。

## Technical Notes

- 设计来源：`.trellis/tasks/06-06-external-integration-anticorruption-layer/prd.md`
- 实现来源：`.trellis/tasks/06-07-creem-payment-integration-slice/prd.md`
- 重点代码：`api/models/integration.go`、`api/usecase/payment.go`、`api/routes/payment.go`、相关测试。
