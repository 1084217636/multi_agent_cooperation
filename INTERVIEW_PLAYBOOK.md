# Interview Playbook

这份文档的目标很直接：

- 让你能把这个项目讲清楚
- 让你能回答追问
- 让你遇到场景题时不慌

## 1. 30 秒版本

我做了一个面向 Go 工程场景的研发智能体原型。它的核心不是桌宠 UI，而是把研发辅助链路串起来：先做复杂度评估和模型路由，再结合 AST 和 RAG 做上下文增强，然后进入代码生成或补丁、工程预检、自动修复轮次、Docker 沙盒验证和 Snapshot 留痕。项目用 Go 做主控，LangGraph 通过 HTTP 后端接管 `planning + codegen` 阶段，Redis 负责最近运行状态恢复。

## 2. 1 分钟版本

这个项目定位是“面向 Go 单仓研发场景的 Coding Agent 原型”。我主要解决的是传统 AI 只会给建议、不懂真实工程上下文的问题，所以做了三层能力。第一层是理解层，用复杂度路由、多 Provider、AST 符号扫描和仓内 RAG 给模型真实上下文，减少函数和依赖幻觉。第二层是执行层，支持独立工作区脚手架生成和当前仓库 patch，后面接工程预检、自动修复轮次、Docker 沙盒验证。第三层是留痕和恢复层，用 Snapshot、diff、原始模型输出、结构化报告和 Redis 状态恢复，让一次运行可追踪、可回滚、可排障。LangGraph 这块我没有硬切成默认本地 runtime，而是做成了可切换的阶段编排后端，这样本地 Go 主链能保持轻量，图编排能力也能接进来。

## 3. 3 分钟版本

这个项目一开始我想做成桌宠式数字员工，但后面我把方向收敛到了技术主线，定位成“面向 Go 工程的研发智能体原型”。  

一次任务进来后，先做复杂度评估和模型路由，低复杂度优先低成本模型，在线模型失败时回退到 mock。然后我会扫描当前仓库的 AST，把包、函数、结构体、import 和基础调用关系整理成符号快照，再结合 README、配置、设计文档、历史运行结果做轻量 RAG，把这些上下文一起送给模型。  

如果任务是文档或分析类，就输出结构化计划、Markdown 报告和导出文档；如果任务是 scaffold 或 closed loop，就进入代码生成。代码生成分两种模式，一种是生成独立工作区脚手架，一种是对当前仓库做 patch。当前仓库 patch 加了白名单约束，只允许修改候选文件或同目录安全辅助文件，避免模型越界改仓库。  

生成完之后不会直接结束，而是进入工程预检和自动修复链。预检支持按目标包范围执行，不是每次都全仓 `./...`。自动修复会串 `gofmt`、`goimports`、`go mod tidy`、`go test`、`go vet`，根据失败签名和工作区变更数决定是否继续。之后还会在 Docker 沙盒里做隔离验证，并把日志、退出码和摘要写回报告。  

为了把过程做成可追踪和可恢复，我又加了 Snapshot、`workspace_diff.patch`、raw model response、preflight.json、workflow_summary.json、sandbox.log 这些工件。Redis 会持久化 workflow、screen 和 latest_run，服务重启后还能恢复最近状态。LangGraph 方面，我没有把 Go 主控完全替掉，而是用 HTTP 后端接管 `planning + codegen` 两个阶段，并在报告里写出 `workflow_trace`。  

如果让我总结，这个项目的价值就是：它不是一个只会聊天的工具，而是一个结合真实工程上下文、能生成、能预检、能修复、能验证、还能留痕的 Go 研发代理原型。

## 4. 背诵版项目亮点

这 6 句建议直接背熟：

1. 我这个项目的核心卖点不是 UI，而是把 Go 工程研发辅助链路做成闭环。
2. 为了减少模型幻觉，我用 AST 和 RAG 给模型注入真实项目上下文。
3. 为了平衡成本和效果，我做了复杂度驱动的多 Provider 路由。
4. 为了让生成结果可验证，我接了 preflight、自修复轮次和 Docker 沙盒验证。
5. 为了让过程可追踪、可恢复，我加了 Snapshot、diff、原始模型输出和结构化报告。
6. LangGraph 我没有直接内嵌替换，而是做成可切换的阶段编排后端，接管 `planning + codegen`。

