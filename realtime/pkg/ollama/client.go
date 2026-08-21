// Package ollama provides a lightweight HTTP client for Ollama LLM-based
// sentiment analysis of XAUUSD news headlines.
//
// Design: graceful degradation — if Ollama is unavailable, times out, or
// returns an error, the sentiment score defaults to 0.0 (neutral). This
// never blocks the primary inference path.
package ollama

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// OllamaClient wraps an HTTP client for the Ollama API.
type OllamaClient struct {
	host    string
	model   string
	timeout time.Duration
	client  *http.Client
	enabled bool
	mu      sync.Mutex
}

// DefaultClient creates an OllamaClient from environment variables.
// OLLAMA_HOST defaults to http://localhost:11434.
// OLLAMA_MODEL defaults to deepseek-v4-pro:cloud.
// OLLAMA_TIMEOUT defaults to 2s.
func DefaultClient() *OllamaClient {
	host := os.Getenv("OLLAMA_HOST")
	if host == "" {
		host = "http://localhost:11434"
	}
	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = "deepseek-v4-pro:cloud"
	}
	timeoutStr := os.Getenv("OLLAMA_TIMEOUT")
	timeout := 2 * time.Second
	if timeoutStr != "" {
		if d, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = d
		}
	}
	return &OllamaClient{
		host:    host,
		enabled: true,
		model:   model,
		timeout: timeout,
		client:  &http.Client{Timeout: timeout},
	}
}

// NewClient creates an OllamaClient with explicit parameters.
func NewClient(host, model string, timeout time.Duration) *OllamaClient {
	return &OllamaClient{
		host:    host,
		enabled: true,
		model:   model,
		timeout: timeout,
		client:  &http.Client{Timeout: timeout},
	}
}

// sentimentResponse is the JSON we expect from the LLM.
type sentimentResponse struct {
	Sentiment float64 `json:"sentiment"`
	Reason    string  `json:"reason"`
}

// ollamaRequest is the Ollama API request body.
type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// ollamaResponse is the Ollama API response body.
type ollamaResponse struct {
	Model     string `json:"model"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
}

// GetNewsSentiment sends headlines to Ollama and returns a sentiment score
// from -1.0 (bearish) to 1.0 (bullish). Returns 0.0 on any error (graceful
// degradation — never blocks trading).
func (c *OllamaClient) GetNewsSentiment(headlines []string) (float64, error) {
	if len(headlines) == 0 {
		return 0.0, nil
	}

	// Build the prompt
	headlineList := strings.Join(headlines, "\n- ")
	prompt := fmt.Sprintf(
		"Analyze the following gold (XAUUSD) related headlines. "+
			"Return ONLY a JSON object with keys: 'sentiment' (float from -1.0 bearish to 1.0 bullish) "+
			"and 'reason' (string). Headlines:\n- %s",
		headlineList,
	)

	// Build request
	reqBody := ollamaRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
	}
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return 0.0, fmt.Errorf("marshal request: %w", err)
	}

	// POST to Ollama API
	req, err := http.NewRequest("POST", c.host+"/api/generate", strings.NewReader(string(reqBytes)))
	if err != nil {
		return 0.0, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		// Graceful degradation: return neutral on network error/timeout
		return 0.0, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0.0, fmt.Errorf("ollama HTTP %d: %s", resp.StatusCode, string(body[:min(len(body), 200)]))
	}

	// Parse Ollama response
	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return 0.0, fmt.Errorf("decode ollama response: %w", err)
	}

	// The LLM response is in ollamaResp.Response — parse the JSON sentiment
	// It may be wrapped in markdown code blocks or have extra text
	sentimentText := extractJSON(ollamaResp.Response)
	if sentimentText == "" {
		return 0.0, fmt.Errorf("no JSON found in LLM response")
	}

	var sentiment sentimentResponse
	if err := json.Unmarshal([]byte(sentimentText), &sentiment); err != nil {
		return 0.0, fmt.Errorf("parse sentiment JSON: %w", err)
	}

	// Clamp to [-1.0, 1.0]
	score := sentiment.Sentiment
	if score > 1.0 {
		score = 1.0
	} else if score < -1.0 {
		score = -1.0
	}

	return score, nil
}

// extractJSON finds the first JSON object in a string that may contain
// markdown code blocks or extra text.
func extractJSON(s string) string {
	// Try to find ```json ... ``` block
	if idx := strings.Index(s, "```json"); idx >= 0 {
		start := idx + 7
		end := strings.Index(s[start:], "```")
		if end >= 0 {
			return strings.TrimSpace(s[start : start+end])
		}
	}
	// Try to find ``` ... ``` block
	if idx := strings.Index(s, "```"); idx >= 0 {
		start := idx + 3
		end := strings.Index(s[start:], "```")
		if end >= 0 {
			return strings.TrimSpace(s[start : start+end])
		}
	}
	// Try to find { ... } directly
	start := strings.Index(s, "{")
	if start >= 0 {
		end := strings.LastIndex(s, "}")
		if end > start {
			return s[start : end+1]
		}
	}
	return ""
}

// IsEnabled returns true if the Ollama client should be used.
func (c *OllamaClient) IsEnabled() bool {
	return c != nil && c.host != "" && c.enabled
}

// SetEnabled enables or disables the Ollama client at runtime.
func (c *OllamaClient) SetEnabled(enabled bool) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = enabled
}
