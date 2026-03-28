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
- `generated_root`：新生成项目的独立输出目录
- `auto_open_browser`：是否自动打开浏览器

## 2.1 项目边界配置

项目边界用于控制系统分析和生成代码的范围，避免污染上下文或误操作。

### `workspace`（分析边界）

- **作用**：定义当前项目的根目录，所有 AST 扫描、RAG 索引、快照生成都以此为边界
- **配置建议**：
  - 单体项目：设置为 `"."`（当前目录）
  - 多模块项目：设置为项目根目录，如 `"/path/to/project"`
  - 子目录项目：设置为子目录路径，如 `"subproject"`
- **注意**：此目录内的文件会被符号扫描和索引，排除列表会自动跳过 `workspace_runs` 等生成目录

### `generated_root`（输出边界）

- **作用**：新生成代码的独立存放区，与分析边界分离
- **配置建议**：
  - 默认：`"workspace_runs"`（相对于项目根）
  - 绝对路径：如 `"/tmp/generated"`
  - 相对路径：如 `"output"` 或 `"generated"`
- **注意**：每次任务生成会在此目录下创建时间戳子目录，如 `20260327172702/`，包含所有生成文件

### 配置示例

**单体 Go 项目**：
```yaml
app:
  workspace: "."
  generated_root: "workspace_runs"
```

**多项目工作区**：
```yaml
app:
  workspace: "/home/user/projects/myapp"
  generated_root: "/home/user/projects/generated"
```

**子目录开发**：
```yaml
app:
  workspace: "src"
  generated_root: "../generated"
```

通过合理配置这两个边界，可以确保：
- 分析时不污染生成目录
- 生成文件独立存放，便于管理
- 支持不同项目结构的灵活适配

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

- `backend`：`builtin`、`auto` 或 `langgraph_http`
- `langgraph_endpoint`：外部 LangGraph HTTP bridge 地址，推荐写成带 `{operation}` 的模板
- `langgraph_timeout_sec`：调用外部图编排服务的超时

说明：

- 今天可直接用的是内置 Go 编排
- 如果你已经有 Python / LangGraph 服务，可以把 `backend` 改成 `langgraph_http`
- 如果你想兼顾本地兜底和远端图编排，保持 `backend: auto`
- 推荐配置：

```yaml
workflow:
  backend: "auto"
  langgraph_endpoint: "http://127.0.0.1:8000/{operation}"
```

这里 `{operation}` 会被替换成：

- `plan`
- `codegen`

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
