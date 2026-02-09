package injection

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type ydotoolBackend struct{}

func NewYdotoolBackend() Backend {
	return &ydotoolBackend{}
}

func (y *ydotoolBackend) Name() string {
	return "ydotool"
}

func (y *ydotoolBackend) Available() error {
	if _, err := exec.LookPath("ydotool"); err != nil {
		return fmt.Errorf("ydotool not found: %w (install ydotool package)", err)
	}

	// Only check socket if ydotoold exists
	if _, err := exec.LookPath("ydotoold"); err == nil {
		socketPath := y.getSocketPath()
		if socketPath == "" {
			return fmt.Errorf("ydotoold socket not found - ensure ydotoold is running")
		}

		conn, err := net.DialTimeout("unix", socketPath, 500*time.Millisecond)
		if err != nil {
			return fmt.Errorf("ydotoold not responding at %s: %w", socketPath, err)
		}
		conn.Close()
	}

	return nil
}

func (y *ydotoolBackend) getSocketPath() string {
	// Check YDOTOOL_SOCKET env var first
	if sock := os.Getenv("YDOTOOL_SOCKET"); sock != "" {
		if _, err := os.Stat(sock); err == nil {
			return sock
		}
	}

	// Check common locations
	paths := []string{
		"/run/user/" + fmt.Sprint(os.Getuid()) + "/.ydotool_socket",
		"/tmp/.ydotool_socket",
	}

	// Also check XDG_RUNTIME_DIR
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		paths = append([]string{filepath.Join(xdg, ".ydotool_socket")}, paths...)
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}

func (y *ydotoolBackend) Inject(ctx context.Context, text string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := y.Available(); err != nil {
		return err
	}

	// ydotool type -- "text"
	cmd := exec.CommandContext(ctx, "ydotool", "type", "--", text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ydotool failed: %w", err)
	}

	return nil
}

// ydotool key code mapping for vim special keys (Linux kernel keycodes)
var ydotoolKeyCode = map[string]int{
	"<Esc>":   1,   // KEY_ESC
	"<CR>":    28,  // KEY_ENTER
	"<BS>":    14,  // KEY_BACKSPACE
	"<Tab>":   15,  // KEY_TAB
	"<Space>": 57,  // KEY_SPACE
	"<Up>":    103, // KEY_UP
	"<Down>":  108, // KEY_DOWN
	"<Left>":  105, // KEY_LEFT
	"<Right>": 106, // KEY_RIGHT
}

// charKeyCode maps characters to Linux kernel keycodes for use with ydotool key
var charKeyCode = map[byte]int{
	'a': 30, 'b': 48, 'c': 46, 'd': 32, 'e': 18, 'f': 33, 'g': 34,
	'h': 35, 'i': 23, 'j': 36, 'k': 37, 'l': 38, 'm': 50, 'n': 49,
	'o': 24, 'p': 25, 'q': 16, 'r': 19, 's': 31, 't': 20, 'u': 22,
	'v': 47, 'w': 17, 'x': 45, 'y': 21, 'z': 44,
	'0': 11, '1': 2, '2': 3, '3': 4, '4': 5, '5': 6, '6': 7,
	'7': 8, '8': 9, '9': 10,
}

func (y *ydotoolBackend) InjectKeys(ctx context.Context, keys string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := y.Available(); err != nil {
		return err
	}

	tokens := parseVimKeys(keys)

	for _, tok := range tokens {
		var err error
		switch {
		case ydotoolKeyCode[tok] != 0:
			// Special key: press and release via ydotool key
			code := ydotoolKeyCode[tok]
			cmd := exec.CommandContext(ctx, "ydotool", "key", fmt.Sprintf("%d:1", code), fmt.Sprintf("%d:0", code))
			err = cmd.Run()
		case strings.HasPrefix(tok, "<C-") && strings.HasSuffix(tok, ">"):
			// Control key combo: press Ctrl + key together, then release both
			inner := strings.ToLower(tok[3 : len(tok)-1])
			if len(inner) != 1 {
				err = fmt.Errorf("invalid control combo %q: inner key must be a single character", tok)
				break
			}
			keyCode, ok := charKeyCode[inner[0]]
			if !ok {
				err = fmt.Errorf("invalid control combo %q: unknown key %q", tok, inner)
				break
			}
			// Single atomic command: Ctrl down, key down, key up, Ctrl up
			cmd := exec.CommandContext(ctx, "ydotool", "key",
				"29:1",                        // KEY_LEFTCTRL press
				fmt.Sprintf("%d:1", keyCode),  // inner key press
				fmt.Sprintf("%d:0", keyCode),  // inner key release
				"29:0",                        // KEY_LEFTCTRL release
			)
			err = cmd.Run()
		default:
			// Regular character: use ydotool type
			cmd := exec.CommandContext(ctx, "ydotool", "type", "--", tok)
			err = cmd.Run()
		}
		if err != nil {
			return fmt.Errorf("ydotool InjectKeys failed on token %q: %w", tok, err)
		}
	}

	return nil
}
