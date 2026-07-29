package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	// maxNumCtx caps KV-cache for ~8GB VRAM machines. 32k forces CPU spill/OOM thrash.
	maxNumCtx = 16384
	minNumCtx = 8192
	// Extra room for the model's reply after the prompt tokens.
	generationHeadroomTokens = 2048
)

type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Stream   bool            `json:"stream"`
	Think    bool            `json:"think"`
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

// fitNumCtx picks a context window big enough for the prompt, but not a fixed
// oversized desk. OLLAMA_NUM_CTX always wins when set.
func fitNumCtx(promptChars int) int {
	if raw := strings.TrimSpace(os.Getenv("OLLAMA_NUM_CTX")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err == nil && n > 0 {
			return n
		}
	}

	est := promptChars/4 + generationHeadroomTokens
	n := ((est + 1023) / 1024) * 1024
	if n < minNumCtx {
		n = minNumCtx
	}
	if n > maxNumCtx {
		n = maxNumCtx
	}
	return n
}

func buildOllamaChatRequest(model string, prompt reviewPrompt, numCtx int) ollamaChatRequest {
	if numCtx <= 0 {
		numCtx = maxNumCtx
	}
	return ollamaChatRequest{
		Model:  model,
		Stream: false,
		// Thinking burns context; structured reviews don't need it.
		Think: false,
		Messages: []ollamaMessage{
			{Role: "system", Content: prompt.System},
			{Role: "user", Content: prompt.User},
		},
		Options: map[string]any{
			"temperature": 0.1,
			"num_ctx":     numCtx,
		},
	}
}

func callOllama(host, model string, prompt reviewPrompt, numCtx int) (string, error) {
	host = strings.TrimRight(host, "/")
	url := host + "/api/chat"

	body, err := json.Marshal(buildOllamaChatRequest(model, prompt, numCtx))
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 15 * time.Minute}
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
