package golden

import (
	"flag"
	"os"
	"strings"
	"testing"
)

// update is the -update flag of every test binary that imports this package (go test ./x -update):
// a case whose exit code, stdout or stderr differs is re-recorded instead of failed. Only its want
// and stderr sections change; a tree difference is still a failure. A package that imports golden must not
// declare a flag of the same name.
var update bool

func init() {
	if testing.Testing() { // a program that imports the package for BuildJar gets no flag
		flag.BoolVar(&update, "update", false, "re-record the want and stderr sections of the golden cases that differ")
	}
}

// record rewrites the want and stderr sections of the case file at path (an absolute path, as the run may
// have changed the working directory), keeping its mode and every other byte.
func record(path, want, stderr string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(rewrite(string(data), want, stderr)), fi.Mode().Perm())
}

// rewrite is case text with its want and stderr sections set to want and stderr; a case without a stderr
// section gets one after want when stderr is not empty.
func rewrite(text, want, stderr string) string {
	hasStderr := false
	for _, sec := range sections(text, "-- ", " --\n") {
		hasStderr = hasStderr || sec[0] == "stderr" || sec[0] == "stderr (noeol)"
	}
	var b strings.Builder
	skip := false
	for _, ln := range strings.SplitAfter(text, "\n") {
		if strings.HasPrefix(ln, "-- ") && strings.HasSuffix(ln, " --\n") {
			skip = false
			switch strings.TrimSuffix(strings.TrimPrefix(ln, "-- "), " --\n") {
			case "want", "want (noeol)":
				b.WriteString(spell("want", want))
				if !hasStderr && stderr != "" {
					b.WriteString(spell("stderr", stderr))
				}
				skip = true
			case "stderr", "stderr (noeol)":
				b.WriteString(spell("stderr", stderr))
				skip = true
			}
			if skip {
				continue
			}
		}
		if !skip {
			b.WriteString(ln)
		}
	}
	return b.String()
}

// spell is one output section: the plain header when body is empty or ends in a newline, else "(noeol)".
func spell(name, body string) string {
	if body == "" || strings.HasSuffix(body, "\n") {
		return "-- " + name + " --\n" + body
	}
	return "-- " + name + " (noeol) --\n" + body + "\n"
}
