package llm

import (
	"strings"
	"testing"
)

func TestBuildSystemPrompt(t *testing.T) {
	tests := []struct {
		name     string
		opts     PostProcessingOptions
		keywords []string
		contains []string
	}{
		{
			name: "all options enabled",
			opts: PostProcessingOptions{
				RemoveStutters:    true,
				AddPunctuation:    true,
				FixGrammar:        true,
				RemoveFillerWords: true,
			},
			keywords: nil,
			contains: []string{
				"Remove stutters",
				"Add proper punctuation",
				"Fix grammar",
				"Remove filler words",
			},
		},
		{
			name: "only grammar",
			opts: PostProcessingOptions{
				FixGrammar: true,
			},
			keywords: nil,
			contains: []string{
				"Fix grammar",
			},
		},
		{
			name: "with keywords",
			opts: PostProcessingOptions{
				RemoveStutters: true,
			},
			keywords: []string{"Kubernetes", "TypeScript", "hyprvoice"},
			contains: []string{
				"Kubernetes",
				"TypeScript",
				"hyprvoice",
				"Context keywords",
			},
		},
		{
			name:     "no options - should have default",
			opts:     PostProcessingOptions{},
			keywords: nil,
			contains: []string{
				"Clean up the text",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := BuildSystemPrompt(tc.opts, tc.keywords)
			for _, expected := range tc.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("expected prompt to contain %q, got: %s", expected, result)
				}
			}
		})
	}
}

func TestBuildUserPrompt(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		customPrompt string
		expected     string
	}{
		{
			name:         "no custom prompt",
			text:         "hello world",
			customPrompt: "",
			expected:     "hello world",
		},
		{
			name:         "with custom prompt",
			text:         "hello world",
			customPrompt: "Format as a haiku",
			expected:     "Format as a haiku\n\nText to process:\nhello world",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := BuildUserPrompt(tc.text, tc.customPrompt)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestNewAdapter(t *testing.T) {
	// Test OpenAI adapter creation
	openaiCfg := Config{
		Provider: "openai",
		APIKey:   "sk-test-key",
		Model:    "gpt-4o-mini",
	}
	adapter, err := NewAdapter(openaiCfg)
	if err != nil {
		t.Fatalf("failed to create openai adapter: %v", err)
	}
	if _, ok := adapter.(*OpenAIAdapter); !ok {
		t.Error("expected OpenAIAdapter type")
	}

	// Test Groq adapter creation
	groqCfg := Config{
		Provider: "groq",
		APIKey:   "gsk_test-key",
		Model:    "llama-3.3-70b-versatile",
	}
	adapter, err = NewAdapter(groqCfg)
	if err != nil {
		t.Fatalf("failed to create groq adapter: %v", err)
	}
	if _, ok := adapter.(*GroqAdapter); !ok {
		t.Error("expected GroqAdapter type")
	}

	// Test OpenCode adapter creation (no API key required)
	opencodeCfg := Config{
		Provider: "opencode",
		Model:    "opencode/kimi-k2.5-free",
	}
	adapter, err = NewAdapter(opencodeCfg)
	if err != nil {
		t.Fatalf("failed to create opencode adapter: %v", err)
	}
	if _, ok := adapter.(*OpenCodeAdapter); !ok {
		t.Error("expected OpenCodeAdapter type")
	}

	// Test missing API key
	noKeyCfg := Config{
		Provider: "openai",
		APIKey:   "",
	}
	_, err = NewAdapter(noKeyCfg)
	if err == nil {
		t.Error("expected error for missing API key")
	}

	// Test unsupported provider
	badCfg := Config{
		Provider: "unsupported",
		APIKey:   "key",
	}
	_, err = NewAdapter(badCfg)
	if err == nil {
		t.Error("expected error for unsupported provider")
	}
}

func TestOpenCodeAdapter_FallbackModels(t *testing.T) {
	cfg := Config{
		Provider: "opencode",
		Model:    "opencode/kimi-k2.5-free",
	}
	adapter := NewOpenCodeAdapter(cfg)

	if len(adapter.fallbackModels) == 0 {
		t.Fatal("expected fallback models to be populated")
	}

	for _, fb := range adapter.fallbackModels {
		if fb == cfg.Model {
			t.Errorf("fallback models should not contain primary model %q", cfg.Model)
		}
	}
}

func TestOpenCodeAdapter_FallbackModels_DefaultModel(t *testing.T) {
	cfg := Config{
		Provider: "opencode",
		Model:    "", // should default to opencode/kimi-k2.5-free
	}
	adapter := NewOpenCodeAdapter(cfg)

	if len(adapter.fallbackModels) == 0 {
		t.Fatal("expected fallback models to be populated")
	}

	for _, fb := range adapter.fallbackModels {
		if fb == "opencode/kimi-k2.5-free" {
			t.Errorf("fallback models should not contain default primary model %q", "opencode/kimi-k2.5-free")
		}
	}
}

func TestOpenCodeAdapter_DefaultServerURL(t *testing.T) {
	adapter := NewOpenCodeAdapter(Config{
		Provider: "opencode",
		Model:    "opencode/kimi-k2.5-free",
	})
	if adapter.serverURL != defaultOpenCodeURL {
		t.Errorf("expected default server URL %q, got %q", defaultOpenCodeURL, adapter.serverURL)
	}
}

func TestOpenCodeAdapter_CustomServerURL(t *testing.T) {
	custom := "http://localhost:9999"
	adapter := NewOpenCodeAdapter(Config{
		Provider:    "opencode",
		Model:       "opencode/kimi-k2.5-free",
		OpenCodeURL: custom,
	})
	if adapter.serverURL != custom {
		t.Errorf("expected custom server URL %q, got %q", custom, adapter.serverURL)
	}
}

func TestParseModelID(t *testing.T) {
	tests := []struct {
		input      string
		providerID string
		modelID    string
	}{
		{"opencode/kimi-k2.5-free", "opencode", "kimi-k2.5-free"},
		{"opencode/big-pickle", "opencode", "big-pickle"},
		{"other-provider/some-model", "other-provider", "some-model"},
		{"bare-model-name", "opencode", "bare-model-name"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			pID, mID := parseModelID(tc.input)
			if pID != tc.providerID {
				t.Errorf("parseModelID(%q) providerID = %q, want %q", tc.input, pID, tc.providerID)
			}
			if mID != tc.modelID {
				t.Errorf("parseModelID(%q) modelID = %q, want %q", tc.input, mID, tc.modelID)
			}
		})
	}
}
