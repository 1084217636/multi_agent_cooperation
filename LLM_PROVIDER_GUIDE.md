# LLM Provider Guide

当前工程实际支持的 Provider 只有这几类：

- `mock`：默认离线回退，保证没有 API 时也能演示完整链路
- `ollama`：本地模型，适合低成本联调
- `groq`：在线高速推理
- `openai`：高质量通用模型
- `siliconflow`：国内可选的开源模型聚合平台

## 配置方式

推荐直接使用 [config.example.yaml](/home/root1/go_learn/multi_agent_cooperation/config.example.yaml) 或 [config.yaml](/home/root1/go_learn/multi_agent_cooperation/config.yaml)，然后通过环境变量注入 API Key：

```bash
export GROQ_API_KEY=...
export OPENAI_API_KEY=...
export SILICONFLOW_API_KEY=...
```

## 路由策略

- 低复杂度：优先 `mock` / `ollama`
- 中复杂度：优先本地模型，失败后尝试在线
- 高复杂度：优先高能力在线模型，失败后自动降级

## 注意

- 项目不再内置任何硬编码 API Key
- 如果在线 Provider 不可用，工作台会自动回退到 `mock`
- `mock` 模式主要用于桌面工作台演示、文档联调和工程验证，不替代真实推理质量
