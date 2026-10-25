// Package golden runs a rule's golden cases: testdata/<case>.txt files recorded from the bash lint the rule
// replaces, by pointing that lint's bats suite at a recording wrapper (go/README.md, Adding a lint rule).
//
// A case file holds "-- args --" (one argument per line), "-- want --" ("exit <n>" then the exact stdout),
// "-- stderr --" and one "-- <path> --" section per file of the tree; "-- <path> (noeol) --" is a file whose
// last line has no "\n". "@DIR@" stands for the tree's absolute path in every section.
//
// A file that is not plain text is spelled by a marker after its path: "(base64)" (the body is the bytes in
// base64), "(class N)" (no body: an 8-byte class header, CAFEBABE, minor 0, major N), "(dir)" (an empty
// directory) and "(jar)": a zip whose entries are the body's "--- <entry> (stored|deflated)[ <marker>] ---"
// sub-sections, in archive order, each spelled as a file is (an entry name ending in "/" is a directory).
package golden

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// Case is one recorded run of the bash lint. Files maps a tree path to its bytes; Dirs are empty directories.
type Case struct {
	Args         []string
	Want, Stderr string
	Files        map[string]string
	Dirs         []string
}

// Read parses one case file.
func Read(path string) (Case, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Case{}, err
	}
	c := Case{Files: map[string]string{}}
	var bad error
	for _, sec := range sections(string(data), "-- ", " --\n") {
		switch name, body := sec[0], sec[1]; {
		case name == "args":
			c.Args = strings.Split(strings.TrimSuffix(body, "\n"), "\n")
			if body == "" {
				c.Args = nil
			}
		case name == "want", name == "want (noeol)":
			c.Want = noeol(name, body)
		case name == "stderr", name == "stderr (noeol)":
			c.Stderr = noeol(name, body)
		case strings.HasSuffix(name, " (dir)"):
			c.Dirs = append(c.Dirs, strings.TrimSuffix(name, " (dir)"))
		case strings.HasSuffix(name, " (jar)"):
			jar, err := BuildJar(body)
			if err != nil && bad == nil {
				bad = fmt.Errorf("%s: %s: %w", path, name, err)
			}
			c.Files[strings.TrimSuffix(name, " (jar)")] = jar
		default:
			p, text, err := decode(name, body)
			if err != nil && bad == nil {
				bad = fmt.Errorf("%s: %s: %w", path, name, err)
			}
			c.Files[p] = text
		}
	}
	return c, bad
}

// noeol is body without its last "\n" when name ends in " (noeol)": output whose last line has none.
func noeol(name, body string) string {
	if strings.HasSuffix(name, " (noeol)") {
		return strings.TrimSuffix(body, "\n")
	}
	return body
}

// sections splits text at its header lines (open + name + close) into {name, body} pairs; text before the
// first header is dropped.
func sections(text, open, close string) [][2]string {
	var out [][2]string
	for _, ln := range strings.SplitAfter(text, "\n") {
		if strings.HasPrefix(ln, open) && strings.HasSuffix(ln, close) {
			out = append(out, [2]string{strings.TrimSuffix(strings.TrimPrefix(ln, open), close), ""})
		} else if len(out) > 0 {
			out[len(out)-1][1] += ln
		}
	}
	return out
}

// decode is one file or jar entry section: its path without the marker and its bytes.
func decode(name, body string) (string, string, error) {
	switch {
	case strings.HasSuffix(name, " (noeol)"):
		return strings.TrimSuffix(name, " (noeol)"), strings.TrimSuffix(body, "\n"), nil
	case strings.HasSuffix(name, " (base64)"):
		b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(body))
		return strings.TrimSuffix(name, " (base64)"), string(b), err
	}
	var major int
	if i := strings.LastIndex(name, " (class "); i >= 0 && strings.HasSuffix(name, ")") {
		if _, err := fmt.Sscanf(name[i:], " (class %d)", &major); err == nil {
			return name[:i], string([]byte{0xca, 0xfe, 0xba, 0xbe, 0, 0, 0, byte(major)}), nil
		}
	}
	return name, body, nil
}

// BuildJar is the zip a "(jar)" section spells: one "--- <entry> (stored|deflated)[ <marker>] ---" header
// per entry, in archive order, its body spelled as a file section is (internal/storedrepack/jarlist builds its
// input jars with it too).
func BuildJar(body string) (string, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, sec := range sections(body, "--- ", " ---\n") {
		method, name := zip.Deflate, sec[0]
		for marker, m := range map[string]uint16{" (stored)": zip.Store, " (deflated)": zip.Deflate} {
			if i := strings.LastIndex(name, marker); i >= 0 {
				method, name = m, name[:i]+name[i+len(marker):]
			}
		}
		entry, data, err := decode(name, sec[1])
		if err != nil {
			return "", err
		}
		w, err := zw.CreateHeader(&zip.FileHeader{Name: entry, Method: method})
		if err == nil {
			_, err = w.Write([]byte(data))
		}
		if err != nil {
			return "", err
		}
	}
	err := zw.Close()
	return buf.String(), err
}

// Run checks that spec, run alone as its toolbelt wrapper runs it, prints every case's recorded exit code,
// stdout and stderr byte for byte. It fails when testdata holds fewer than min cases.
func Run(t *testing.T, spec lint.Spec, min int) {
	t.Helper()
	command := lint.Command(spec)
	RunMain(t, func(args []string, s cli.Stdio) int { return command(append([]string{spec.ID()}, args...), s) }, min)
}

// RunMain is Run for a command that is not a lint rule: m gets each case's arguments as recorded. It is a
// Runner for at least min cases, so every case also proves the run left its tree untouched.
func RunMain(t *testing.T, m cli.Main, min int) {
	t.Helper()
	Runner{Cases: min, AtLeast: true, Main: func(_, args []string, s cli.Stdio) int { return m(args, s) }}.Run(t)
}

// Unreadable checks the awk_lint_run failure contract of a rule that sets ContinueOnWalkError: under a root
// with an unreadable sub-directory and an unreadable file beside a.java (which holds src), the rule prints
// the walk env line, then the scan env line, prints the rows a.java alone gives (at least one) and exits 3.
func Unreadable(t *testing.T, spec lint.Spec, src string) {
	t.Helper()
	if skip, msg := rootVerdict(os.Geteuid(), os.Getenv(RootOptIn), []string{"a/U.java", "locked"}); skip {
		t.Skip(msg)
	} else if msg != "" {
		t.Fatal(msg)
	}
	run := func(dir string) (int, string, string) {
		var o, e bytes.Buffer
		code := lint.Command(spec)([]string{spec.ID(), dir}, cli.Stdio{Out: &o, Err: &e})
		return code, strings.ReplaceAll(o.String(), dir, "@DIR@"), e.String()
	}
	clean, dir := t.TempDir(), t.TempDir()
	for _, d := range []string{clean, dir} {
		if err := os.MkdirAll(filepath.Join(d, "a"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "a", "A.java"), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	u, locked := filepath.Join(dir, "a", "U.java"), filepath.Join(dir, "locked")
	if err := os.WriteFile(u, []byte(src), 0); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	_, rows, _ := run(clean)
	code, out, stderr := run(dir)
	want := spec.ID() + ": cannot list every file under " + dir + ": find: '" + locked + "': Permission denied\n" +
		spec.ID() + ": cannot scan " + u + ": awk: fatal: cannot open file `" + u + "' for reading: Permission denied\n"
	if rows == "" || code != 3 || out != rows || stderr != want {
		t.Errorf("exit %d out %q err %q, want 3, %q and %q", code, out, stderr, rows, want)
	}
}
