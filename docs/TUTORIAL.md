# 开发教学文档

这份文档帮助你继续把项目往“面向 Go 工程的研发智能体”推进。

## 1. 当前代码结构

- `cmd/`：主入口，支持 `desktop` / `run` / `inspect`
- `companion/`：新核心，负责配置、路由、服务编排和本地工作台
- `rag/`：轻量本地检索器
- `mcp/`：AST 静态分析与符号快照
- `preflight/`：工程预检
- `llm/providers/`：Provider 实现，包括离线 `mock`
- `executor/`：Docker 隔离执行器

## 2. 一次任务是怎么跑完的

1. `cmd/main.go` 读取配置并启动服务
2. `companion.Service` 初始化 Provider、符号快照、RAG 索引
3. 输入目标后，`router.go` 评估复杂度并做阶梯式路由
4. `service.go` 组装符号快照和 RAG 命中，调用 Provider
5. 返回结构化计划，并跑 `preflight` 输出工程健康摘要
6. 报告落盘到 `data/reports` 和 `data/runs`

## 3. 如果你要继续增强 RAG

第一步：

- 扩大 `knowledge.include_paths`
- 把 `docs/`、设计稿、接口文档都纳入

第二步：

- 给 `rag/index.go` 增加更好的分词和打分
- 增加 chunk 元信息，比如标签、模块名、更新时间

第三步：

- 如果后续有 API 额度，再接入 embedding + 向量库

## 4. 如果你要继续增强路由

当前实现是轻量级规则路由。后续可以加：

- 成本预算
- 成功率历史
- 失败次数与自动提权
- 针对代码任务与文档任务的不同模型策略

入口文件：

- [router.go](/home/root1/go_learn/multi_agent_cooperation/companion/router.go)
- [service.go](/home/root1/go_learn/multi_agent_cooperation/companion/service.go)

## 5. 如果你要继续增强研发闭环

当前更值得补的是执行主链，而不是角色化外壳。后续建议：

1. 增加代码写回和文件落盘能力
2. 增加 `goimports`、`golangci-lint` 与失败重试
3. 增加任务状态机、重试阈值与熔断
4. 增加更稳定的 IDE / 终端上下文接入

## 6. 如果你要接真实 API

先配置环境变量：

```bash
export GROQ_API_KEY=...
```

再运行：

```bash
GOTOOLCHAIN=local go run ./cmd -mode inspect
```

确认 Provider 变成 `ready` 后，再启动 `desktop` 模式。

## 7. 推荐迭代顺序

1. 先把文档和配置维护好
2. 再升级 RAG 和路由
3. 再引入代码执行审批
4. 最后再决定是否需要托盘、通知或语音入口
