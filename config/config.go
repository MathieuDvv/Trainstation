package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"trainstation/provider"
)

type Config struct {
	Workspace  string               `yaml:"-"`
	Router     RouterConfig         `yaml:"router"`
	Providers  map[string]Provider  `yaml:"providers"`
	Agents     AgentsConfig         `yaml:"agents"`
}

type RouterConfig struct {
	Provider      string `yaml:"provider"`
	Model         string `yaml:"model"`
	ThinkingLevel string `yaml:"thinking_level"`
}

type Provider struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
}

type AgentsConfig struct {
	Claude      AgentConfig `yaml:"claude"`
	OpenCode    AgentConfig `yaml:"opencode"`
	Codex       AgentConfig `yaml:"codex"`
	Antigravity AgentConfig `yaml:"antigravity"`
}

type AgentConfig struct {
	Enabled         bool     `yaml:"enabled"`
	Strengths       []string `yaml:"strengths"`
	SkipPermissions bool     `yaml:"skip_permissions"`
	ExtraArgs       []string `yaml:"extra_args"`
}

func defaults() Config {
	return Config{
		Workspace: ".",
		Router: RouterConfig{
			Provider:      "deepseek",
			Model:         "deepseek-chat",
			ThinkingLevel: "medium",
		},
		Providers: map[string]Provider{},
		Agents: AgentsConfig{
			Claude: AgentConfig{
				Enabled:   true,
				Strengths: []string{"architecture", "code-review", "refactoring", "complex-reasoning", "debugging"},
			},
			OpenCode: AgentConfig{
				Enabled:   true,
				Strengths: []string{"general-coding", "open-source", "privacy-sensitive", "local-tasks", "documentation"},
			},
			Codex: AgentConfig{
				Enabled:   true,
				Strengths: []string{"quick-scripts", "api-integration", "openai-ecosystem", "prototyping", "data-processing"},
			},
			Antigravity: AgentConfig{
				Enabled:   true,
				Strengths: []string{"web-apps", "google-cloud", "android", "prototyping", "ui-development"},
			},
		},
	}
}

func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".trainstation", "config.yaml"), nil
}

func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	cfg := defaults()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]Provider)
	}
	
	if cwd, err := os.Getwd(); err == nil {
		cfg.Workspace = cwd
	}
	
	return &cfg, nil
}

func ConfigExists() (bool, error) {
	path, err := ConfigPath()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

func Default() *Config {
	cfg := defaults()
	workspace, _ := os.Getwd()
	cfg.Workspace = workspace
	return &cfg
}

func Save(cfg *Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (c *Config) GetAPIKey(providerName string) string {
	if p, ok := c.Providers[providerName]; ok {
		return p.APIKey
	}
	return ""
}

func (c *Config) GetBaseURL(providerName string) string {
	if p, ok := c.Providers[providerName]; ok && p.BaseURL != "" {
		return p.BaseURL
	}
	if def := provider.Get(providerName); def != nil {
		return def.BaseURL
	}
	return ""
}

func (c *Config) SetProvider(name, apiKey string) {
	def := provider.Get(name)
	baseURL := ""
	if def != nil {
		baseURL = def.BaseURL
	}
	c.Providers[name] = Provider{APIKey: apiKey, BaseURL: baseURL}
}

func (c *Config) ConfiguredProviders() []string {
	var result []string
	for _, def := range provider.Definitions {
		if c.GetAPIKey(def.Name) != "" {
			result = append(result, def.Name)
		}
	}
	return result
}
