package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Stream   bool            `json:"stream"`
	Messages []ollamaMessage `json:"messages"`
	Options  map[string]any  `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatResponse struct {
	Message ollamaMessage `json:"message"`
	Error   string        `json:"error,omitempty"`
}

func callOllama(host, model string, prompt reviewPrompt) (string, error) {
	host = strings.TrimRight(host, "/")
	url := host + "/api/chat"

	body, err := json.Marshal(ollamaChatRequest{
		Model:  model,
		Stream: false,
		Messages: []ollamaMessage{
			{Role: "system", Content: prompt.System},
			{Role: "user", Content: prompt.User},
		},
		Options: map[string]any{
			"temperature": 0.1,
		},
	})
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 10 * time.Minute}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama request failed (is Ollama running at %s?): %w", host, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("ollama HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var parsed ollamaChatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("decode ollama response: %w", err)
	}
	if parsed.Error != "" {
		return "", fmt.Errorf("ollama: %s", parsed.Error)
	}
	content := strings.TrimSpace(parsed.Message.Content)
	if content == "" {
		return "", fmt.Errorf("ollama returned empty content")
	}
	return normalizeReviewOutput(content), nil
}

// normalizeReviewOutput unwraps accidental JSON / fence wrappers so the
// saved review is readable markdown.
func normalizeReviewOutput(content string) string {
	content = strings.TrimSpace(content)

	if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSpace(content)
		if nl := strings.IndexByte(content, '\n'); nl >= 0 {
			lang := strings.TrimSpace(content[:nl])
			if lang == "json" || lang == "markdown" || lang == "md" || lang == "" {
				content = content[nl+1:]
			}
		}
		content = strings.TrimSpace(content)
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}

	if strings.HasPrefix(content, "{") {
		var wrap struct {
			Response string `json:"response"`
			Content  string `json:"content"`
			Review   string `json:"review"`
		}
		if err := json.Unmarshal([]byte(content), &wrap); err == nil {
			for _, candidate := range []string{wrap.Response, wrap.Content, wrap.Review} {
				candidate = strings.TrimSpace(candidate)
				if candidate != "" {
					// JSON may store markdown with escaped newlines; Unmarshal already unescapes.
					return strings.TrimSpace(candidate)
				}
			}
		}
	}

	return content
}
