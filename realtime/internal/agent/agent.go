// Package agent implements the AI Agent Mesh (check.md 2026-08-30).
// Multi-provider consensus for Live Market bias, Risk assessment, Sentiment.
// Providers: 3 local Ollama models + optional OpenAI / DeepSeek / Tencent.
package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ─── Types ──

type AgentConsensus struct {
	Providers  []ProviderResult `json:"providers"`
	Active     int              `json:"providers_active"`
	Consensus  float64          `json:"consensus_score"`
	Agreement  float64          `json:"agreement_pct"`
	RiskLevel  float64          `json:"risk_level"`
	Confidence float64          `json:"confidence"`
	Timestamp  time.Time        `json:"timestamp"`
}

type ProviderResult struct {
	Provider  string  `json:"provider"`
	Active    bool    `json:"active"`
	Score     float64 `json:"score"`
	LatencyMs int64   `json:"latency_ms"`
}

type AgentMesh struct {
	ollamaHost  string
	ollamaModel string
	openAIKey   string
	deepSeekKey string
	tencentKey  string
	timeout     time.Duration
}

func NewAgentMesh() *AgentMesh {
	return &AgentMesh{
		ollamaHost:  strings.TrimRight(envFrom("OLLAMA_HOST", "http://host.docker.internal:11434"), "/"),
		ollamaModel: envFrom("OLLAMA_MODEL", "glm-5.3-flash:cloud"),
		openAIKey:   os.Getenv("OPENAI_API_KEY"),
		deepSeekKey: os.Getenv("DEEPSEEK_API_KEY"),
		tencentKey:  os.Getenv("TENCENT_AI_API_KEY"),
		timeout:     10 * time.Second,
	}
}

// ─── Public agents ──

func (m *AgentMesh) LiveMarketAgent(price, rsi, adx float64, vedic string, marketClosed bool) *AgentConsensus {
	prompt := fmt.Sprintf(
		"You are a XAUUSD trading AI analyst. Live: price %.2f, RSI %.1f, ADX %.1f. Vedic: %s. Give ONLY a bias score -100 to +100. No other text.",
		price, rsi, adx, vedic, marketClosed)
	return m.runMesh("live_market", prompt)
}

func (m *AgentMesh) RiskAgent(spread, exposure, newsRisk string) *AgentConsensus {
	return m.runMesh("risk", fmt.Sprintf(
		"Assess XAUUSD live trade risk. Spread %s, exposure %s, news risk %s. Give ONLY risk score 0-100.", spread, exposure, newsRisk))
}

func (m *AgentMesh) SentimentAgent(headlines string) *AgentConsensus {
	return m.runMesh("sentiment", fmt.Sprintf(
		"Analyze XAUUSD headlines: %s. Give ONLY score -1 to +1.", headlines))
}

// ─── Mesh executor ──

func (m *AgentMesh) runMesh(label string, prompt string) *AgentConsensus {
	result := &AgentConsensus{Timestamp: time.Now().UTC(), Providers: []ProviderResult{}}

	type Result struct {
		name  string
		score float64
		ok    bool
		lat   int64
	}

	providers := m.getProviderMap()
	results := make(chan ProviderResult, len(providers))
	var wg sync.WaitGroup
	for name, fn := range providers {
		wg.Add(1)
		go func(name string, fn func(string) string) {
			defer wg.Done()
			start := time.Now()
			resp := fn(prompt)
			lat := time.Since(start).Milliseconds()
			score := parseScoreLLM(resp)
			results <- ProviderResult{Provider: name, Active: score != 0.0, Score: score, LatencyMs: lat}
		}(name, fn)
	}

	go func() { wg.Wait(); close(results) }()
	for r := range results {
		result.Providers = append(result.Providers, r)
	}

	// Aggregate
	var totalScore float64
	var buy, sell, neutral int
	for _, p := range result.Providers {
		if !p.Active {
			continue
		}
		result.Active++
		totalScore += p.Score
	}
	if result.Active > 0 {
		result.Consensus = totalScore / float64(result.Active)
	}

	agreeCount := 0
	if result.Consensus > 0 {
		agreeCount = buy
	} else if result.Consensus < 0 {
		agreeCount = sell
	} else {
		agreeCount = neutral
	}
	if result.Active > 0 {
		result.Agreement = float64(agreeCount) / float64(result.Active) * 100
	}
	result.RiskLevel = 100 - result.Agreement
	result.Confidence = (result.Agreement / 100) * float64(result.Active) / 3
	return result
}

// ─── Provider registry ──

func (m *AgentMesh) getProviderMap() map[string]func(string) string {
	out := map[string]func(string) string{}
	out["ollama_glm"] = func(prompt string) string {
		return m.httpPostJSON(m.ollamaModel, prompt, m.ollamaHost+"/api/generate")
	}
	out["ollama_deepseek"] = func(prompt string) string {
		return m.httpPostJSON("deepseek-v4-pro:cloud", prompt, m.ollamaHost+"/api/generate")
	}
	out["ollama_gptoss"] = func(prompt string) string {
		return m.httpPostJSON("gpt-oss:120b-cloud", prompt, m.ollamaHost+"/api/generate")
	}
	if m.openAIKey != "" {
		out["openai_gpt4o"] = func(prompt string) string {
			return m.openAICompatible(m.openAIKey, "https://api.openai.com/v1/chat/completions", "gpt-4o-mini", prompt)
		}
	}
	if m.deepSeekKey != "" {
		out["deepseek_cloud"] = func(prompt string) string {
			return m.openAICompatible(m.deepSeekKey, "https://api.deepseek.com/v1/chat/completions", "deepseek-chat", prompt)
		}
	}
	if m.tencentKey != "" {
		out["tencent_hunyuan"] = func(prompt string) string {
			return m.openAICompatible(m.tencentKey, "https://api.hunyuan.cloud.tencent.com/v1/chat/completions", "hunyuan-lite", prompt)
		}
	}
	return out
}

func (m *AgentMesh) httpPostJSON(model, prompt, url string) string {
	body, _ := json.Marshal(map[string]any{"model": model, "prompt": prompt, "stream": false})
	client := http.Client{Timeout: m.timeout}
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var out struct {
		Response string `json:"response"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return out.Response
}

func (m *AgentMesh) openAICompatible(key, url, model, prompt string) string {
	body, _ := json.Marshal(map[string]any{
		"model":    model,
		"messages": []map[string]string{{"role": "user", "content": prompt}},
	})
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	client := http.Client{Timeout: m.timeout}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var out struct {
		Choices []struct{ Message struct{ Content string } }
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if len(out.Choices) > 0 {
		return out.Choices[0].Message.Content
	}
	return ""
}

// ─── Helpers ──

func envOr(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

func extractNumericScore(text string) float64 {
	compiled := regexp.MustCompile(`-?\d+\.?\d*`)
	for _, match := range compiled.FindAllString(text, -1) {
		v, err := strconv.ParseFloat(match, 64)
		if err != nil {
			continue
		}
		if v >= -100 && v <= 100 {
			return v / 100.0
		}
	}
	return 0.0
}

// silence unused imports
var (
	_ = os.Stdout
)

func envFrom(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v != "" {
		return v
	}
	return def
}

func parseScoreLLM(text string) float64 {
	for _, m := range regexp.MustCompile("-?[0-9]+.?[0-9]*").FindAllString(text, -1) {
		v, err := strconv.ParseFloat(m, 64)
		if err != nil {
			continue
		}
		if v >= -100 && v <= 100 {
			return v / 100.0
		}
	}
	return 0.0
}
