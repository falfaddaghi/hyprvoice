package injection

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type wtypeBackend struct{}

func NewWtypeBackend() Backend {
	return &wtypeBackend{}
}

func (w *wtypeBackend) Name() string {
	return "wtype"
}

func (w *wtypeBackend) Available() error {
	if _, err := exec.LookPath("wtype"); err != nil {
		return fmt.Errorf("wtype not found: %w (install wtype package)", err)
	}

	if os.Getenv("WAYLAND_DISPLAY") == "" {
		return fmt.Errorf("WAYLAND_DISPLAY not set - wtype requires Wayland session")
	}

	if os.Getenv("XDG_RUNTIME_DIR") == "" {
		return fmt.Errorf("XDG_RUNTIME_DIR not set - wtype requires proper session environment")
	}

	return nil
}

func (w *wtypeBackend) Inject(ctx context.Context, text string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := w.Available(); err != nil {
		return err
	}

	// Brief pause so modifier keys (e.g. Super from keybind) are released
	// before wtype starts typing, preventing Super+<key> combos.
	time.Sleep(50 * time.Millisecond)

	cmd := exec.CommandContext(ctx, "wtype", "--", text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wtype failed: %w", err)
	}

	return nil
}

func (w *wtypeBackend) InjectKeys(ctx context.Context, keys string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := w.Available(); err != nil {
		return err
	}

	// Brief pause so modifier keys (e.g. Super from keybind) are released
	time.Sleep(50 * time.Millisecond)

	args := buildWtypeKeysArgs(keys)
	cmd := exec.CommandContext(ctx, "wtype", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wtype InjectKeys failed: %w", err)
	}

	return nil
}

// buildWtypeKeysArgs parses a vim keystroke string and builds wtype arguments.
// Handles <Esc>, <CR>, <C-x> notation and regular characters.
func buildWtypeKeysArgs(keys string) []string {
	var args []string
	tokens := parseVimKeys(keys)

	for _, tok := range tokens {
		switch {
		case tok == "<Esc>":
			args = append(args, "-k", "Escape")
		case tok == "<CR>":
			args = append(args, "-k", "Return")
		case tok == "<BS>":
			args = append(args, "-k", "BackSpace")
		case tok == "<Tab>":
			args = append(args, "-k", "Tab")
		case tok == "<Space>":
			args = append(args, "-k", "space")
		case tok == "<Up>":
			args = append(args, "-k", "Up")
		case tok == "<Down>":
			args = append(args, "-k", "Down")
		case tok == "<Left>":
			args = append(args, "-k", "Left")
		case tok == "<Right>":
			args = append(args, "-k", "Right")
		case strings.HasPrefix(tok, "<C-") && strings.HasSuffix(tok, ">"):
			// Control key: <C-x> -> -M ctrl -k x -m ctrl
			inner := tok[3 : len(tok)-1]
			args = append(args, "-M", "ctrl", "-k", inner, "-m", "ctrl")
		default:
			// Regular character
			args = append(args, "--", tok)
		}
	}

	return args
}

// parseVimKeys splits a vim keystroke string into individual tokens.
// Special keys like <Esc>, <CR>, <C-w> are returned as single tokens.
// Regular characters are returned individually.
func parseVimKeys(keys string) []string {
	var tokens []string
	i := 0
	for i < len(keys) {
		if keys[i] == '<' {
			// Look for closing >
			end := strings.IndexByte(keys[i:], '>')
			if end != -1 {
				tokens = append(tokens, keys[i:i+end+1])
				i += end + 1
				continue
			}
		}
		// Regular character
		tokens = append(tokens, string(keys[i]))
		i++
	}
	return tokens
}
