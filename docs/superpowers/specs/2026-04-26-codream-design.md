# CoDream 系统设计规格说明书

**版本**: v1.0  
**日期**: 2026-04-26  
**状态**: Draft

---

## 1. 项目概述

### 1.1 项目背景
CoDream 是一个 AI 驱动的研发全流程引擎,通过多智能体协作自动化完成从需求分析到代码交付的完整开发流程。

### 1.2 核心目标
- 构建可配置的 Pipeline 引擎,支持阶段化执行和依赖管理
- 实现多 Agent 编排,每个 Agent 具备明确的角色和权限
- 提供 Human-in-the-Loop 检查点,确保关键决策的人工审批
- 采用 API-First 架构,支持前端集成和第三方扩展
- 支持多 Pipeline 并发执行,满足多用户场景

### 1.3 技术栈
- **后端**: Golang
- **数据库**: PostgreSQL
- **消息队列**: Redis
- **实时通信**: WebSocket
- **代码索引**: 向量数据库(Qdrant/Milvus)

---

## 2. 整体架构

### 2.1 架构模式
采用**混合架构**:单体 API Server + 独立 Worker Pool + 共享基础设施层

```
用户 → API Server → Redis队列 → Worker Pool → Agent执行
                ↓                              ↓
            WebSocket推送 ← 检查点/状态更新 ← PostgreSQL
```

### 2.2 核心层次

**API Server 层**
- 处理所有 HTTP 和 WebSocket 请求
- 管理 Pipeline 生命周期(启动、暂停、恢复、终止)
- 维护用户会话和 WebSocket 连接
- 推送实时通知(检查点、状态变更)

**Worker Pool 层**
- 独立工作进程池,通过 Redis 消息队列与 API Server 通信
- 异步执行 Agent 任务(Plan、Execute、Check 阶段)
- 提供资源隔离,支持水平扩展

**Infrastructure 层**
- PostgreSQL: 存储 Pipeline 定义、执行状态、检查点记录
- Git 服务: 管理目标代码库的克隆、分支、提交
- 代码索引服务: 语义检索(向量数据库)
- 文件系统: Agent 工作目录、临时文件

---

## 3. 数据模型

### 3.1 核心实体

**Pipeline (流水线定义)**
```
pipelines
- id (UUID, PK)
- name (string)
- description (text)
- config (JSONB) - Pipeline配置,包括stages定义
- status (enum: pending/running/paused/completed/failed)
- created_by (string)
- created_at, updated_at (timestamp)
```

**PipelineExecution (执行实例)**
```
pipeline_executions
- id (UUID, PK)
- pipeline_id (UUID, FK)
- status (enum: pending/running/paused/completed/failed)
- current_stage_index (int)
- input (JSONB) - 用户提交的初始输入
- output (JSONB) - 最终输出结果
- started_at, completed_at (timestamp)
```

**StageExecution (阶段执行)**
```
stage_executions
- id (UUID, PK)
- execution_id (UUID, FK)
- stage_name (string)
- stage_order (int)
- status (enum: pending/planning/executing/checking/completed/failed)
- input (JSONB)
- output (JSONB)
- plan (JSONB) - Plan Phase生成的工作项
- retry_count (int)
- started_at, completed_at (timestamp)
```

**Checkpoint (检查点)**
```
checkpoints
- id (UUID, PK)
- execution_id (UUID, FK)
- stage_id (UUID, FK)
- position (enum: before/after)
- status (enum: pending/approved/rejected)
- artifacts (JSONB) - 需要审查的产出物
- decision (JSONB) - 审批决策
- created_at, decided_at (timestamp)
```

**AgentTask (Agent任务)**
```
agent_tasks
- id (UUID, PK)
- stage_execution_id (UUID, FK)
- agent_type (string)
- status (enum: queued/running/completed/failed)
- input (JSONB)
- output (JSONB)
- model_config (JSONB)
- token_usage (JSONB)
- started_at, completed_at (timestamp)
```

---

## 4. LLM 抽象层设计

### 4.1 核心接口

