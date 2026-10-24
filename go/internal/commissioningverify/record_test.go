package commissioningverify

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// section is a case-file section header line.
var section = regexp.MustCompile(`(?m)^-- .* --$`)

// TestRecord re-records the want and stderr sections of every recorded (non truth-*) case from a bash
// commissioning-verify.sh (testdata/README.md): CV_RECORD=<script> go test -run TestRecord. The script runs as
// commissioning-verify.sh of a toolbelt copy (the case's stubs included), in the case tree, with minimalEnv.
func TestRecord(t *testing.T) {
	script := os.Getenv("CV_RECORD")
	if script == "" {
		t.Skip("set CV_RECORD=<bash commissioning-verify.sh> to re-record the testdata cases")
	}
	bash, err := os.ReadFile(script)
	must(t, err)
	repo, err := filepath.Abs("../../..")
	must(t, err)
	// Read and root-check every recorded case before the first rewrite, so testdata is never left half re-recorded.
	var paths []string
	var all []golden.Case
	for _, p := range cases(t, repo) {
		if strings.HasPrefix(filepath.Base(p), "truth-") {
			continue
		}
		c, err := golden.Read(p)
		must(t, err)
		unreadableAsRoot(t, c.Files)
		paths, all = append(paths, p), append(all, c)
	}
	for i, p := range paths {
		c := all[i]
		c.Files["toolbelt/commissioning-verify.sh"] = string(bash)
		dir := t.TempDir()
		tb := materialize(t, dir, repo, c)
		sub := strings.NewReplacer("@DIR@", dir, "@REPO@", repo)
		args := make([]string, len(c.Args))
		for i, a := range c.Args {
			args[i] = sub.Replace(a)
		}
		var o, e bytes.Buffer
		cmd := exec.Command(filepath.Join(tb, "commissioning-verify.sh"), args...)
		cmd.Dir, cmd.Env, cmd.Stdout, cmd.Stderr = dir, minimalEnv(), &o, &e
		code := 0
		if err := cmd.Run(); err != nil {
			var ee *exec.ExitError
			if !errors.As(err, &ee) {
				t.Fatal(err)
			}
			code = ee.ExitCode()
		}
		back := strings.NewReplacer(dir, "@DIR@", repo, "@REPO@")
		raw, err := os.ReadFile(p)
		must(t, err)
		text := string(raw)
		start := strings.Index(text, "-- want --\n")
		after := strings.Index(text, "-- stderr --\n") + len("-- stderr --\n")
		end := len(text)
		if m := section.FindStringIndex(text[after:]); m != nil {
			end = after + m[0]
		}
		recorded := fmt.Sprintf("-- want --\nexit %d\n%s-- stderr --\n%s", code, back.Replace(o.String()), back.Replace(e.String()))
		must(t, os.WriteFile(p, []byte(text[:start]+recorded+text[end:]), 0o644))
	}
}
