package buildmodule

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// scriptName is the launcher in the toolbelt directory; its header comment is the usage text.
const scriptName = "build.sh"

// usageText is the usage: the header comment of the invoked build.sh, read when it is printed, as the bash
// tool read its own file. The text has one home (the script) and no copy to keep equal.
func usageText(s cli.Stdio) string {
	script := filepath.Join(s.Env(ToolbeltEnv), scriptName)
	data, err := os.ReadFile(script)
	if err != nil {
		return "build.sh: usage text unavailable: cannot read " + script + "\n"
	}
	return scriptHeader(string(data))
}

// scriptHeader is the comment block of a script from line 2 up to its first line that is not a comment,
// without the leading "#" and one space.
func scriptHeader(script string) string {
	lines := strings.Split(script, "\n")
	var header strings.Builder
	for _, line := range lines[min(1, len(lines)):] {
		if !strings.HasPrefix(line, "#") {
			break
		}
		header.WriteString(strings.TrimPrefix(strings.TrimPrefix(line, "#"), " ") + "\n")
	}
	return header.String()
}
