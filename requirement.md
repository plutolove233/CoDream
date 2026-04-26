# 系统架构：中心化星型拓扑结构

```tex
plaintext
        用户
          ↕
    主 Agent
  (Query Loop)   ← 唯一的中央控制节点
   ↙  ↓  ↘
子代理A 子代理B 子代理C

    × 互不通信 ×

说明：
顶层：用户
中间层：主 Agent（Query Loop），标注为「唯一的中央控制节点」
底层：子代理 A、子代理 B、子代理 C
说明：子代理之间互不通信，所有交互、调度全部经由主 Agent 中转
```

通信方式：工具调用，将Agent实例化作为一个工具

消息传递：只返回执行结果，隔离每个子Agent的思考过程。

# 执行流程
```
用户提交 Pipeline（含需求描述）
  │
  ▼
Main Orchestrator Agent 启动
  │
  ├─► for each Stage (按 order 顺序):
  │     │
  │     ├─► [CheckpointConfig.position == "before"] 触发 Human 检查点
  │     │     ├── Approve → 继续
  │     │     └── Reject  → 携带理由，回退到上一阶段重做（或终止）
  │     │
  │     ├─► Stage 内部执行循环：
  │     │     1. Plan Phase（预设 Agent 自动规划子任务）
  │     │          └── 拆解出 work1, work2, ... workN
  │     │     2. Execute Phase（Agent 按计划执行各 work item）
  │     │          └── 可调用 Tool
  │     │     3. Check Phase（Skill 验证输出质量）
  │     │          ├── Pass  → 汇总结果（执行步骤 + 产出物）
  │     │          └── Fail  → 触发重试（RetryPolicy）
  │     │              ├── 未超最大重试 → 重新执行 Plan + Execute
  │     │              └── 超出最大重试 → Stage 置为 failed，等待人工介入
  │     │
  │     ├─► Stage Output 传递给下一 Stage 作为 Input
  │     │
  │     └─► [CheckpointConfig.position == "after"] 触发 Human 检查点
  │           ├── Approve → 进入下一 Stage
  │           └── Reject  → 携带理由，当前 Stage 重做
  │
  └─► 所有 Stage 完成 → Pipeline 置为 completed，输出最终交付物
```

## Stage结构
```
Stage
  ├── Plan Phase
  │     - 预设 Agent 分析 Stage Input
  │     - 自动拆解为若干 WorkItem（work1, work2, ...）
  │     - 生成执行计划
  │
  ├── Execute Phase
  │     - 按 WorkItem 顺序（或并行）调用 Agent
  │     - Agent 可调用 Tool
  │     - 记录每个 WorkItem 的执行结果
  │
  └── Check Phase（Skill）
        - 验证 Execute Phase 产出质量
        - 输出：{ passed: bool, issues: string[], suggestions: string[] }
        - 结果连同执行摘要一并返回给 Main Orchestrator
```



# 预设的pipeline

Agent tool：

- 架构：React

- prompt
- name
- skills
- tools

```tex
# CoDream预设的pipeine

用户需求 -> planner -> Code Generator -> Code Reviewer → Security Reviewer → QA Engineer

- planner： 
model：高质量推理模型，Claude系列模型/DS 
tools权限：Read Glob Grep
Planner → 「ambitious scope」：避免功能规划不足，主动提议更完整的方案。
	- 需求澄清Agent：苏格拉底式提问法，理解用户意图，澄清歧义  
	- 需求方案生成Agent/skills：根据需求方案模板（支持预设），输出需求方案
		- 预设的需求方案模板包括：项目背景与意义，技术栈要求，API设计，文件变更清单
	- 测试方案生成Agent/skills：根据需求方案，专门调用一个Agent，生成**测试方案**，
	- 项目排期Agent/skills：根据需求方案 + 测试方案，生成项目排期方案，每个task包含taskid、任务描述、文件变更diff、验收标准、task依赖关系，是后续项目推进的依据
		- 分析子任务的执行顺序，解析子任务项目依赖，同一排期（如A）中，若task无前置依赖项可以并行执行
	- agent check

- Code Generator
model：高吞吐量的模型
tools权限：Read Write Edit Bash Glob / Grep
Generator → 「incremental and testable」：确保每步产出都可验证，而非一次性生成整个工程。
	- 支持加载不同skills，成为不同功能的Agent：前端设计Agent，go/python后端开发Agent
	- 根据项目排期，逐个task执行
	- 每个task执行完成后，自动消亡，新建Code Generator，隔离上下文

评估方法：如何评估代码生成质量？ -> Code Reviewer、Security Reviewer

- Code Reviewer
model：高质量推理模型，GPT系列模型
tools权限： Read Bash Glob / Grep，只给读权限，而不给写权限，防止模型“顺手帮你修改了”
Code Reviewer → 「critical and skeptical」：抵消模型天然的宽大倾向，明确要求「对每个函数都假设它有问题」。
	- 支持加载Code Reviewer的对应语言的skills
	- 逐个task进行Review
	- 给出结构化的输出，输出 VERDICT: PASS / FAIL，将主观的"代码看起来还行"转化为二元的通过/失败信号，使得流水线可以基于这个信号做出自动化决策。
	
- Security Reviewer：与Code Reviewer并行执行
model：高质量推理模型，GPT系列模型
tools权限： Read Bash Glob / Grep，只给读权限，而不给写权限，防止模型“顺手帮你修改了”
	- 逐个task进行Review
	- 给出结构化的输出，输出 VERDICT: PASS / FAIL，将主观的"代码看起来还行"转化为二元的通过/失败信号，使得流水线可以基于这个信号做出自动化决策。

QA Engineer
model：sonnet
tools权限：Read Bash Glob / Grep
功能：通过测试、构建和回归验证确认任务是否真的满足完成标准。
	- 逐个task执行测试
```



