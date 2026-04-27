# CoDream 系统实现计划 - 2人团队协作版

> **团队规模**: 2人  
> **并发策略**: 模块化分工 + 接口先行  
> **预计周期**: 3-4周

**目标**: 构建 AI 驱动的研发全流程引擎，支持 Pipeline 编排、多 Agent 协作、Human-in-the-Loop 检查点

**架构**: API Server + Worker Pool + 共享基础设施层

**技术栈**: Golang, PostgreSQL, Redis, WebSocket, 向量数据库

---

## 团队分工策略

### Person A: 基础设施 + API 层
负责数据持久化、LLM 抽象、对外接口

### Person B: 执行引擎 + Agent 编排
负责 Pipeline 引擎、Worker Pool、Agent 任务调度

### 协作接口
两人需要在 **Phase 1** 结束时完成接口约定，确保后续并行开发

---

## Phase 1: 基础设施搭建（Week 1）

### Person A 任务组

#### A1: 项目初始化与数据库设计（刘浩宇）
**优先级**: P0  
**依赖**: 无  
**并发**: 可与 B1 并行

**职责**:
- 初始化 Go 项目结构（go.mod, 目录结构）
- 设计并创建 PostgreSQL 数据库 schema
  - pipelines 表
  - pipeline_executions 表
  - stage_executions 表
  - checkpoints 表
  - agent_tasks 表
- 编写数据库迁移脚本
- 实现基础的数据库连接和 ORM 配置

**交付物**:
- 项目骨架代码
- 数据库迁移文件
- 数据模型定义（Go structs）

---

#### A2: LLM 抽象层实现（肖志鸿）
**优先级**: P0  
**依赖**: A1  
**并发**: 可与 B2 并行

**职责**:
- 设计 LLMProvider 接口
  - Chat() 方法
  - ChatStream() 方法
  - GetModelInfo() 方法
- 实现 Anthropic Claude Provider
- 实现 OpenAI GPT Provider（可选）
- 实现 ProviderFactory 工厂模式
- 实现工具权限管理器（PermissionManager）
  - CanCallTool() 权限检查
  - ValidateToolCall() 参数验证
  - LogToolCall() 审计日志

**交付物**:
- LLM 抽象层代码包
- Provider 实现
- 权限管理器
- 单元测试

---

#### A3: REST API 框架搭建（刘浩宇）
**优先级**: P1  
**依赖**: A1, A2  
**并发**: 可与 B3 并行

**职责**:
- 选择并配置 Web 框架（推荐 Gin 或 Echo）
- 实现 API 路由结构
  - Pipeline 管理接口（CRUD）
  - 执行管理接口（启动、查询、控制）
  - 检查点管理接口（查询、审批）
- 实现统一的错误处理和响应格式
- 实现请求验证中间件
- 实现认证/授权中间件（基础版）

**交付物**:
- API Server 框架代码
- 路由定义
- 中间件实现
- API 文档（Swagger/OpenAPI）

---

#### A4: WebSocket 实时通信（刘浩宇）
**优先级**: P1  
**依赖**: A3  
**并发**: 可与 B4 并行

**职责**:
- 实现 WebSocket 连接管理
  - 连接建立和认证
  - 心跳检测（ping/pong）
  - 连接池管理
- 实现事件推送机制
  - execution.status_changed
  - stage.completed
  - checkpoint.created
  - error.occurred
  - token_usage.updated
- 实现订阅/取消订阅逻辑

**交付物**:
- WebSocket 服务代码
- 事件推送系统
- 连接管理器

---

### Person B 任务组

#### B1: Redis 消息队列配置（刘浩宇）
**优先级**: P0  
**依赖**: 无  
**并发**: 可与 A1 并行

**职责**:
- 配置 Redis 连接
- 设计消息队列结构
  - 任务队列（task queue）
  - 结果队列（result queue）
  - 延迟队列（delay queue）
