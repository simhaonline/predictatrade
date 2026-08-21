package ollama

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDefaultClient(t *testing.T) {
	c := DefaultClient()
	if c.host == "" {
		t.Error("Default client host should not be empty")
	}
	if c.model == "" {
		t.Error("Default client model should not be empty")
	}
	if c.timeout == 0 {
		t.Error("Default client timeout should not be zero")
	}
}

func TestNewClient(t *testing.T) {
	c := NewClient("http://test:1234", "test-model", 5*time.Second)
	if c.host != "http://test:1234" || c.model != "test-model" {
		t.Error("Client params mismatch")
	}
}

func TestExtractJSON(t *testing.T) {
	// Test markdown json block
	s := "Here is the result:\n```json\n{\"sentiment\": 0.5, \"reason\": \"bullish\"}\n```\nDone."
	result := extractJSON(s)
	if result == "" {
		t.Error("extractJSON should find markdown json block")
	}
	var sr sentimentResponse
	if err := json.Unmarshal([]byte(result), &sr); err != nil {
		t.Errorf("extracted JSON should be valid: %v", err)
	}
	if sr.Sentiment != 0.5 {
		t.Errorf("sentiment should be 0.5, got %f", sr.Sentiment)
	}
}

func TestExtractJSONRaw(t *testing.T) {
	// Test raw JSON
	s := `Some text {"sentiment": -0.3, "reason": "bearish"} more text`
	result := extractJSON(s)
	var sr sentimentResponse
	if err := json.Unmarshal([]byte(result), &sr); err != nil {
		t.Errorf("raw JSON extraction failed: %v", err)
	}
	if sr.Sentiment != -0.3 {
		t.Errorf("sentiment should be -0.3, got %f", sr.Sentiment)
	}
}

func TestExtractJSONEmpty(t *testing.T) {
	result := extractJSON("no json here")
	if result != "" {
		t.Error("extractJSON should return empty for no JSON")
	}
}

func TestGetNewsSentimentEmpty(t *testing.T) {
	c := DefaultClient()
	score, err := c.GetNewsSentiment([]string{})
	if err != nil {
		t.Errorf("Empty headlines should not error: %v", err)
	}
	if score != 0.0 {
		t.Errorf("Empty headlines should return 0.0, got %f", score)
	}
}

func TestGetNewsSentimentMockServer(t *testing.T) {
	// Mock Ollama server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"model":    "test-model",
			"response": `{"sentiment": 0.8, "reason": "bullish gold news"}`,
			"done":     true,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-model", 2*time.Second)
	score, err := c.GetNewsSentiment([]string{"Gold prices surge on weak dollar"})
	if err != nil {
		t.Errorf("Mock server should not error: %v", err)
	}
	if score != 0.8 {
		t.Errorf("Sentiment should be 0.8, got %f", score)
	}
}

func TestGetNewsSentimentMockServerMarkdown(t *testing.T) {
	// Mock server returning markdown-wrapped JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"model":    "test-model",
			"response": "```json\n{\"sentiment\": -0.6, \"reason\": \"bearish\"}\n```",
			"done":     true,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-model", 2*time.Second)
	score, err := c.GetNewsSentiment([]string{"Gold drops sharply"})
	if err != nil {
		t.Errorf("Markdown mock should not error: %v", err)
	}
	if score != -0.6 {
		t.Errorf("Sentiment should be -0.6, got %f", score)
	}
}

func TestGetNewsSentimentServerError(t *testing.T) {
	// Mock server returning 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-model", 2*time.Second)
	score, err := c.GetNewsSentiment([]string{"test"})
	if err == nil {
		t.Error("Server error should return error")
	}
	if score != 0.0 {
		t.Errorf("Error should return 0.0, got %f", score)
	}
}

func TestGetNewsSentimentClamp(t *testing.T) {
	// Mock server returning out-of-range sentiment
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"response": `{"sentiment": 5.0, "reason": "extreme"}`,
			"done":     true,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient(server.URL, "test-model", 2*time.Second)
	score, _ := c.GetNewsSentiment([]string{"test"})
	if score != 1.0 {
		t.Errorf("Sentiment should be clamped to 1.0, got %f", score)
	}
}

func TestIsEnabled(t *testing.T) {
	c := DefaultClient()
	if !c.IsEnabled() {
		t.Error("Default client should be enabled")
	}
}
