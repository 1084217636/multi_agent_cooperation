# Learning Map

这份文档是给“先把项目搞通弄透，再补简历差距”的学习路线图。

建议你不要一上来全仓乱看，按下面顺序过。

## 0. 先知道项目现在是什么

这个项目现在最准确的定位是：

`面向 Go 工程的研发智能体原型`

它的主链不是做桌宠 UI，而是做下面这条技术链：

`任务理解 -> 路由 -> 上下文增强 -> 代码生成/补丁 -> 预检 -> 自动修复 -> Docker 验证 -> 快照留痕 -> 报告导出`

先读：

- [README.md](/home/root1/go_learn/multi_agent_cooperation/README.md)
- [PROJECT_TEACHING_GUIDE.md](/home/root1/go_learn/multi_agent_cooperation/PROJECT_TEACHING_GUIDE.md)
- [TERM_GLOSSARY.md](/home/root1/go_learn/multi_agent_cooperation/TERM_GLOSSARY.md)

## 1. 启动与配置

目标：先搞清楚项目怎么跑起来，配置从哪里进。

看这些文件：

- [cmd/main.go](/home/root1/go_learn/multi_agent_cooperation/cmd/main.go)
- [companion/config.go](/home/root1/go_learn/multi_agent_cooperation/companion/config.go)
- [config.yaml](/home/root1/go_learn/multi_agent_cooperation/config.yaml)

你要学会回答：

- 项目有哪些启动模式
- 配置默认值从哪里来
- `.env.local` 和 `config.yaml` 谁负责模型密钥

## 2. Provider 与模型路由

目标：理解为什么要多 Provider，怎么做复杂度驱动路由。

看这些文件：

- [companion/router.go](/home/root1/go_learn/multi_agent_cooperation/companion/router.go)
- [llm/provider.go](/home/root1/go_learn/multi_agent_cooperation/llm/provider.go)
- [llm/providers/groq.go](/home/root1/go_learn/multi_agent_cooperation/llm/providers/groq.go)
- [LLM_PROVIDER_GUIDE.md](/home/root1/go_learn/multi_agent_cooperation/LLM_PROVIDER_GUIDE.md)

你要学会回答：

- 复杂度评估用什么信号
- 为什么低复杂度优先低成本模型
- 模型失败后怎么回退

## 3. 上下文增强：AST / MCP 风格 / RAG

目标：理解模型为什么不会完全“瞎编”。

看这些文件：

- [mcp/inspector.go](/home/root1/go_learn/multi_agent_cooperation/mcp/inspector.go)
- [mcp/inspector_test.go](/home/root1/go_learn/multi_agent_cooperation/mcp/inspector_test.go)
- [rag/index.go](/home/root1/go_learn/multi_agent_cooperation/rag/index.go)
- [companion/service.go](/home/root1/go_learn/multi_agent_cooperation/companion/service.go)

你要学会回答：

- AST 提取了哪些结构信息
- 现在是不是标准 MCP
- RAG 索引了哪些仓内内容

## 4. 执行主链

目标：理解一次任务从输入到落盘完整怎么走。

重点看：

- [companion/service.go](/home/root1/go_learn/multi_agent_cooperation/companion/service.go)

重点函数：

- `Execute`
- `generatePlan`
- `persistRun`
- `persistRunArtifacts`

你要学会回答：

- 一次任务有哪些阶段
- 哪些阶段是分析，哪些阶段是真执行
- 报告、导出文档、JSON 分别在哪写出

## 5. 代码生成与当前仓库补丁

目标：理解“独立工作区生成”和“当前仓库 patch”有什么区别。

看这些文件：

- [companion/codegen.go](/home/root1/go_learn/multi_agent_cooperation/companion/codegen.go)
- [companion/validation_scope.go](/home/root1/go_learn/multi_agent_cooperation/companion/validation_scope.go)
- [companion/validation_scope_test.go](/home/root1/go_learn/multi_agent_cooperation/companion/validation_scope_test.go)

