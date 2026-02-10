package provider

// OpenCodeProvider implements Provider for OpenCode services.
// Models are accessed via the locally installed opencode CLI (opencode run).
type OpenCodeProvider struct{}

func (p *OpenCodeProvider) Name() string {
	return ProviderOpenCode
}

func (p *OpenCodeProvider) RequiresAPIKey() bool {
	return false
}

func (p *OpenCodeProvider) ValidateAPIKey(key string) bool {
	return true
}

func (p *OpenCodeProvider) APIKeyURL() string {
	return ""
}

func (p *OpenCodeProvider) IsLocal() bool {
	return false
}

func (p *OpenCodeProvider) Models() []Model {
	return []Model{
		{
			ID:                "opencode/kimi-k2.5-free",
			Name:              "Kimi K2.5 Free",
			Description:       "Free Kimi model via opencode CLI; strong reasoning, good default",
			Type:              LLM,
			SupportsBatch:     true,
			SupportsStreaming: false,
			Local:             false,
			AdapterType:       AdapterOpenCode,
		},
		{
			ID:                "opencode/big-pickle",
			Name:              "Big Pickle",
			Description:       "Free model via opencode CLI; fast and capable",
			Type:              LLM,
			SupportsBatch:     true,
			SupportsStreaming: false,
			Local:             false,
			AdapterType:       AdapterOpenCode,
		},
		{
			ID:                "opencode/minimax-m2.1-free",
			Name:              "MiniMax M2.1 Free",
			Description:       "Free MiniMax model via opencode CLI; fast responses",
			Type:              LLM,
			SupportsBatch:     true,
			SupportsStreaming: false,
			Local:             false,
			AdapterType:       AdapterOpenCode,
		},
		{
			ID:                "opencode/gpt-5-nano",
			Name:              "GPT-5 Nano",
			Description:       "Free GPT-5 Nano via opencode CLI; compact with reasoning",
			Type:              LLM,
			SupportsBatch:     true,
			SupportsStreaming: false,
			Local:             false,
			AdapterType:       AdapterOpenCode,
		},
	}
}

func (p *OpenCodeProvider) DefaultModel(t ModelType) string {
	switch t {
	case LLM:
		return "opencode/kimi-k2.5-free"
	}
	return ""
}
