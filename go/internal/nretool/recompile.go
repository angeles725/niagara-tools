package nretool

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

const (
	recompileClass = "program:com.tridium.program.ui.RecompileTool"
	recompileUsage = "usage: nre-tool recompile <niagara_home> <dir|bog> [-alias:<a>] [-tsaUrl:<u>]"
)

// recompile runs RecompileTool <dir|bogfile> [-alias:] [-tsaUrl:] (B1453 §1453.4; documented as
// nre program:com.tridium.program.ui.RecompileTool). It never passes -password: argv shows in the process list
// and shell history (§1453.5), and RecompileTool prompts when it is omitted (RecompileTool.java:110-111), so
// stdin and the terminal go to nre. main exits 0 after a failure it prints (File not found :57, exit :92, ERROR
// lines :157 and :169), so the output decides: an ERROR line, or no "--- Processed <file>!" line, is exit 1.
func recompile(args []string, s cli.Stdio) int {
	if len(args) < 2 {
		fmt.Fprintln(s.Err, recompileUsage)
		return exitUsage
	}
	home, target, flags := args[0], "", []string(nil)
	for _, a := range args[1:] {
		low := strings.ToLower(a)
		switch {
		case strings.HasPrefix(low, "-password"):
			fmt.Fprintln(s.Err, "nre-tool: recompile never passes the password in argv; nre prompts for it")
			return exitUsage
		case strings.HasPrefix(low, "-alias:"), strings.HasPrefix(low, "-tsaurl:"):
			flags = append(flags, a)
		case strings.HasPrefix(a, "-"):
			name, _, _ := strings.Cut(a, ":")
			fmt.Fprintf(s.Err, "nre-tool: recompile: unknown flag %s (allowed: -alias:, -tsaUrl:)\n", name)
			return exitUsage
		case target == "":
			target = a
		default:
			fmt.Fprintln(s.Err, recompileUsage)
			return exitUsage
		}
	}
	if target == "" {
		fmt.Fprintln(s.Err, recompileUsage)
		return exitUsage
	}
	if _, err := os.Stat(target); err != nil {
		fmt.Fprintf(s.Err, "nre-tool: recompile: %s does not exist\n", target)
		return exitUsage
	}
	nre := findNre(home)
	if nre == "" {
		fmt.Fprintf(s.Err, "nre-tool: no bin/nre.exe or bin/nre under %s\n", home)
		return exitUsage
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		fmt.Fprintf(s.Err, "nre-tool: recompile: %v\n", err)
		return exitUsage
	}
	code, out := runStreaming(nre, append([]string{recompileClass, nativePath(nre, abs)}, flags...), s)
	if code != 0 {
		return code
	}
	return judge(out, s)
}

// judge is exit 1 when RecompileTool printed an ERROR line or never printed that it processed a bog file.
func judge(out string, s cli.Stdio) int {
	processed := false
	for _, l := range strings.Split(out, "\n") {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "ERROR") {
			fmt.Fprintln(s.Err, "nre-tool: RecompileTool reported ERROR (it exits 0 after printing it)")
			return exitFound
		}
		processed = processed || (strings.HasPrefix(l, "--- Processed ") && strings.HasSuffix(l, "!"))
	}
	if !processed {
		fmt.Fprintln(s.Err, "nre-tool: RecompileTool processed no bog file (no \"--- Processed <file>!\" line; "+
			"a missing file or a bad signing key exits 0)")
		return exitFound
	}
	return 0
}
