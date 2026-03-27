# Go R&D Agent Workbench

一个基于 Go 的本地研发智能体工作台，面向 Go 工程场景提供任务分析、模型路由、AST 符号注入、轻量 RAG、工程预检、快照回滚和文档导出能力。

当前版本的重点不是角色化外壳，而是把研发辅助链路先跑通：

- 复杂度评估与多 Provider 路由
- AST 符号扫描与上下文增强
- 轻量 RAG 检索
- 工程预检与结构化排障
- Snapshot 差异记录与回滚
- 导出 Markdown / JSON / 交付文档
- 可选 Docker 自愈动作与 Redis 状态存储

## 核心定位

这个项目更适合被理解为：

`Go 工程研发辅助智能体 + 本地工作台 + 执行闭环原型`

而不是一个以拟人交互为核心卖点的产品。

## 当前能做什么

- 启动本地工作台，在浏览器中完成任务输入、运行查看和结果导出
- 对 Go 项目做复杂度评估，并根据可用 Provider 自动路由
- 扫描 Go AST，向模型注入真实包、结构体、函数和签名信息
- 为 README、配置、设计文档和历史运行结果建立轻量 RAG 索引
- 运行 `go test`、`go vet`、`goimports`/`golangci-lint` 可用性检查
- 记录执行前后快照，生成差异、回滚建议和结构化排障报告
- 按目标包范围执行预检、自动修复和 Docker 验证，而不是始终全仓跑 `./...`
- 导出 Markdown 报告、JSON 运行结果和交付文档
- 保留 raw model response、preflight.json、repair_rounds.json、sandbox.log 等留痕工件
- 在需要时执行 Docker 自愈、重新预检、自动修复和快照回滚动作
- 可选接入 LangGraph 阶段编排后端，支持把 `planning + codegen` 节点委托给外部图服务
- Redis 状态恢复和屏幕上下文采集

## 快速开始

前置条件：

- Go 1.25.x

启动工作台：

```bash
GOTOOLCHAIN=local go run ./cmd -mode desktop
```

单次运行任务：

```bash
GOTOOLCHAIN=local go run ./cmd -mode run -goal "围绕当前 Go 项目生成技术分析、交付文档或闭环开发方案"
```

查看当前状态：

```bash
GOTOOLCHAIN=local go run ./cmd -mode inspect
```

运行测试：

```bash
GOTOOLCHAIN=local go test ./...
```

## 输出位置

- 工作台地址：`http://127.0.0.1:18888`
- Markdown 报告：`data/reports/*.md`
- JSON 运行产物：`data/runs/*.json`
- 导出文档与计划：`data/exports/<run_id>/*`
- 屏幕截图工件：`data/captures/*`

## 关键模块

- 入口与模式：[cmd/main.go](/home/root1/go_learn/multi_agent_cooperation/cmd/main.go)
- 执行主链：[companion/service.go](/home/root1/go_learn/multi_agent_cooperation/companion/service.go)
- 路由与复杂度评估：[companion/router.go](/home/root1/go_learn/multi_agent_cooperation/companion/router.go)
- AST 符号扫描：[mcp/inspector.go](/home/root1/go_learn/multi_agent_cooperation/mcp/inspector.go)
- 工程预检：[preflight/health.go](/home/root1/go_learn/multi_agent_cooperation/preflight/health.go)
- Docker 自愈动作：[companion/actions_advanced.go](/home/root1/go_learn/multi_agent_cooperation/companion/actions_advanced.go)
- 快照与回滚：[snapshot/manifest.go](/home/root1/go_learn/multi_agent_cooperation/snapshot/manifest.go)
- Redis 状态存储：[companion/redis_store.go](/home/root1/go_learn/multi_agent_cooperation/companion/redis_store.go)
- LangGraph 阶段编排桥接：[companion/langgraph_bridge.go](/home/root1/go_learn/multi_agent_cooperation/companion/langgraph_bridge.go)

## 文档

- 项目教学文档：[PROJECT_TEACHING_GUIDE.md](/home/root1/go_learn/multi_agent_cooperation/PROJECT_TEACHING_GUIDE.md)
- 学习地图：[LEARNING_MAP.md](/home/root1/go_learn/multi_agent_cooperation/LEARNING_MAP.md)
- 术语表：[TERM_GLOSSARY.md](/home/root1/go_learn/multi_agent_cooperation/TERM_GLOSSARY.md)
- AI 小白讲义：[AI_BEGINNER_GUIDE.md](/home/root1/go_learn/multi_agent_cooperation/AI_BEGINNER_GUIDE.md)
- 面试口径包：[INTERVIEW_PLAYBOOK.md](/home/root1/go_learn/multi_agent_cooperation/INTERVIEW_PLAYBOOK.md)
- 14 天学习计划：[STUDY_PLAN_14_DAYS.md](/home/root1/go_learn/multi_agent_cooperation/STUDY_PLAN_14_DAYS.md)
- 使用文档：[docs/USAGE.md](/home/root1/go_learn/multi_agent_cooperation/docs/USAGE.md)
- 配置文档：[docs/CONFIGURATION.md](/home/root1/go_learn/multi_agent_cooperation/docs/CONFIGURATION.md)
- 测试文档：[docs/TESTING_GUIDE.md](/home/root1/go_learn/multi_agent_cooperation/docs/TESTING_GUIDE.md)
- LangGraph 取舍说明：[docs/LANGGRAPH_DECISION.md](/home/root1/go_learn/multi_agent_cooperation/docs/LANGGRAPH_DECISION.md)
- Provider 说明：[LLM_PROVIDER_GUIDE.md](/home/root1/go_learn/multi_agent_cooperation/LLM_PROVIDER_GUIDE.md)

## 当前边界

- 这版最强的是“分析、预检、导出、回滚建议”，不是完整的自动写代码主链
- LangGraph 目前已能接管 `planning + codegen` 两个核心节点，但仍是外部 HTTP 图后端，不是本地单进程图运行时
- Redis 目前已承担最近任务 / workflow / screen 恢复，但不是完整的调度内核
- 屏幕上下文是可选辅助能力，不是项目主卖点

## 安全边界

- 只实现合规的本地工作台、知识检索、模型接入和工程辅助能力
- 不包含账号滥用、批量注册、绕过平台额度等自动化能力
