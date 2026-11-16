package skillinstall

import (
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// harness is one agent runtime a skill installs into. The table below is data; the logic reads it.
type harness struct {
	name string
	// dir is the harness directory relative to the home; envDir, when set in the environment, replaces it.
	dir, envDir string
	// optional: an absent directory is an "absent" row, never created. Claude's is created, as it always was.
	optional bool
	// prompts: a bare /<skill> needs <dir>/prompts/<skill>.md (pi and its derivatives).
	prompts bool
}

// harnesses is the --harness table, in the order --harness all installs and checks them.
var harnesses = []harness{
	{name: "claude", dir: ".claude"},
	{name: "pi", dir: ".pi/agent", envDir: "PI_CODING_AGENT_DIR", optional: true, prompts: true},
	{name: "gentle-shell", dir: ".gentle-shell/agent", optional: true, prompts: true},
}

const (
	defaultHarness = "claude"
	allHarnesses   = "all"
)

// selectHarnesses resolves a --harness value; ok is false for an unknown name.
func selectHarnesses(name string) (selected []harness, ok bool) {
	if name == allHarnesses {
		return harnesses, true
	}
	for _, h := range harnesses {
		if h.name == name {
			return []harness{h}, true
		}
	}
	return nil, false
}

// harnessChoices is the accepted --harness values for a message ("claude, pi, gentle-shell or all").
func harnessChoices() string {
	names := strings.Split(harnessNames("|"), "|")
	return strings.Join(names[:len(names)-1], ", ") + " or " + allHarnesses
}

// root is the harness directory under home, or the directory its environment variable names.
func (h harness) root(home string, s cli.Stdio) string {
	if h.envDir != "" {
		if dir := s.Env(h.envDir); dir != "" {
			return dir
		}
	}
	return filepath.Join(home, h.dir)
}

// skillFile is where a skill's launcher is installed in this harness.
func (h harness) skillFile(root, skill string) (dir, file string) {
	dir = filepath.Join(root, "skills", skill)
	return dir, filepath.Join(dir, "SKILL.md")
}

// promptFile is the prompt template that gives the harness a bare /<skill>; "" when it needs none.
func (h harness) promptFile(root, skill string) string {
	if !h.prompts {
		return ""
	}
	return filepath.Join(root, "prompts", skill+".md")
}

// label is the skill column of a drift row: the bare skill for the default line, skill[harness] otherwise.
func label(skill string, h harness, explicit bool) string {
	if !explicit {
		return skill
	}
	return skill + "[" + h.name + "]"
}

// harnessFlag is the --harness part of a remedy command ("" for the default harness).
func harnessFlag(h harness) string {
	if h.name == defaultHarness {
		return ""
	}
	return " --harness " + h.name
}

// harnessNames joins the --harness values, "all" last, for the usage line.
func harnessNames(sep string) string {
	all := make([]string, 0, len(harnesses)+1)
	for _, h := range harnesses {
		all = append(all, h.name)
	}
	return strings.Join(append(all, allHarnesses), sep)
}