**LLM Provider 接口**
```
interface LLMProvider {
  // 发送消息并获取响应
  Chat(ctx, request: ChatRequest) -> ChatResponse
  
  // 流式响应
  ChatStream(ctx, request: ChatRequest) -> Stream<ChatChunk>
  
  // 获取模型信息
  GetModelInfo(modelID: string) -> ModelInfo
}

type ChatRequest {
  model: string
  messages: Message[]
  tools: Tool[]
  temperature: float
  max_tokens: int
  system_prompt: string
}

type ChatResponse {
  content: string
  tool_calls: ToolCall[]
  usage: TokenUsage
  model: string
}

type TokenUsage {
  input_tokens: int
  output_tokens: int
  cache_creation_tokens: int
  cache_read_tokens: int
}
```

### 4.2 Provider 实现

**支持的 Provider**
- Anthropic Claude (主要)
- OpenAI GPT (备选)
- 本地模型接口 (扩展)

**Provider 工厂**
```
interface ProviderFactory {
  CreateProvider(providerType: string, config: ProviderConfig) -> LLMProvider
  GetProvider(providerType: string) -> LLMProvider
}

type ProviderConfig {
  api_key: string
  base_url: string
  timeout: duration
  retry_policy: RetryPolicy
}
```

### 4.3 工具权限管理

**工具定义与权限**
```
type Tool {
  id: string
  name: string
  description: string
  input_schema: JSONSchema
  permissions: ToolPermission[]
  rate_limit: RateLimit
}

type ToolPermission {
  agent_type: string
  stage: string
  allowed: bool
  constraints: map[string]any
}

type RateLimit {
  calls_per_minute: int
  tokens_per_hour: int
}
```

**权限检查接口**
```
interface PermissionManager {
  // 检查 Agent 是否有权限调用工具
  CanCallTool(agentType: string, toolID: string, stage: string) -> bool
  
  // 验证工具调用参数
  ValidateToolCall(toolID: string, params: map[string]any) -> ValidationResult
  
  // 记录工具调用审计日志
  LogToolCall(agentType: string, toolID: string, params: map[string]any, result: any)
}
```

**工具库**
- 代码检索工具 (CodeSearch)
- 文件操作工具 (FileOps)
- Git 操作工具 (GitOps)
- 代码执行工具 (CodeExec)
- 测试工具 (TestRunner)
- 文档生成工具 (DocGen)

---

## 5. Pipeline 执行流程

### 5.1 三阶段模型

**Plan Phase (规划阶段)**
- Agent 分析输入,生成执行计划
- 输出: 工作项列表、资源需求、依赖关系
- 检查点: 人工审批计划

**Execute Phase (执行阶段)**
- 按计划执行具体任务
- 支持并行执行、重试、超时控制
- 输出: 执行结果、中间产物

**Check Phase (检查阶段)**
- 验证执行结果是否满足要求
- 生成质量报告
- 决策: 通过、修改、重试

### 5.2 状态机

```
Pipeline Execution 状态流转:
pending → running → [paused] → completed/failed

Stage Execution 状态流转:
pending → planning → [checkpoint] → executing → checking → [checkpoint] → completed/failed

重试流程:
failed → queued → running (retry_count++)
```

### 5.3 检查点机制

**检查点类型**
- Before Checkpoint: 阶段执行前的审批
- After Checkpoint: 阶段执行后的审批

**检查点工作流**
```
interface CheckpointManager {
  // 创建检查点
  CreateCheckpoint(executionID: string, stageID: string, position: string, artifacts: any) -> Checkpoint
  
  // 获取待审批检查点
  GetPendingCheckpoints(userID: string) -> []Checkpoint
  
  // 审批决策
  ApproveCheckpoint(checkpointID: string, decision: CheckpointDecision) -> bool
  
  // 获取检查点历史
  GetCheckpointHistory(executionID: string) -> []Checkpoint
}

type CheckpointDecision {
  status: enum(approved/rejected/needs_revision)
  feedback: string
  suggested_changes: any
}
```

### 5.4 重试策略

```
type RetryPolicy {
  max_retries: int
  backoff_strategy: enum(exponential/linear/fixed)
  backoff_base: duration
  max_backoff: duration
  retry_on_errors: []string
}

// 默认策略
default_retry_policy = {
  max_retries: 3,
  backoff_strategy: exponential,
  backoff_base: 1s,
  max_backoff: 5m,
  retry_on_errors: [timeout, rate_limit, transient_error]
}
```

---

## 6. API 与事件接口

### 6.1 REST API 设计