- 实现消息生产者接口
- 实现消息消费者接口
- 实现消息重试机制

**交付物**:
- Redis 配置代码
- 消息队列抽象层
- 生产者/消费者实现

---

#### B2: Pipeline 执行引擎核心（肖志鸿）
**优先级**: P0  
**依赖**: B1  
**并发**: 可与 A2 并行

**职责**:
- 实现 Pipeline 状态机
  - pending → running → paused → completed/failed
- 实现 Stage 执行状态机
  - pending → planning → executing → checking → completed/failed
- 实现三阶段执行模型
  - Plan Phase 逻辑
  - Execute Phase 逻辑
  - Check Phase 逻辑
- 实现重试策略
  - 指数退避
  - 最大重试次数
  - 错误类型过滤

**交付物**:
- Pipeline 引擎核心代码
- 状态机实现
- 重试策略实现

---

#### B3: Worker Pool 实现（刘浩宇）
**优先级**: P1  
**依赖**: B1, B2  
**并发**: 可与 A3 并行

**职责**:
- 实现 Worker 进程池
  - Worker 启动和停止
  - 任务分发逻辑
  - 负载均衡
- 实现任务执行器
  - 从队列拉取任务
  - 调用 Agent 执行
  - 结果回写
- 实现资源隔离
  - 工作目录管理
  - 超时控制
  - 内存限制

**交付物**:
- Worker Pool 代码
- 任务执行器
- 资源管理器

---

#### B4: 检查点机制实现（肖志鸿）
**优先级**: P1  
**依赖**: B2  
**并发**: 可与 A4 并行

**职责**:
- 实现 CheckpointManager
  - CreateCheckpoint() 创建检查点
  - GetPendingCheckpoints() 查询待审批
  - ApproveCheckpoint() 审批决策
  - GetCheckpointHistory() 历史记录
- 实现检查点工作流
  - Before Checkpoint 逻辑
  - After Checkpoint 逻辑
  - 审批通过后的流程恢复
  - 审批拒绝后的回退逻辑

**交付物**:
- 检查点管理器代码
- 工作流控制逻辑

---

## Phase 2: 代码库上下文接口（Week 2）

### Person A 任务组

#### A5: 文件操作服务
**优先级**: P1  
**依赖**: A1  
**并发**: 可与 B5 并行

**职责**:
- 实现 FileService 接口
  - ReadFile() 读取文件
  - WriteFile() 写入文件
  - ListDir() 列表目录
  - GetFileInfo() 获取元数据
  - Delete() 删除文件/目录
- 实现文件权限检查
- 实现路径安全验证（防止路径遍历攻击）

**交付物**:
- 文件服务代码
- 安全检查逻辑

---

#### A6: Git 操作服务
**优先级**: P1  
**依赖**: A5  
**并发**: 可与 B6 并行

**职责**:
- 实现 GitService 接口
  - Clone() 克隆仓库
  - CreateBranch() 创建分支
  - CheckoutBranch() 切换分支
  - Commit() 提交变更
  - Push() 推送分支
  - GetDiff() 获取 diff
  - GetCommitHistory() 获取提交历史
- 实现 Git 凭证管理
- 实现工作目录隔离

**交付物**:
- Git 服务代码
- 凭证管理器

---

### Person B 任务组

#### B5: 语义检索服务
**优先级**: P2  
**依赖**: 无（可独立开发）  
**并发**: 可与 A5 并行

**职责**:
- 选择向量数据库（Qdrant 或 Milvus）
- 实现 SemanticSearchService 接口
  - IndexFile() 索引文件
  - Search() 语义搜索
  - GetRelevantSnippets() 获取相关代码片段
  - UpdateIndex() 更新索引
- 实现代码解析和向量化
  - 支持多语言（Go, Python, JavaScript 等）
  - 代码分块策略
  - Embedding 生成

