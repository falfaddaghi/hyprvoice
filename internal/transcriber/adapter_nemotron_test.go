package transcriber

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNemotronAdapter_ImplementsBatchAdapter(t *testing.T) {
	var _ BatchAdapter = (*NemotronAdapter)(nil)
}

func TestNemotronAdapter_EmptyAudio(t *testing.T) {
	adapter := NewNemotronAdapter("http://localhost:9999")
	text, err := adapter.Transcribe(context.Background(), []byte{})
	if err != nil {
		t.Errorf("expected no error for empty audio, got: %v", err)
	}
	if text != "" {
		t.Errorf("expected empty text for empty audio, got: %q", text)
	}
}

func TestNemotronAdapter_MissingURL(t *testing.T) {
	adapter := NewNemotronAdapter("")
	audioData := make([]byte, 32000)

	_, err := adapter.Transcribe(context.Background(), audioData)
	if err == nil {
		t.Error("expected error for missing URL")
	}
	if err != nil && !contains(err.Error(), "nemotron server URL not configured") {
		t.Errorf("expected 'nemotron server URL not configured' error, got: %v", err)
	}
}

func TestNemotronAdapter_SuccessfulTranscription_JSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/transcribe" {
			t.Errorf("expected /transcribe, got %s", r.URL.Path)
		}

		ct := r.Header.Get("Content-Type")
		if ct == "" || !contains(ct, "multipart/form-data") {
			t.Errorf("expected multipart/form-data content type, got: %s", ct)
		}

		// verify audio file is present
		file, _, err := r.FormFile("audio")
		if err != nil {
			t.Errorf("expected audio form file, got error: %v", err)
		} else {
			file.Close()
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"text": "hello world"}`)
	}))
	defer server.Close()

	adapter := NewNemotronAdapter(server.URL)
	audioData := make([]byte, 32000)

	text, err := adapter.Transcribe(context.Background(), audioData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "hello world" {
		t.Errorf("expected 'hello world', got: %q", text)
	}
}

func TestNemotronAdapter_SuccessfulTranscription_PlainText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "hello world\n")
	}))
	defer server.Close()

	adapter := NewNemotronAdapter(server.URL)
	audioData := make([]byte, 32000)

	text, err := adapter.Transcribe(context.Background(), audioData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "hello world" {
		t.Errorf("expected 'hello world', got: %q", text)
	}
}

func TestNemotronAdapter_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "internal error")
	}))
	defer server.Close()

	adapter := NewNemotronAdapter(server.URL)
	audioData := make([]byte, 32000)

	_, err := adapter.Transcribe(context.Background(), audioData)
	if err == nil {
		t.Error("expected error for server error response")
	}
	if err != nil && !contains(err.Error(), "status 500") {
		t.Errorf("expected 'status 500' in error, got: %v", err)
	}
}

func TestNemotronAdapter_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// block until context is cancelled
		<-r.Context().Done()
	}))
	defer server.Close()

	adapter := NewNemotronAdapter(server.URL)
	audioData := make([]byte, 32000)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := adapter.Transcribe(ctx, audioData)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestNemotronAdapter_ServerUnreachable(t *testing.T) {
	adapter := NewNemotronAdapter("http://localhost:1") // port 1 is unlikely to be open
	audioData := make([]byte, 32000)

	_, err := adapter.Transcribe(context.Background(), audioData)
	if err == nil {
		t.Error("expected error for unreachable server")
	}
	if err != nil && !contains(err.Error(), "is the server running") {
		t.Errorf("expected helpful error about server running, got: %v", err)
	}
}
