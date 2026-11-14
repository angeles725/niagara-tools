package scaffoldmodule

import (
	"os"
	"path/filepath"
	"testing"
)

// add-concern runs <toolbelt>/module-map.sh with every argument and returns its output and exit code.
func TestAddConcernPassthrough(t *testing.T) {
	tb := t.TempDir()
	stub := "#!/bin/sh\nprintf 'args:'; for a in \"$@\"; do printf ' [%s]' \"$a\"; done; printf '\\n'\nprintf 'to-stderr\\n' >&2\nexit 7\n"
	if err := os.WriteFile(filepath.Join(tb, "module-map.sh"), []byte(stub), 0o755); err != nil {
		t.Fatal(err)
	}
	r := run(t, tb, "add-concern", "Battery", "--root", "a b", "--adapter")
	if r.code != 7 || r.out != "args: [add-concern] [Battery] [--root] [a b] [--adapter]\n" || r.err != "to-stderr\n" {
		t.Errorf("code=%d out=%q err=%q", r.code, r.out, r.err)
	}
	// add-concern alone is still the passthrough, not a usage error of this tool
	if r := run(t, tb, "add-concern"); r.code != 7 || r.out != "args: [add-concern]\n" {
		t.Errorf("alone: code=%d out=%q err=%q", r.code, r.out, r.err)
	}
}

// A toolbelt without module-map.sh is the shell's "command not found" exit, with one line on stderr.
func TestAddConcernWithoutModuleMap(t *testing.T) {
	tb := t.TempDir()
	r := run(t, tb, "add-concern", "Battery")
	if r.code != 127 || r.out != "" || r.err != "scaffold-module: cannot run "+tb+"/module-map.sh: no such file or directory\n" {
		t.Errorf("code=%d out=%q err=%q", r.code, r.out, r.err)
	}
}
