package scaffoldmodule

import (
	"fmt"
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// stripCommand is an internal mode, not part of the usage: `scaffold-module --strip-offload-markers <dir>` removes
// the offload marker lines (and the row suffix) from every file under dir in place, keeping every offload line, which
// is what --with-offload emits. CI and the bats round trip use it on a copy of the fixture, so one definition
// (filterOffload) decides what a marker is. A refused run changes no file.
const (
	stripCommand = "--strip-offload-markers"
	stripUsage   = "usage: scaffold-module.sh --strip-offload-markers <dir>\n"
)

func stripMarkers(args []string, s cli.Stdio) int {
	if len(args) != 2 {
		fmt.Fprint(s.Err, stripUsage)
		return ExitUsage
	}
	files, err := regularFiles(args[1], "")
	if err != nil {
		fmt.Fprintf(s.Err, writeFault, err)
		return ExitEnv
	}
	rewrites := map[string]string{}
	for _, rel := range files {
		data, err := os.ReadFile(args[1] + "/" + rel)
		if err != nil {
			fmt.Fprintf(s.Err, writeFault, err)
			return ExitEnv
		}
		text := string(data)
		if !strings.Contains(text, "offload:") {
			continue
		}
		out, err := filterOffload(text, true)
		if err != nil {
			fmt.Fprintf(s.Err, writeFault, fmt.Errorf("%s: %w", rel, err))
			return ExitEnv
		}
		if out != text {
			rewrites[rel] = out
		}
	}
	for rel, out := range rewrites {
		path := args[1] + "/" + rel
		st, err := os.Stat(path)
		if err == nil {
			err = os.WriteFile(path, []byte(out), st.Mode().Perm())
		}
		if err != nil {
			fmt.Fprintf(s.Err, writeFault, err)
			return ExitEnv
		}
	}
	return ExitOK
}
