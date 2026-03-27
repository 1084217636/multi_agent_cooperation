# 14-Day Study Plan

这份计划默认你每天能投入 2 到 4 小时。

目标不是把所有代码都背下来，而是让你在 1 到 2 周内达到：

- 能讲清项目定位
- 能讲清核心技术
- 能回答常见追问
- 能扛住一轮基于项目的面试追问

## 学习原则

每天都做 4 件事：

1. 读文档
2. 看代码
3. 跑命令
4. 复述和背诵

## Day 1：搞清项目到底是什么

读：

- [README.md](/home/root1/go_learn/multi_agent_cooperation/README.md)
- [TERM_GLOSSARY.md](/home/root1/go_learn/multi_agent_cooperation/TERM_GLOSSARY.md)
- [AI_BEGINNER_GUIDE.md](/home/root1/go_learn/multi_agent_cooperation/AI_BEGINNER_GUIDE.md)

做：

```bash
GOTOOLCHAIN=local go test ./...
GOTOOLCHAIN=local go run ./cmd -mode inspect
```

今天必须背：

- 项目定位一句话
- 复杂度路由一句话
- AST + RAG 一句话

## Day 2：把启动和配置搞懂

读：

- [LEARNING_MAP.md](/home/root1/go_learn/multi_agent_cooperation/LEARNING_MAP.md) 的第 1 节
- [companion/config.go](/home/root1/go_learn/multi_agent_cooperation/companion/config.go)
- [config.yaml](/home/root1/go_learn/multi_agent_cooperation/config.yaml)

做：

- 自己解释 `app / runtime / workflow / redis / providers`

今天必须背：

- `.env.local` 和 `config.yaml` 的分工
- 为什么默认要保留 `mock`

## Day 3：模型路由

读：

- [companion/router.go](/home/root1/go_learn/multi_agent_cooperation/companion/router.go)
- [LLM_PROVIDER_GUIDE.md](/home/root1/go_learn/multi_agent_cooperation/LLM_PROVIDER_GUIDE.md)

做：

- 找出复杂度评估和 provider 选择逻辑
- 自己口头讲 3 分钟：为什么不能所有任务都用最强模型

今天必须背：

- “复杂度低优先低成本模型，失败时升级高能力模型”

## Day 4：AST 和 RAG

读：

- [mcp/inspector.go](/home/root1/go_learn/multi_agent_cooperation/mcp/inspector.go)
- [rag/index.go](/home/root1/go_learn/multi_agent_cooperation/rag/index.go)
- [PROJECT_TEACHING_GUIDE.md](/home/root1/go_learn/multi_agent_cooperation/PROJECT_TEACHING_GUIDE.md)

做：

- 说清 AST 提取了什么
- 说清 RAG 索引了什么

今天必须背：

- “AST 用来减少符号幻觉，RAG 用来减少文档层面的拍脑袋输出”

## Day 5：执行主链

读：

- [companion/service.go](/home/root1/go_learn/multi_agent_cooperation/companion/service.go)

做：

- 画出一次任务流程图
- 看 `Execute` 主链

今天必须背：

- `任务理解 -> 路由 -> 上下文增强 -> codegen/patch -> preflight -> repair -> docker -> snapshot -> report`

## Day 6：代码生成与 patch

读：

- [companion/codegen.go](/home/root1/go_learn/multi_agent_cooperation/companion/codegen.go)
- [companion/validation_scope.go](/home/root1/go_learn/multi_agent_cooperation/companion/validation_scope.go)

做：

- 理解 `isolated_workspace` 和 `current_repo_patch`
- 理解 `patch_candidates` 和 `rejected_files`

今天必须背：

- 为什么 current repo patch 要有限制

## Day 7：工程预检和自动修复

读：

- [preflight/health.go](/home/root1/go_learn/multi_agent_cooperation/preflight/health.go)
- [companion/auto_loop.go](/home/root1/go_learn/multi_agent_cooperation/companion/auto_loop.go)
- [companion/actions_advanced.go](/home/root1/go_learn/multi_agent_cooperation/companion/actions_advanced.go)

做：

- 解释 `gofmt / goimports / go mod tidy / go test / go vet`

