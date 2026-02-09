package provider

// OpenCodeProvider implements Provider for OpenCode services
type OpenCodeProvider struct{}

func (p *OpenCodeProvider) Name() string {
	return ProviderOpenCode
}

func (p *OpenCodeProvider) RequiresAPIKey() bool {
	return true
}

func (p *OpenCodeProvider) ValidateAPIKey(key string) bool {
	return len(key) > 0
}

func (p *OpenCodeProvider) APIKeyURL() string {
	return "https://opencode.ai/zen"
}

func (p *OpenCodeProvider) IsLocal() bool {
	return false
}

func (p *OpenCodeProvider) Models() []Model {
	return []Model{
		{
			ID:                "opencode/glm-4.7-free",
			Name:              "GLM 4.7 Free",
			Description:       "Free GLM model; good default for text processing",
			Type:              LLM,
			SupportsBatch:     true,
			SupportsStreaming: false,
			Local:             false,
			AdapterType:       AdapterOpenAI,
			Endpoint:          &EndpointConfig{BaseURL: "https://opencode.ai/zen", Path: "/v1/chat/completions"},
		},
		{
			ID:                "opencode/kimi-k2.5-free",
			Name:              "Kimi K2.5 Free",
			Description:       "Free Kimi model; strong reasoning capabilities",
			Type:              LLM,
			SupportsBatch:     true,
			SupportsStreaming: false,
			Local:             false,
			AdapterType:       AdapterOpenAI,
			Endpoint:          &EndpointConfig{BaseURL: "https://opencode.ai/zen", Path: "/v1/chat/completions"},
		},
		{
			ID:                "opencode/minimax-m2.1-free",
			Name:              "MiniMax M2.1 Free",
			Description:       "Free MiniMax model; fast and capable",
			Type:              LLM,
			SupportsBatch:     true,
			SupportsStreaming: false,
			Local:             false,
			AdapterType:       AdapterOpenAI,
			Endpoint:          &EndpointConfig{BaseURL: "https://opencode.ai/zen", Path: "/v1/chat/completions"},
		},
		{
			ID:                "opencode/gpt-5-nano",
			Name:              "GPT-5 Nano",
			Description:       "Free GPT-5 Nano model; compact but capable",
			Type:              LLM,
			SupportsBatch:     true,
			SupportsStreaming: false,
			Local:             false,
			AdapterType:       AdapterOpenAI,
			Endpoint:          &EndpointConfig{BaseURL: "https://opencode.ai/zen", Path: "/v1/chat/completions"},
		},
	}
}

func (p *OpenCodeProvider) DefaultModel(t ModelType) string {
	switch t {
	case LLM:
		return "opencode/glm-4.7-free"
	}
	return ""
}
