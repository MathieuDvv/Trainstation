package usage

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"trainstation/config"
	"trainstation/provider"
)

var httpClient = &http.Client{
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

type AgentUsage struct {
	Name     string
	LoggedIn bool
	Plan     string
	Detail   string
	Error    string
}

func (u AgentUsage) StatusLine() string {
	if u.Error != "" {
		return u.Error
	}
	if !u.LoggedIn {
		return "not logged in"
	}
	parts := []string{}
	if u.Plan != "" {
		parts = append(parts, u.Plan)
	}
	if u.Detail != "" {
		parts = append(parts, u.Detail)
	}
	if len(parts) == 0 {
		return "active"
	}
	return strings.Join(parts, " · ")
}

type ProviderUsage struct {
	Name    string
	Balance string
	Error   string
}

type Snapshot struct {
	Agents    map[string]AgentUsage
	Providers map[string]ProviderUsage
	FetchedAt time.Time
}

func (s *Snapshot) AvailableAgents() []string {
	var avail []string
	for name, u := range s.Agents {
		if u.Error == "" && u.LoggedIn {
			// For claude, check if we're not explicitly out of usage
			if strings.Contains(strings.ToLower(u.Detail), "no usage left") || strings.Contains(strings.ToLower(u.StatusLine()), "limit reached") {
				continue
			}
			avail = append(avail, name)
		}
	}
	return avail
}

func FetchAll(ctx context.Context, cfg *config.Config, enabledAgents map[string]bool) *Snapshot {
	snap := &Snapshot{
		Agents:    make(map[string]AgentUsage),
		Providers: make(map[string]ProviderUsage),
		FetchedAt: time.Now(),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	agentFetchers := map[string]func(context.Context) AgentUsage{
		"claude":      fetchClaude,
		"codex":       fetchCodex,
		"opencode":    fetchOpenCode,
		"antigravity": fetchAntigravity,
	}

	for name, fn := range agentFetchers {
		if !enabledAgents[name] {
			continue
		}
		wg.Add(1)
		go func(name string, fn func(context.Context) AgentUsage) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					mu.Lock()
					snap.Agents[name] = AgentUsage{Name: name, Error: "fetch panicked"}
					mu.Unlock()
				}
			}()
			u := fn(ctx)
			u.Name = name
			mu.Lock()
			snap.Agents[name] = u
			mu.Unlock()
		}(name, fn)
	}

	for _, provName := range cfg.ConfiguredProviders() {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					mu.Lock()
					snap.Providers[name] = ProviderUsage{Name: name, Error: "fetch panicked"}
					mu.Unlock()
				}
			}()
			u := fetchProviderBalance(ctx, cfg, name)
			mu.Lock()
			snap.Providers[name] = u
			mu.Unlock()
		}(provName)
	}

	wg.Wait()
	return snap
}

func runWithTimeout(ctx context.Context, name string, args ...string) (output []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("command panicked: %v", r)
			output = nil
		}
	}()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Output()
}

func fetchClaude(ctx context.Context) AgentUsage {
	u := AgentUsage{Name: "claude"}

	data, err := runWithTimeout(ctx, "claude", "auth", "status")
	if err != nil {
		u.Error = "unavailable"
		return u
	}

	checkCmd := exec.CommandContext(ctx, "sh", "-c", "echo '' | claude usage 2>&1")
	out, _ := checkCmd.CombinedOutput()
	if strings.Contains(string(out), "disabled Claude subscription access") {
		u.Error = "no usage left"
		return u
	}

	var status struct {
		LoggedIn         bool   `json:"loggedIn"`
		AuthMethod       string `json:"authMethod"`
		Email            string `json:"email"`
		SubscriptionType string `json:"subscriptionType"`
	}
	if err := json.Unmarshal(data, &status); err != nil {
		u.Error = "parse error"
		return u
	}

	u.LoggedIn = status.LoggedIn
	if status.SubscriptionType != "" {
		u.Plan = strings.ToUpper(status.SubscriptionType[:1]) + status.SubscriptionType[1:]
	} else if status.AuthMethod == "claude.ai" {
		u.Plan = "Claude.ai"
	}
	if status.AuthMethod == "apiKey" {
		u.Plan = "API"
	}
	return u
}

