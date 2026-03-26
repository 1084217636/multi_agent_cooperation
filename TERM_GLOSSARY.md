# Term Glossary

这份术语表是给“看项目、看面试题、看我回答时，经常被英文术语卡住”的情况准备的。

读法建议：

- 先看你不懂的词
- 再看“这个项目里具体指什么”
- 最后回到代码和文档里对照

## 1. Code Review / Git 相关

| 术语 | 中文意思 | 这个项目里具体指什么 |
| --- | --- | --- |
| Code Review | 代码评审 | 审查代码改动有没有 bug、风险、风格问题、可维护性问题。你这个项目的 AST、RAG、预检、Docker 验证，其实都能服务于 code review。 |
| PR | Pull Request | 提交代码变更后，请团队审核并合并的一次请求。GitHub/GitLab 常见。 |
| PR Diff | PR 的代码差异 | 一次 PR 里哪些行被加了、删了、改了。做 code review 时最核心的输入之一。 |
| PR Review | PR 评审 | 围绕一个 PR 给出评论、风险判断、修改建议、是否通过。 |
| Commit | 提交 | Git 里一次代码快照。 |
| Branch | 分支 | Git 里隔离开发的一条线。 |
| Merge | 合并 | 把一个分支的改动并到另一个分支。 |
| Rollback | 回滚 | 把改动恢复到之前状态。你项目里的 Snapshot 和 before/after diff 就是在支撑回滚。 |
| Patch | 补丁 | 一组针对现有文件的修改。你项目里的 `current_repo_patch` 就是“对当前仓库打补丁”。 |
| Diff | 差异 | 两个版本之间的不同。你项目里的 `workspace_diff.patch` 就是差异文件。 |

## 2. Agent / 大模型相关

| 术语 | 中文意思 | 这个项目里具体指什么 |
| --- | --- | --- |
| Agent | 智能体 | 会接收任务、做判断、调用工具、生成结果的一套执行逻辑。 |
| Coding Agent | 编码智能体 | 面向代码和工程任务的 Agent。你这个项目本质上就是一个面向 Go 工程的 Coding Agent 原型。 |
| Prompt | 提示词 | 发给模型的指令文本。 |
| System Prompt | 系统提示词 | 给模型设定角色、边界和输出格式的提示词。 |
| User Prompt | 用户提示词 | 面向当前任务的具体输入。 |
| Context | 上下文 | 模型回答前参考的信息。你项目里的上下文包括 AST、RAG、屏幕信息、任务描述。 |
| LLM | 大语言模型 | 比如 Groq 后面的模型、OpenAI、Ollama 里的模型。 |
| Provider | 模型供应方 | 比如 `groq`、`openai`、`ollama`、`siliconflow`。 |
| Routing | 路由 | 根据任务复杂度、模型状态选择哪个 Provider。 |
| Fallback | 回退 | 主要模型失败后改用备用路径，比如回退到 `mock`。 |
| Retry | 重试 | 第一次失败后再试一次。 |
| Mock | 模拟实现 | 不依赖真实在线模型的兜底实现。你项目没 API 额度时也能跑，靠的就是 mock。 |
| Hallucination | 幻觉 | 模型编造不存在的函数、类型、依赖。你项目用 AST / RAG 就是在减少这个问题。 |
| Codegen | 代码生成 | 让模型输出文件内容、项目骨架或补丁。 |
| Scaffold | 脚手架 | 一套最小可运行项目骨架。你项目里 `scaffold` 模式会生成独立工作区代码包。 |
| Closed Loop | 闭环 | 不只是生成建议，还要继续预检、修复、验证、留痕。 |
| Workflow | 工作流 | 一次任务从输入到完成的阶段链。 |
| Node | 节点 | 工作流里的一个阶段，比如 `planning`、`codegen`、`preflight`。 |
| Trace | 轨迹 | 记录每个节点执行了什么、成功还是失败。你项目里的 `workflow_trace` 就是这个。 |
| Bridge | 桥接层 | 用来连接外部系统的适配层。你项目的 LangGraph 不是内嵌，而是 HTTP bridge。 |
| Endpoint | 接口地址 | 比如 `http://127.0.0.1:8000/{operation}`。 |
| Durable Execution | 持久执行 | 任务中断后还能恢复继续。LangGraph 常强调这个能力。 |
| Checkpoint | 检查点 | 长任务执行过程中的中间保存点。 |

## 3. 代码工程相关

| 术语 | 中文意思 | 这个项目里具体指什么 |
| --- | --- | --- |
| AST | 抽象语法树 | 代码的语法结构树。你项目用它提取包、函数、结构体、import、调用关系。 |
| MCP | Model Context Protocol | 一种把工具和上下文暴露给模型的协议。你项目现在更准确地说是 “MCP 风格上下文注入”。 |
| RAG | 检索增强生成 | 先检索文档，再把结果喂给模型。 |
| Symbol | 符号 | 代码里的包、函数、结构体、字段、方法等。 |
| Import | 导包 | Go 文件里 `import` 的包。 |
| Call Graph | 调用图 | 谁调用了谁的关系图。 |
| Call Edge | 调用边 | 调用图里的一条边，比如 `A -> B`。 |
| Preflight | 预检 | 执行正式闭环前先做工程检查。你项目会跑 `go test`、`go vet` 和工具检查。 |
| Lint / Linter | 静态风格和质量检查 / 检查器 | 比如 `golangci-lint`。 |
| gofmt | Go 官方格式化工具 | 自动把 Go 代码格式化。 |
| goimports | Go 导包整理工具 | 自动补 import、删无用 import。 |
| go vet | Go 静态检查工具 | 用来发现可疑代码问题。 |
| Unit Test | 单元测试 | 验证某个函数或模块行为的小测试。 |
| Smoke Test | 烟雾测试 | 先做一轮最基本的可用性验证。 |
| Sandbox | 沙盒 | 隔离的执行环境。你项目里主要就是 Docker 容器。 |
| Snapshot | 快照 | 某一时刻工作区文件状态的记录。 |
| Artifact | 产物 | 运行后落盘的文件，比如报告、JSON、patch、log。 |
| Raw Response | 原始响应 | 大模型原始输出，方便后续排查为什么解析失败。 |
| Validation Scope | 验证范围 | 这次只验证哪些包，不一定全仓 `./...`。 |
| Regression | 回归问题 | 改一个地方，把原来能用的东西搞坏了。 |

## 4. 你这个项目里最常出现、最该先记住的词

如果你现在先只记住 12 个词，优先记这些：

- Code Review
- PR Diff
- Prompt
- Context
- Provider
- Routing
- AST
- RAG
- Preflight
- Snapshot
- Patch
- Sandbox

## 5. 一句最通俗的理解

把这套项目先想成：

`一个会看代码、会查文档、会选模型、会做预检、会留痕、还能尝试修复的 Go 工程助手`

你后面再看到上面这些英文词，基本都能挂回这条主线。

