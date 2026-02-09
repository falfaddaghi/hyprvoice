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

	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}

	m := models[0]
	if m.ID != "nemotron-speech-0.6b" {
		t.Errorf("expected model ID 'nemotron-speech-0.6b', got '%s'", m.ID)
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
	m := models[0]

	if len(m.SupportedLanguages) != 1 || m.SupportedLanguages[0] != "en" {
		t.Errorf("expected SupportedLanguages=['en'], got %v", m.SupportedLanguages)
	}
	if !m.SupportsLanguage("en") {
		t.Error("SupportsLanguage('en') should be true")
	}
	if m.SupportsLanguage("es") {
		t.Error("SupportsLanguage('es') should be false")
	}
	if !m.SupportsLanguage("") {
		t.Error("SupportsLanguage('') should be true (auto always supported)")
	}
}

func TestNemotronProvider_DefaultModel(t *testing.T) {
	p := &NemotronProvider{}
	if p.DefaultModel(Transcription) != "nemotron-speech-0.6b" {
		t.Errorf("expected DefaultModel(Transcription)='nemotron-speech-0.6b', got '%s'", p.DefaultModel(Transcription))
	}
	if p.DefaultModel(LLM) != "" {
		t.Errorf("expected DefaultModel(LLM)='', got '%s'", p.DefaultModel(LLM))
	}
}