**Pipeline 管理**
```
POST   /api/v1/pipelines              创建 Pipeline
GET    /api/v1/pipelines              列表 Pipeline
GET    /api/v1/pipelines/{id}         获取 Pipeline 详情
PUT    /api/v1/pipelines/{id}         更新 Pipeline
DELETE /api/v1/pipelines/{id}         删除 Pipeline
```

**执行管理**
```
POST   /api/v1/pipelines/{id}/execute 启动执行
GET    /api/v1/executions/{id}        获取执行状态
GET    /api/v1/executions/{id}/stages 获取阶段列表
PATCH  /api/v1/executions/{id}        暂停/恢复/终止
```

**检查点管理**
```
GET    /api/v1/checkpoints            获取待审批检查点
POST   /api/v1/checkpoints/{id}/approve 审批通过
POST   /api/v1/checkpoints/{id}/reject  审批拒绝
```

### 6.2 WebSocket 事件

**事件类型**
```
// 执行状态变更
event: execution.status_changed
payload: {
  execution_id: string
  status: string
  timestamp: datetime
}

// 阶段完成
event: stage.completed
payload: {
  execution_id: string
  stage_id: string
  stage_name: string
  output: any
}

// 检查点创建
event: checkpoint.created
payload: {
  checkpoint_id: string
  execution_id: string
  artifacts: any
  position: string
}

// 错误发生
event: error.occurred
payload: {
  execution_id: string
  error_code: string
  error_message: string
  stage_id: string
}

// Token 使用统计
event: token_usage.updated
payload: {
  execution_id: string
  input_tokens: int
  output_tokens: int
  total_cost: float
}
```

**WebSocket 连接**
```
WS /api/v1/ws/executions/{execution_id}

连接后自动订阅该执行的所有事件
支持心跳检测 (ping/pong)
自动重连机制
```

### 6.3 API 响应约定

**成功响应**
```
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

**错误响应**
```
{
  "code": 4xx/5xx,
  "message": "error description",
  "error_code": "ERROR_CODE",
  "details": { ... }
}
```

---

## 7. 代码库上下文接口

### 7.1 文件访问接口

```
interface FileService {
  // 读取文件
  ReadFile(path: string, encoding: string) -> (content: string, err: error)
  
  // 写入文件
  WriteFile(path: string, content: string, mode: int) -> error
  
  // 列表目录
  ListDir(path: string, recursive: bool) -> ([]FileInfo, error)
  
  // 获取文件元数据
  GetFileInfo(path: string) -> (FileInfo, error)
  
  // 删除文件/目录
  Delete(path: string, recursive: bool) -> error
}

type FileInfo {
  path: string
  name: string
  size: int64
  mode: int
  modified_time: datetime
  is_dir: bool
}
```

### 7.2 语义检索接口

```
interface SemanticSearchService {
  // 索引代码文件
  IndexFile(path: string, content: string, language: string) -> error
  
  // 语义搜索
  Search(query: string, language: string, limit: int) -> []SearchResult
  
  // 获取相关代码片段
  GetRelevantSnippets(query: string, context_size: int) -> []CodeSnippet
  
  // 更新索引
  UpdateIndex(path: string, content: string) -> error
}

type SearchResult {
  file_path: string
  line_number: int
  snippet: string
  relevance_score: float
  language: string
}

type CodeSnippet {
  file_path: string
  start_line: int
  end_line: int
  content: string
  language: string
}
```

### 7.3 Git 操作接口

```
interface GitService {
  // 克隆仓库
  Clone(repo_url: string, target_dir: string) -> error
  
  // 创建分支
  CreateBranch(branch_name: string, base_branch: string) -> error
  
  // 切换分支
  CheckoutBranch(branch_name: string) -> error
  
  // 提交变更
  Commit(message: string, files: []string) -> (commit_hash: string, error)
  
  // 推送分支
  Push(branch_name: string, force: bool) -> error
  
  // 获取 diff
  GetDiff(base_branch: string, target_branch: string) -> (diff: string, error)
  
  // 获取提交历史
  GetCommitHistory(branch: string, limit: int) -> ([]Commit, error)
}

type Commit {
  hash: string
  author: string
  message: string
  timestamp: datetime
  files_changed: int
}
```

---

## 8. 错误处理与测试验收

### 8.1 分层错误模型

**错误分类**
```
// 用户错误 (4xx)
- InvalidInput (400): 输入参数不合法
- Unauthorized (401): 认证失败
- Forbidden (403): 权限不足
- NotFound (404): 资源不存在
- Conflict (409): 资源冲突

