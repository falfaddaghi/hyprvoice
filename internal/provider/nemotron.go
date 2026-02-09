package provider

// NemotronProvider implements Provider for NVIDIA Nemotron Speech ASR local transcription.
// Requires a separately-run NeMo inference server (HTTP endpoint).
type NemotronProvider struct{}

func (p *NemotronProvider) Name() string {
	return ProviderNemotron
}

func (p *NemotronProvider) RequiresAPIKey() bool {
	return false
}

func (p *NemotronProvider) ValidateAPIKey(key string) bool {
	return true
}

func (p *NemotronProvider) APIKeyURL() string {
	return ""
}

func (p *NemotronProvider) IsLocal() bool {
	return true
}

func (p *NemotronProvider) Models() []Model {
	return []Model{
		{
			ID:                 "nemotron-speech-0.6b",
			Name:               "Nemotron Speech ASR 0.6B",
			Description:        "NVIDIA 600M-param streaming ASR; English-only, ~2.4GB VRAM, requires local NeMo server",
			Type:               Transcription,
			SupportsBatch:      true,
			SupportsStreaming:   false,
			Local:              true,
			AdapterType:        AdapterNemotron,
			SupportedLanguages: nemotronTranscriptionLanguages,
			Endpoint:           nil,
			LocalInfo:          nil, // detect-only, no managed download
			DocsURL:            "https://huggingface.co/nvidia/parakeet-tdt-0.6b-v2",
		},
	}
}

func (p *NemotronProvider) DefaultModel(t ModelType) string {
	switch t {
	case Transcription:
		return "nemotron-speech-0.6b"
	}
	return ""
}