今天必须背：

- 自动修复现在是命令级修复，不是完全模型改代码闭环

## Day 8：Docker 和隔离验证

读：

- [executor/docker_run.go](/home/root1/go_learn/multi_agent_cooperation/executor/docker_run.go)
- [docs/TESTING_GUIDE.md](/home/root1/go_learn/multi_agent_cooperation/docs/TESTING_GUIDE.md)

做：

- 说清 Docker 在这里为什么不是部署，而是隔离验证

今天必须背：

- “Docker 在这个项目里主要承担沙盒验证，不是部署链。”

## Day 9：Snapshot / Diff / Rollback

读：

- [snapshot/manifest.go](/home/root1/go_learn/multi_agent_cooperation/snapshot/manifest.go)
- [companion/diff_artifact.go](/home/root1/go_learn/multi_agent_cooperation/companion/diff_artifact.go)

做：

- 解释 snapshot 和 diff 的区别

今天必须背：

- “Snapshot 记录状态，diff 记录变化，rollback 负责恢复。”

## Day 10：Redis 和 LangGraph

读：

- [companion/redis_store.go](/home/root1/go_learn/multi_agent_cooperation/companion/redis_store.go)
- [companion/langgraph_bridge.go](/home/root1/go_learn/multi_agent_cooperation/companion/langgraph_bridge.go)
- [docs/LANGGRAPH_DECISION.md](/home/root1/go_learn/multi_agent_cooperation/docs/LANGGRAPH_DECISION.md)

做：

- 说清 Redis 现在做了什么
- 说清 LangGraph 现在做了什么

今天必须背：

- “当前是 Go 主控 + LangGraph 可切换阶段编排，不是本地单进程 LangGraph runtime。”

## Day 11：跑一遍测试清单

按这个顺序跑：

```bash
GOTOOLCHAIN=local go test ./...
GOTOOLCHAIN=local go run ./cmd -mode inspect
GOTOOLCHAIN=local go run ./cmd -mode run -goal "根据当前仓库输出一份技术梳理并导出文档"
GOTOOLCHAIN=local go run ./cmd -mode run -goal "生成一个最小 Go HTTP 服务项目，包含 README、测试和运行说明"
```

做：

- 打开生成的 `data/reports`、`data/runs`、`data/exports`

今天必须背：

- 每类产物保存在哪里

## Day 12：面试口径

读：

- [INTERVIEW_PLAYBOOK.md](/home/root1/go_learn/multi_agent_cooperation/INTERVIEW_PLAYBOOK.md)
- [RESUME_PROJECT_ENTRY.md](/home/root1/go_learn/multi_agent_cooperation/RESUME_PROJECT_ENTRY.md)

做：

- 背 30 秒版
- 背 1 分钟版
- 自己讲 3 分钟版

## Day 13：场景题

重点练：

- code review 怎么设计
- 为什么不用 Python 做主控
- AST / RAG 为什么能减少幻觉
- Docker 在这里有什么价值

今天必须背：

- Code review 四步答法

## Day 14：模拟面试

自己对着录音或者找同学问你：

1. 你这个项目是做什么的
2. 为什么要多 Provider
3. AST 和 RAG 有什么区别
4. LangGraph 到底用了没有
5. Redis 为什么需要
6. Docker 在这里是做什么的
7. Snapshot 和 diff 有什么区别
8. 如果让我设计 code review agent，你怎么做

最后检查：

- 你能不能不看文档，讲 1 分钟
- 你能不能画出主链
- 你能不能说出 3 个真实边界

## 面试前一天只复习什么

只看：

- [INTERVIEW_PLAYBOOK.md](/home/root1/go_learn/multi_agent_cooperation/INTERVIEW_PLAYBOOK.md)
- [AI_BEGINNER_GUIDE.md](/home/root1/go_learn/multi_agent_cooperation/AI_BEGINNER_GUIDE.md)
- [TERM_GLOSSARY.md](/home/root1/go_learn/multi_agent_cooperation/TERM_GLOSSARY.md)
- [docs/RESUME_ALIGNMENT.md](/home/root1/go_learn/multi_agent_cooperation/docs/RESUME_ALIGNMENT.md)

