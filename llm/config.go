package llm

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config 定义LLM配置结构
type Config struct {
	Providers []ProviderConfig `yaml:"providers" json:"providers"` // Provider列表
	Default   string           `yaml:"default" json:"default"`     // 默认Provider
}

// LoadConfig 从YAML文件加载配置
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// SaveConfig 保存配置到YAML文件
func SaveConfig(config *Config, path string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetProviderConfig 获取指定Provider的配置
func (c *Config) GetProviderConfig(name string) (ProviderConfig, error) {
	for _, provider := range c.Providers {
		if provider.Name == name {
			return provider, nil
		}
	}
	return ProviderConfig{}, fmt.Errorf("provider %s not found in config", name)
}

// GetDefaultProviderConfig 获取默认Provider的配置
func (c *Config) GetDefaultProviderConfig() (ProviderConfig, error) {
	if c.Default == "" {
		if len(c.Providers) > 0 {
			return c.Providers[0], nil
		}
		return ProviderConfig{}, fmt.Errorf("no providers configured")
	}
	return c.GetProviderConfig(c.Default)
}

// Manager 管理多个LLM Provider
type Manager struct {
	config    *Config
	providers map[string]Provider
}

// NewManager 创建LLM Manager
func NewManager(config *Config) (*Manager, error) {
	manager := &Manager{
		config:    config,
		providers: make(map[string]Provider),
	}

	// 初始化所有配置的Provider
	for _, providerConfig := range config.Providers {
		provider, err := Create(providerConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create provider %s: %w", providerConfig.Name, err)
		}
		manager.providers[providerConfig.Name] = provider
	}

	return manager, nil
}

// GetProvider 获取指定名称的Provider
func (m *Manager) GetProvider(name string) (Provider, error) {
	provider, ok := m.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %s not found", name)
	}
	return provider, nil
}

// GetDefaultProvider 获取默认Provider
func (m *Manager) GetDefaultProvider() (Provider, error) {
	config, err := m.config.GetDefaultProviderConfig()
	if err != nil {
		return nil, err
	}
	return m.GetProvider(config.Name)
}

// GetProviderByModel 根据模型名称获取Provider
func (m *Manager) GetProviderByModel(modelName string) (Provider, error) {
	for name, provider := range m.providers {
		for _, model := range provider.GetModelList() {
			if model == modelName {
				return m.providers[name], nil
			}
		}
	}
	return nil, fmt.Errorf("no provider found for model %s", modelName)
}

// ListProviders 列出所有可用的Provider
func (m *Manager) ListProviders() []string {
	providers := make([]string, 0, len(m.providers))
	for name := range m.providers {
		providers = append(providers, name)
	}
	return providers
}

// ListAllModels 列出所有可用的模型
func (m *Manager) ListAllModels() map[string][]string {
	models := make(map[string][]string)
	for name, provider := range m.providers {
		models[name] = provider.GetModelList()
	}
	return models
}