func fetchCodex(ctx context.Context) AgentUsage {
	u := AgentUsage{Name: "codex"}

	data, err := runWithTimeout(ctx, "codex", "login", "status")
	if err != nil {
		data2, err2 := readCodexAuthFile()
		if err2 != nil {
			u.Error = "unavailable"
			return u
		}
		data = data2
	} else {
		output := strings.TrimSpace(string(data))
		u.LoggedIn = !strings.Contains(strings.ToLower(output), "not logged in")
		if strings.Contains(output, "ChatGPT") {
			u.Plan = "ChatGPT"
		} else if strings.Contains(output, "API key") {
			u.Plan = "API"
		}
		if u.LoggedIn {
			return u
		}
		data, _ = readCodexAuthFile()
	}

	var auth struct {
		AuthMode string `json:"auth_mode"`
	}
	_ = json.Unmarshal(data, &auth)

	if auth.AuthMode == "chatgpt" {
		u.LoggedIn = true
		u.Plan = "ChatGPT"
		var full struct {
			Tokens struct {
				IDToken string `json:"id_token"`
			} `json:"tokens"`
		}
		_ = json.Unmarshal(data, &full)
		if full.Tokens.IDToken != "" {
			planType := extractJWTClaim(full.Tokens.IDToken, "chatgpt_plan_type")
			if planType != "" {
				u.Plan = strings.Title(planType)
			}
			until := extractJWTClaim(full.Tokens.IDToken, "chatgpt_subscription_active_until")
			if len(until) >= 10 {
				u.Detail = "until " + until[:10]
			} else if until != "" {
				u.Detail = "until " + until
			}
		}
	} else if auth.AuthMode == "apikey" {
		u.LoggedIn = true
		u.Plan = "API"
	}

	return u
}

func readCodexAuthFile() ([]byte, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return os.ReadFile(filepath.Join(home, ".codex", "auth.json"))
}

func extractJWTClaim(token, claim string) string {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return ""
	}
	payload := parts[1]
	if l := len(payload) % 4; l != 0 {
		payload += strings.Repeat("=", 4-l)
	}
	decoded, err := base64Decode(payload)
	if err != nil {
		return ""
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return ""
	}
	apiAuth, ok := claims["https://api.openai.com/auth"].(map[string]interface{})
	if !ok {
		return ""
	}
	val, _ := apiAuth[claim].(string)
	return val
}

func base64Decode(s string) ([]byte, error) {
	return base64.URLEncoding.DecodeString(s)
}

func fetchOpenCode(ctx context.Context) AgentUsage {
	u := AgentUsage{Name: "opencode"}

	data, err := runWithTimeout(ctx, "opencode", "stats")
	if err != nil {
		u.Error = "unavailable"
		return u
	}

	output := string(data)
	u.LoggedIn = true

	costRe := regexp.MustCompile(`Total Cost\s+\$([\d.]+)`)
	if m := costRe.FindStringSubmatch(output); len(m) > 1 {
		u.Detail = "$" + m[1] + " used"
	}

	msgRe := regexp.MustCompile(`Messages\s+(\d+)`)
	if m := msgRe.FindStringSubmatch(output); len(m) > 1 {
		if u.Detail != "" {
			u.Detail += " · " + m[1] + " msgs"
		} else {
			u.Detail = m[1] + " msgs"
		}
	}

	return u
}

func fetchAntigravity(ctx context.Context) AgentUsage {
	u := AgentUsage{Name: "antigravity"}

	home, err := os.UserHomeDir()
	if err == nil {
		settingsData, err := os.ReadFile(filepath.Join(home, ".gemini", "antigravity-cli", "settings.json"))
		if err == nil {
			var settings struct {
				Model string `json:"model"`
			}
			_ = json.Unmarshal(settingsData, &settings)
			if settings.Model != "" {
				u.LoggedIn = true
				u.Plan = "Gemini"
				u.Detail = settings.Model
				return u
			}
		}
	}

	data, err := runWithTimeout(ctx, "agy", "models")
	if err != nil {
		u.Error = "unavailable"
		return u
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) > 0 {
		u.LoggedIn = true
		u.Plan = strconv.Itoa(len(lines)) + " models"
	}

	return u
}

func fetchProviderBalance(ctx context.Context, cfg *config.Config, provName string) (u ProviderUsage) {
	defer func() {
		if r := recover(); r != nil {
			u = ProviderUsage{Name: provName, Error: "fetch error"}
		}
	}()
	u = ProviderUsage{Name: provName}

	def := provider.Get(provName)
	if def == nil || def.BalanceURL == "" {
		u.Error = "no balance API"
		return u
	}

	apiKey := cfg.GetAPIKey(provName)
	if apiKey == "" {
		u.Error = "no API key"
		return u
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", def.BalanceURL, nil)
	if err != nil {
		u.Error = "request error"
		return u
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		u.Error = "fetch error"
		return u
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		u.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		return u
	}

	var body struct {
		IsAvailable  bool `json:"is_available"`
		BalanceInfos []struct {
			Currency      string `json:"currency"`
			TotalBalance  string `json:"total_balance"`
			GrantedBalance string `json:"granted_balance"`
		} `json:"balance_infos"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		u.Error = "parse error"
		return u
	}

	if len(body.BalanceInfos) > 0 {
		info := body.BalanceInfos[0]
		balance := info.TotalBalance
		if balance != "" {
			u.Balance = "$" + balance
			if info.Currency == "CNY" {
				u.Balance = "¥" + balance
			}
		}
	}

	if u.Balance == "" {
		u.Error = "no balance data"
	}

	return u
}
