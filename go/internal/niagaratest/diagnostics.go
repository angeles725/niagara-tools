package niagaratest

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// writeRunLog writes the redacted run log, mode 0600 in a 0700 dir, and prints the row naming it. A failed run
// always keeps one; a green run only for an explicit --log-dir or --keep-home. It returns false when a log was
// due and could not be written.
func (h host) writeRunLog(p *plan, ev evidence, code int, s cli.Stdio) bool {
	if code == exitPass && !p.logDirSet && !p.keepHome {
		return true
	}
	status, label := "FAIL", "diagnostics kept"
	if code == exitPass {
		status, label = "PASS", "run log"
	}
	if p.logDir == "" {
		row(s, "FAIL", "diagnostics: no log dir (no $XDG_STATE_HOME or home); pass --log-dir")
		return false
	}
	var b strings.Builder
	fmt.Fprintf(&b, "run-niagara-test run log\nmodule: %s\npart: %s\ninstall: %s\nrun id: %s\nwritten: %s\n",
		p.module, p.part, p.install.win, p.runID, time.Now().UTC().Format(time.RFC3339))
	section := func(title string, lines []string) {
		fmt.Fprintf(&b, "\n== %s ==\n", title)
		for _, l := range lines {
			b.WriteString(l + "\n")
		}
	}
	rows := []string{"(none)"}
	if p.rows != nil {
		rows, _ = p.rows.snapshot()
	}
	section("runner rows", rows)
	section("test.exe stderr", ev.stderr)
	section("test.exe stdout", ev.stdout)
	section("test.exe -output: dir", ev.listing)
	path, err := createLog(p.logDir, p.module)
	if err != nil {
		row(s, "FAIL", "diagnostics: cannot write the run log in "+p.logDir+": "+err.Error())
		return false
	}
	red, kept := h.redactorFor(p), true
	if code != exitPass {
		stem := strings.TrimSuffix(path.name, ".log")
		rows, kerr := keepTestNG(ev.testng, stem, red)
		if kerr == nil {
			var more []string
			more, kerr = keepStderr(ev.fullStderr, stem, red)
			rows = append(rows, more...)
		}
		if kerr != nil {
			rows, kept = append(rows, "cannot keep the fail evidence: "+kerr.Error()), false
		}
		section("kept evidence", rows)
	}
	name := exitNames[code]
	if name == "" {
		name = "unknown"
	}
	fmt.Fprintf(&b, "\nexit: %d (%s)\n", code, name)
	_, err = path.f.WriteString(red.apply(b.String()))
	if cerr := path.f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		row(s, "FAIL", "diagnostics: cannot write the run log in "+p.logDir+": "+err.Error())
		return false
	}
	p.logPath = path.name
	row(s, status, label+": "+path.name)
	if !kept {
		row(s, "FAIL", "diagnostics: cannot keep the fail evidence beside "+path.name)
	}
	return kept
}

type logFile struct {
	f    *os.File
	name string
}

// createLog makes <dir>/<UTC timestamp>-<module>.log exclusively (a -<n> suffix when that second is taken).
func createLog(dir, module string) (logFile, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return logFile{}, err
	}
	stamp := time.Now().UTC().Format("20060102T150405Z")
	for i := 1; i <= 100; i++ {
		n := filepath.Join(dir, fmt.Sprintf("%s-%s.log", stamp, module))
		if i > 1 {
			n = filepath.Join(dir, fmt.Sprintf("%s-%s-%d.log", stamp, module, i))
		}
		f, err := os.OpenFile(n, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		return logFile{f, n}, err
	}
	return logFile{}, fmt.Errorf("100 logs named %s-%s already", stamp, module)
}
