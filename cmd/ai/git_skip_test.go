package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldSkipUntracked_SkipsDocsAndMarkdowns(t *testing.T) {
	assert.True(t, shouldSkipUntracked("docs/TENANT_WHATSAPP_PHONE.md"))
	assert.True(t, shouldSkipUntracked("markdowns/MVP_HAPPY_PATH_CHECKLIST.md"))
	assert.True(t, shouldSkipUntracked(".ai/reviews/review-1.md"))
	assert.False(t, shouldSkipUntracked("internal/handlers/orders.go"))
	assert.False(t, shouldSkipUntracked("migrations/000045_add_whatsapp_phone_to_tenants.up.sql"))
}

func TestReviewDiffPathspecs_ExcludesDocTrees(t *testing.T) {
	specs := reviewDiffPathspecs()
	assert.Contains(t, specs, ".")
	assert.Contains(t, specs, ":(exclude)docs")
	assert.Contains(t, specs, ":(exclude)markdowns")
	assert.Contains(t, specs, ":(exclude).ai/reviews")
}