// 系统错误 (5xx)
- InternalError (500): 内部错误
- ServiceUnavailable (503): 服务不可用
- Timeout (504): 超时

// 业务错误 (custom codes)
- PipelineExecutionFailed: Pipeline 执行失败
- CheckpointRejected: 检查点被拒绝
- AgentTaskFailed: Agent 任务失败
- ToolCallFailed: 工具调用失败
- RateLimitExceeded: 速率限制超出
```

**错误响应结构**
```
type ErrorResponse {
  code: int
  error_code: string
  message: string
  details: map[string]any
  trace_id: string
  timestamp: datetime
}
```

### 8.2 测试分层

**单元测试**
- LLM Provider 接口实现
- 权限管理逻辑
- 状态机转移
- 工具调用验证

**集成测试**
- Pipeline 执行流程 (Plan → Execute → Check)
- 检查点工作流
- 重试机制
- WebSocket 事件推送
- 数据库事务一致性

**端到端测试**
- 完整 Pipeline 执行 (从创建到完成)
- 多并发执行
- 错误恢复
- 性能基准测试

### 8.3 MVP 验收标准

**功能验收**
- [ ] 支持创建和执行单个 Pipeline
- [ ] 三阶段执行流程完整运行
- [ ] 检查点审批机制可用
- [ ] WebSocket 实时通知正常推送
- [ ] 支持基本的重试和错误恢复

**性能验收**
- [ ] 单个 Pipeline 执行时间 < 5 分钟 (不含 LLM 调用)
- [ ] 支持 10+ 并发执行
- [ ] API 响应时间 < 500ms (p95)
- [ ] WebSocket 消息延迟 < 100ms

**可靠性验收**
- [ ] 错误恢复率 > 95%
- [ ] 数据一致性检查通过
- [ ] 日志完整性验证通过
- [ ] 无内存泄漏 (运行 1 小时)

**安全验收**
- [ ] 工具权限检查生效
- [ ] API 认证/授权正常
- [ ] 敏感信息不在日志中泄露
- [ ] SQL 注入防护有效

---

## 9. 部署与运维

### 9.1 部署架构

**开发环境**
- 单机部署: API Server + Worker + PostgreSQL + Redis

**生产环境**
- API Server: 多副本 (负载均衡)
- Worker Pool: 独立集群 (可水平扩展)
- PostgreSQL: 主从复制 + 备份
- Redis: 集群模式 + 持久化
- 向量数据库: 独立部署

### 9.2 监控指标

**系统指标**
- CPU/内存/磁盘使用率
- 网络 I/O
- 数据库连接数

**业务指标**
- Pipeline 执行成功率
- 平均执行时间
- 检查点审批时间
- Agent 任务失败率
- Token 使用成本

### 9.3 日志与追踪

**日志级别**
- DEBUG: 详细执行流程
- INFO: 关键事件 (执行开始/完成)
- WARN: 异常情况 (重试、降级)
- ERROR: 错误信息

**分布式追踪**
- 使用 trace_id 关联请求链路
- 记录每个阶段的耗时
- 记录 LLM 调用详情

---

## 10. 预设 Pipeline 定义

### 10.1 标准开发流程 Pipeline

CoDream 提供一个预设的标准开发流程 Pipeline,包含以下 Stage:

**Stage 1: Planner (需求规划)**
- Agent类型: planner
- 模型: Claude Opus / 高质量推理模型
- 工具权限: ReadFile, SearchCode, ListDirectory (只读)
- 职责:
  - 需求澄清(苏格拉底式提问)
  - 生成需求方案(项目背景、技术栈、API设计、文件变更清单)
  - 生成测试方案
  - 生成项目排期(任务拆解、依赖关系、验收标准)
- Checkpoint: Before - 人工审批需求方案和排期

**Stage 2: Code Generator (代码生成)**
- Agent类型: code_generator
- 模型: Claude Sonnet / 高吞吐量模型
- 工具权限: ReadFile, WriteFile, SearchCode, ExecuteCommand (受限)
- 职责:
  - 根据排期逐个task执行代码生成
  - 支持加载不同skills(前端/后端开发)
  - 每个task完成后隔离上下文
- 执行策略: 按依赖关系顺序执行,无依赖的task可并行

**Stage 3: Code Reviewer (代码审查)**
- Agent类型: code_reviewer
- 模型: GPT-4 / 高质量推理模型
- 工具权限: ReadFile, SearchCode, GitDiff (只读)
- 职责:
  - 逐个task进行代码审查
  - 评估维度: 设计质量、风格一致性、工艺性、功能性
  - 输出结构化报告: VERDICT (PASS/FAIL) + 具体问题清单
- Checkpoint: After - 如果FAIL,携带问题清单回退到Stage 2

**Stage 4: Security Reviewer (安全审查)**
- Agent类型: security_reviewer
- 模型: GPT-4 / 高质量推理模型
- 工具权限: ReadFile, SearchCode, SecurityScan (只读)
- 职责:
  - 并行于Code Reviewer执行
  - 检查安全漏洞(OWASP Top 10、敏感信息泄露等)
  - 输出结构化报告: VERDICT (PASS/FAIL) + 安全问题清单
- Checkpoint: After - 如果FAIL,携带问题清单回退到Stage 2

**Stage 5: QA Engineer (质量保证)**
- Agent类型: qa_engineer
- 模型: Claude Sonnet
- 工具权限: ReadFile, ExecuteCommand (测试命令), GitCheckout
- 职责:
  - 逐个task执行测试
  - 运行单元测试、集成测试
  - 验证功能是否满足验收标准
  - 输出测试报告
- Checkpoint: After - 人工确认最终交付

### 10.2 Pipeline 配置示例

```json
{
  "name": "standard-dev-pipeline",
  "description": "标准开发流程",
  "stages": [
    {
      "name": "planner",
      "order": 1,
      "agent_type": "planner",
      "model": "claude-opus-4",
      "checkpoint": {
        "position": "before",
        "required": true
      },
      "retry_policy": {
        "max_retries": 3,
        "backoff_strategy": "exponential"
      }
    },
    {
      "name": "code_generator",
      "order": 2,
      "agent_type": "code_generator",
      "model": "claude-sonnet-4",
      "retry_policy": {
        "max_retries": 2,
        "backoff_strategy": "linear"
      }
    },
    {
      "name": "code_reviewer",
      "order": 3,
      "agent_type": "code_reviewer",
      "model": "gpt-4",
      "checkpoint": {
        "position": "after",
        "required": true
      }
    },
    {
      "name": "security_reviewer",
      "order": 3,
      "agent_type": "security_reviewer",
      "model": "gpt-4",
      "parallel_with": ["code_reviewer"],
      "checkpoint": {
        "position": "after",
        "required": true
      }
    },
    {
      "name": "qa_engineer",
      "order": 4,
      "agent_type": "qa_engineer",
      "model": "claude-sonnet-4",
      "checkpoint": {
        "position": "after",
        "required": true
      }
    }
  ]
}
```

---

## 11. 扩展性设计

### 10.1 新 Agent 类型接入

```
interface Agent {
  // Agent 初始化
  Initialize(config: AgentConfig) -> error
  
