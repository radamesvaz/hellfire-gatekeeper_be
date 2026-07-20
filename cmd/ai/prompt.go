package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type reviewPrompt struct {
	System string
	User   string
}

func buildReviewPrompt(root, diff string) (reviewPrompt, error) {
	reviewer, err := readAIFile(root, "prompts", "reviewer.md")
	if err != nil {
		return reviewPrompt{}, err
	}
	architecture, err := readAIFile(root, "memory", "architecture.md")
	if err != nil {
		return reviewPrompt{}, err
	}
	checklist, err := readAIFile(root, "memory", "checklist.md")
	if err != nil {
		return reviewPrompt{}, err
	}

	allowed := extractDiffPaths(diff)

	var user strings.Builder
	user.WriteString("# PROJECT GUIDELINES\n\n")
	user.WriteString(strings.TrimSpace(architecture))
	user.WriteString("\n\n")
	user.WriteString(strings.TrimSpace(checklist))
	user.WriteString("\n\n")
	user.WriteString("# ALLOWED PATHS\n\n")
	user.WriteString("You may cite ONLY these paths (from the git diff).\n")
	user.WriteString("If a finding is not about one of these paths, do not report it.\n\n")
	if len(allowed) == 0 {
		user.WriteString("- (none extracted — report Findings as `- none`)\n")
	} else {
		for _, p := range allowed {
			user.WriteString("- ")
			user.WriteString(p)
			user.WriteByte('\n')
		}
	}
	user.WriteString("\n")
	user.WriteString("# GIT DIFF TO REVIEW\n\n")
	user.WriteString("```diff\n")
	user.WriteString(diff)
	if !strings.HasSuffix(diff, "\n") {
		user.WriteByte('\n')
	}
	user.WriteString("```\n\n")
	user.WriteString("# TASK\n")
	user.WriteString("Review the git diff against the guidelines above.\n")
	user.WriteString("Only flag issues with clear evidence in the diff.\n")
	user.WriteString("Do not invent files, tools, or generic security TODOs.\n\n")
	user.WriteString("Severity:\n")
	user.WriteString("- BLOCK = clear architecture/security/data violation\n")
	user.WriteString("- WARN = likely issue or missing tests/migrations\n")
	user.WriteString("- NOTE = minor nit\n\n")
	user.WriteString("Write ONLY this plain markdown template.\n")
	user.WriteString("Do NOT use JSON. Do NOT wrap the answer in code fences.\n\n")
	user.WriteString("## Verdict\n")
	user.WriteString("OK | WARN | BLOCK\n\n")
	user.WriteString("## Summary\n")
	user.WriteString("One or two sentences on guideline compliance.\n\n")
	user.WriteString("## Findings\n")
	user.WriteString("- [SEVERITY] path — issue — guideline — fix\n")
	user.WriteString("or\n")
	user.WriteString("- none\n\n")
	user.WriteString("## Cursor action list\n")
	user.WriteString("1. Concrete fix for a finding above (must reference an ALLOWED PATH)\n")
	user.WriteString("or\n")
	user.WriteString("none\n\n")
	user.WriteString("Begin now with ## Verdict\n")

	return reviewPrompt{
		System: strings.TrimSpace(reviewer),
		User:   user.String(),
	}, nil
}

func (p reviewPrompt) combined() string {
	return p.System + "\n\n" + p.User
}

func readAIFile(root string, parts ...string) (string, error) {
	pathParts := append([]string{root, ".ai"}, parts...)
	path := filepath.Join(pathParts...)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return string(data), nil
}