## 5. 高频追问怎么答

### 5.1 为什么用 Go，不直接用 Python + LangGraph

答法：

因为这个项目的核心能力大量依赖本地工程工具链，比如 AST 扫描、工程预检、快照、Docker 动作、HTTP 工作台，这些都更适合直接在 Go 里做主控。Python 和 LangGraph 在复杂图编排上更强，所以我没有排斥，而是把它做成外部阶段编排后端，接管 `planning + codegen`，这样本地仍然保持单进程 Go 主链。

### 5.2 你这个项目怎么减少模型乱编函数

答法：

核心做法有两层。第一层是 AST，把真实包、函数、结构体、import、调用关系提取出来，直接提供给模型。第二层是 RAG，把 README、配置文档、历史运行结果也召回给模型。这样模型不是凭空写，而是参考真实仓内上下文做生成。

### 5.3 为什么还要 Redis

答法：

Redis 在这里不是复杂调度内核，而是轻量状态层。我用它保存 workflow、screen 和 latest_run，让服务重启后还能恢复最近一次运行状态，避免每次都从零开始。

### 5.4 LangGraph 现在是不是主运行时

答法：

不是本地单进程主运行时，而是外部 HTTP 阶段编排后端。当前已经能接管 `planning + codegen` 两个核心节点，但 Go 内置主链仍然保留兜底，这样既能本地轻量运行，也能扩展图编排。

### 5.5 你这个项目和 code review 有什么关系

答法：

它不是专门的 code review 平台，但具备了 code review 的关键基础能力。因为 code review 需要 diff、上下文增强、风险判断、编译测试验证和留痕，而我这个项目已经有 AST、RAG、preflight、Docker 验证、Snapshot、diff 和结构化报告。所以如果要做 code review agent，这个项目就是很好的底座。

## 6. Code Review 场景题答法

### 6.1 标准答法

如果面试官问“设计一个 code review 系统怎么做”，你可以这样答：

我会把 code review 设计成一个阶段化代理，而不是只让模型直接看 PR 文本。第一步先拿 PR diff、改动文件和提交说明，确定 review 范围。第二步做上下文增强，包括 AST 符号、import、调用关系、仓内文档和规范。第三步做风险分析，重点看接口兼容、依赖变化、并发安全、错误处理、测试覆盖和回归风险。第四步对高风险改动做定向验证，比如跑目标包测试、静态检查，必要时放到 Docker 沙盒里隔离验证。最后输出 review findings、风险等级、修改建议和验证日志。  

如果结合我的项目来做，其实我的 AST、RAG、Preflight、Snapshot、Docker Validation、workflow trace 这些能力都已经是一个 code review agent 的底座。

### 6.2 背诵版

直接记这 4 步：

1. 拿 diff，确定 review 范围。
2. 做 AST + 文档检索上下文增强。
3. 做静态风险判断和定向验证。
4. 输出 findings、建议、日志和可追溯工件。

## 7. 哪些话不要说太满

这些不要直接说：

- 我做了完整自动编程机器人
- 我做了完整 LangGraph runtime
- 我已经能全自动修任何 Go 工程问题
- 我这个就是成熟 code review 平台

更稳的说法：

- 我做了研发智能体原型
- 已经完成主链基础能力
- LangGraph 已接入阶段编排
- 可以自然扩展到 code review 场景

## 8. 面试前最后 1 小时复习什么

只看这些：

- [RESUME_PROJECT_ENTRY.md](/home/root1/go_learn/multi_agent_cooperation/RESUME_PROJECT_ENTRY.md)
- [docs/RESUME_ALIGNMENT.md](/home/root1/go_learn/multi_agent_cooperation/docs/RESUME_ALIGNMENT.md)
- 本文档的 “1 分钟版本”
- 本文档的 “高频追问怎么答”
- 本文档的 “Code Review 场景题答法”

