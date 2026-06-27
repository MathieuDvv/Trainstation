package provider

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ProviderDef struct {
	Name       string
	Label      string
	BaseURL    string
	Models     []ModelDef
	EnvVar     string
	BalanceURL string

	// ModelsPath is the path to the models endpoint (default: /v1/models)
	// Set to empty to skip API fetching for this provider
	ModelsPath string
	// APIType: "openai" (standard /v1/models), "anthropic", "google"
	APIType string
	// FilterPrefixes: only include models whose ID starts with one of these (empty = all)
	FilterPrefixes []string
	// ExcludePatterns: model IDs containing any of these are excluded
	ExcludePatterns []string
}

type ModelDef struct {
	ID       string
	Label    string
	Reasoner bool
}

var Definitions = []ProviderDef{
	{
		Name:       "deepseek",
		Label:      "Deepseek",
		BaseURL:    "https://api.deepseek.com",
		EnvVar:     "DEEPSEEK_API_KEY",
		BalanceURL: "https://api.deepseek.com/user/balance",
		ModelsPath: "/models",
		APIType:    "openai",
		Models: []ModelDef{
			{ID: "deepseek-chat", Label: "Deepseek Chat"},
			{ID: "deepseek-reasoner", Label: "Deepseek Reasoner (R1)", Reasoner: true},
		},
	},
	{
		Name:           "openai",
		Label:          "OpenAI",
		BaseURL:        "https://api.openai.com/v1",
		EnvVar:         "OPENAI_API_KEY",
		ModelsPath:     "/models",
		APIType:        "openai",
		FilterPrefixes: []string{"gpt-", "o1", "o3", "o4", "chatgpt-"},
		ExcludePatterns: []string{"-realtime", "-audio", "-tts", "-whisper", "-embedding", "dall-e", "text-", "babbage", "davinci-002", "moderation"},
		Models: []ModelDef{
			{ID: "gpt-4.1-mini", Label: "GPT-4.1 Mini"},
			{ID: "gpt-4.1", Label: "GPT-4.1"},
			{ID: "gpt-4o-mini", Label: "GPT-4o Mini"},
			{ID: "gpt-4o", Label: "GPT-4o"},
			{ID: "o4-mini", Label: "o4-mini", Reasoner: true},
			{ID: "o3", Label: "o3", Reasoner: true},
		},
	},
	{
		Name:       "anthropic",
		Label:      "Anthropic",
		BaseURL:    "https://api.anthropic.com/v1",
		EnvVar:     "ANTHROPIC_API_KEY",
		ModelsPath: "/models",
		APIType:    "anthropic",
		Models: []ModelDef{
			{ID: "claude-3-5-haiku-latest", Label: "Claude 3.5 Haiku"},
			{ID: "claude-3-5-sonnet-latest", Label: "Claude 3.5 Sonnet"},
			{ID: "claude-sonnet-4-20250514", Label: "Claude Sonnet 4"},
			{ID: "claude-opus-4-20250514", Label: "Claude Opus 4"},
		},
	},
	{
		Name:           "groq",
		Label:          "Groq",
		BaseURL:        "https://api.groq.com/openai/v1",
		EnvVar:         "GROQ_API_KEY",
		ModelsPath:     "/models",
		APIType:        "openai",
		ExcludePatterns: []string{"whisper", "guard", "distil", "gemma-7b-it", "llava"},
		Models: []ModelDef{
			{ID: "llama-3.3-70b-versatile", Label: "Llama 3.3 70B"},
			{ID: "llama-3.1-8b-instant", Label: "Llama 3.1 8B Instant"},
			{ID: "deepseek-r1-distill-llama-70b", Label: "Deepseek R1 Distill 70B", Reasoner: true},
		},
	},
	{
		Name:           "together",
		Label:          "Together AI",
		BaseURL:        "https://api.together.xyz/v1",
		EnvVar:         "TOGETHER_API_KEY",
		ModelsPath:     "/models",
		APIType:        "openai",
		ExcludePatterns: []string{"embedding", "code", "vision", "inpaint", "flux", "stable", "sdxl"},
		Models: []ModelDef{
			{ID: "meta-llama/Llama-3.3-70B-Instruct-Turbo", Label: "Llama 3.3 70B Turbo"},
			{ID: "meta-llama/Meta-Llama-3.1-8B-Instruct-Turbo", Label: "Llama 3.1 8B Turbo"},
			{ID: "deepseek-ai/DeepSeek-R1", Label: "Deepseek R1", Reasoner: true},
		},
	},
	{
		Name:       "mistral",
		Label:      "Mistral",
		BaseURL:    "https://api.mistral.ai/v1",
		EnvVar:     "MISTRAL_API_KEY",
		ModelsPath: "/models",
		APIType:    "openai",
		ExcludePatterns: []string{"embed", "codestral", "pixtral", "ministral"},
		Models: []ModelDef{
			{ID: "mistral-small-latest", Label: "Mistral Small"},
			{ID: "mistral-large-latest", Label: "Mistral Large"},
		},
	},
	{
		Name:       "xai",
		Label:      "xAI (Grok)",
		BaseURL:    "https://api.x.ai/v1",
		EnvVar:     "XAI_API_KEY",
		ModelsPath: "/models",
		APIType:    "openai",
		Models: []ModelDef{
			{ID: "grok-3-mini", Label: "Grok 3 Mini", Reasoner: true},
			{ID: "grok-3", Label: "Grok 3"},
			{ID: "grok-2", Label: "Grok 2"},
		},
	},
	{
		Name:       "perplexity",
		Label:      "Perplexity",
		BaseURL:    "https://api.perplexity.ai",
		EnvVar:     "PERPLEXITY_API_KEY",
		ModelsPath: "/models",
		APIType:    "openai",
		Models: []ModelDef{
			{ID: "sonar", Label: "Sonar"},
			{ID: "sonar-pro", Label: "Sonar Pro"},
			{ID: "sonar-reasoning", Label: "Sonar Reasoning", Reasoner: true},
		},
	},
	{
		Name:       "openrouter",
		Label:      "OpenRouter",
		BaseURL:    "https://openrouter.ai/api/v1",
		EnvVar:     "OPENROUTER_API_KEY",
		ModelsPath: "/models",
		APIType:    "openai",
		Models: []ModelDef{
			{ID: "anthropic/claude-3.5-sonnet", Label: "Claude 3.5 Sonnet"},
			{ID: "openai/gpt-4o", Label: "GPT-4o"},
			{ID: "google/gemini-2.5-pro", Label: "Gemini 2.5 Pro"},
			{ID: "deepseek/deepseek-chat", Label: "Deepseek Chat"},
		},
	},
	{
		Name:       "google",
		Label:      "Google AI",
		BaseURL:    "https://generativelanguage.googleapis.com/v1beta/openai",
		EnvVar:     "GOOGLE_API_KEY",
		ModelsPath: "/models",
		APIType:    "openai",
		FilterPrefixes: []string{"gemini-"},
		Models: []ModelDef{
			{ID: "gemini-2.0-flash", Label: "Gemini 2.0 Flash"},
			{ID: "gemini-2.5-pro", Label: "Gemini 2.5 Pro"},
			{ID: "gemini-2.5-flash", Label: "Gemini 2.5 Flash"},
		},
	},
}

