# CoDream 数据库配置指南

## 概述

本文档说明如何设置和使用 CoDream 项目的 PostgreSQL 数据库。

## 数据库架构

### 核心表

1. **pipelines** - 流水线定义
   - 存储 Pipeline 配置、状态和元数据
   - 支持 JSONB 配置，类型安全的 PipelineConfig 结构

2. **pipeline_executions** - 执行实例
   - 记录每次 Pipeline 执行的状态和结果
   - 跟踪当前执行阶段和输入/输出

3. **stage_executions** - 阶段执行
   - 记录每个 Stage 的执行详情
   - 包含 Plan、Execute、Check 三阶段的数据

4. **checkpoints** - 检查点
   - Human-in-the-Loop 审批点
   - 存储审批决策和产出物

5. **agent_tasks** - Agent 任务
   - 记录 Agent 执行的具体任务
   - 包含 Token 使用统计和模型配置

## 快速开始

### 1. 启动数据库服务

```bash
# 启动 PostgreSQL 和 Redis
docker-compose up -d postgres redis

# 检查服务状态
docker-compose ps
```

### 2. 运行数据库迁移

使用 GORM 自动迁移（推荐用于开发环境）：

```bash
go run cmd/migrate/main.go
```

或使用 SQL 脚本（推荐用于生产环境）：

```bash
# 连接到数据库
docker exec -it codream_postgres psql -U codream -d codream

# 执行迁移脚本
\i /path/to/migrations/001_init_schema.sql
```

### 3. 验证数据库

```bash
# 进入 PostgreSQL 容器
docker exec -it codream_postgres psql -U codream -d codream

# 查看所有表
\dt

# 查看表结构
\d pipelines
\d pipeline_executions
\d stage_executions
\d checkpoints
\d agent_tasks
```

## 使用示例

### 连接数据库

```go
package main

import (
    "log"
    "github.com/plutolove233/co-dream/internal/database"
)

func main() {
    // 创建配置
    config := database.NewConfig()
    
    // 连接数据库
    db, err := database.Connect(config)
    if err != nil {
        log.Fatal(err)
    }
    defer database.Close()
    
    // 使用 db 进行操作...
}
```

### 创建 Pipeline

```go
pipeline := &models.Pipeline{
    Name:        "Standard Dev Pipeline",
    Description: "标准开发流程",
    Config: models.PipelineConfig{
        Name: "standard-dev",
        Stages: []models.StageConfig{
            {
                Name:      "planner",
                Order:     1,
                AgentType: "planner",
                Model:     "claude-opus-4",
            },
        },
    },
    Status:    models.PipelineStatusPending,
    CreatedBy: "user@example.com",
}

if err := db.Create(pipeline).Error; err != nil {
    log.Fatal(err)
}
```

## 环境变量

在 `.env` 文件中配置以下变量：

```env
# PostgreSQL
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=codream
POSTGRES_PASSWORD=codream_secret
POSTGRES_DB=codream
POSTGRES_SSLMODE=disable

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=codream_redis_secret
```

## 数据库维护

### 备份

```bash
# 备份数据库
docker exec codream_postgres pg_dump -U codream codream > backup.sql

# 恢复数据库
docker exec -i codream_postgres psql -U codream codream < backup.sql
```

### 清理测试数据

```bash
# 进入数据库
docker exec -it codream_postgres psql -U codream -d codream

# 清空所有表（保留结构）
TRUNCATE pipelines, pipeline_executions, stage_executions, checkpoints, agent_tasks CASCADE;
```

## 性能优化

### 索引策略

数据库已创建以下关键索引：
- 状态字段索引（用于快速查询执行状态）
- 时间戳索引（用于按时间排序）
- 外键索引（用于关联查询）
- 复合索引（用于常见查询模式）

### 连接池配置

在 `database.go` 中已配置：
- MaxIdleConns: 10
- MaxOpenConns: 100
- ConnMaxLifetime: 1 hour

## 故障排查

### 连接失败

```bash
# 检查 PostgreSQL 是否运行
docker-compose ps postgres

# 查看日志
docker-compose logs postgres

# 测试连接
docker exec codream_postgres pg_isready -U codream
```

### 迁移失败

```bash
# 查看 GORM 日志
# 在代码中设置 logger.LogMode(logger.Info)

# 手动检查表结构
docker exec -it codream_postgres psql -U codream -d codream -c "\d+ pipelines"
```

## 下一步

- 实现 API Server 层
- 添加 Worker Pool
- 集成 Redis 消息队列
- 实现 WebSocket 实时通知
