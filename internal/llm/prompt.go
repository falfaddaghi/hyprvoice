package llm

import (
	"fmt"
	"strings"
)

// PostProcessingOptions controls which cleanup operations to request
type PostProcessingOptions struct {
	RemoveStutters    bool
	AddPunctuation    bool
	FixGrammar        bool
	RemoveFillerWords bool
}

// BuildSystemPrompt generates the system prompt for text cleanup
func BuildSystemPrompt(opts PostProcessingOptions, keywords []string) string {
	var tasks []string

	if opts.RemoveStutters {
		tasks = append(tasks, "Remove stutters and repeated words/phrases")
	}
	if opts.AddPunctuation {
		tasks = append(tasks, "Add proper punctuation")
	}
	if opts.FixGrammar {
		tasks = append(tasks, "Fix grammar errors")
	}
	if opts.RemoveFillerWords {
		tasks = append(tasks, "Remove filler words (um, uh, like, you know, etc.)")
	}

	// If no tasks, just clean up generally
	if len(tasks) == 0 {
		tasks = append(tasks, "Clean up the text while preserving meaning")
	}

	prompt := "You are a text cleanup assistant. Your job is to clean up speech-to-text transcriptions.\n\n"
	prompt += "Tasks:\n"
	for _, task := range tasks {
		prompt += fmt.Sprintf("- %s\n", task)
	}

	prompt += "\nRules:\n"
	prompt += "- Preserve the original meaning and intent\n"
	prompt += "- Keep the same language as the input\n"
	prompt += "- Do not add any new information\n"
	prompt += "- Do not remove meaningful content\n"
	prompt += "- Output ONLY the cleaned text, nothing else\n"
	prompt += "- If the input is empty or nonsensical, return it as-is\n"

	if len(keywords) > 0 {
		prompt += fmt.Sprintf("\nContext keywords (use correct spelling for these terms): %s\n", strings.Join(keywords, ", "))
	}

	return prompt
}

// BuildUserPrompt generates the user prompt with the text to process
func BuildUserPrompt(text string, customPrompt string) string {
	if customPrompt != "" {
		return fmt.Sprintf("%s\n\nText to process:\n%s", customPrompt, text)
	}
	return text
}

// BuildVimSystemPrompt generates the system prompt for vim keystroke translation
func BuildVimSystemPrompt(customActions []string, customPrompt string) string {
	prompt := `You are a vim command translator. Your job is to convert spoken English descriptions into exact vim keystrokes.

Default whitelist of allowed vim actions:

Motions: h j k l w b e 0 $ ^ gg G { } %
Editing: i a o O x dd yy p P u <C-r>
Search: / ? n N
Navigation: gd gD gr ]q [q
Window/Buffer: <C-w>s <C-w>v gt gT`

	if len(customActions) > 0 {
		prompt += fmt.Sprintf("\nCustom actions: %s", strings.Join(customActions, " "))
	}

	prompt += `

Rules:
- Output ONLY the vim keystrokes, nothing else
- Use <Esc> <CR> <C-x> notation for special keys (e.g., <C-w> for Ctrl+W, <Esc> for Escape, <CR> for Enter)
- Support numeric prefixes (e.g., "move down 5 lines" -> "5j")
- For search commands, include the search term and <CR> (e.g., "search for error" -> "/error<CR>")
- If the spoken text is unclear or doesn't map to a vim action, return an empty string
- Never output text that is not a valid vim keystroke sequence
- Combine keystrokes as needed (e.g., "delete 3 words" -> "3dw", "change inside quotes" -> "ci\"")
- For insert mode commands, only output the keystroke to enter insert mode, not the text to type`

	if customPrompt != "" {
		prompt += "\n\n" + customPrompt
	}

	return prompt
}

// BuildVimUserPrompt generates the user prompt for vim keystroke translation
func BuildVimUserPrompt(spokenText string) string {
	return fmt.Sprintf("Convert this spoken command to vim keystrokes: %s", spokenText)
}
