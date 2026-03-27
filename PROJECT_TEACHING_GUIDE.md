# 项目教学文档：从零理解这个 Go 研发智能体

这份文档面向两个目标：

1. 让你搞清楚这个项目现在到底在做什么
2. 让你把这里涉及的技术栈基础八股补到能讲、能答、能继续改代码

如果你现在对大模型框架、RAG、LangGraph、Redis 这些概念还不熟，建议按下面顺序读：

1. 先看“项目总览”
2. 再看“执行主链”
3. 再看“每个技术栈是什么”
4. 最后看“当前项目还缺什么”

如果你经常被英文术语卡住，先配合看：

- [TERM_GLOSSARY.md](/home/root1/go_learn/multi_agent_cooperation/TERM_GLOSSARY.md)
- 如果你是 AI 小白，先看 [AI_BEGINNER_GUIDE.md](/home/root1/go_learn/multi_agent_cooperation/AI_BEGINNER_GUIDE.md)
- 如果你要准备面试，配合看 [INTERVIEW_PLAYBOOK.md](/home/root1/go_learn/multi_agent_cooperation/INTERVIEW_PLAYBOOK.md)

---

## 1. 项目总览

这个项目当前不是完整的“自动写完整项目代码”的 Agent 产品，而是一个：

`面向 Go 工程的研发智能体工作台`

它的核心目标是：

- 接收一个研发任务
- 收集代码和文档上下文
- 选择合适的大模型
- 生成结构化方案和交付文档
- 对工程做预检
- 记录快照、回滚建议和排障信息

它最像什么？

- 一个本地研发辅助中台
- 一个 Go 工程上下文增强器
- 一个“分析 + 预检 + 导出 + 修复动作”的闭环原型

它不像什么？

- 不是完整 IDE Copilot
- 不是成熟的自动编程机器人
- 不是完整 LangGraph 工作流平台
- 不是角色化产品本身

---

## 2. 代码结构怎么读

建议从这几个文件开始：

- [cmd/main.go](/home/root1/go_learn/multi_agent_cooperation/cmd/main.go)
  入口。决定用 `desktop`、`run`、`inspect` 哪种模式启动。

- [companion/service.go](/home/root1/go_learn/multi_agent_cooperation/companion/service.go)
  核心服务。任务执行主链基本都在这里。

- [companion/router.go](/home/root1/go_learn/multi_agent_cooperation/companion/router.go)
  复杂度评估和 Provider 路由逻辑。

- [mcp/inspector.go](/home/root1/go_learn/multi_agent_cooperation/mcp/inspector.go)
  Go AST 扫描器。负责把真实函数、结构体、包信息提取出来。

- [preflight/health.go](/home/root1/go_learn/multi_agent_cooperation/preflight/health.go)
  工程预检。负责 `go test`、`go vet` 和工具可用性检查。

- [snapshot/manifest.go](/home/root1/go_learn/multi_agent_cooperation/snapshot/manifest.go)
  快照和恢复机制。

- [companion/actions_advanced.go](/home/root1/go_learn/multi_agent_cooperation/companion/actions_advanced.go)
  Docker 自愈、自动修复、自动提交等动作。

- [companion/redis_store.go](/home/root1/go_learn/multi_agent_cooperation/companion/redis_store.go)
  Redis 状态存储。

- [companion/langgraph_bridge.go](/home/root1/go_learn/multi_agent_cooperation/companion/langgraph_bridge.go)
  LangGraph HTTP bridge。注意这里只是 bridge，不是主运行时。

---

## 3. 执行主链怎么走

当前一次任务的大致流程是：

```text
用户输入任务
  -> 复杂度评估
  -> Provider 路由
  -> 扫描 Go AST 符号
  -> 检索 RAG 上下文
  -> 可选收集屏幕上下文
  -> 生成结构化计划
  -> 工程预检
  -> 写入 Markdown / JSON / 导出文档
  -> 生成快照与排障报告
  -> 必要时再人工触发修复动作
```

你可以把它理解成两层：

### 第一层：分析层

- 理解任务
- 理解代码
- 理解文档
- 理解工程状态

### 第二层：执行层

- 导出报告
- 预检工程
- 生成回滚点
- 提供修复动作

