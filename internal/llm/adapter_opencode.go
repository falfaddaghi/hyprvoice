package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/leonardotrapani/hyprvoice/internal/provider"
)

const defaultOpenCodeURL = "http://127.0.0.1:14500"

// OpenCodeAdapter implements Adapter using the OpenCode REST server (opencode serve).
type OpenCodeAdapter struct {
	config         Config
	serverURL      string
	client         *http.Client
	fallbackModels []string
}

// NewOpenCodeAdapter creates a new OpenCode LLM adapter that talks to
// the OpenCode REST server over HTTP.
func NewOpenCodeAdapter(cfg Config) *OpenCodeAdapter {
	primaryModel := cfg.Model
	if primaryModel == "" {
		primaryModel = "opencode/kimi-k2.5-free"
	}

	serverURL := cfg.OpenCodeURL
	if serverURL == "" {
		serverURL = defaultOpenCodeURL
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
		serverURL:      serverURL,
		client:         &http.Client{Timeout: 60 * time.Second},
		fallbackModels: fallbacks,
	}
}

// sessionResponse is the JSON response from POST /session.
type sessionResponse struct {
	ID string `json:"id"`
}

// messageRequest is the JSON body for POST /session/{id}/message.
type messageRequest struct {
	Model  messageModel  `json:"model"`
	System string        `json:"system"`
	Parts  []messagePart `json:"parts"`
}

type messageModel struct {
	ProviderID string `json:"providerID"`
	ModelID    string `json:"modelID"`
}

type messagePart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// messageResponse is the JSON response from POST /session/{id}/message.
type messageResponse struct {
	Parts []messagePart `json:"parts"`
}

// parseModelID splits "opencode/kimi-k2.5-free" into ("opencode", "kimi-k2.5-free").
func parseModelID(model string) (providerID, modelID string) {
	if i := strings.IndexByte(model, '/'); i >= 0 {
		return model[:i], model[i+1:]
	}
	return "opencode", model
}

func (a *OpenCodeAdapter) createSession(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.serverURL+"/session", nil)
	if err != nil {
		return "", fmt.Errorf("opencode: create session request: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("opencode: create session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("opencode: create session: status %d: %s", resp.StatusCode, body)
	}

	var sr sessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return "", fmt.Errorf("opencode: decode session response: %w", err)
	}
	if sr.ID == "" {
		return "", fmt.Errorf("opencode: create session: empty session ID")
	}
	return sr.ID, nil
}

func (a *OpenCodeAdapter) callModel(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
	start := time.Now()

	sessionID, err := a.createSession(ctx)
	if err != nil {
		return "", err
	}

	providerID, modelID := parseModelID(model)

	body := messageRequest{
		Model:  messageModel{ProviderID: providerID, ModelID: modelID},
		System: systemPrompt,
		Parts:  []messagePart{{Type: "text", Text: userPrompt}},
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("opencode: marshal message: %w", err)
	}

	url := fmt.Sprintf("%s/session/%s/message", a.serverURL, sessionID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyJSON))
	if err != nil {
		return "", fmt.Errorf("opencode: create message request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	duration := time.Since(start)
	if err != nil {
		log.Printf("opencode-llm-adapter: model %s failed after %v: %v", model, duration, err)
		return "", fmt.Errorf("opencode message (%s): %w", model, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("opencode-llm-adapter: model %s failed after %v: status %d: %s", model, duration, resp.StatusCode, respBody)
		return "", fmt.Errorf("opencode message (%s): status %d: %s", model, resp.StatusCode, respBody)
	}

	var mr messageResponse
	if err := json.NewDecoder(resp.Body).Decode(&mr); err != nil {
		return "", fmt.Errorf("opencode: decode message response: %w", err)
	}

	var textParts []string
	for _, p := range mr.Parts {
		if p.Type == "text" && p.Text != "" {
			textParts = append(textParts, p.Text)
		}
	}

	result := strings.TrimSpace(strings.Join(textParts, ""))
	if result == "" {
		return "", fmt.Errorf("opencode message (%s): no text in response", model)
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
