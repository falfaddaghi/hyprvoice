package llm

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/provider"
	"github.com/sashabaranov/go-openai"
)

// OpenCodeAdapter implements Adapter using OpenCode's OpenAI-compatible API
type OpenCodeAdapter struct {
	client         *openai.Client
	config         Config
	fallbackModels []string
}

// NewOpenCodeAdapter creates a new OpenCode LLM adapter
func NewOpenCodeAdapter(cfg Config) *OpenCodeAdapter {
	clientConfig := openai.DefaultConfig(cfg.APIKey) // APIKey can be empty; OpenCode Zen is free
	clientConfig.BaseURL = "https://opencode.ai/zen/v1"

	primaryModel := cfg.Model
	if primaryModel == "" {
		primaryModel = "opencode/glm-4.7-free"
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
		client:         openai.NewClientWithConfig(clientConfig),
		config:         cfg,
		fallbackModels: fallbacks,
	}
}

func (a *OpenCodeAdapter) callModel(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
	req := openai.ChatCompletionRequest{
		Model: model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
		Temperature: 0.3,
	}

	start := time.Now()
	resp, err := a.client.CreateChatCompletion(ctx, req)
	duration := time.Since(start)

	if err != nil {
		log.Printf("opencode-llm-adapter: model %s failed after %v: %v", model, duration, err)
		return "", fmt.Errorf("opencode chat completion (%s): %w", model, err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("opencode chat completion (%s): no response choices", model)
	}

	result := resp.Choices[0].Message.Content
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
		primaryModel = "opencode/glm-4.7-free"
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
