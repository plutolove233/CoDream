# GORM Gen 使用指南

## 快速开始

### 1. 初始化

在应用启动时设置数据库连接：

```go
import (
    "github.com/plutolove233/co-dream/internal/database"
    "github.com/plutolove233/co-dream/internal/dal/query"
)

// 初始化数据库
ctx := context.Background()
config := database.NewConfig()
db, err := database.NewDatabase(ctx, config)
if err != nil {
    log.Fatal(err)
}

// 设置默认连接（全局单例）
gen.SetDefault(db.DB())
```

### 2. 基本 CRUD 操作

#### CREATE - 创建

```go
p := gen.Pipeline

newPipeline := &models.Pipeline{
    Name:        "我的流水线",
    Description: "流水线描述",
    Status:      models.PipelineStatusPending,
    CreatedBy:   "user123",
}

err := p.WithContext(ctx).Create(newPipeline)
```

#### READ - 查询

```go
p := gen.Pipeline

// 根据ID查询
pipeline, err := p.WithContext(ctx).
    Where(p.ID.Eq(pipelineID)).
    First()

// 查询列表（带条件）
pipelines, err := p.WithContext(ctx).
    Where(p.Status.Eq(string(models.PipelineStatusPending))).
    Order(p.CreatedAt.Desc()).
    Limit(10).
    Find()

// 多条件查询
pipelines, err := p.WithContext(ctx).
    Where(
        p.Status.In("pending", "running"),
        p.CreatedBy.Eq("user123"),
    ).
    Find()

// 模糊查询
pipelines, err := p.WithContext(ctx).
    Where(p.Name.Like("%关键词%")).
    Find()
```

#### UPDATE - 更新

```go
p := gen.Pipeline

// 更新单个字段
_, err := p.WithContext(ctx).
    Where(p.ID.Eq(pipelineID)).
    Update(p.Status, "running")

// 更新多个字段
_, err := p.WithContext(ctx).
    Where(p.ID.Eq(pipelineID)).
    Updates(map[string]interface{}{
        "status":      "completed",
        "description": "已完成",
    })
```

#### DELETE - 删除

```go
p := query.Pipeline

// 软删除（推荐）
_, err := p.WithContext(ctx).
    Where(p.ID.Eq(pipelineID)).
    Delete()

// 永久删除
_, err := p.WithContext(ctx).
    Unscoped().
    Where(p.ID.Eq(pipelineID)).
    Delete()
```

### 3. 高级查询

#### 关联查询

```go
p := query.Pipeline

// 预加载关联数据
pipeline, err := p.WithContext(ctx).
    Preload(p.Executions).
    Where(p.ID.Eq(pipelineID)).
    First()

// 访问关联数据
for _, exec := range pipeline.Executions {
    fmt.Println(exec.Status)
}
```

#### 聚合查询

```go
p := query.Pipeline

// 统计数量
count, err := p.WithContext(ctx).
    Where(p.Status.Eq("pending")).
    Count()

// 分页查询
page := 1
pageSize := 20
pipelines, err := p.WithContext(ctx).
    Offset((page - 1) * pageSize).
    Limit(pageSize).
    Find()
```

#### 事务处理

```go
err := query.Q.Transaction(func(tx *query.Query) error {
    // 在事务中执行多个操作
    if err := tx.Pipeline.WithContext(ctx).Create(pipeline); err != nil {
        return err
    }
    
    if err := tx.PipelineExecution.WithContext(ctx).Create(execution); err != nil {
        return err
    }
    
    return nil
})
```

### 4. 查询条件方法

| 方法 | 说明 | 示例 |
|------|------|------|
| `Eq(value)` | 等于 | `p.Status.Eq("pending")` |
| `Neq(value)` | 不等于 | `p.Status.Neq("failed")` |
| `Gt(value)` | 大于 | `p.CreatedAt.Gt(time.Now())` |
| `Gte(value)` | 大于等于 | `p.CreatedAt.Gte(startTime)` |
| `Lt(value)` | 小于 | `p.CreatedAt.Lt(endTime)` |
| `Lte(value)` | 小于等于 | `p.CreatedAt.Lte(time.Now())` |
| `In(values...)` | 在列表中 | `p.Status.In("pending", "running")` |
| `NotIn(values...)` | 不在列表中 | `p.Status.NotIn("failed", "cancelled")` |
| `Like(pattern)` | 模糊匹配 | `p.Name.Like("%test%")` |
| `IsNull()` | 为空 | `p.DeletedAt.IsNull()` |
| `IsNotNull()` | 不为空 | `p.CompletedAt.IsNotNull()` |

### 5. 重新生成代码

当数据库模型变更后，重新运行生成器：

```bash
go run ./cmd/gen
```

## 完整示例

查看 `examples/dal_usage_example.go` 获取完整的使用示例。

## 参考资料

- [GORM Gen 官方文档](https://gorm.io/gen/)
- [GORM 官方文档](https://gorm.io/docs/)
