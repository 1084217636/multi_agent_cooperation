# 使用文档

## 1. 启动桌面工作台

```bash
GOTOOLCHAIN=local go run ./cmd -mode desktop
```

默认会：

- 读取根目录 `config.yaml`
- 启动本地服务 `http://127.0.0.1:18888`
- 自动打开浏览器
- 加载 Provider 状态、符号快照、RAG 索引和历史运行结果
- 提供“截图一次”按钮，按需写入 `data/captures`
- 默认尝试接入 Redis 状态存储和屏幕 OCR / 多模态分析

## 2. 单次执行

```bash
GOTOOLCHAIN=local go run ./cmd -mode run -goal "把当前仓库升级为成熟的桌面 AI 项目"
```

终端会输出：

- 使用的 Provider
- 复杂度等级
- 核心概述
- Markdown 报告路径
- JSON 产物路径

## 3. 查看工程快照

```bash
GOTOOLCHAIN=local go run ./cmd -mode inspect
```

适合快速确认：

- 当前 Provider 是否配置好
- 符号扫描是否成功
- RAG 索引是否建立
- Docker CLI 是否可见

## 4. 桌面工作台里能看到什么

- `Mission Input`：输入你的改造目标
- `Screen Context`：按需截图、查看最近一次屏幕上下文
- `Workflow Backend`：当前是内置 Go 编排还是外部 LangGraph bridge
- `Loop Engine`：查看当前闭环状态和执行步骤
- `System Snapshot`：当前仓库的 Provider / 符号 / RAG 摘要
- `Provider Ladder`：路由可用性
- `Recent Runs`：历史 Markdown 报告
- `Latest Companion Run`：最近一次任务的动作建议、创新点、RAG 场景、当前缺口、风险、快照和排障建议
- `Approval Actions`：审批后执行“重新预检”“快照回滚”“Docker 自愈执行闭环”或“自动修复后自检并提交”

## 5. 产物目录

- `data/captures/`：最近的屏幕截图工件
- `data/reports/`：Markdown 报告
- `data/runs/`：JSON 运行结果
- `data/snapshots/`：任务前后工作区快照

## 6. 常见使用方式

场景一：没有 API 额度，先演示项目能力

```bash
GOTOOLCHAIN=local go run ./cmd -mode desktop
```

场景二：想检查真实 Provider 是否接好了

```bash
export GROQ_API_KEY=...
GOTOOLCHAIN=local go run ./cmd -mode inspect
```

场景三：想把一次分析结果沉淀成文档

```bash
GOTOOLCHAIN=local go run ./cmd -mode run -goal "补齐 README、配置文档和教学文档"
```

场景四：想在本地工作台里做一次强闭环演示

1. 启动 `desktop` 模式
2. 如果任务依赖当前页面，再点击“截图一次”
3. 输入任务目标并点击“启动闭环任务”
4. 如果出现 `Manual Recovery Actions`，再按需点击执行
5. 查看同一份 JSON / Markdown 报告里新增的审批步骤和动作输出