注意：

当前“执行层”还不是完全自动的，它还是偏“辅助执行”，而不是“完全自主执行”。

---

## 4. 这个项目里每个技术栈到底是什么

下面按“技术八股”的角度来讲。

### 4.1 Go

Go 是这个项目的主语言。

你需要知道的基础点：

- Go 编译快，适合做 CLI、本地服务、工具链
- 标准库强，HTTP、文件系统、并发支持很好
- 适合做工程工具、扫描器、执行器、预检器

在这个项目里，Go 主要负责：

- 本地 HTTP 工作台
- 任务主链编排
- AST 静态分析
- 工程预检
- 快照和回滚
- Docker 动作调用

面试常见问法：

- 为什么用 Go 做 Agent，而不是 Python？
- Go 在工具链和本地工程场景里有什么优势？
- Go 的并发和标准库在这个项目里用到了哪里？

### 4.2 LLM / 多 Provider 路由

LLM 就是大语言模型。

多 Provider 的意义：

- 不同模型能力不同
- 成本不同
- 响应速度不同
- 可用性不同

这个项目里做了一个复杂度驱动的路由：

- 简单任务优先轻量模型
- 复杂任务优先高能力模型
- 在线模型失败时回退到 mock

为什么这很重要？

- 可以平衡成本和效果
- 不会因为一个 Provider 出问题让整条链路不可用

你要能讲清楚：

- 为什么要做复杂度分发
- 路由规则按什么维度设计
- fallback 和 retry 的边界是什么

### 4.3 AST 静态分析

AST 是抽象语法树。

为什么要扫 AST？

- 因为大模型最容易“编函数、编类型、编字段”
- 你把真实的符号信息提前给它，它会更少胡说

这个项目里 AST 扫描会拿到：

- 包名
- 函数名
- 结构体名
- 字段信息
- 签名信息

你要理解：

- AST 是对代码结构的语法层表示
- 它比直接喂原始代码更适合做结构摘要
- 它是“上下文增强”的关键来源之一

### 4.4 MCP

严格说，这个项目现在不是完整 MCP 协议实现。

更准确的说法应该是：

`MCP 风格的上下文暴露`

什么意思？

- 把本地代码结构暴露成模型可读的上下文
- 让模型像“调用一个本地上下文服务”一样使用真实工程信息

如果你面试被问到：

- 现在这个项目是不是标准 MCP？

你应该回答：

- 现在是 MCP 风格的符号暴露和上下文注入
- 还不是完整标准协议 server/client 互通实现

### 4.5 RAG

RAG 是 Retrieval-Augmented Generation，检索增强生成。

核心思想：

- 先检索相关文档
- 再让模型生成答案

这个项目里检索的对象主要是：

- README
- 配置文档
- 教学文档
- 设计说明
- 历史运行产物

为什么需要它？

- 让模型知道“这个项目以前怎么设计的”
- 让回答更贴近真实代码仓库
- 减少拍脑袋输出

RAG 最基本的三步：

1. 切分文档
2. 建索引
3. 按 query 检索后拼到 prompt

### 4.6 Docker

Docker 在这里不是做部署，而是做隔离执行。

意义：

- 避免宿主机环境差异太大
- 在容器里跑测试、格式化、自愈动作
- 减少“我机器上能跑，你机器上不行”

当前项目里 Docker 主要用于：

- 容器内执行 `go test`
- 容器内执行 `go mod tidy`
- 容器内执行 `gofmt`
- 容器内执行 `go vet`

注意：

现在是 Docker CLI 调用，不是 Docker API 深度封装。

### 4.7 Redis

Redis 在这个项目里现在主要做：

- 工作流状态存储
- 屏幕上下文状态存储
- 最近一次运行结果存储

你要知道 Redis 的基础点：

- 它是内存型 KV 存储
- 速度快
- 适合状态、缓存、计数器、轻任务状态

但现在这个项目还没做到：

- 把 Redis 作为真正跨周期任务调度中心
- 用 Redis 做重试队列、熔断计数和恢复点管理

### 4.8 LangGraph

LangGraph 是一种更适合做工作流编排的 Agent 框架。

你可以把它理解成：

