package main

import (
	"fmt"
	"regexp"
	"strings"
)

// Path ends at whitespace or a dash separator (ASCII/Unicode) used in the template.
var findingLineRe = regexp.MustCompile(`(?i)^-\s*\[(BLOCK|WARN|NOTE)\]\s+([^\s—]+)`)

// sanitizeReview drops findings that cite paths outside the diff allowlist.
// If nothing valid remains, Findings become "- none", action list "none", and Verdict "OK".
// Returns the cleaned review and how many finding lines were dropped.
func sanitizeReview(review string, allowedPaths []string) (string, int) {
	original := review
	review = strings.TrimSpace(review)
	if review == "" {
		return original, 0
	}

	allowed := make(map[string]struct{}, len(allowedPaths))
	for _, p := range allowedPaths {
		p = filepathToSlash(strings.TrimSpace(p))
		if p != "" {
			allowed[p] = struct{}{}
		}
	}

	sections := splitMarkdownSections(review)
	findingsIdx := -1
	actionsIdx := -1
	verdictIdx := -1
	for i, s := range sections {
		switch strings.ToLower(strings.TrimSpace(s.Title)) {
		case "findings":
			findingsIdx = i
		case "cursor action list":
			actionsIdx = i
		case "verdict":
			verdictIdx = i
		}
	}
	if findingsIdx < 0 {
		return original, 0
	}

	kept, dropped := filterFindingLines(sections[findingsIdx].Body, allowed)
	if dropped == 0 {
		return original, 0
	}

	if len(kept) == 0 {
		sections[findingsIdx].Body = "- none\n"
		if actionsIdx >= 0 {
			sections[actionsIdx].Body = "none\n"
		}
		if verdictIdx >= 0 {
			sections[verdictIdx].Body = "OK\n"
		}
	} else {
		sections[findingsIdx].Body = strings.Join(kept, "\n") + "\n"
		if verdictIdx >= 0 {
			sections[verdictIdx].Body = verdictFromFindings(kept) + "\n"
		}
	}

	return joinMarkdownSections(sections), dropped
}

func filterFindingLines(body string, allowed map[string]struct{}) (kept []string, dropped int) {
	for _, line := range strings.Split(body, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" {
			continue
		}
		if strings.EqualFold(trim, "- none") || strings.EqualFold(trim, "none") {
			kept = append(kept, trim)
			continue
		}
		m := findingLineRe.FindStringSubmatch(trim)
		if m == nil {
			// Preserve non-finding lines under Findings (rare).
			kept = append(kept, trim)
			continue
		}
		path := filepathToSlash(strings.Trim(m[2], "`\"'"))
		path = strings.TrimPrefix(path, "./")
		if pathAllowed(path, allowed) {
			kept = append(kept, trim)
			continue
		}
		dropped++
	}
	// If we only kept non-finding junk after dropping all real findings, treat as empty.
	if dropped > 0 && !hasFindingOrNone(kept) {
		return nil, dropped
	}
	return kept, dropped
}

func hasFindingOrNone(lines []string) bool {
	for _, line := range lines {
		if strings.EqualFold(line, "- none") || strings.EqualFold(line, "none") {
			return true
		}
		if findingLineRe.MatchString(line) {
			return true
		}
	}
	return false
}

func pathAllowed(path string, allowed map[string]struct{}) bool {
	if _, ok := allowed[path]; ok {
		return true
	}
	// Allow basename-only citations when uniquely resolvable in the allowlist.
	if !strings.Contains(path, "/") {
		var matches int
		for a := range allowed {
			if strings.HasSuffix(a, "/"+path) || a == path {
				matches++
			}
		}
		return matches == 1
	}
	return false
}

func verdictFromFindings(lines []string) string {
	verdict := "OK"
	for _, line := range lines {
		m := findingLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		switch strings.ToUpper(m[1]) {
		case "BLOCK":
			return "BLOCK"
		case "WARN":
			verdict = "WARN"
		}
	}
	return verdict
}

type mdSection struct {
	Title string // empty for preamble before first heading
	Body  string
}

func splitMarkdownSections(doc string) []mdSection {
	lines := strings.Split(doc, "\n")
	var sections []mdSection
	cur := mdSection{}
	var body []string

	flush := func() {
		cur.Body = strings.Join(body, "\n")
		if cur.Title != "" || strings.TrimSpace(cur.Body) != "" {
			sections = append(sections, cur)
		}
		body = nil
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			flush()
			cur = mdSection{Title: strings.TrimSpace(strings.TrimPrefix(line, "## "))}
			continue
		}
		body = append(body, line)
	}
	flush()
	return sections
}

func joinMarkdownSections(sections []mdSection) string {
	var b strings.Builder
	for i, s := range sections {
		if s.Title != "" {
			if b.Len() > 0 && !strings.HasSuffix(b.String(), "\n\n") {
				if !strings.HasSuffix(b.String(), "\n") {
					b.WriteByte('\n')
				}
				b.WriteByte('\n')
			}
			fmt.Fprintf(&b, "## %s\n", s.Title)
		}
		body := s.Body
		if !strings.HasSuffix(body, "\n") && body != "" {
			body += "\n"
		}
		b.WriteString(body)
		if i < len(sections)-1 && !strings.HasSuffix(b.String(), "\n") {
			b.WriteByte('\n')
		}
	}
	return strings.TrimSpace(b.String()) + "\n"
}