**交付物**:
- 语义检索服务代码
- 向量数据库集成
- 代码解析器

---

#### B6: Agent 编排器
**优先级**: P1  
**依赖**: B2, A2（LLM 抽象层）  
**并发**: 可与 A6 并行

**职责**:
- 实现 Agent 接口定义
  - Initialize() 初始化
  - Execute() 执行任务
  - GetMetadata() 获取元数据
- 实现预设 Agent 类型
  - Planner Agent
  - Code Generator Agent
  - Code Reviewer Agent
  - Security Reviewer Agent
  - QA Engineer Agent
- 实现 Agent 工具调用
  - 工具注册
  - 工具权限验证
  - 工具执行

**交付物**:
- Agent 编排器代码
- 预设 Agent 实现
- 工具库

---

## Phase 3: 集成与测试（Week 3）

### 联合任务（A + B 协作）

#### AB1: 端到端集成
**优先级**: P0  
**依赖**: 所有 Phase 1 和 Phase 2 任务

**职责**:
- Person A: 集成 API Server 与 Worker Pool
  - API 调用触发任务入队
  - WebSocket 推送 Worker 事件
- Person B: 集成 Pipeline 引擎与 Agent 编排器
  - Pipeline 调用 Agent 执行
  - Agent 调用工具服务
- 联合调试接口对接问题

**交付物**:
- 完整的端到端流程
- 集成测试用例

---

#### AB2: 标准 Pipeline 实现
**优先级**: P1  
**依赖**: AB1

**职责**:
- Person A: 实现 Pipeline 配置加载
  - 从数据库加载 Pipeline 定义
  - 验证配置合法性
- Person B: 实现标准开发流程 Pipeline
  - 5 个 Stage 的完整流程
  - Stage 间依赖关系
  - 并行执行逻辑（Code Reviewer + Security Reviewer）

**交付物**:
- 标准 Pipeline 配置
- Pipeline 执行演示

---

#### AB3: 错误处理与监控
**优先级**: P1  
**依赖**: AB1

**职责**:
- Person A: 实现错误响应标准化
  - 错误码定义
  - 错误日志记录
  - 分布式追踪（trace_id）
- Person B: 实现重试和恢复机制
  - 任务失败重试
  - Pipeline 暂停/恢复
  - 状态持久化

**交付物**:
- 错误处理框架
- 监控指标定义
- 日志系统

---

## Phase 4: 测试与优化（Week 4）

### Person A 任务组

#### A7: API 测试与文档
**优先级**: P1  
**依赖**: AB1

**职责**:
- 编写 API 集成测试
  - Pipeline CRUD 测试
  - 执行流程测试
  - 检查点审批测试
- 完善 API 文档
  - Swagger/OpenAPI 规范
  - 示例请求/响应
- 性能测试
  - 并发请求测试
  - 响应时间测试

**交付物**:
- API 测试套件
- API 文档
- 性能测试报告

---

### Person B 任务组

#### B7: Pipeline 测试与优化
**优先级**: P1  
**依赖**: AB2

**职责**:
- 编写 Pipeline 单元测试
  - 状态机测试
  - 重试逻辑测试
  - 并行执行测试
- 编写 Agent 测试
  - Mock LLM 响应
  - 工具调用测试
- 性能优化
  - 减少数据库查询
  - 优化消息队列性能

**交付物**:
- Pipeline 测试套件
- Agent 测试套件
- 性能优化报告

---

## 接口约定（Phase 1 结束时确定）

### 1. API Server → Worker Pool 接口

**消息格式**:
```json
{
  "task_id": "uuid",
  "execution_id": "uuid",
  "stage_id": "uuid",
  "agent_type": "planner|code_generator|...",
  "input": { ... },
  "model_config": { ... }
}
```

**结果格式**:
```json
{
  "task_id": "uuid",
  "status": "completed|failed",
  "output": { ... },
  "error": "...",
  "token_usage": { ... }
}
```

