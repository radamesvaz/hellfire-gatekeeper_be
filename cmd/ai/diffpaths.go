package main

import (
	"sort"
	"strings"
)

// extractDiffPaths returns unique repo-relative paths mentioned in a unified diff.
func extractDiffPaths(diff string) []string {
	seen := make(map[string]struct{})
	for _, line := range strings.Split(diff, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "diff --git "):
			// diff --git a/path b/path
			rest := strings.TrimPrefix(line, "diff --git ")
			parts := strings.Fields(rest)
			for _, p := range parts {
				if path, ok := stripDiffPathPrefix(p); ok {
					seen[path] = struct{}{}
				}
			}
		case strings.HasPrefix(line, "+++ "), strings.HasPrefix(line, "--- "):
			field := strings.Fields(line)
			if len(field) < 2 {
				continue
			}
			raw := field[1]
			if raw == "/dev/null" || raw == "NUL" {
				continue
			}
			if path, ok := stripDiffPathPrefix(raw); ok {
				seen[path] = struct{}{}
			}
		}
	}

	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func stripDiffPathPrefix(p string) (string, bool) {
	p = filepathToSlash(strings.TrimSpace(p))
	if p == "" || p == "/dev/null" || p == "NUL" {
		return "", false
	}
	if strings.HasPrefix(p, "a/") || strings.HasPrefix(p, "b/") {
		p = p[2:]
	}
	p = strings.TrimPrefix(p, "./")
	if p == "" {
		return "", false
	}
	return p, true
}
