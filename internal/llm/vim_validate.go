package llm

import (
	"regexp"
	"strings"
)

// noisePatterns are transcription outputs that indicate no useful speech was captured.
var noisePatterns = []string{
	"[blank_audio]",
	"[silence]",
	"[no speech]",
	"[inaudible]",
	"(blank audio)",
	"(silence)",
	"(inaudible)",
}

// IsNoiseTranscription returns true if the transcription text is empty or
// contains only ASR noise markers (e.g. [BLANK_AUDIO], [silence]).
func IsNoiseTranscription(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return true
	}
	lower := strings.ToLower(trimmed)
	for _, p := range noisePatterns {
		if lower == p {
			return true
		}
	}
	return false
}

// vimKeysPattern matches valid vim keystroke sequences:
// - Single ASCII characters (motions, operators, registers)
// - Numeric prefixes (e.g. 3dw, 5j)
// - Special keys in angle bracket notation: <Esc>, <CR>, <C-w>, <C-r>, <S-Tab>
// - Search patterns: /term<CR>, ?term<CR>
// - Quoted strings within keystrokes: ci", ci', yi(
// The pattern rejects anything that looks like natural language (spaces between
// words, parenthetical remarks, long runs of alphabetic characters).
var vimKeysPattern = regexp.MustCompile(
	`^[a-zA-Z0-9\[\]{}()~!@#$%^&*\-_=+;:'",./<>?\\|` + "`" + `]+$`,
)

// maxVimKeysLength is a sanity cap. Real vim commands from voice are short.
const maxVimKeysLength = 128

// ValidateVimKeys checks whether the LLM response looks like valid vim
// keystrokes rather than a natural-language explanation. Returns the cleaned
// keystrokes or empty string if invalid.
func ValidateVimKeys(response string) string {
	s := strings.TrimSpace(response)
	if s == "" {
		return ""
	}

	// Strip wrapping markdown code fences
	s = stripCodeFences(s)
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	// Too long to be a real vim command
	if len(s) > maxVimKeysLength {
		return ""
	}

	// Vim keystrokes never contain spaces (except inside search terms after / or ?).
	// If there are spaces in a non-search context, it's natural language.
	if containsNaturalLanguage(s) {
		return ""
	}

	return s
}

// stripCodeFences removes common markdown code fence wrapping.
func stripCodeFences(s string) string {
	// ```vim\n...\n``` or ```\n...\n```
	if strings.HasPrefix(s, "```") {
		lines := strings.Split(s, "\n")
		if len(lines) >= 3 && strings.HasPrefix(lines[len(lines)-1], "```") {
			inner := strings.Join(lines[1:len(lines)-1], "\n")
			return strings.TrimSpace(inner)
		}
	}
	// Single backtick wrapping: `dd`
	if len(s) > 2 && s[0] == '`' && s[len(s)-1] == '`' && strings.Count(s, "`") == 2 {
		return s[1 : len(s)-1]
	}
	return s
}

// containsNaturalLanguage detects if the string looks like an English sentence
// rather than vim keystrokes. Vim keystrokes may contain spaces only inside
// search terms (e.g. "/hello world<CR>").
func containsNaturalLanguage(s string) bool {
	// Search commands can contain spaces in the search term
	if isSearchCommand(s) {
		return false
	}

	// Any space in a non-search command is natural language
	if strings.Contains(s, " ") {
		return true
	}

	// Check for parenthetical remarks: (something)
	if strings.Contains(s, "(") && strings.Contains(s, ")") {
		inner := s[strings.Index(s, "(")+1 : strings.LastIndex(s, ")")]
		if strings.Contains(inner, " ") || len(inner) > 10 {
			return true
		}
	}

	return false
}

// isSearchCommand checks if the string is a vim search command (starts with / or ?
// and ends with <CR>).
func isSearchCommand(s string) bool {
	if len(s) < 2 {
		return false
	}
	if s[0] != '/' && s[0] != '?' {
		return false
	}
	return strings.HasSuffix(s, "<CR>")
}
