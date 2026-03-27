package companion

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"multi_agent_cooperation/llm"
)

// AppConfig 定义桌面伴侣的入口配置。
type AppConfig struct {
	Name            string `yaml:"name" json:"name"`
	HTTPAddr        string `yaml:"http_addr" json:"http_addr"`
	DataDir         string `yaml:"data_dir" json:"data_dir"`
	Workspace       string `yaml:"workspace" json:"workspace"`
	GeneratedRoot   string `yaml:"generated_root" json:"generated_root"`
	AutoOpenBrowser bool   `yaml:"auto_open_browser" json:"auto_open_browser"`
}

// RuntimeConfig 定义运行期开关。
type RuntimeConfig struct {
	DefaultMode            string `yaml:"default_mode" json:"default_mode"`
	EnableSymbols          bool   `yaml:"enable_symbols" json:"enable_symbols"`
	EnableRAG              bool   `yaml:"enable_rag" json:"enable_rag"`
	EnablePreflight        bool   `yaml:"enable_preflight" json:"enable_preflight"`
	EnableDocker           bool   `yaml:"enable_docker" json:"enable_docker"`
	EnableDockerValidation bool   `yaml:"enable_docker_validation" json:"enable_docker_validation"`
	EnableSnapshots        bool   `yaml:"enable_snapshots" json:"enable_snapshots"`
	EnableSandboxSmoke     bool   `yaml:"enable_sandbox_smoke" json:"enable_sandbox_smoke"`
	AutoRepairRounds       int    `yaml:"auto_repair_rounds" json:"auto_repair_rounds"`
	MaxKnowledgeResults    int    `yaml:"max_knowledge_results" json:"max_knowledge_results"`
	ProviderTimeoutSec     int    `yaml:"provider_timeout_sec" json:"provider_timeout_sec"`
}

// WorkflowConfig 描述工作流编排后端。
type WorkflowConfig struct {
	Backend             string `yaml:"backend" json:"backend"`
	LangGraphEndpoint   string `yaml:"langgraph_endpoint" json:"langgraph_endpoint"`
	LangGraphTimeoutSec int    `yaml:"langgraph_timeout_sec" json:"langgraph_timeout_sec"`
}

// RedisConfig 描述状态持久化配置。
type RedisConfig struct {
	Enabled            bool   `yaml:"enabled" json:"enabled"`
	URL                string `yaml:"url" json:"url"`
	Namespace          string `yaml:"namespace" json:"namespace"`
	AutoStartContainer bool   `yaml:"auto_start_container" json:"auto_start_container"`
	ContainerName      string `yaml:"container_name" json:"container_name"`
}

// VisionConfig 描述屏幕 OCR / 多模态分析配置。
type VisionConfig struct {
	Enabled          bool   `yaml:"enabled" json:"enabled"`
	Provider         string `yaml:"provider" json:"provider"`
	Model            string `yaml:"model" json:"model"`
	AnalyzeOnCapture bool   `yaml:"analyze_on_capture" json:"analyze_on_capture"`
	TimeoutSec       int    `yaml:"timeout_sec" json:"timeout_sec"`
}

// KnowledgeConfig 控制本地知识库索引范围。
type KnowledgeConfig struct {
	IncludePaths  []string `yaml:"include_paths" json:"include_paths"`
	ExcludeNames  []string `yaml:"exclude_names" json:"exclude_names"`
	MaxFileSizeKB int      `yaml:"max_file_size_kb" json:"max_file_size_kb"`
	ChunkSize     int      `yaml:"chunk_size" json:"chunk_size"`
}

// Config 是桌面伴侣的总配置。
type Config struct {
	ConfigPath string               `yaml:"-" json:"config_path"`
	ConfigDir  string               `yaml:"-" json:"config_dir"`
	App        AppConfig            `yaml:"app" json:"app"`
	Runtime    RuntimeConfig        `yaml:"runtime" json:"runtime"`
	Workflow   WorkflowConfig       `yaml:"workflow" json:"workflow"`
	Redis      RedisConfig          `yaml:"redis" json:"redis"`
	Vision     VisionConfig         `yaml:"vision" json:"vision"`
	Knowledge  KnowledgeConfig      `yaml:"knowledge" json:"knowledge"`
	Providers  []llm.ProviderConfig `yaml:"providers" json:"providers"`
}

