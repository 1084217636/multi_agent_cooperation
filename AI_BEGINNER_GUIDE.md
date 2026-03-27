# AI Beginner Guide

这份文档是专门写给“AI 小白”的。

目标不是让你一次学会全部，而是让你先建立一个不会乱的脑图：

- 这个技术是什么
- 它在这个项目里干什么
- 哪些要理解
- 哪些要背

## 1. 先建立总脑图

你先把整个项目记成一句话：

`一个会看代码、会查文档、会选模型、会做预检、会留痕、还能尝试修复的 Go 工程助手`

然后再把项目拆成 8 块：

1. Go 主控
2. 模型路由
3. AST 上下文增强
4. RAG 文档检索
5. 代码生成 / patch
6. 工程预检与自修复
7. Docker 沙盒验证
8. Snapshot / Redis / LangGraph 留痕与恢复

## 2. 每个技术到底是什么

### 2.1 Go

是什么：

- 一门后端和工程工具很强的语言

在项目里干什么：

- 起 HTTP 服务
- 跑主流程
- 扫 AST
- 做 preflight
- 做 Snapshot
- 调 Docker

你要理解：

- Go 适合做工程工具和本地服务

你要背：

- “我用 Go 做主控，是因为本地工程扫描、预检、快照和工具链整合更自然。”

### 2.2 LLM

是什么：

- 大语言模型

在项目里干什么：

- 负责理解任务
- 生成计划
- 生成代码包或补丁

你要理解：

- 模型本身不懂你的项目，必须给它上下文

你要背：

- “模型负责生成，但上下文增强和验证链负责把结果拉回真实工程。”

### 2.3 Provider

是什么：

- 模型供应方，比如 Groq、OpenAI、Ollama

在项目里干什么：

- 给项目提供不同模型

你要理解：

- 不同 Provider 的成本、速度、能力不同

你要背：

- “我做了多 Provider 路由，按复杂度选择模型，在在线模型失败时回退到 mock。”

### 2.4 Routing

是什么：

- 路由就是“这次任务该走哪个模型”

在项目里干什么：

- 根据复杂度、关键词和模型状态选 provider

你要理解：

- 不是所有任务都需要最强模型

你要背：

- “复杂度低时优先低成本模型，复杂度高或失败时再升级高能力模型。”

### 2.5 AST

是什么：

- 抽象语法树，代码的结构化表示

在项目里干什么：

- 提取包、函数、结构体、import、调用关系

你要理解：

- AST 比直接把整份代码塞给模型更适合做结构摘要

你要背：

- “我用 AST 提取真实符号和依赖关系，减少模型编造函数和类型。”

### 2.6 MCP

是什么：

- Model Context Protocol，把工具和上下文暴露给模型的一种协议思路

在项目里干什么：

- 现在不是完整标准 MCP，而是 MCP 风格上下文注入

你要理解：

- 你这里重点不是协议实现，而是“把本地真实上下文暴露给模型”

你要背：

- “当前项目是 MCP 风格上下文暴露，不是完整标准协议实现。”

### 2.7 RAG

是什么：

- Retrieval-Augmented Generation，检索增强生成

在项目里干什么：

- 从 README、docs、配置、历史报告里先检索相关内容，再喂给模型

你要理解：

- 模型回答前先查资料

你要背：

- “我用 RAG 把仓内文档和历史运行结果作为生成上下文，减少拍脑袋输出。”

### 2.8 Codegen

是什么：

- 代码生成

在项目里干什么：

- 生成独立工作区代码包
- 或者对当前仓库生成 patch

你要理解：

- `isolated_workspace` 更安全
- `current_repo_patch` 更接近真实工程修改

你要背：

- “代码生成分独立工作区和当前仓库 patch 两种模式，后者有限制白名单避免越界修改。”

### 2.9 Preflight

是什么：

- 正式执行前先做工程检查

在项目里干什么：

- 跑 `go test`、`go vet`
- 检查 `goimports`、`golangci-lint`

