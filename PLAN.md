# Code Review Plan: `feat/nemotron` Branch

## Branch Overview

This branch contains **two logically independent features**:
1. **Committed:** NVIDIA Nemotron ASR provider + Docker infrastructure (22 files, +886 lines)
2. **Unstaged:** Vim voice command mode (16 modified files)

### Build Status: All Green
- `go build` - passes
- `go vet` - passes
- `go test -short` - all 14 packages pass

---

## Issues Found

### Critical

| # | Issue | Location |
|---|-------|----------|
| 1 | **Vim pipeline: transcription text duplicated in LLM prompt.** `BuildVimUserPrompt(text)` is set as `CustomPrompt`, then `adapter.Process(ctx, text)` passes the same text again. The LLM receives the transcription twice. | `pipeline.go:374-389` |
| 2 | **ydotool `<C-x>` is fundamentally broken.** Uses `ydotool type` for the inner key while Ctrl is held, but `type` generates text input events that don't respect held modifiers. All Ctrl+key combos will silently fail. | `ydotool.go:142` |

### High

| # | Issue | Location |
|---|-------|----------|
| 3 | **Deprecated `strings.Title()` used** (deprecated since Go 1.18, project uses 1.24). Should use `cases.Title(language.English)` from `golang.org/x/text/cases`. | `validate.go:82,151,194` |
| 4 | **ydotool Ctrl key can get stuck.** If the `type` call fails between Ctrl press and release, `key 29:0` (Ctrl release) is never sent, leaving Ctrl held system-wide. | `ydotool.go:136-147` |

### Medium

| # | Issue | Location |
|---|-------|----------|
| 5 | **`docker setup.md` is a scratchpad file** with a space in filename, committed to the repo. Contains raw terminal notes, not polished docs. | root directory |
| 6 | **Non-reproducible Docker builds.** Both whisper Dockerfiles clone `HEAD` without pinning to a tag/commit. Builds may break unexpectedly. | `scripts/Dockerfile.whisper*` |
| 7 | **Vim pipeline: status not reset on error.** If LLM processing or key injection fails, pipeline status stays stuck at "Processing"/"Injecting" instead of returning to "Idle". | `pipeline.go:390-406` |
| 8 | **Two features in one branch.** Nemotron ASR and vim mode are independent features mixed together, making review and bisect harder. | branch-level |

### Low

| # | Issue | Location |
|---|-------|----------|
| 9 | **No validation of vim key names.** `<C->` (empty inner) or `<C-123>` pass through without error. | `wtype.go:100` |
| 10 | **ydotool spawns many processes.** One process per keystroke token, plus 3 for Ctrl combos. Slow for long sequences with timing gaps. | `ydotool.go:127-156` |
| 11 | **No linter configured.** No `.golangci.yml` exists — easy to miss deprecated APIs, unused code, etc. | project-level |
| 12 | **Nemotron Flask dev server.** Uses `app.run()` instead of gunicorn. Acceptable for local single-user use but fragile. | `scripts/nemotron_server.py` |

---

## Proposed Plan

### Phase 1: Critical Bug Fixes

1. **Fix vim pipeline text duplication** — Restructure `handleVimInjection` so the transcription text is only passed once to the LLM. Either pass it solely through `Process(ctx, text)` with an empty `CustomPrompt`, or set `CustomPrompt` and pass empty string to `Process`.

2. **Fix ydotool `<C-x>` handling** — Replace `ydotool type` with `ydotool key` using proper kernel keycodes for the inner character. Add a character-to-keycode mapping (at minimum a-z, 0-9, and common symbols). Also add a `defer` to guarantee Ctrl release even on error.

### Phase 2: High Priority Fixes

3. **Replace deprecated `strings.Title()`** — Use `cases.Title(language.English).String()` from `golang.org/x/text/cases` in all 3 locations in `validate.go`.

4. **Fix ydotool Ctrl-stuck risk** — Restructure the Ctrl combo code to use `defer` for the key release, or combine into a single `ydotool key 29:1 <keycode>:1 <keycode>:0 29:0` command.

### Phase 3: Cleanup

5. **Remove or relocate `docker setup.md`** — Either delete it (it's personal notes) or rename to `docs/nemotron-setup.md` and clean up the content.

6. **Pin Docker build versions** — Add a specific whisper.cpp tag or commit hash to both Dockerfiles for reproducible builds.

7. **Fix vim pipeline status reset** — Ensure status returns to `Idle` when LLM processing or key injection errors occur (consistent with how dictation mode handles errors).

### Phase 4: Quality Improvements

8. **Add vim key validation** — Validate that `<C-x>` has a non-empty, single-character inner value. Reject malformed keys early with a clear error.

9. **Optimize ydotool keystroke batching** — Where possible, batch multiple keycodes into a single `ydotool key` call (e.g., `ydotool key 1:1 1:0 2:1 2:0`) instead of spawning a process per token.

10. **Add `.golangci.yml`** — Configure a linter to catch deprecated APIs, unused code, and common issues in CI.

### Phase 5: Branch Hygiene (Optional)

11. **Split features into separate PRs** — Consider separating the Nemotron ASR (committed, clean) from vim voice mode (unstaged, has bugs) into two branches/PRs for cleaner review and merge.