// DefaultConfig 返回可直接启动的默认配置。
func DefaultConfig(workdir string) *Config {
	return &Config{
		App: AppConfig{
			Name:            "Desk Companion AI",
			HTTPAddr:        "127.0.0.1:18888",
			DataDir:         filepath.Join(workdir, "data"),
			Workspace:       workdir,
			GeneratedRoot:   filepath.Join(workdir, "workspace_runs"),
			AutoOpenBrowser: true,
		},
		Runtime: RuntimeConfig{
			DefaultMode:            "desktop",
			EnableSymbols:          true,
			EnableRAG:              true,
			EnablePreflight:        true,
			EnableDocker:           true,
			EnableDockerValidation: true,
			EnableSnapshots:        true,
			EnableSandboxSmoke:     false,
			AutoRepairRounds:       2,
			MaxKnowledgeResults:    4,
			ProviderTimeoutSec:     20,
		},
		Workflow: WorkflowConfig{
			Backend:             "auto",
			LangGraphEndpoint:   "",
			LangGraphTimeoutSec: 25,
		},
		Redis: RedisConfig{
			Enabled:            true,
			URL:                "redis://127.0.0.1:6379/0",
			Namespace:          "desk_companion",
			AutoStartContainer: true,
			ContainerName:      "desk-companion-redis",
		},
		Vision: VisionConfig{
			Enabled:          true,
			Provider:         "groq",
			Model:            "meta-llama/llama-4-scout-17b-16e-instruct",
			AnalyzeOnCapture: true,
			TimeoutSec:       20,
		},
		Knowledge: KnowledgeConfig{
			IncludePaths: []string{
				"README.md",
				"docs",
				"cmd",
				"companion",
				"agent",
				"engine",
				"executor",
				"llm",
				"mcp",
				"rag",
			},
			ExcludeNames: []string{
				".git",
				".vscode",
				"bin",
				"accounts",
				"node_modules",
				"data",
			},
			MaxFileSizeKB: 512,
			ChunkSize:     700,
		},
		Providers: []llm.ProviderConfig{
			{
				Name:         "mock",
				DefaultModel: "companion-mock-v1",
				Models:       []string{"companion-mock-v1"},
			},
			{
				Name:         "ollama",
				BaseURL:      "http://localhost:11434/v1",
				DefaultModel: "qwen2.5-coder:7b",
				Models: []string{
					"qwen2.5-coder:7b",
					"llama3.1",
					"deepseek-coder",
				},
			},
			{
				Name:         "groq",
				APIKey:       "${GROQ_API_KEY}",
				BaseURL:      "https://api.groq.com/openai/v1",
				DefaultModel: "openai/gpt-oss-120b",
				Models: []string{
					"openai/gpt-oss-120b",
					"llama-3.3-70b-versatile",
					"llama-3.1-8b-instant",
				},
			},
			{
				Name:         "openai",
				APIKey:       "${OPENAI_API_KEY}",
				BaseURL:      "https://api.openai.com/v1",
				DefaultModel: "gpt-4o-mini",
				Models: []string{
					"gpt-4o-mini",
					"gpt-4o",
				},
			},
			{
				Name:         "siliconflow",
				APIKey:       "${SILICONFLOW_API_KEY}",
				BaseURL:      "https://api.siliconflow.cn/v1",
				DefaultModel: "Qwen/Qwen2.5-72B-Instruct",
				Models: []string{
					"Qwen/Qwen2.5-72B-Instruct",
					"deepseek-ai/DeepSeek-V2.5",
				},
			},
		},
	}
}

// LoadConfig 从 YAML 读取配置；若文件不存在，则回退为默认配置。
func LoadConfig(path, workdir string) (*Config, error) {
	cfg := DefaultConfig(workdir)
	cfg.ConfigPath = path
	cfg.ConfigDir = filepath.Dir(path)

	loadLocalEnv(filepath.Join(workdir, ".env.local"))

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg.resolvePaths(workdir)
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	cfg.expandEnv()
	cfg.resolvePaths(workdir)
	return cfg, nil
}

// ProviderConfig 根据名称获取 Provider 配置。
func (c *Config) ProviderConfig(name string) (llm.ProviderConfig, bool) {
	for _, provider := range c.Providers {
		if provider.Name == name {
			return provider, true
		}
	}
	return llm.ProviderConfig{}, false
}

func (c *Config) expandEnv() {
	c.App.HTTPAddr = os.ExpandEnv(c.App.HTTPAddr)
	c.App.DataDir = os.ExpandEnv(c.App.DataDir)
	c.App.Workspace = os.ExpandEnv(c.App.Workspace)
	c.App.GeneratedRoot = os.ExpandEnv(c.App.GeneratedRoot)
	c.Workflow.LangGraphEndpoint = os.ExpandEnv(c.Workflow.LangGraphEndpoint)
	c.Redis.URL = os.ExpandEnv(c.Redis.URL)
	c.Redis.Namespace = os.ExpandEnv(c.Redis.Namespace)
	c.Redis.ContainerName = os.ExpandEnv(c.Redis.ContainerName)
	c.Vision.Provider = os.ExpandEnv(c.Vision.Provider)
	c.Vision.Model = os.ExpandEnv(c.Vision.Model)

	for idx := range c.Knowledge.IncludePaths {
		c.Knowledge.IncludePaths[idx] = os.ExpandEnv(c.Knowledge.IncludePaths[idx])
	}

	for idx := range c.Providers {
		c.Providers[idx].APIKey = os.ExpandEnv(c.Providers[idx].APIKey)
		c.Providers[idx].BaseURL = os.ExpandEnv(c.Providers[idx].BaseURL)
		c.Providers[idx].DefaultModel = os.ExpandEnv(c.Providers[idx].DefaultModel)
	}
}

func loadLocalEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if key == "" || value == "" {
			continue
		}
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, value)
		}
	}
}

func (c *Config) resolvePaths(workdir string) {
	if c.ConfigDir == "" {
		c.ConfigDir = filepath.Dir(c.ConfigPath)
	}
	if c.ConfigDir == "." || c.ConfigDir == "" {
		c.ConfigDir = workdir
	}

	if !filepath.IsAbs(c.App.Workspace) {
		c.App.Workspace = filepath.Join(workdir, c.App.Workspace)
	}
	if strings.TrimSpace(c.App.GeneratedRoot) == "" {
		c.App.GeneratedRoot = filepath.Join(workdir, "workspace_runs")
	}
	if !filepath.IsAbs(c.App.GeneratedRoot) {
		c.App.GeneratedRoot = filepath.Join(workdir, c.App.GeneratedRoot)
	}
	if !filepath.IsAbs(c.App.DataDir) {
		c.App.DataDir = filepath.Join(workdir, c.App.DataDir)
	}

	for idx, include := range c.Knowledge.IncludePaths {
		if filepath.IsAbs(include) {
			continue
		}
		c.Knowledge.IncludePaths[idx] = filepath.Join(c.App.Workspace, include)
	}
}
