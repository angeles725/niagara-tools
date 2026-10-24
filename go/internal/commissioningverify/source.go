package commissioningverify

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

// srcDir is one source directory the lints run on and the label of its rows (the -rt artifact name).
type srcDir struct{ path, label string }

// sourceDirs lists <root>/*/*-rt/src like `find -H <root> -mindepth 3 -maxdepth 3 -type d -name src | grep --
// '-rt/src$' | sort` (the root followed, no symlink below it, unreadable directories skipped), else <root>/src
// when that is a directory (labelled with the root's base name).
func sourceDirs(root string) []srcDir {
	var out []srcDir
	for _, a := range subdirs(root) {
		for _, b := range subdirs(join(root, a)) {
			if !strings.HasSuffix(b, "-rt") {
				continue
			}
			for _, c := range subdirs(join(join(root, a), b)) {
				if c == "src" {
					out = append(out, srcDir{join(join(join(root, a), b), c), b})
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].path < out[j].path })
	if len(out) == 0 && isDir(root+"/src") {
		label := strings.TrimRight(root, "/") // basename "$(dirname <root>/src)", no path cleaning
		if label = label[strings.LastIndexByte(label, '/')+1:]; label == "" {
			label = "/"
		}
		out = append(out, srcDir{root + "/src", label})
	}
	return out
}

// subdirs names the entries of dir that are directories themselves (a symlink is not).
func subdirs(dir string) []string {
	entries, _ := os.ReadDir(dir)
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}

// join is find's path spelling: no separator is added after a directory that already ends in one.
func join(dir, name string) string {
	if strings.HasSuffix(dir, "/") {
		return dir + name
	}
	return dir + "/" + name
}

// sourceLint is one per -rt/src lint: its check name, toolbelt script, the row statuses it relays and the
// detail of its PASS row.
type sourceLint struct {
	check, script string
	warn          bool
	clean         string
}

var sourceLints = []sourceLint{
	{"config-sanity", "lint-config-sanity", true, "CS1/CS2/CS3 clean"},
	{"status-parity", "lint-status-parity", true, "N:M config/status parity clean"},
	{"recovery-path", "lint-recovery-path", false, "protection-output recovery path clean"},
	{"link-target-flags", "lint-link-target-flags", false, ""},
}

// sourceChecks runs every source lint on src. Exit 3 is a SKIP row, the lint's FAIL (and WARN) rows are relayed
// with the lint prefix dropped, a non-zero exit with no such row is a FAIL row (never PASS), else the check
// PASSes; link-target-flags gets the wiring map and names it in its Table 2 reason and PASS detail.
func (r *report) sourceChecks(tb string, src srcDir, strict bool, wmap string) {
	for _, l := range sourceLints {
		var args []string
		clean, ltf := l.clean, l.check == "link-target-flags"
		switch {
		case l.check == "status-parity" && strict:
			args = append(args, "--strict")
		case ltf && wmap != "":
			args, clean = append(args, "--wiring-map", wmap), "LTF1/LTF2 clean (wiring map: "+wmap+")"
		case ltf:
			clean = "LTF1/LTF2 clean (no wiring map)"
		}
		out, code := run(filepath.Join(tb, l.script+".sh"), r.env, append(args, src.path)...)
		if ltf && wmap != "" {
			out = strings.ReplaceAll(out, "(wiring-map Table 2)", "(wiring-map Table 2: "+wmap+")")
		}
		check := l.check + ":" + src.label
		var rows [][2]string
		for _, ln := range strings.Split(out, "\n") {
			for _, st := range []string{failed, warn} {
				if strings.HasPrefix(ln, st) && (st == failed || l.warn) {
					rows = append(rows, [2]string{st, strings.TrimPrefix(ln, st+"  "+l.script+"  ")})
				}
			}
		}
		switch {
		case code == 3:
			r.emit(skip, check, "env fault (exit 3) — run "+l.script+".sh manually")
		case len(rows) > 0:
			for _, row := range rows {
				r.emit(row[0], check, row[1])
			}
		case code != 0: // 127 = lint missing, 126 not executable, a crash: never PASS
			r.emit(failed, check, fmt.Sprintf("%s.sh exited %d with no FAIL row (crash or unexpected output) -- run it manually", l.script, code))
		default:
			r.emit(pass, check, clean)
		}
	}
}

// run executes a toolbelt script with environment env plus LC_ALL=C and returns its stdout and stderr interleaved, like the shell's
// "$(cmd 2>&1)" (NUL bytes and trailing newlines dropped), and its exit status (127 not found, 126 not
// executable, 128+n killed by signal n).
func run(path string, env []string, args ...string) (string, int) {
	var b bytes.Buffer
	cmd := exec.Command(path, args...)
	cmd.Stdout, cmd.Stderr, cmd.Env = &b, &b, append(append([]string(nil), env...), "LC_ALL=C")
	err := cmd.Run()
	out := strings.TrimRight(strings.ReplaceAll(b.String(), "\x00", ""), "\n")
	var ee *exec.ExitError
	switch {
	case err == nil:
		return out, 0
	case errors.As(err, &ee):
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			return out, 128 + int(ws.Signal())
		}
		return out, ee.ExitCode()
	case errors.Is(err, fs.ErrNotExist):
		return out, 127
	default:
		return out, 126
	}
}

// isDir and isFile are the shell's [ -d ] and [ -f ]: the path, symlinks followed, is a directory or a regular file.
func isDir(p string) bool { st, err := os.Stat(p); return err == nil && st.IsDir() }

func isFile(p string) bool { st, err := os.Stat(p); return err == nil && st.Mode().IsRegular() }
