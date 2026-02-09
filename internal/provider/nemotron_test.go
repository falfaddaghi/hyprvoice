package provider

import "testing"

func TestNemotronProvider_GetProvider(t *testing.T) {
	p := GetProvider("nemotron")
	if p == nil {
		t.Fatal("GetProvider('nemotron') returned nil")
	}
	if p.Name() != "nemotron" {
		t.Errorf("expected name 'nemotron', got '%s'", p.Name())
	}
}

func TestNemotronProvider_Models(t *testing.T) {
	p := &NemotronProvider{}
	models := p.Models()

	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}

	// First model should be 1.1b (default/primary)
	m := models[0]
	if m.ID != "nemotron-speech-1.1b" {
		t.Errorf("expected first model ID 'nemotron-speech-1.1b', got '%s'", m.ID)
	}
	if !m.Local {
		t.Error("expected Local=true")
	}
	if m.LocalInfo != nil {
		t.Error("expected LocalInfo=nil (detect-only)")
	}
	if m.AdapterType != AdapterNemotron {
		t.Errorf("expected AdapterType='nemotron', got '%s'", m.AdapterType)
	}
	if m.Type != Transcription {
		t.Error("expected Type=Transcription")
	}
	if m.Endpoint != nil {
		t.Error("expected Endpoint=nil for local model")
	}
	if !m.SupportsBatch {
		t.Error("expected SupportsBatch=true")
	}
	if m.SupportsStreaming {
		t.Error("expected SupportsStreaming=false")
	}

	// Second model should be 0.6b
	m2 := models[1]
	if m2.ID != "nemotron-speech-0.6b" {
		t.Errorf("expected second model ID 'nemotron-speech-0.6b', got '%s'", m2.ID)
	}
	if m2.AdapterType != AdapterNemotron {
		t.Errorf("expected AdapterType='nemotron', got '%s'", m2.AdapterType)
	}
}

func TestNemotronProvider_RequiresAPIKey(t *testing.T) {
	p := &NemotronProvider{}
	if p.RequiresAPIKey() {
		t.Error("RequiresAPIKey() should return false")
	}
}

func TestNemotronProvider_IsLocal(t *testing.T) {
	p := &NemotronProvider{}
	if !p.IsLocal() {
		t.Error("IsLocal() should return true")
	}
}

func TestNemotronProvider_EnglishOnly(t *testing.T) {
	p := &NemotronProvider{}
	models := p.Models()

	for _, m := range models {
		if len(m.SupportedLanguages) != 1 || m.SupportedLanguages[0] != "en" {
			t.Errorf("model %s: expected SupportedLanguages=['en'], got %v", m.ID, m.SupportedLanguages)
		}
		if !m.SupportsLanguage("en") {
			t.Errorf("model %s: SupportsLanguage('en') should be true", m.ID)
		}
		if m.SupportsLanguage("es") {
			t.Errorf("model %s: SupportsLanguage('es') should be false", m.ID)
		}
		if !m.SupportsLanguage("") {
			t.Errorf("model %s: SupportsLanguage('') should be true (auto always supported)", m.ID)
		}
	}
}

func TestNemotronProvider_DefaultModel(t *testing.T) {
	p := &NemotronProvider{}
	if p.DefaultModel(Transcription) != "nemotron-speech-1.1b" {
		t.Errorf("expected DefaultModel(Transcription)='nemotron-speech-1.1b', got '%s'", p.DefaultModel(Transcription))
	}
	if p.DefaultModel(LLM) != "" {
		t.Errorf("expected DefaultModel(LLM)='', got '%s'", p.DefaultModel(LLM))
	}
}
