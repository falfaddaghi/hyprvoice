# Plan: Connect Vim Mode to OpenCode Zen Free Models

## Goal
Use OpenCode Zen's free LLM models for vim mode via their OpenAI-compatible API.

## Key Info
- **API base URL**: `https://opencode.ai/zen/v1`
- **Chat endpoint**: `https://opencode.ai/zen/v1/chat/completions`
- **Free models**: `opencode/glm-4.7-free`, `opencode/kimi-k2.5-free`, `opencode/minimax-m2.1-free`, `opencode/gpt-5-nano`
- **Auth**: API key from OpenCode Zen dashboard (user needs to sign up at opencode.ai/zen)
- **Format**: OpenAI-compatible (works with go-openai SDK by setting custom BaseURL)

## Implementation Steps

### 1. Add `opencode` LLM adapter (`internal/llm/adapter_opencode.go`)
- New file, modeled after `adapter_groq.go` (Groq already uses custom BaseURL with go-openai)
- `OpenCodeAdapter` struct with `openai.Client` configured with BaseURL `https://opencode.ai/zen/v1`
- Default model: `opencode/glm-4.7-free`
- Same `Process()` logic as existing adapters (SystemPrompt override, CustomPrompt, etc.)
- **Note**: Verify the BaseURL doesn't cause `/v1/v1/` duplication — the go-openai SDK may append `/v1` itself. If so, use `https://opencode.ai/zen` as the BaseURL instead.

### 2. Register `opencode` in `llm.NewAdapter()` (`internal/llm/llm.go`)
- Add `case "opencode"` that creates `OpenCodeAdapter`
- Require API key like other providers

### 3. Add provider constants (`internal/provider/names.go`)
- Add `ProviderOpenCode = "opencode"` constant
- Add `EnvOpenCodeKey = "OPENCODE_API_KEY"` constant
- Add `case ProviderOpenCode: return EnvOpenCodeKey` to `EnvVarForProvider()` switch

### 4. Register `opencode` as a provider (`internal/provider/opencode.go`)
- New file defining `OpenCodeProvider` struct implementing the `Provider` interface
- LLM-only provider (no transcription models)
- Models: the free ones listed above, each with `Type: LLM`, `AdapterType: AdapterOpenAI`
- `ValidateAPIKey`: check `len(key) > 0` (no known key prefix for OpenCode)
- `APIKeyURL`: `"https://opencode.ai/zen"`
- `DefaultModel(LLM)`: `"opencode/glm-4.7-free"`
- Endpoint: `&EndpointConfig{BaseURL: "https://opencode.ai/zen", Path: "/v1/chat/completions"}` (verify no `/v1` duplication)

### 5. Register provider in init (`internal/provider/provider.go`)
- Add `Register(&OpenCodeProvider{})` to the `init()` function

### 6. Add env var mapping in config validation (`internal/config/validate.go`)
- Add `case "opencode": return "OPENCODE_API_KEY"` to the `envVarForProvider()` switch (this is a separate duplicate mapping from the one in `names.go`)

### 7. Update tests
- `internal/provider/provider_test.go` — add opencode entry to `TestProviderInterface` test table
- `internal/llm/llm_test.go` — add opencode adapter creation test case

### 8. User config (not a code change)
- User adds `[vim]` section with `enabled = true`, provider = "opencode", model = one of the free models
- User sets `OPENCODE_API_KEY` env var or adds to `[providers.opencode]`

## Files to Create
- `internal/llm/adapter_opencode.go`
- `internal/provider/opencode.go`

## Files to Modify
- `internal/llm/llm.go` — add "opencode" case to `NewAdapter()`
- `internal/provider/names.go` — add `ProviderOpenCode`, `EnvOpenCodeKey` constants + `EnvVarForProvider()` case
- `internal/provider/provider.go` — add `Register(&OpenCodeProvider{})` in `init()`
- `internal/config/validate.go` — add "opencode" case to `envVarForProvider()`
- `internal/provider/provider_test.go` — add opencode to `TestProviderInterface` test table
- `internal/llm/llm_test.go` — add opencode adapter test

## Testing
1. `go build ./cmd/hyprvoice`
2. `go test ./...`
3. Set OPENCODE_API_KEY, start daemon, test vim-start/vim-stop
