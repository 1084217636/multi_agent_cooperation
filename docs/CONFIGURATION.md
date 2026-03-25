# 配置文档

主配置文件是根目录的 [config.yaml](/home/root1/go_learn/multi_agent_cooperation/config.yaml)。

## 1. 顶层结构

```yaml
app:
runtime:
workflow:
redis:
vision:
knowledge:
providers:
```

## 2. `app`

- `name`：工作台标题
- `http_addr`：本地桌面工作台监听地址
- `data_dir`：报告与 JSON 产物目录
- `workspace`：要索引和扫描的工程目录
- `auto_open_browser`：是否自动打开浏览器

## 3. `runtime`

- `default_mode`：建议保持 `desktop`
- `enable_symbols`：是否启用 AST / MCP 风格符号快照
- `enable_rag`：是否启用本地轻量检索
- `enable_preflight`：是否在每次执行后跑工程预检
- `enable_docker`：为后续沙箱执行预留
- `enable_snapshots`：是否在任务前后生成工作区快照
- `enable_sandbox_smoke`：是否在任务中跑一次 Docker 沙箱烟雾验证
- `max_knowledge_results`：RAG 最多返回多少条结果
- `provider_timeout_sec`：单次模型尝试的基础超时时间；低复杂度更短，高复杂度更长

## 4. `knowledge`

- `include_paths`：哪些目录或文件进入 RAG 索引
- `exclude_names`：跳过哪些目录
- `max_file_size_kb`：单文件大小上限
- `chunk_size`：切片大小

建议：

- 文档多时把 `README.md`、`docs/`、核心包目录都纳入
- 大二进制目录、构建产物目录不要纳入

## 5. `workflow`

- `backend`：`builtin` 或 `langgraph_http`
- `langgraph_endpoint`：外部 LangGraph HTTP bridge 地址
- `langgraph_timeout_sec`：调用外部图编排服务的超时

说明：

- 今天可直接用的是内置 Go 编排
- 如果你已经有 Python / LangGraph 服务，可以把 `backend` 改成 `langgraph_http`

## 6. `redis`

- `enabled`：是否启用 Redis 状态持久化
- `url`：Redis 连接地址
- `namespace`：键名前缀
- `auto_start_container`：本地没起 Redis 时，是否尝试用 Docker 拉起一个 `redis:7-alpine`
- `container_name`：自动拉起时使用的容器名

## 7. `vision`

- `enabled`：是否启用屏幕 OCR / 多模态分析
- `provider`：当前默认支持 `groq`
- `model`：默认用 `meta-llama/llama-4-scout-17b-16e-instruct`
- `analyze_on_capture`：每次新屏幕帧写入后是否自动分析
- `timeout_sec`：单次视觉分析超时

## 8. `providers`

推荐最少保留：

- `mock`
- `ollama`
- `groq`
- `openai`
- `siliconflow`

API Key 推荐通过环境变量提供：

```bash
export GROQ_API_KEY=...
export OPENAI_API_KEY=...
export SILICONFLOW_API_KEY=...
```

如果你不想每次手动 `export`，可以在项目根目录创建 `.env.local`：

```bash
GROQ_API_KEY=...
```

当前项目会在启动时自动加载这个文件。

对应 YAML：

```yaml
api_key: "${GROQ_API_KEY}"
```

## 9. 离线优先配置

如果你当前没有 API 额度，保留默认配置即可。系统会自动使用 `mock`。

## 10. 本地模型配置

如果你安装了 Ollama：

```bash
ollama serve
ollama pull qwen2.5-coder:7b
```

然后保持：

```yaml
base_url: "http://localhost:11434/v1"
default_model: "qwen2.5-coder:7b"
```