---

### 2. Pipeline Engine → Agent 接口

**Agent.Execute() 输入**:
```json
{
  "stage_name": "planner",
  "input": { ... },
  "context": {
    "repo_path": "/path/to/repo",
    "branch": "feature-x"
  }
}
```

**Agent.Execute() 输出**:
```json
{
  "output": { ... },
  "artifacts": [ ... ],
  "next_action": "checkpoint|continue|retry"
}
```

---

### 3. Agent → Tool 接口

**工具调用格式**:
```json
{
  "tool_id": "code_search",
  "params": {
    "query": "function name",
    "language": "go"
  }
}
```

**工具响应格式**:
```json
{
  "result": { ... },
  "error": null
}
```

---

## 里程碑与验收标准

### Milestone 1: 基础设施完成（Week 1 结束）
- [ ] 数据库 schema 创建完成
- [ ] LLM 抽象层可用
- [ ] Redis 消息队列可用
- [ ] Pipeline 引擎核心逻辑完成
- [ ] API 框架搭建完成

### Milestone 2: 核心功能完成（Week 2 结束）
- [ ] 文件操作服务可用
- [ ] Git 操作服务可用
- [ ] Agent 编排器可用
- [ ] WebSocket 实时通信可用
- [ ] 检查点机制可用

### Milestone 3: 端到端集成（Week 3 结束）
- [ ] API Server 与 Worker Pool 集成完成
- [ ] Pipeline 引擎与 Agent 编排器集成完成
- [ ] 标准 Pipeline 可以完整运行
- [ ] 错误处理和监控就绪

### Milestone 4: MVP 交付（Week 4 结束）
- [ ] 所有测试通过
- [ ] API 文档完善
- [ ] 性能达标（见 spec 第 8.3 节）
- [ ] 安全验收通过

---

## 并发工作建议

### Week 1
- **Day 1-2**: A1 + B1 并行（项目初始化 + Redis 配置）
- **Day 3-4**: A2 + B2 并行（LLM 抽象层 + Pipeline 引擎）
- **Day 5**: A3 + B3 并行（API 框架 + Worker Pool）

### Week 2
- **Day 1-2**: A4 + B4 并行（WebSocket + 检查点）
- **Day 3-4**: A5 + B5 并行（文件服务 + 语义检索）
- **Day 5**: A6 + B6 并行（Git 服务 + Agent 编排）

### Week 3
- **Day 1-3**: AB1 联合集成
- **Day 4**: AB2 标准 Pipeline
- **Day 5**: AB3 错误处理

### Week 4
- **Day 1-3**: A7 + B7 并行（测试与优化）
- **Day 4-5**: 联合验收和文档整理

---

## 风险与依赖

### 关键依赖
1. **LLM API 可用性**: A2 依赖 Anthropic/OpenAI API 稳定性
2. **向量数据库选型**: B5 需要尽早确定 Qdrant 或 Milvus
3. **接口约定**: Phase 1 结束时必须完成接口定义

### 风险缓解
1. **LLM API 故障**: 实现 Provider 降级和重试机制
2. **性能瓶颈**: 预留 Week 4 进行性能优化
3. **集成问题**: Week 3 预留充足的联合调试时间

---

## 总结

这个计划将 CoDream 系统分解为两个并行开发轨道：
- **Person A**: 专注于对外接口和基础设施（API、数据库、LLM、文件/Git 服务）
- **Person B**: 专注于核心执行引擎（Pipeline、Worker、Agent、检查点）

两人在 Phase 1 和 Phase 2 可以高度并行工作，Phase 3 进行联合集成，Phase 4 进行测试和优化。

关键成功因素：
1. **接口先行**: Phase 1 结束时必须完成接口约定
2. **频繁同步**: 每天至少一次同步会议
3. **增量交付**: 每个 Phase 结束时都有可演示的功能

