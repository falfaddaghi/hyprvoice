package llm

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sashabaranov/go-openai"
)

// OpenCodeAdapter implements Adapter using OpenCode's OpenAI-compatible API
type OpenCodeAdapter struct {
	client *openai.Client
	config Config
}

// NewOpenCodeAdapter creates a new OpenCode LLM adapter
func NewOpenCodeAdapter(cfg Config) *OpenCodeAdapter {
	clientConfig := openai.DefaultConfig(cfg.APIKey)
	clientConfig.BaseURL = "https://opencode.ai/zen/v1"
	return &OpenCodeAdapter{
		client: openai.NewClientWithConfig(clientConfig),
		config: cfg,
	}
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

	model := a.config.Model
	if model == "" {
		model = "opencode/glm-4.7-free"
	}

	req := openai.ChatCompletionRequest{
		Model: model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
		Temperature: 0.3, // Low temperature for consistent cleanup
	}

	start := time.Now()
	resp, err := a.client.CreateChatCompletion(ctx, req)
	duration := time.Since(start)

	if err != nil {
		log.Printf("opencode-llm-adapter: API call failed after %v: %v", duration, err)
		return "", fmt.Errorf("opencode chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("opencode chat completion: no response choices")
	}

	result := resp.Choices[0].Message.Content
	log.Printf("opencode-llm-adapter: processed in %v: %q -> %q", duration, text, result)
	return result, nil
}
