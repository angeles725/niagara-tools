package skillinstall

import (
	"embed"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// promptName is a kit's prompt template, beside its tracked SKILL.md: <kit>/skill/prompt.md. It is the one
// source of the prompt file a pi-style harness needs for a bare /<skill>.
const promptName = "prompt.md"

// defaultPrompt is the template for a skill whose kit has no prompt.md (a skill added to the SKILLS table
// without one). @SKILL@ is the skill name and @SKILL_FILE@ the installed SKILL.md path, in both.
//
//go:embed prompts/default.md
var defaultPrompt embed.FS

// promptSource is the template file beside the tracked launcher.
func promptSource(tracked string) string { return filepath.Join(filepath.Dir(tracked), promptName) }

// promptLabel names the template renderPrompt reads, for a report: the kit's prompt.md, or the built-in
// default when the kit has none (the same test as renderPrompt), so a report never names a missing file.
func promptLabel(tracked string) string {
	source := promptSource(tracked)
	if _, err := os.Lstat(source); errors.Is(err, fs.ErrNotExist) {
		return "the built-in default (the kit has no " + source + ")"
	}
	return source
}

// renderPrompt is the prompt file for skill whose tracked launcher is tracked and whose installed launcher
// is skillFile, made absolute so the harness can read it from any working directory. A kit template that
// exists but cannot be read is an error, never the default.
func renderPrompt(skill, tracked, skillFile string) ([]byte, error) {
	abs, err := filepath.Abs(skillFile)
	if err != nil {
		return nil, err
	}
	text, err := os.ReadFile(promptSource(tracked))
	if errors.Is(err, fs.ErrNotExist) {
		text, err = defaultPrompt.ReadFile("prompts/default.md")
	}
	if err != nil {
		return nil, err
	}
	r := strings.NewReplacer("@SKILL_FILE@", abs, "@SKILL@", skill)
	return []byte(r.Replace(string(text))), nil
}
