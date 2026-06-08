# Standardize Order/Product/Admin DTO Boundary

## Goal

Extend the internal API DTO boundary beyond users/auth by making order responses explicit and documenting how order, product, and admin surfaces should decide between model structs, DTOs, and simple message helpers.

## What I Already Know

* User/auth request and response DTO boundaries are already standardized.
* `api/routes/order.go` still returns `models.Order` and `models.OrderItem` directly in create/list/detail responses.
* `api/models/order.go` and `api/models/product.go` are storage/business-layer structs.
* There are currently no product HTTP routes; product data is read by order creation to capture price.
* `ReloadSharedDB` returns `okMessage` / `internalServerError`, which already matches the internal route helper contract.
* Internal API contracts should stay simple and should not adopt the public Open API envelope.

## Requirements

* Add explicit order response DTOs in `api/routes`.
* Replace direct order model JSON responses in order routes with DTO mapping helpers.
* Keep existing endpoint shapes compatible as practical:
  * `POST /api/orders` returns `{ "message": "...", "order": {...} }`.
  * `GET /api/orders/user/:user_id` returns an array of order objects.
  * `GET /api/orders/:id` returns `{ "order": {...}, "items": [...] }`.
  * `PATCH /api/orders/:id/status` keeps the simple message helper response.
* Keep `CreateOrderRequest` as a route request DTO and make its item type explicit.
* Use a named request DTO for order status updates.
* Do not add product routes or expose product DTOs where no route exists.
* Do not add admin DTOs for simple message-only admin operations.
* Update backend API contract docs to describe order/product/admin DTO boundary rules.
* Add focused tests for order DTO mapping and response JSON shapes.

## Acceptance Criteria

* [x] Order create/list/detail routes no longer return `models.Order` or `models.OrderItem` directly.
* [x] Order create/list/detail response shapes remain compatible.
* [x] Order status update request uses a named route request DTO.
* [x] Product model data is not exposed through a new API response in this task.
* [x] Admin reload remains a simple message/error helper route.
* [x] Focused order DTO tests pass.
* [x] `go test ./...` passes.
* [x] Backend API contract/spec docs describe order/product/admin DTO boundary rules.

## Out of Scope

* Adding product HTTP routes.
* Changing database models or migrations.
* Changing order business logic, inventory reservation, or transaction flow.
* Adding a full internal API envelope.
* Changing frontend behavior.
* Changing Open API DTOs.
* Fixing unrelated database runtime files.

## Technical Approach

* Add `OrderResponse`, `OrderItemResponse`, `CreateOrderResponse`, and `OrderDetailResponse`.
* Add mapping helpers:
  * `toOrderResponse(order *models.Order) OrderResponse`
  * `toOrderResponses(orders []models.Order) []OrderResponse`
  * `toOrderItemResponse(item *models.OrderItem) OrderItemResponse`
  * `toOrderItemResponses(items []models.OrderItem) []OrderItemResponse`
* Replace direct `order`, `orders`, and `items` JSON responses in `api/routes/order.go`.
* Replace the inline status update request struct with `UpdateOrderStatusRequest`.
* Add route-package tests for mapping and JSON shapes.
* Extend `.trellis/spec/backend/api-contracts.md`.

## Technical Notes

* Relevant files:
  * `api/routes/order.go`
  * `api/models/order.go`
  * `api/models/product.go`
  * `api/routes/admin.go`
  * `.trellis/spec/backend/api-contracts.md`
  * `.trellis/spec/backend/route-handler-guidelines.md`
* Repo inspection:
  * `order.go` is the only internal route file currently returning order model structs directly.
  * `product.go` has model functions but no route handlers.
  * `admin.go` uses internal response helpers and does not return resource models.
