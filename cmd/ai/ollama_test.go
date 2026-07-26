package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildOllamaChatRequest_SetsNumCtxAndDisablesThink(t *testing.T) {
	req := buildOllamaChatRequest("qwen3:8b", reviewPrompt{
		System: "system",
		User:   "user with a diff",
	}, 32768)

	assert.Equal(t, "qwen3:8b", req.Model)
	assert.False(t, req.Stream)
	assert.False(t, req.Think)
	require.Len(t, req.Messages, 2)
	assert.Equal(t, "system", req.Messages[0].Role)
	assert.Equal(t, "user", req.Messages[1].Role)
	assert.Equal(t, 0.1, req.Options["temperature"])
	assert.Equal(t, 32768, req.Options["num_ctx"])

	raw, err := json.Marshal(req)
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"num_ctx":32768`)
	assert.Contains(t, string(raw), `"think":false`)
}

func TestDefaultNumCtx_IsLargeEnoughForReviewPrompts(t *testing.T) {
	// Ollama defaults to 4096 on most machines; review prompts are typically 6–12k tokens.
	assert.GreaterOrEqual(t, defaultNumCtx, 16384)
}
