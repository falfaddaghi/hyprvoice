package llm

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/provider"
)

// OpenCodeAdapter implements Adapter using the opencode CLI (opencode run).
type OpenCodeAdapter struct {
	config         Config
	fallbackModels []string
}

// NewOpenCodeAdapter creates a new OpenCode LLM adapter that shells out to
// the locally installed opencode CLI.
func NewOpenCodeAdapter(cfg Config) *OpenCodeAdapter {
	primaryModel := cfg.Model
	if primaryModel == "" {
		primaryModel = "opencode/kimi-k2.5-free"
	}

	var fallbacks []string
	if p := provider.GetProvider("opencode"); p != nil {
		for _, m := range p.Models() {
			if m.Type == provider.LLM && m.ID != primaryModel {
				fallbacks = append(fallbacks, m.ID)
			}
		}
	}

	return &OpenCodeAdapter{
		config:         cfg,
		fallbackModels: fallbacks,
	}
}

// opencodeEvent represents a JSON event from `opencode run --format json`.
type opencodeEvent struct {
	Type string `json:"type"`
	Part struct {
		Text string `json:"text"`
	} `json:"part"`
}

func (a *OpenCodeAdapter) callModel(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
	// Combine system + user prompt into a single message for the CLI.
	prompt := systemPrompt + "\n\n" + userPrompt

	args := []string{"run", "-m", model, "--format", "json", prompt}

	start := time.Now()
	cmd := exec.CommandContext(ctx, "opencode", args...)

	out, err := cmd.Output()
	duration := time.Since(start)

	if err != nil {
		stderr := ""
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = string(exitErr.Stderr)
		}
		log.Printf("opencode-llm-adapter: model %s failed after %v: %v (stderr: %s)", model, duration, err, stderr)
		return "", fmt.Errorf("opencode run (%s): %w", model, err)
	}

	// Parse JSON-lines output and collect text parts.
	var textParts []string
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		var ev opencodeEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			continue
		}
		if ev.Type == "text" && ev.Part.Text != "" {
			textParts = append(textParts, ev.Part.Text)
		}
	}

	result := strings.TrimSpace(strings.Join(textParts, ""))
	if result == "" {
		return "", fmt.Errorf("opencode run (%s): no text in response", model)
	}

	log.Printf("opencode-llm-adapter: model %s processed in %v", model, duration)
	return result, nil
}

func (a *OpenCodeAdapter) Process(ctx context.Context, text string) (string, error) {
	if text == "" {
		return "", nil
	}

	var systemPrompt string
	if a.config.SystemPrompt != "" {
		systemPrompt = a.config.SystemPrompt
	} else {
		opts := PostProcessingOptions{
			RemoveStutters:    a.config.RemoveStutters,
			AddPunctuation:    a.config.AddPunctuation,
			FixGrammar:        a.config.FixGrammar,
			RemoveFillerWords: a.config.RemoveFillerWords,
		}
		systemPrompt = BuildSystemPrompt(opts, a.config.Keywords)
	}
	userPrompt := BuildUserPrompt(text, a.config.CustomPrompt)

	primaryModel := a.config.Model
	if primaryModel == "" {
		primaryModel = "opencode/kimi-k2.5-free"
	}

	result, err := a.callModel(ctx, primaryModel, systemPrompt, userPrompt)
	if err == nil {
		log.Printf("opencode-llm-adapter: %q -> %q", text, result)
		return result, nil
	}

	var lastErr error = err
	for _, model := range a.fallbackModels {
		log.Printf("opencode-llm-adapter: trying fallback model %s", model)
		result, err = a.callModel(ctx, model, systemPrompt, userPrompt)
		if err == nil {
			log.Printf("opencode-llm-adapter: fallback %s succeeded: %q -> %q", model, text, result)
			return result, nil
		}
		lastErr = err
	}

	return "", fmt.Errorf("opencode: all models failed, last error: %w", lastErr)
}