  // 执行任务
  Execute(ctx, input: any) -> (output: any, error)
  
  // 获取 Agent 元数据
  GetMetadata() -> AgentMetadata
}

type AgentMetadata {
  name: string
  version: string
  supported_stages: []string
  required_tools: []string
  capabilities: []string
}
```

### 10.2 新 Provider 接入

遵循 LLM Provider 接口规范,实现 Chat/ChatStream/GetModelInfo 方法

### 10.3 新工具接入

```
interface Tool {
  // 工具初始化
  Initialize(config: ToolConfig) -> error
  
  // 执行工具
  Execute(ctx, params: map[string]any) -> (result: any, error)
  
  // 获取工具定义
  GetDefinition() -> ToolDefinition
}
```

---

## 附录

### A. 术语表

| 术语 | 定义 |
|------|------|
| Pipeline | 可配置的工作流定义,包含多个 Stage |
| Stage | Pipeline 中的执行单元,包含 Plan/Execute/Check 三个阶段 |
| Checkpoint | 人工审批点,用于关键决策 |
| Agent | 具备特定能力的 AI 智能体 |
| Tool | Agent 可调用的工具/能力 |
| Execution | Pipeline 的一次运行实例 |

### B. 参考资源

- PostgreSQL 文档: https://www.postgresql.org/docs/
- Redis 文档: https://redis.io/documentation
- Anthropic API: https://docs.anthropic.com/
- OpenAI API: https://platform.openai.com/docs/