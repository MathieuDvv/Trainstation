package router

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"

	"trainstation/agent"
	"trainstation/config"
	"trainstation/provider"
)

var safeHTTPClient = &http.Client{
	Timeout: 60 * time.Second,
	Transport: &http.Transport{
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          5,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   15 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSClientConfig: &tls.Config{
			NextProtos: []string{"http/1.1"},
		},
	},
}

type Router struct {
	client    *openai.Client
	provider  string
	model     string
	thinking  string
	registry  *agent.Registry
	strengths map[string][]string
	available []string
}

func New(cfg *config.Config, registry *agent.Registry) (*Router, error) {
	apiKey := cfg.GetAPIKey(cfg.Router.Provider)
	if apiKey == "" {
		return nil, fmt.Errorf("no API key configured for provider %s — use /provider to add one", cfg.Router.Provider)
	}

	baseURL := cfg.GetBaseURL(cfg.Router.Provider)

	clientOpts := []option.RequestOption{
		option.WithAPIKey(apiKey),
		option.WithHTTPClient(safeHTTPClient),
	}
	if baseURL != "" {
		clientOpts = append(clientOpts, option.WithBaseURL(baseURL))
	}

	client := openai.NewClient(clientOpts...)

	thinking := cfg.Router.ThinkingLevel
	if thinking == "" {
		thinking = "medium"
	}

	return &Router{
		client:    &client,
		provider:  cfg.Router.Provider,
		model:     cfg.Router.Model,
		thinking:  thinking,
		registry:  registry,
		strengths: registry.Strengths(cfg),
		available: registry.Available(),
	}, nil
}

func (r *Router) Route(ctx context.Context, userPrompt string) (result *TaskPlan, err error) {
	defer func() {
		if rv := recover(); rv != nil {
			result = nil
			err = fmt.Errorf("router panicked: %v", rv)
		}
	}()

	systemPrompt := buildSystemPrompt(r.strengths, r.available, r.thinking)

	jsonObjectParam := shared.NewResponseFormatJSONObjectParam()

	params := openai.ChatCompletionNewParams{
		Model:    shared.ChatModel(r.model),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(userPrompt),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &jsonObjectParam,
		},
	}

	if provider.IsReasoner(r.provider, r.model) {
		switch r.thinking {
		case "low":
			params.ReasoningEffort = shared.ReasoningEffortLow
		case "high":
			params.ReasoningEffort = shared.ReasoningEffortHigh
		case "max":
			params.ReasoningEffort = shared.ReasoningEffortHigh
		default:
			params.ReasoningEffort = shared.ReasoningEffortMedium
		}
	}

	resp, err := r.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("router LLM call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("router returned no choices")
	}

	content := resp.Choices[0].Message.Content
	if content == "" {
		return nil, fmt.Errorf("router returned empty content")
	}

	content = cleanJSON(content)

	var plan TaskPlan
	if err := json.Unmarshal([]byte(content), &plan); err != nil {
		return nil, fmt.Errorf("failed to parse router response: %w\nraw: %s", err, content)
	}

	if len(plan.Tasks) == 0 {
		return nil, fmt.Errorf("router returned no tasks")
	}

	plan = r.validatePlan(plan)

	return &plan, nil
}

func (r *Router) validatePlan(plan TaskPlan) TaskPlan {
	availableSet := make(map[string]bool)
	for _, a := range r.available {
		availableSet[a] = true
	}

	for i := range plan.Tasks {
		if !availableSet[plan.Tasks[i].Agent] {
			if len(r.available) > 0 {
				plan.Tasks[i].Agent = r.available[0]
			}
		}
		if plan.Tasks[i].DependsOn == nil {
			plan.Tasks[i].DependsOn = []int{}
		}
	}

	return plan
}

func (r *Router) Model() string         { return r.model }
func (r *Router) ProviderName() string  { return r.provider }
func (r *Router) Thinking() string      { return r.thinking }
func (r *Router) SetThinking(level string) { r.thinking = level }

func (r *Router) UpdateAvailable(available []string) {
	if len(available) > 0 {
		r.available = available
	}
}

func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
