# UUID Generation

> 本文定义项目内 UUID 生成约定。只要代码需要生成新的 UUID 字符串，无论是否最终作为数据库主键，都默认使用 UUID v7。

---

## Convention

项目使用 `github.com/google/uuid` 生成 UUID。新生成的 UUID 字符串统一使用：

```go
uuid.Must(uuid.NewV7()).String()
```

不要新增：

```go
uuid.New().String()
uuid.NewString()
```

原因：

* UUID v7 带时间有序前缀，写入 `TEXT` 主键、事件、调度执行、临时 task/client/request ID 时都能保持更好的时间局部性。
* 统一一种生成方式，避免同一系统里混用 v4/v7 导致排序和排查行为不一致。
* `uuid.New()` / `uuid.NewString()` 本身会在随机源失败时 panic；`uuid.Must(uuid.NewV7())` 保持同一错误处理风格。

---

## Scope

适用范围：

* 数据库主键、业务实体 ID、事件 ID、delivery/execution ID。
* request ID、realtime client ID、导出/通知 task ID 等非持久化或短生命周期 UUID。
* 未来新增的任何项目内 UUID 字符串生成点。

例外：

* 解析、校验或比较外部传入的 UUID 时，按场景使用 `uuid.Parse` / `uuid.Validate`。
* 第三方协议明确要求 UUID v4 时，必须在调用点附近注释说明协议要求，并在相关 spec/PRD 中记录。

---

## Review Checklist

修改 UUID 生成相关代码时：

* 运行 `rg "uuid\\.New\\(|uuid\\.NewString\\(" api`，应无生产调用点。
* 运行 `go test ./...`。
* 如果引入第三方协议要求 v4，更新对应 topic spec，说明为什么不能使用 v7。