func Get(name string) *ProviderDef {
	for i := range Definitions {
		if Definitions[i].Name == name {
			return &Definitions[i]
		}
	}
	return nil
}

func Labels() []string {
	labels := make([]string, len(Definitions))
	for i, p := range Definitions {
		labels[i] = p.Name
	}
	return labels
}

// --- Dynamic model cache ---

var (
	modelCache   = make(map[string][]ModelDef)
	cacheTime    = make(map[string]time.Time)
	cacheMu      sync.RWMutex
	cacheTTL     = 1 * time.Hour

	safeHTTPClient = &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			ForceAttemptHTTP2:     false,
			MaxIdleConns:          5,
			IdleConnTimeout:       30 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSClientConfig: &tls.Config{
				NextProtos: []string{"http/1.1"},
			},
		},
	}
)

// GetModels returns the models for a provider, fetching from the API if needed.
// Falls back to hardcoded models if the API call fails.
func GetModels(ctx context.Context, providerName, apiKey string) []ModelDef {
	def := Get(providerName)
	if def == nil {
		return nil
	}

	if apiKey == "" || def.ModelsPath == "" {
		return def.Models
	}

	cacheMu.RLock()
	cached, ok := modelCache[providerName]
	cachedAt, hasTime := cacheTime[providerName]
	cacheMu.RUnlock()

	if ok && hasTime && time.Since(cachedAt) < cacheTTL {
		return cached
	}

	fetched := fetchModelsFromAPI(ctx, def, apiKey)
	if len(fetched) == 0 {
		return def.Models
	}

	cacheMu.Lock()
	modelCache[providerName] = fetched
	cacheTime[providerName] = time.Now()
	cacheMu.Unlock()

	return fetched
}

