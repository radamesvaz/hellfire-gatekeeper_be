package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractDiffPaths(t *testing.T) {
	diff := `diff --git a/model/users/user.go b/model/users/user.go
index 111..222 100644
--- a/model/users/user.go
+++ b/model/users/user.go
@@ -1,3 +1,4 @@
+const UserRoleSuperAdmin = 3
diff --git a/migrations/000040_add_superadmin_role.up.sql b/migrations/000040_add_superadmin_role.up.sql
new file mode 100644
--- /dev/null
+++ b/migrations/000040_add_superadmin_role.up.sql
@@ -0,0 +1,2 @@
+INSERT INTO roles (id_role, name) VALUES (3, 'superadmin');
`
	got := extractDiffPaths(diff)
	assert.Equal(t, []string{
		"migrations/000040_add_superadmin_role.up.sql",
		"model/users/user.go",
	}, got)
}

func TestSanitizeReview_DropsHallucinatedFindings(t *testing.T) {
	review := `## Verdict
BLOCK

## Summary
Critical issues found.

## Findings
- [BLOCK] api/user_data.go — SQL injection — Security — Use parameterized queries
- [WARN] model/users/user.go — Role stored as string — Use enum — Define role enum
- [NOTE] ci/ci.yml — Missing Clair — Security scan — Configure Clair

## Cursor action list
1. Implement parameterized queries
2. Define role enum
`

	cleaned, dropped := sanitizeReview(review, []string{"model/users/user.go"})
	require.Equal(t, 2, dropped)
	assert.Contains(t, cleaned, "## Verdict\nWARN")
	assert.Contains(t, cleaned, "- [WARN] model/users/user.go")
	assert.NotContains(t, cleaned, "api/user_data.go")
	assert.NotContains(t, cleaned, "ci/ci.yml")
}

func TestSanitizeReview_AllHallucinatedBecomesNone(t *testing.T) {
	review := `## Verdict
BLOCK

## Summary
Critical issues found.

## Findings
- [BLOCK] api/user_data.go — SQL injection — Security — Use parameterized queries
- [NOTE] logs/config.go — Logs need encryption — Add TLS

## Cursor action list
1. Implement parameterized queries
2. Add TLS for log transport
`

	cleaned, dropped := sanitizeReview(review, []string{"model/users/user.go"})
	require.Equal(t, 2, dropped)
	assert.Contains(t, cleaned, "## Verdict\nOK")
	assert.Contains(t, cleaned, "## Findings\n- none")
	assert.Contains(t, cleaned, "## Cursor action list\nnone")
	assert.NotContains(t, cleaned, "api/user_data.go")
}

func TestSanitizeReview_NoChangeWhenAllAllowed(t *testing.T) {
	review := `## Verdict
WARN

## Summary
One issue.

## Findings
- [WARN] model/users/user.go — missing test — Testing — add coverage

## Cursor action list
1. Add test for model/users/user.go
`
	cleaned, dropped := sanitizeReview(review, []string{"model/users/user.go"})
	assert.Equal(t, 0, dropped)
	assert.Equal(t, review, cleaned)
}

func TestPathAllowed_UniqueBasename(t *testing.T) {
	allowed := map[string]struct{}{
		"model/users/user.go": {},
	}
	assert.True(t, pathAllowed("user.go", allowed))
	assert.True(t, pathAllowed("model/users/user.go", allowed))
	assert.False(t, pathAllowed("api/user_data.go", allowed))
}
