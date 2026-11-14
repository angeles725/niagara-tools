package installtools

import (
	"fmt"
	"io"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/gitcmd"
)

// Scope selects the configuration files GitConfig reads.
type Scope bool

const (
	Effective Scope = false // system, global, local and worktree, as git itself resolves a key
	Local     Scope = true  // the repository's own config only (`git config --local`)
)

// GitConfig is the value `git -C dir config [--local] --get <key>` prints ("" when the key is unset, the
// last value when it is set several times). Unlike `--get || true`, a git that cannot read the
// configuration is an error and never reads as "unset": the whole list is read (`-z --list`, exit 0 with
// or without the key) and searched here. key is lower case, as git lists section and variable names.
func GitConfig(dir string, errTo io.Writer, scope Scope, key string) (string, error) {
	args := []string{"config"}
	if scope == Local {
		args = append(args, "--local")
	}
	out, ok := gitcmd.Run(dir, errTo, append(args, "-z", "--list")...)
	if !ok {
		return "", fmt.Errorf("git config --list failed in %s", dir)
	}
	value := ""
	for _, entry := range strings.Split(out, "\x00") {
		name, v, _ := strings.Cut(entry, "\n")
		if name == key {
			value = v
		}
	}
	return value, nil
}