// InvalidateCache clears the model cache for a provider (or all if empty)
func InvalidateCache(providerName string) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if providerName == "" {
		modelCache = make(map[string][]ModelDef)
		cacheTime = make(map[string]time.Time)
	} else {
		delete(modelCache, providerName)
		delete(cacheTime, providerName)
	}
}

func fetchModelsFromAPI(ctx context.Context, def *ProviderDef, apiKey string) (result []ModelDef) {
	defer func() {
		if r := recover(); r != nil {
			result = nil
		}
	}()

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	url := strings.TrimRight(def.BaseURL, "/") + def.ModelsPath

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil
	}

	switch def.APIType {
	case "anthropic":
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	default:
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := safeHTTPClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil
	}

	var apiResp struct {
		Data []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil
	}

	for _, m := range apiResp.Data {
		if m.ID == "" {
			continue
		}
		if !shouldIncludeModel(def, m.ID) {
			continue
		}
		label := m.DisplayName
		if label == "" {
			label = prettifyModelID(m.ID)
		}
		result = append(result, ModelDef{
			ID:       m.ID,
			Label:    label,
			Reasoner: guessReasoner(m.ID),
		})
	}

	return result
}

func shouldIncludeModel(def *ProviderDef, modelID string) bool {
	// Check exclusions first
	for _, pattern := range def.ExcludePatterns {
		if strings.Contains(strings.ToLower(modelID), strings.ToLower(pattern)) {
			return false
		}
	}

	// Check prefix filter
	if len(def.FilterPrefixes) > 0 {
		matched := false
		for _, prefix := range def.FilterPrefixes {
			if strings.HasPrefix(modelID, prefix) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

func prettifyModelID(id string) string {
	parts := strings.Split(id, "/")
	name := parts[len(parts)-1]
	return strings.ToUpper(name[:1]) + name[1:]
}

// guessReasoner tries to determine if a model is a reasoning model based on its name
func guessReasoner(modelID string) bool {
	lower := strings.ToLower(modelID)
	patterns := []string{"reasoner", "reasoning", "r1", "o1", "o3", "o4-mini", "think", "deepseek-r1", "grok-3-mini"}
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func ModelLabel(providerName, modelID string) string {
	p := Get(providerName)
	if p == nil {
		return modelID
	}
	for _, m := range p.Models {
		if m.ID == modelID {
			return m.Label
		}
	}
	// Check cache
	cacheMu.RLock()
	cached, ok := modelCache[providerName]
	cacheMu.RUnlock()
	if ok {
		for _, m := range cached {
			if m.ID == modelID {
				return m.Label
			}
		}
	}
	return prettifyModelID(modelID)
}

func IsReasoner(providerName, modelID string) bool {
	p := Get(providerName)
	if p != nil {
		for _, m := range p.Models {
			if m.ID == modelID {
				return m.Reasoner
			}
		}
	}
	// Check cache
	cacheMu.RLock()
	cached, ok := modelCache[providerName]
	cacheMu.RUnlock()
	if ok {
		for _, m := range cached {
			if m.ID == modelID {
				return m.Reasoner
			}
		}
	}
	// Fallback: guess from name
	return guessReasoner(modelID)
}

// PrefetchModels fetches models for all configured providers in parallel
func PrefetchModels(ctx context.Context, providers map[string]string) {
	var wg sync.WaitGroup
	for name, apiKey := range providers {
		if apiKey == "" {
			continue
		}
		wg.Add(1)
		go func(name, apiKey string) {
			defer wg.Done()
			defer func() { recover() }()
			GetModels(ctx, name, apiKey)
		}(name, apiKey)
	}
	wg.Wait()
}

func FormatModelsList(models []ModelDef) string {
	var sb strings.Builder
	for i, m := range models {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("%s (%s)", m.Label, m.ID))
	}
	return sb.String()
}
