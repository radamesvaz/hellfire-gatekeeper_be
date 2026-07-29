// Command ai runs a local architecture review via Ollama.
//
// Prototype usage:
//
//	go run ./cmd/ai review
//	go run ./cmd/ai review -staged
//	go run ./cmd/ai review -base main
//	go run ./cmd/ai review -dry-run
//	go run ./cmd/ai review -save
//
// Env overrides: OLLAMA_HOST, OLLAMA_MODEL (default qwen2.5-coder:7b),
// OLLAMA_NUM_CTX (optional; otherwise sized to the prompt, capped at 16384 for ~8GB VRAM).
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const defaultReviewModel = "qwen2.5-coder:7b"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "review":
		if err := runReview(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		printUsage()
		os.Exit(2)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `ai — local architecture review with Ollama

Usage:
  go run ./cmd/ai review [flags]

Flags:
  -staged          review staged changes only (git diff --staged)
  -base string     review merge-base diff against ref (e.g. master)
  -model string    Ollama model (default: OLLAMA_MODEL or qwen2.5-coder:7b)
  -host string     Ollama host (default: OLLAMA_HOST or http://localhost:11434)
  -dry-run         print assembled prompt; do not call Ollama
  -save            write review under .ai/reviews/

Examples:
  go run ./cmd/ai review -staged
  go run ./cmd/ai review -base master -save
  go run ./cmd/ai review -dry-run
`)
}

func runReview(args []string) error {
	fs := flag.NewFlagSet("review", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	staged := fs.Bool("staged", false, "use staged diff")
	base := fs.String("base", "", "diff against this git ref")
	model := fs.String("model", envOr("OLLAMA_MODEL", defaultReviewModel), "Ollama model name")
	host := fs.String("host", envOr("OLLAMA_HOST", "http://localhost:11434"), "Ollama base URL")
	dryRun := fs.Bool("dry-run", false, "print prompt only")
	save := fs.Bool("save", false, "save review to .ai/reviews")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *staged && *base != "" {
		return fmt.Errorf("flags -staged and -base are mutually exclusive")
	}

	root, err := findRepoRoot()
	if err != nil {
		return err
	}

	diff, diffLabel, err := getGitDiff(root, *staged, *base)
	if err != nil {
		return err
	}
	if diff == "" {
		fmt.Println("No changes to review (empty diff).")
		return nil
	}

	prompt, err := buildReviewPrompt(root, diff)
	if err != nil {
		return err
	}

	promptChars := len(prompt.combined())
	estTokens := promptChars / 4
	numCtx := fitNumCtx(promptChars)

	fmt.Fprintf(os.Stderr, "diff: %s (%d bytes)\n", diffLabel, len(diff))
	fmt.Fprintf(os.Stderr, "model: %s @ %s\n", *model, *host)
	fmt.Fprintf(os.Stderr, "num_ctx: %d (override with OLLAMA_NUM_CTX)\n", numCtx)
	fmt.Fprintf(os.Stderr, "prompt: ~%d chars (~%d tokens est.)\n", promptChars, estTokens)

	if !promptFitsContext(estTokens, numCtx) {
		return fmt.Errorf(
			"prompt too large for local review (~%d tokens; effective context %d).\n"+
				"Workarounds:\n"+
				"  - review a smaller slice: ./run.sh review -staged\n"+
				"  - after committing: ./run.sh review -base master\n"+
				"  - docs/ and markdowns/ are already excluded from the diff\n"+
				"  - or raise OLLAMA_NUM_CTX if your machine has enough VRAM/RAM",
			estTokens, numCtx,
		)
	}

	if *dryRun {
		fmt.Println(prompt.combined())
		return nil
	}

	review, err := callOllama(*host, *model, prompt, numCtx)
	if err != nil {
		return err
	}

	allowed := extractDiffPaths(diff)
	cleaned, dropped := sanitizeReview(review, allowed)
	if dropped > 0 {
		fmt.Fprintf(os.Stderr, "sanitize: dropped %d finding(s) citing paths outside the diff\n", dropped)
		review = cleaned
	}

	fmt.Println(review)

	if *save {
		path, err := saveReview(root, review)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "saved: %s\n", path)
	}
	return nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from %s", wd)
		}
		dir = parent
	}
}

func saveReview(root, review string) (string, error) {
	dir := filepath.Join(root, ".ai", "reviews")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("review-%s.md", time.Now().Format("20060102-150405"))
	path := filepath.Join(dir, name)
	content := fmt.Sprintf("# Architecture review\n\nGenerated: %s\n\n%s\n",
		time.Now().Format(time.RFC3339), review)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	return path, nil
}
