package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildOllamaChatRequest_SetsNumCtxAndDisablesThink(t *testing.T) {
	req := buildOllamaChatRequest("qwen2.5-coder:7b", reviewPrompt{
		System: "system",
		User:   "user with a diff",
	}, 16384)

	assert.Equal(t, "qwen2.5-coder:7b", req.Model)
	assert.False(t, req.Stream)
	assert.False(t, req.Think)
	require.Len(t, req.Messages, 2)
	assert.Equal(t, "system", req.Messages[0].Role)
	assert.Equal(t, "user", req.Messages[1].Role)
	assert.Equal(t, 0.1, req.Options["temperature"])
	assert.Equal(t, 16384, req.Options["num_ctx"])

	raw, err := json.Marshal(req)
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"num_ctx":16384`)
	assert.Contains(t, string(raw), `"think":false`)
}

func TestFitNumCtx_ScalesToPromptAndCapsFor8GBVRAM(t *testing.T) {
	t.Setenv("OLLAMA_NUM_CTX", "")

	assert.Equal(t, minNumCtx, fitNumCtx(1000))    // tiny → floor
	assert.Equal(t, 12288, fitNumCtx(40_000))      // ~10k tokens + headroom
	assert.Equal(t, maxNumCtx, fitNumCtx(200_000)) // huge → 16k cap (8GB VRAM)
}

func TestFitNumCtx_EnvOverrideWins(t *testing.T) {
	t.Setenv("OLLAMA_NUM_CTX", "24576")
	assert.Equal(t, 24576, fitNumCtx(200_000))
}

func TestMaxNumCtx_Fits8GBVRAM(t *testing.T) {
	assert.Equal(t, 16384, maxNumCtx)
	assert.GreaterOrEqual(t, maxNumCtx, minNumCtx)
}

func TestDefaultReviewModel_IsCoder7b(t *testing.T) {
	assert.Equal(t, "qwen2.5-coder:7b", defaultReviewModel)
}