# CoDream关键设计决策

**模型选择的差异化**：Planner 和 Reviewer 使用 opus（更强的推理能力），Generator 和 QA 使用 sonnet（更快的执行速度）。规划和审查需要深度思考，而代码生成和测试执行需要高效的工具调用吞吐量。

**工具集的权限隔离**：Code Reviewer 和 Security Reviewer 没有 Write 和 Edit 权限。评审者只能阅读和分析，不能修改代码。这确保了评审的独立性——如果评审者可以直接修改代码，它就会倾向于"修复后通过"而不是"标记问题并拒绝"。

**轮次预算的约束**：每个智能体都有硬性的轮次上限。这不仅是成本控制，更是一种认知卫生：它迫使每个智能体在有限的"注意力预算"内完成工作，防止无止境的自我修正循环。



# 每个Agent产物评估策略

- planner

  ```tex
  每个方案评估：
  1. 完整性：计划是否覆盖了输入需求的所有要点？有无明显遗漏？
  2. 可执行性：每个子任务是否足够具体，可以被 AI Agent 直接执行？
     （模糊描述如"处理登录逻辑"不可接受，"修改 src/auth/login.ts 第42行的验证函数"可接受）
  3. 顺序合理性：子任务的执行顺序是否合理？是否存在依赖倒置？ 
  4. 边界清晰性：子任务之间是否有明确边界，避免重叠和重复劳动？
  5. 风险识别：计划是否识别并标注了高风险步骤？
  ```

- Code Generator - 见Code Reviewer

- Code Reviewer

  ```tex
  - 结构化评分框架——从以下4各维度进行评估：
  		- go最佳实践规范，支持配置
      - Design Quality（设计质量 / 风格一致性）：模块间的命名规范、错误处理模式、API 设计风格是否全局统一。
      - Originality（原创性）：是否有自定义的设计决策，而非套用模板。Anthropic 特别提到要惩罚「AI slop」——千篇一律的模板代码输出。
      - Craft（工艺性）：边界条件覆盖、类型安全、性能热点处理、日志充分等技术执行的精细度。
      - Functionality（功能性）：功能是否按预期工作、测试是否通过、用户路径是否畅通——独立于代码美学的可用性判断。
  
  - QA 反馈的具体性原则
  优秀的评估器不是输出「代码质量良好」，而是输出可执行的 Bug 报告。每个 VERDICT: FAIL 必须包含三要素：
      - 在哪里：文件路径、函数名、行号。不说「拖拽功能有问题」，而说「Tool 只在拖拽起点和终点放置瓷砖，未填充中间区域」。
      - 什么现象：实际行为 vs 预期行为。不说「API 报错」，而说「FastAPI 将 reorder 字符串匹配为 frame_id 的整数参数，返回 422」。
      - 为什么：根因分析。这样 Debugger 才能快速定位问题，而不是在 60 轮预算中浪费时间重复发现同一个 Bug。
  ```

- Security Reviewer

  ```tex
  安全性评估：
  	安全性skill
  	
  - QA 反馈的具体性原则
  优秀的评估器不是输出「代码质量良好」，而是输出可执行的 Bug 报告。每个 VERDICT: FAIL 必须包含三要素：
      - 在哪里：文件路径、函数名、行号。不说「拖拽功能有问题」，而说「Tool 只在拖拽起点和终点放置瓷砖，未填充中间区域」。
      - 什么现象：实际行为 vs 预期行为。不说「API 报错」，而说「FastAPI 将 reorder 字符串匹配为 frame_id 的整数参数，返回 422」。
      - 为什么：根因分析。这样 Debugger 才能快速定位问题，而不是在 60 轮预算中浪费时间重复发现同一个 Bug。
  ```