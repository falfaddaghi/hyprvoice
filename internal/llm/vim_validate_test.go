package llm

import "testing"

func TestIsNoiseTranscription(t *testing.T) {
	tests := []struct {
		input string
		noise bool
	}{
		{"", true},
		{"   ", true},
		{"[BLANK_AUDIO]", true},
		{"[blank_audio]", true},
		{"[Silence]", true},
		{"[NO SPEECH]", true},
		{"[inaudible]", true},
		{"(blank audio)", true},
		{"(silence)", true},
		{"(inaudible)", true},
		{"hello world", false},
		{"page down", false},
		{"delete three lines", false},
		{"[some other tag]", false},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := IsNoiseTranscription(tc.input)
			if got != tc.noise {
				t.Errorf("IsNoiseTranscription(%q) = %v, want %v", tc.input, got, tc.noise)
			}
		})
	}
}

func TestValidateVimKeys(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Valid keystrokes
		{"simple motion", "5j", "5j"},
		{"delete line", "dd", "dd"},
		{"ctrl combo", "3<C-f>", "3<C-f>"},
		{"escape insert", "<Esc>ciw", "<Esc>ciw"},
		{"search command", "/error<CR>", "/error<CR>"},
		{"search with spaces", "/hello world<CR>", "/hello world<CR>"},
		{"backward search", "?pattern<CR>", "?pattern<CR>"},
		{"window split", "<C-w>v", "<C-w>v"},
		{"yank", "yy", "yy"},
		{"change inside quotes", `ci"`, `ci"`},
		{"complex combo", "3dw", "3dw"},
		{"go to definition", "gd", "gd"},

		// Wrapped in code fences — should be unwrapped
		{"markdown fenced", "```vim\ndd\n```", "dd"},
		{"backtick wrapped", "`3j`", "3j"},
		{"generic fence", "```\n5k\n```", "5k"},

		// Natural language — should be rejected
		{"english sentence", "You've provided the instructions but didn't include the spoken command", ""},
		{"parenthetical remark", "(empty response - no valid vim command was provided)", ""},
		{"explanation with command", "The command is dd", ""},
		{"chatty response", "Here are the keystrokes: dd", ""},
		{"multiword", "move down five lines", ""},

		// Edge cases
		{"empty", "", ""},
		{"whitespace only", "   ", ""},
		{"very long", string(make([]byte, 200)), ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ValidateVimKeys(tc.input)
			if got != tc.want {
				t.Errorf("ValidateVimKeys(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestStripCodeFences(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no fences", "dd", "dd"},
		{"vim fence", "```vim\ndd\n```", "dd"},
		{"generic fence", "```\n5j\n```", "5j"},
		{"backticks", "`dd`", "dd"},
		{"single backtick only", "`", "`"},
		{"empty backticks", "``", "``"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := stripCodeFences(tc.input)
			if got != tc.want {
				t.Errorf("stripCodeFences(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestContainsNaturalLanguage(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"dd", false},
		{"3<C-f>", false},
		{"/hello world<CR>", false},
		{"?search term<CR>", false},
		{"this is english", true},
		{"The command is dd", true},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := containsNaturalLanguage(tc.input)
			if got != tc.want {
				t.Errorf("containsNaturalLanguage(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestIsSearchCommand(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"/error<CR>", true},
		{"?pattern<CR>", true},
		{"/hello world<CR>", true},
		{"/no-enter", false},
		{"dd", false},
		{"", false},
		{"/", false},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := isSearchCommand(tc.input)
			if got != tc.want {
				t.Errorf("isSearchCommand(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}