你要理解：

- 不是生成完就结束，要先验证工程基本健康

你要背：

- “我把 Go 工程约束前置到 preflight 阶段，避免模型生成结果直接污染主流程。”

### 2.10 Auto Repair

是什么：

- 自动修复轮次

在项目里干什么：

- 跑 `gofmt`
- 跑 `goimports`
- 跑 `go mod tidy`
- 重新 `go test` / `go vet`

你要理解：

- 当前以命令级修复为主，不是模型重写全部代码

你要背：

- “自动修复重点先放在 Go 工程的确定性纠偏，比如格式化、导包整理、依赖收敛和预检复跑。”

### 2.11 Docker

是什么：

- 容器隔离环境

在项目里干什么：

- 在容器里跑验证，不直接依赖宿主机状态

你要理解：

- Docker 这里的用途是隔离验证，不是部署

你要背：

- “我把 Docker 用在沙盒验证链，而不是部署链。”

### 2.12 Snapshot

是什么：

- 某一时刻工作区文件状态的快照

在项目里干什么：

- 记录运行前后文件状态
- 支持 diff 和回滚

你要理解：

- Snapshot 解决的是“出问题后怎么恢复”

你要背：

- “我用 Snapshot 记录运行前后状态，支撑回滚和结构化排障。”

### 2.13 Redis

是什么：

- 键值存储

在项目里干什么：

- 保存 workflow、screen、latest_run

你要理解：

- 它现在是状态层，不是复杂调度中心

你要背：

- “Redis 在当前项目里主要承担最近运行状态恢复，不是完整调度内核。”

### 2.14 LangGraph

是什么：

- 一个更适合做阶段图、多节点编排的 Agent 工作流框架

在项目里干什么：

- 通过 HTTP 后端接管 `planning + codegen`

你要理解：

- 它现在是外部图后端，不是本地默认 runtime

你要背：

- “当前项目是 Go 主控 + LangGraph 可切换阶段编排，LangGraph 已能接管 planning 和 codegen。”

## 3. 哪些内容必须背，哪些只要理解

### 必须背

- 项目定位
- 复杂度路由
- AST + RAG 作用
- preflight + auto repair
- Docker 沙盒验证
- Snapshot / diff / rollback
- Redis 恢复
- LangGraph 的真实边界

### 只要理解

- 具体每个函数的实现细节
- 每个 JSON 字段完整结构
- 所有前端页面细节

## 4. 面试最容易卡住的点

### 卡点 1：把项目说成“全自动编程”

正确说法：

- 这是研发智能体原型
- 已经完成主链基础能力
- 不是完整成熟自动编程系统

### 卡点 2：把 LangGraph 说成默认运行时

正确说法：

- 当前是外部 HTTP 阶段编排
- Go 主链仍保留兜底

### 卡点 3：把 Redis 说成调度中枢

正确说法：

- 当前 Redis 主要做状态恢复

## 5. 你现在该怎么学

先按这个顺序：

1. [TERM_GLOSSARY.md](/home/root1/go_learn/multi_agent_cooperation/TERM_GLOSSARY.md)
2. [PROJECT_TEACHING_GUIDE.md](/home/root1/go_learn/multi_agent_cooperation/PROJECT_TEACHING_GUIDE.md)
3. [LEARNING_MAP.md](/home/root1/go_learn/multi_agent_cooperation/LEARNING_MAP.md)
4. [INTERVIEW_PLAYBOOK.md](/home/root1/go_learn/multi_agent_cooperation/INTERVIEW_PLAYBOOK.md)
5. [docs/TESTING_GUIDE.md](/home/root1/go_learn/multi_agent_cooperation/docs/TESTING_GUIDE.md)

## 6. 一句最后的记忆法

如果你一紧张就忘，先背这一句：

`我这个项目是一个面向 Go 工程的研发智能体原型，核心是复杂度路由、AST+RAG 上下文增强、预检修复、Docker 验证和 Snapshot 留痕。`

