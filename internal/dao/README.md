# DAO 层

DAO (Data Access Object) 层为 CoDream 项目提供了简化的数据访问接口，基于 GORM 实现。

## 特性

- **类型安全**: 使用 Go 泛型和 Entity 接口约束，提供编译时类型检查
- **统一接口**: 提供一致的 CRUD 操作接口
- **软删除**: 自动支持软删除，保证数据安全
- **事务支持**: 内置事务管理
- **分页查询**: 简化的分页操作
- **工厂模式**: 全局工厂管理 DAO 实例

## 快速开始

### 1. 初始化

```go
import (
    "context"
    "github.com/plutolove233/co-dream/internal/dao"
    "github.com/plutolove233/co-dream/internal/database"
)

// 初始化数据库连接
ctx := context.Background()
config := database.NewConfig()
db, err := database.NewDatabase(ctx, config)
if err != nil {
    log.Fatal(err)
}
defer db.Close(ctx)

// 初始化 DAO 工厂
if err := dao.InitDAOFactory(db.DB()); err != nil {
    log.Fatal(err)
}
```

### 2. 基础 CRUD 操作

```go
import "github.com/plutolove233/co-dream/internal/dal/models"

// 获取 DAO 实例（注意：必须使用指针类型）
pipelineDAO := dao.GetDAO[*models.Pipeline]()

// 创建
pipeline := &models.Pipeline{
    Name:        "示例流水线",
    Description: "这是一个示例流水线",
    Status:      models.PipelineStatusPending,
    CreatedBy:   "user123",
}
err := pipelineDAO.Create(ctx, pipeline)

// 查询
found, err := pipelineDAO.FindByID(ctx, pipeline.ID)

// 更新
pipeline.Status = models.PipelineStatusRunning
err = pipelineDAO.Update(ctx, pipeline)

// 删除（软删除）
err = pipelineDAO.Delete(ctx, pipeline)
```

### 3. 查询操作

```go
// 查询所有
all, err := pipelineDAO.FindAll(ctx)

// 统计
count, err := pipelineDAO.Count(ctx)

// 检查存在性
exists, err := pipelineDAO.Exists(ctx, id)

// 条件查询
pipelines, err := pipelineDAO.FindByCondition(ctx, func(db *gorm.DB) *gorm.DB {
    return db.Where("status = ?", models.PipelineStatusRunning).
        Order("created_at DESC").
        Limit(10)
})
```

### 4. 分页查询

```go
// 第 1 页，每页 10 条
items, total, err := pipelineDAO.Paginate(ctx, 1, 10)

page := 1
pageSize := 10
totalPages := (int(total) + pageSize - 1) / pageSize

log.Printf("第 %d 页，共 %d 页，总共 %d 条记录", page, totalPages, total)
```

### 5. 事务操作

```go
err := pipelineDAO.Transaction(ctx, func(txDAO dao.DAO[*models.Pipeline]) error {
    // 在事务中创建 Pipeline
    newPipeline := &models.Pipeline{
        Name:      "事务中创建的流水线",
        Status:    models.PipelineStatusPending,
        CreatedBy: "user123",
    }
    if err := txDAO.Create(ctx, newPipeline); err != nil {
        return err
    }

    // 在事务中创建关联的 Execution
    execution := &models.PipelineExecution{
        PipelineID: newPipeline.ID,
        Status:     models.ExecutionStatusPending,
    }
    executionDAO := dao.GetDAO[*models.PipelineExecution]()
    if err := executionDAO.Create(ctx, execution); err != nil {
        return err
    }

    return nil
})
```

## Entity 接口

所有实体必须实现 `Entity` 接口：

```go
type Entity interface {
    GetID() uuid.UUID
    SetID(uuid.UUID)
}
```

**重要**: 使用 DAO 时必须使用指针类型（如 `*models.Pipeline`），因为 `SetID` 方法有指针接收器。

详细说明请参考 [ENTITY_INTERFACE.md](./ENTITY_INTERFACE.md)。

## 支持的实体

DAO 层支持以下实体（使用时需要加 `*` 指针）：

- `*models.Pipeline` - 流水线定义
- `*models.PipelineExecution` - 流水线执行记录
- `*models.StageExecution` - 阶段执行记录
- `*models.AgentTask` - Agent 任务
- `*models.Checkpoint` - 检查点

## 错误处理

```go
retrieved, err := pipelineDAO.FindByID(ctx, id)
if err != nil {
    if dao.IsNotFound(err) {
        log.Println("记录不存在")
        return
    }
    log.Printf("查询失败: %v", err)
    return
}
```

## API 参考

### Entity 接口

```go
type Entity interface {
    GetID() uuid.UUID
    SetID(uuid.UUID)
}
```

### DAO 接口

```go
type DAO[T Entity] interface {
    // 基础 CRUD 操作
    Create(ctx context.Context, entity *T) error
    FindByID(ctx context.Context, id uuid.UUID) (*T, error)
    Update(ctx context.Context, entity *T) error
    Delete(ctx context.Context, entity *T) error

    // 扩展查询操作
    FindAll(ctx context.Context) ([]*T, error)
    FindByCondition(ctx context.Context, conditionFunc func(db *gorm.DB) *gorm.DB) ([]*T, error)
    Count(ctx context.Context) (int64, error)
    Exists(ctx context.Context, id uuid.UUID) (bool, error)

    // 分页操作
    Paginate(ctx context.Context, page, pageSize int) ([]*T, int64, error)

    // 事务支持
    Transaction(ctx context.Context, fn func(txDAO DAO[T]) error) error
}
```

### 错误类型

- `ErrNotFound` - 记录不存在
- `ErrEntityNil` - 实体为空
- `ErrInvalidPageSize` - 无效的分页大小
- `ErrInvalidPage` - 无效的页码
- `ErrTransactionFailed` - 事务失败

## 设计文档

详细的设计文档请参考 `/docs/superpowers/specs/2026-04-27-dao-layer-design.md`。
