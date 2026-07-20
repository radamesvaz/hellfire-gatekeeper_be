package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func getGitDiff(root string, staged bool, base string) (diff string, label string, err error) {
	var args []string
	switch {
	case staged:
		args = []string{"diff", "--staged"}
		label = "staged"
	case base != "":
		resolved, rerr := resolveBaseRef(root, base)
		if rerr != nil {
			return "", "", rerr
		}
		if resolved != base {
			fmt.Fprintf(os.Stderr, "base: %q not found; using %q\n", base, resolved)
		}
		args = []string{"diff", resolved + "...HEAD"}
		label = resolved + "...HEAD"
	default:
		args = []string{"diff", "HEAD"}
		label = "working tree vs HEAD"
	}

	out, err := runGit(root, args...)
	if err != nil {
		return "", label, err
	}

	// Also include untracked new files when reviewing the working tree,
	// otherwise brand-new files never appear in `git diff HEAD`.
	if !staged && base == "" {
		untracked, uerr := untrackedFileDiffs(root)
		if uerr != nil {
			return "", label, uerr
		}
		if untracked != "" {
			if out != "" {
				out += "\n"
			}
			out += untracked
			label += " + untracked"
		}
	}

	return strings.TrimSpace(out), label, nil
}

// resolveBaseRef maps -base values to an existing git revision.
// Accepts main/master interchangeably and falls back to origin/<name>
// or the remote default branch (origin/HEAD).
func resolveBaseRef(root, base string) (string, error) {
	base = strings.TrimSpace(base)
	if base == "" {
		return "", fmt.Errorf("empty base ref")
	}

	for _, candidate := range baseRefCandidates(base) {
		if _, err := runGit(root, "rev-parse", "--verify", "--quiet", candidate+"^{commit}"); err == nil {
			return candidate, nil
		}
	}

	if sym, err := runGit(root, "symbolic-ref", "--quiet", "refs/remotes/origin/HEAD"); err == nil {
		sym = strings.TrimSpace(sym)
		if strings.HasPrefix(sym, "refs/remotes/") {
			return strings.TrimPrefix(sym, "refs/remotes/"), nil
		}
	}

	return "", fmt.Errorf(
		"unknown base ref %q (tried: %s). Tip: this repo uses master as the default branch",
		base,
		strings.Join(baseRefCandidates(base), ", "),
	)
}

func baseRefCandidates(base string) []string {
	base = strings.TrimSpace(base)
	var out []string
	add := func(s string) {
		for _, existing := range out {
			if existing == s {
				return
			}
		}
		out = append(out, s)
	}

	add(base)
	if !strings.HasPrefix(base, "origin/") {
		add("origin/" + base)
	}

	switch base {
	case "main":
		add("master")
		add("origin/master")
	case "master":
		add("main")
		add("origin/main")
	case "origin/main":
		add("master")
		add("origin/master")
		add("main")
	case "origin/master":
		add("main")
		add("origin/main")
		add("master")
	}

	return out
}

func untrackedFileDiffs(root string) (string, error) {
	list, err := runGit(root, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return "", err
	}
	files := splitGitLines(list)
	if len(files) == 0 {
		return "", nil
	}

	var b strings.Builder
	for _, f := range files {
		// Skip bulky/generated paths; review should stay focused on source.
		if shouldSkipUntracked(f) {
			continue
		}
		diff, err := syntheticNewFileDiff(root, f)
		if err != nil {
			return "", fmt.Errorf("diff untracked %s: %w", f, err)
		}
		if strings.TrimSpace(diff) != "" {
			b.WriteString(diff)
			if !strings.HasSuffix(diff, "\n") {
				b.WriteByte('\n')
			}
		}
	}
	return b.String(), nil
}

func splitGitLines(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var out []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

// syntheticNewFileDiff builds a unified diff for an untracked file without
// relying on OS-specific /dev/null or NUL paths.
func syntheticNewFileDiff(root, rel string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		return "", err
	}
	// Skip obviously binary/large blobs.
	if len(data) > 200_000 || bytes.IndexByte(data, 0) >= 0 {
		return "", nil
	}

	slash := filepathToSlash(rel)
	var b strings.Builder
	fmt.Fprintf(&b, "diff --git a/%s b/%s\n", slash, slash)
	b.WriteString("new file mode 100644\n")
	b.WriteString("--- /dev/null\n")
	fmt.Fprintf(&b, "+++ b/%s\n", slash)

	content := string(data)
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	// Split keeps a trailing empty element if file ends with newline; trim that visual noise.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	fmt.Fprintf(&b, "@@ -0,0 +1,%d @@\n", len(lines))
	for _, line := range lines {
		b.WriteString("+")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String(), nil
}

func shouldSkipUntracked(path string) bool {
	p := strings.ToLower(filepathToSlash(path))
	switch {
	case strings.HasPrefix(p, ".ai/reviews/"):
		return true
	case strings.HasSuffix(p, ".exe"):
		return true
	case strings.HasSuffix(p, ".png"), strings.HasSuffix(p, ".jpg"), strings.HasSuffix(p, ".jpeg"), strings.HasSuffix(p, ".gif"), strings.HasSuffix(p, ".webp"):
		return true
	case strings.HasPrefix(p, "uploads/"), strings.HasPrefix(p, "volumes/"), strings.HasPrefix(p, ".gopath/"):
		return true
	default:
		return false
	}
}

func filepathToSlash(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
}

func runGit(root string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	out := stdout.String()
	if err != nil {
		// git diff --no-index returns exit code 1 when there is a diff.
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 && out != "" {
			return out, nil
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return out, fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return out, nil
}
