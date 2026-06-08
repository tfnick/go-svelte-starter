# 升级会员时自动取消旧订阅

## Goal

当用户升级会员等级（premium → super），自动取消旧的 Creem 订阅，避免重复扣款。

## Requirements

- 升级会员时，自动查询用户所有 active 状态的旧订阅
- 通过 Creem API 取消旧订阅
- 本地标记旧订单 subscription_status = canceled
- 不折算，剩余时长直接作废

## Technical Approach

1. `ports.go`: 新增 `CancelSubscription` 接口方法
2. `creem.go`: 实现 `CancelSubscription` — `POST /v1/subscriptions/{id}/cancel`
3. `usecase/order.go`: `ApplyOrderMembership` 中检测升级场景，取消旧订阅
4. 取消旧订阅在事务外执行（API 调用不应嵌在 DB 事务中）

## Out of Scope

- Prorated 折算
- 取消失败重试

## Technical Notes

- Creem 取消端点: `POST /v1/subscriptions/{id}/cancel`
- 取消模式: scheduled（让用户在当前周期结束前继续享有旧等级服务）