- 一个面向“多步骤、多状态、多分支”的 Agent 工作流工具

这个项目里现在的状态是：

- 已经留了 bridge
- 可以通过 HTTP 接一个外部 LangGraph 服务
- 但默认主链还是 Go 内置编排

所以你讲项目时要说：

- 当前项目“兼容 LangGraph HTTP bridge”
- 不是“主编排完全基于 LangGraph”

### 4.9 Snapshot

Snapshot 就是工作区快照。

它的作用：

- 记录任务执行前后文件状态
- 对比差异
- 出问题时回滚

当前项目里 Snapshot 做的事情：

- 记录 before/after
- 算 diff
- 支持 restore

它是后续做重试和熔断的基础设施。

### 4.10 Mock

Mock 在这个项目里不是“业务模拟器”，而是：

- 当真实模型不可用时的兜底链路

为什么重要？

- 让项目没有 API 额度时也能演示
- 让路由、RAG、快照、预检这些链路能持续联调

但你要注意：

- Mock 只能证明链路通
- 不能证明真实生成质量

---

## 5. 你现在最应该真正搞懂的 5 个问题

如果你想把项目“弄通弄透”，先把这 5 个问题讲明白：

### 5.1 为什么要做复杂度路由？

因为不是所有任务都值得直接上最贵最强模型。

核心权衡：

- 成本
- 时延
- 成功率

### 5.2 为什么 AST 比直接喂代码更重要？

因为模型最容易错在“结构性幻觉”。

AST 给的是：

- 真函数
- 真结构体
- 真签名

这对代码类任务比“只给 README”更有效。

### 5.3 为什么 RAG 不能代替 AST？

因为：

- RAG 更偏文档和语义背景
- AST 更偏真实代码结构

两者解决的问题不同。

### 5.4 为什么 Snapshot 是闭环基础设施？

因为你一旦开始改代码，就必须考虑：

- 改坏了怎么办
- 怎么重试
- 怎么恢复

没有快照，所谓“闭环”就是不可靠的。

### 5.5 为什么当前项目还不算完整 Coding Agent？

因为真正的 Coding Agent 至少要做到：

- 改代码
- 跑测试
- 自动修复
- 再次验证
- 持续迭代直到满足目标或失败退出

而你现在的项目更偏：

- 分析
- 导出
- 预检
- 手动触发修复

---

## 6. 当前项目最真实的优点和问题

### 优点

- 技术链比较完整
- Go 工程上下文增强做得是成立的
- 路由、RAG、AST、预检、快照之间已经串起来了
- 已经具备“继续往强执行闭环演化”的基础

### 问题

- 产品定位容易发散
- 代码分析比代码执行更强
- LangGraph / Redis / Docker 还没有真正变成“主链核心”
- UI 包装和形态叙事曾经占了太多精力

---

## 7. 你接下来应该优先补什么

如果你现在只关注技术，不再花太多时间在 UI 上，优先级应该是：

1. 代码写回主链
2. 自修复主链
3. 重试 / 熔断 / 恢复状态机
4. LangGraph 是否真的要接入主编排
5. Redis 是否真正承担跨周期任务状态

更具体一点：

### 第一优先级

- 让任务不只是“生成方案”，而是“生成文件或修改文件”

### 第二优先级

- 让失败后能自动跑 `gofmt`、`go mod tidy`、`go test`、`go vet`

### 第三优先级

- 给任务加 retry counter、熔断、恢复点

---

## 8. 这份项目你面试时应该怎么讲

一句话版本：

我做了一个面向 Go 工程的研发智能体工作台，通过复杂度路由、AST 符号增强、轻量 RAG、工程预检和快照回滚，把任务分析和研发辅助闭环串在一起。

不要先讲：

- 角色化外壳
- 球体 UI
- 角色化交互

先讲：

- 路由
- AST
- RAG
- 预检
- 快照
- 自愈动作

因为这些才是技术核心。

---

## 9. 最后给你的建议

你现在最该做的，不是继续堆 UI，而是把项目彻底收敛成：

`Go 研发智能体内核 + 一个够用的工作台`

这样你后续无论写简历、面试讲项目，还是继续做功能，都不会跑偏。