重点概念：

- `isolated_workspace`
- `current_repo_patch`
- `patch_candidates`
- `raw_response_path`
- `parse_mode`
- `rejected_files`

你要学会回答：

- 为什么 current repo patch 要有限制
- 为什么要保留原始模型输出
- 为什么需要 parse fallback

## 6. 工程预检、自修复轮次与终止策略

目标：理解 Go 工程约束是怎么接进闭环的。

看这些文件：

- [preflight/health.go](/home/root1/go_learn/multi_agent_cooperation/preflight/health.go)
- [companion/auto_loop.go](/home/root1/go_learn/multi_agent_cooperation/companion/auto_loop.go)
- [companion/actions_advanced.go](/home/root1/go_learn/multi_agent_cooperation/companion/actions_advanced.go)

你要学会回答：

- 预检检查了哪些东西
- 自动修复会跑哪些命令
- 什么情况下提前终止
- 现在是不是已经做到了“模型改代码后自修复”

## 7. Docker 沙盒验证

目标：理解为什么要隔离验证，以及当前做到什么程度。

看这些文件：

- [executor/docker_run.go](/home/root1/go_learn/multi_agent_cooperation/executor/docker_run.go)
- [companion/auto_loop.go](/home/root1/go_learn/multi_agent_cooperation/companion/auto_loop.go)

你要学会回答：

- 为什么不是直接在宿主机跑
- 现在是全仓验证还是局部包验证
- 验证日志落到哪里

## 8. Snapshot、diff、回滚

目标：理解为什么这个项目强调“留痕”和“可恢复”。

看这些文件：

- [snapshot/manifest.go](/home/root1/go_learn/multi_agent_cooperation/snapshot/manifest.go)
- [companion/diff_artifact.go](/home/root1/go_learn/multi_agent_cooperation/companion/diff_artifact.go)
- [companion/service.go](/home/root1/go_learn/multi_agent_cooperation/companion/service.go)

你要学会回答：

- 运行前后快照分别干什么
- `workspace_diff.patch` 和 snapshot 的区别
- 回滚动作怎么触发

## 9. LangGraph 与 Redis

目标：分清“已经做了什么”和“还没完全做到什么”。

看这些文件：

- [companion/langgraph_bridge.go](/home/root1/go_learn/multi_agent_cooperation/companion/langgraph_bridge.go)
- [companion/redis_store.go](/home/root1/go_learn/multi_agent_cooperation/companion/redis_store.go)
- [companion/enhancements.go](/home/root1/go_learn/multi_agent_cooperation/companion/enhancements.go)
- [docs/LANGGRAPH_DECISION.md](/home/root1/go_learn/multi_agent_cooperation/docs/LANGGRAPH_DECISION.md)

你要学会回答：

- LangGraph 现在能接管哪些节点
- Redis 现在保存了哪些状态
- 服务重启后能恢复哪些上下文

## 10. 前端工作台只看必要部分

目标：理解它怎么展示状态，不要把时间都花在 UI 上。

看这些文件：

- [companion/dashboard.go](/home/root1/go_learn/multi_agent_cooperation/companion/dashboard.go)

只需要重点理解：

- 它怎么调用 `/api/run`
- 它怎么展示 `latest_run`
- 它怎么展示 `workflow / preflight / artifacts`

## 学习完成后的自检问题

如果你把上面 10 步走完，至少应该能回答这些问题：

- 这个项目的核心卖点到底是什么
- 多 Provider 路由和 AST / RAG 是怎么串起来的
- 为什么要分 `isolated_workspace` 和 `current_repo_patch`
- Snapshot 和 `workspace_diff.patch` 分别解决什么问题
- 现在和简历差距最大的点还剩什么

## 配套测试文档

读完学习地图后，直接去按功能验收：

- [docs/TESTING_GUIDE.md](/home/root1/go_learn/multi_agent_cooperation/docs/TESTING_GUIDE.md)
