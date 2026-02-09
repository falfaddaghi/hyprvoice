package transcriber

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// NemotronAdapter implements BatchAdapter for NVIDIA Nemotron Speech ASR
// via a locally-run NeMo inference server.
type NemotronAdapter struct {
	serverURL string
}

// NewNemotronAdapter creates a new Nemotron adapter.
// serverURL is the base URL of the NeMo inference server (e.g. "http://localhost:8080").
func NewNemotronAdapter(serverURL string) *NemotronAdapter {
	return &NemotronAdapter{
		serverURL: strings.TrimRight(serverURL, "/"),
	}
}

func (a *NemotronAdapter) Transcribe(ctx context.Context, audioData []byte) (string, error) {
	if len(audioData) == 0 {
		return "", nil
	}

	if a.serverURL == "" {
		return "", fmt.Errorf("nemotron server URL not configured: set transcription.nemotron_url in config")
	}

	// convert raw PCM to WAV
	wavData, err := convertToWAV(audioData)
	if err != nil {
		return "", fmt.Errorf("convert to WAV: %w", err)
	}

	// build multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("audio", "audio.wav")
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(wavData); err != nil {
		return "", fmt.Errorf("write audio data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close multipart writer: %w", err)
	}

	url := a.serverURL + "/transcribe"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 120 * time.Second}
	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start)
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("nemotron server request failed (is the server running at %s?): %w", a.serverURL, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("nemotron server returned status %d: %s", resp.StatusCode, string(respBody))
	}

	text := parseNemotronResponse(respBody)
	log.Printf("nemotron: transcribed %d bytes in %v: %q", len(audioData), duration, text)
	return text, nil
}

// parseNemotronResponse tries JSON {"text": "..."} first, falls back to plain text.
func parseNemotronResponse(body []byte) string {
	var jsonResp struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(body, &jsonResp); err == nil && jsonResp.Text != "" {
		return strings.TrimSpace(jsonResp.Text)
	}
	return strings.TrimSpace(string(body))
}
