# Performance Review — OOM Investigation

_Date: 2026-02-11_

## Summary

Hyprvoice caused the laptop to run OOM. This review identifies all memory hotspots across the Go client and the Nemotron ASR Docker server, ranked by likelihood of being the OOM culprit.

---

## Findings

### 1. CRITICAL — Nemotron Docker container has no memory limits

**Most likely OOM cause.**

- Base image `nvcr.io/nvidia/nemo:24.12` is ~15GB and pulls in all of PyTorch + NeMo
- The 1.1B model requires ~2.5GB weights + PyTorch runtime overhead = **6-10GB total**
- No `docker-compose.yml` exists — no memory limits on the container
- `nemotron_server.py` never calls `torch.cuda.empty_cache()` after inference
- If GPU is unavailable or VRAM insufficient, PyTorch silently falls back to **CPU/system RAM**, consuming 6-10GB of your laptop's RAM
- Flask runs single-threaded (`threaded=False`) which is good, but a single inference still holds the full model in memory permanently

**Files:** `scripts/Dockerfile.nemotron`, `scripts/nemotron_server.py`

### 2. HIGH — Audio triple-buffered during transcription (up to ~230MB peak)

- Default recording timeout is **5 minutes** (`internal/config/defaults.go:15`)
- At 16kHz mono 16-bit: 5 min = ~57MB raw PCM in `SimpleTranscriber.audioBuffer`
- `transcribeAll()` copies the entire buffer (`internal/transcriber/simple_transcriber.go:97-98`)
- `convertToWAV()` creates WAV bytes copy
- `adapter_nemotron.go` multipart encoding creates yet another copy
- **Peak: ~230MB** for a max-length recording — 4 copies of the audio simultaneously in memory
- Not OOM on its own, but combined with Nemotron = death

**Files:** `internal/transcriber/simple_transcriber.go`, `internal/transcriber/adapter_nemotron.go`, `internal/transcriber/audio_utils.go`

### 3. MEDIUM — Untracked goroutines (leak risk, not OOM)

- Pipeline error-forwarding goroutines (`internal/pipeline/pipeline.go:197-207`) not in WaitGroup
- Recording stderr logger (`internal/recording/recording.go:139-144`) not context-aware, not in WaitGroup
- These won't cause OOM but are correctness issues that could accumulate over many toggle cycles

**Files:** `internal/pipeline/pipeline.go`, `internal/recording/recording.go`

### 4. LOW — Waybar poller creates new socket connection every 300ms

- `runWaybar()` in `cmd/hyprvoice/main.go` polls daemon status in a tight loop
- Creates and tears down a Unix socket connection ~3.3 times/sec
- Not a memory issue but unnecessary churn

---

## Recommended Fixes

### Fix 1: Docker memory limits + server cleanup (Finding #1)
- Create `scripts/docker-compose.nemotron.yml` with memory limits
- Add `torch.cuda.empty_cache()` + `gc.collect()` after each inference
- Log memory usage after model load

### Fix 2: Eliminate audio copy chain (Finding #2)
- Pass audio buffer directly instead of copying in `transcribeAll()`
- Stream WAV data directly into multipart writer instead of building full `[]byte`
- Add `MaxAudioBytes` config option with sane default
- Reduce default recording timeout from 5m to 2m

### Fix 3: Track all goroutines (Finding #3)
- Add WaitGroup tracking to pipeline error-forwarding goroutines
- Make recording stderr logger context-aware

### Fix 4: Release audio buffer after transcription
- Set `t.audioBuffer = nil` after `transcribeAll()` to release for GC
