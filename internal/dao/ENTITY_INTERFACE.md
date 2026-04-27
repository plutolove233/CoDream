# Entity 接口与类型安全的 DAO

## 概述

DAO 层现在使用泛型约束来提供类型安全。所有实体必须实现 `Entity` 接口。

## Entity 接口定义

```go
type Entity interface {
    GetID() uuid.UUID
    SetID(uuid.UUID)
}
```

## 已实现的模型

以下模型已经实现了 `Entity` 接口：

- `Pipeline`
- `PipelineExecution`
- `StageExecution`
- `AgentTask`
- `Checkpoint`

## 使用示例

### 正确的用法（使用指针类型）

```go
// 获取 DAO 实例时使用指针类型
pipelineDAO := dao.GetDAO[*models.Pipeline]()
executionDAO := dao.GetDAO[*models.PipelineExecution]()

// 创建实体
pipeline := &models.Pipeline{
    Name:        "示例流水线",
    Description: "这是一个示例",
    Status:      models.PipelineStatusPending,
}

err := pipelineDAO.Create(ctx, pipeline)
```

### 错误的用法（使用值类型）

```go
// ❌ 错误：不要使用值类型
pipelineDAO := dao.GetDAO[models.Pipeline]()  // 编译错误！
```

## 为什么使用指针类型？

在 Go 中，当接口方法有指针接收器时（如 `SetID(*T)`），只有指针类型才满足该接口。

我们的 `SetID` 方法定义为：

```go
func (p *Pipeline) SetID(id uuid.UUID) {
    p.ID = id
}
```

因此，只有 `*Pipeline` 类型才实现了 `Entity` 接口，而不是 `Pipeline` 值类型。

## 为新模型添加 Entity 支持

如果你创建了新的模型，需要添加以下方法：

```go
// GetID 返回实体的 ID
func (m YourModel) GetID() uuid.UUID {
    return m.ID
}

// SetID 设置实体的 ID
func (m *YourModel) SetID(id uuid.UUID) {
    m.ID = id
}
```

然后就可以使用 DAO 了：

```go
yourModelDAO := dao.GetDAO[*models.YourModel]()
```

## 类型安全的好处

1. **编译时检查**：如果模型没有实现 `Entity` 接口，代码将无法编译
2. **IDE 支持**：IDE 可以提供更好的自动完成和类型提示
3. **防止错误**：避免在运行时才发现类型不匹配的问题
4. **清晰的契约**：明确表明所有 DAO 实体必须有 ID 字段和相关方法
