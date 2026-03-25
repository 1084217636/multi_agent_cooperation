# LangGraph 取舍说明

## 为什么这版没有一开始就直接换成 LangGraph

核心原因很现实：这个项目当前主链是 Go，本地能力也几乎都在 Go 里。

- AST / MCP 风格符号扫描是 Go
- 桌面工作台和本地 HTTP 服务是 Go
- Snapshot、预检、Docker 动作、审批执行是 Go

如果一开始就强行把主编排替换成 LangGraph，项目会马上变成双运行时：

- 你要维护 Go 服务
- 还要再维护一套 Python / LangGraph 服务

这样做的问题是：

- 本地启动复杂度会明显上升
- 调试链路会更长
- 对“今天就能跑”的目标不友好

## 什么时候 LangGraph 更好

LangGraph 在下面这些场景会比手写编排更强：

- 需要复杂节点图和动态分支
- 需要更强的 checkpoint / durable execution
- 需要更标准的人机协作节点
- 你的主工程本来就是 Python

## 为什么当前内置 Go 编排更适合这个项目

这版内置编排的优势是：

- 单二进制 / 单运行时，启动成本低
- 和 Go 项目扫描、预检、快照、Docker 执行动作耦合更自然
- 对本地研发工具来说更轻，今天就能直接跑
- 你可以非常明确地控制每一个状态和审批点

## 现在怎么兼容 LangGraph

为了不把路堵死，这版已经加了一个可选的 `langgraph_http` 后端。

你可以在配置里写：

```yaml
workflow:
  backend: "langgraph_http"
  langgraph_endpoint: "http://127.0.0.1:8000/plan"
```

这样项目会把目标、复杂度、符号、屏幕上下文和 RAG 命中发给外部 LangGraph 服务，由外部图编排返回结构化方案。

## 结论

当前默认不直接替换成 LangGraph，不是因为它不好，而是因为：

- 对这个 Go 项目的今天版 MVP 来说，代价大于收益
- 内置 Go 编排更容易把工作台、RAG、MCP、预检、快照和审批执行先跑通

更合理的路线是：

1. 默认继续用内置 Go 编排
2. 有成熟 Python agent graph 时，再切到 `langgraph_http`
