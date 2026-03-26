# Testing Guide

这份文档告诉你怎么验证项目的关键能力是不是已经成功启动。

## 1. 基础编译与自检

命令：

```bash
GOTOOLCHAIN=local go test ./...
GOTOOLCHAIN=local go run ./cmd -mode inspect
```

成功标志：

- `go test ./...` 全部通过
- `inspect` 输出里能看到：
  - Provider 数量
  - Symbols / Knowledge 统计
  - Docker 状态
  - Redis 状态
  - Vision 状态

## 2. 文档 / 分析模式

命令：

```bash
GOTOOLCHAIN=local go run ./cmd -mode run -goal "根据当前仓库输出一份技术梳理并导出文档"
```

成功标志：

- 终端里会输出 `Provider`、`Complexity`、`Markdown report`、`JSON artifact`
- `data/reports/<run_id>.md` 存在
- `data/runs/<run_id>.json` 存在
- `data/exports/<run_id>/document_export.md` 或 `project_package.md` 存在

## 3. 独立工作区代码生成

命令：

```bash
GOTOOLCHAIN=local go run ./cmd -mode run -goal "生成一个最小 Go HTTP 服务项目，包含 README、测试和运行说明"
```

成功标志：

- `data/runs/<run_id>.json` 里：
  - `plan.mode = scaffold`
  - `codegen.target_mode = isolated_workspace`
  - `codegen.provider != fallback-template`
  - `codegen.parse_mode` 有值
- `workspace_runs/<run_id>/` 下有生成的代码
- `data/exports/<run_id>/sandbox.log` 存在
- `data/exports/<run_id>/workflow_summary.json` 存在

## 4. 当前仓库 patch 模式

命令：

```bash
GOTOOLCHAIN=local go run ./cmd -mode run -goal "修复当前项目的预检链路，并保留 diff、日志和回滚信息"
```

成功标志：

- `data/runs/<run_id>.json` 里：
  - `codegen.target_mode = current_repo_patch`
  - `codegen.patch_candidates` 非空
  - `codegen.raw_response_path` 非空
- `data/exports/<run_id>/workspace_diff.patch` 存在
- 如果模型输出越界文件，`codegen.rejected_files` 里能看到被拦截的路径

注意：

- 这个命令会尝试改当前仓库，执行前先看好 Git 状态
- 如果要恢复，使用页面里的回滚动作，或者基于 before snapshot 恢复

## 5. 预检与自动修复轮次

看运行结果里的这些字段：

- `preflight.scope`
- `preflight.targets`
- `repair_rounds`

成功标志：

- 预检不是只会跑 `./...`，而是会根据本轮目标输出验证范围
- 有失败时会出现 `repair_rounds`
- 每轮会记录：
  - `commands`
  - `changed_files`
  - `termination_reason`

## 6. Docker 沙盒验证

先确保本机 Docker 可用：

```bash
docker ps
```

再执行任意 `scaffold` 或 `closed_loop` 任务。

成功标志：

- 结果 JSON 里：
  - `sandbox.enabled = true`
  - `sandbox.status = completed`
  - `sandbox.targets` 非空或默认是 `./...`
- `data/exports/<run_id>/sandbox.log` 存在

## 7. Redis 状态恢复

先跑一次任务，再重启服务：

```bash
GOTOOLCHAIN=local go run ./cmd -mode run -goal "根据当前仓库导出一份说明文档"
GOTOOLCHAIN=local go run ./cmd -mode inspect
```

成功标志：

- `inspect` 输出里 `Redis: redis connected`
- 打开工作台后，最近一次运行结果还能看到
- 屏幕上下文和 workflow 状态不会完全丢掉

如果你想直接检查 Redis：

```bash
docker exec -it desk-companion-redis redis-cli KEYS 'desk_companion:*'
```

正常会看到：

- `desk_companion:workflow`
- `desk_companion:screen`
- `desk_companion:latest_run`

## 8. LangGraph Bridge

这个功能需要你自己提供 LangGraph HTTP 服务端点。

配置：

- [config.yaml](/home/root1/go_learn/multi_agent_cooperation/config.yaml)

关键字段：

```yaml
workflow:
  backend: auto
  langgraph_endpoint: "http://127.0.0.1:8000/{operation}"
```

成功标志：

- 运行结果 JSON 里 `workflow_backend = langgraph_http`
- `attempts` 里出现 `langgraph-http`
- `workflow_trace.nodes` 里 `planning` 的 backend 是 `langgraph_http`
- 如果任务涉及代码生成，`codegen.backend = langgraph_http`

如果 endpoint 没配好，会自动回退到内置编排。

## 9. Snapshot 与回滚

每次运行后检查：

- `data/snapshots/*-before.json`
- `data/snapshots/*-after.json`
- `data/exports/<run_id>/workspace_diff.patch`

成功标志：

- JSON 里 `snapshot.before_path`、`snapshot.after_path` 非空
- `snapshot.changed_files` 大于等于 0
- 工作台里出现 “回滚到运行前快照” 时，点击后能恢复差异文件

## 10. 学项目时最推荐的测试顺序

按这个顺序就够了：

1. `go test ./...`
2. `go run ./cmd -mode inspect`
3. 跑一次文档导出任务
4. 跑一次独立工作区 scaffold 任务
5. 看 `reports / runs / exports / workspace_runs`
6. 最后再试 current repo patch
