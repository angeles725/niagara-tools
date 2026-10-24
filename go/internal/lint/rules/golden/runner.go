package golden

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// RootOptIn lets a run as root skip (instead of fail) the cases that hold an owner-unreadable entry.
const RootOptIn = "GOLDEN_ALLOW_ROOT_SKIP"

// Runner replays a directory of golden cases hermetically (record/README.md has the case format).
// A case's tree is written under a fresh dir, the command runs with only the case's environment, and the
// exit code, stdout, stderr and the whole tree the run leaves must be exactly what the case records.
type Runner struct {
	// Dir holds the cases (*.txt) and, for a set recorded by record/record.sh, its record.env.
	Dir string
	// Cases is the exact number of case files (a minimum with AtLeast).
	Cases   int
	AtLeast bool
	// Subst maps more placeholders to their values (e.g. "@REPO@" to the repository root).
	Subst map[string]string
	// Setup prepares the case dir after its tree is written and before the run; it may add to env. It
	// returns the entries it made, which the tree comparison skips ("p/" skips p and all below it).
	Setup func(t *testing.T, dir string, env map[string]string) []string
	// Main runs the command: env is the case's whole environment, sorted, and s.Getenv reads only it.
	Main func(env, args []string, s cli.Stdio) int
}

// defaultEnv is the environment a case gets for each of these variables it does not declare (record.sh gives
// the bash run the same). The private recorders before the Runner left HOME and LC_ALL unset; the preflight,
// commissioning-verify and new-retro cases re-recorded byte-identical under these defaults, and none of those
// ports reads HOME or LC_ALL itself (commissioning-verify adds LC_ALL=C to its lints' env on its own).
var defaultEnv = map[string]string{"PATH": "/usr/bin:/bin", "HOME": "@DIR@", "LC_ALL": "C"}

// Run replays every case of r.Dir as a subtest, in name order, one at a time.
func (r Runner) Run(t *testing.T) {
	t.Helper()
	paths, _ := filepath.Glob(filepath.Join(r.dir(), "*.txt"))
	if msg := countProblem(len(paths), r.Cases, r.AtLeast); msg != "" {
		t.Fatal(msg)
	}
	defer syscall.Umask(syscall.Umask(0o022)) // files the run creates get the modes the recorder's runs gave them
	for _, p := range paths {
		t.Run(strings.TrimSuffix(filepath.Base(p), ".txt"), func(t *testing.T) {
			problems, skip := r.replay(t, p)
			for _, msg := range problems {
				t.Error(msg)
			}
			if skip != "" && len(problems) == 0 {
				t.Skip(skip)
			}
		})
	}
}

func (r Runner) dir() string {
	if r.Dir == "" {
		return "testdata"
	}
	return r.Dir
}

// countProblem says why n case files are not the want the runner expects ("" when they are).
func countProblem(n, want int, atLeast bool) string {
	if n == want || (atLeast && n > want) {
		return ""
	}
	if atLeast {
		return fmt.Sprintf("found %d golden cases, want at least %d", n, want)
	}
	return fmt.Sprintf("found %d golden cases, want exactly %d", n, want)
}

// rootVerdict decides a case holding the owner-unreadable entries locked for a run as uid with RootOptIn
// set to optIn: no message runs it; otherwise skip says whether to skip (optIn "1") or fail, and the
// message names the first locked entry in path order.
func rootVerdict(uid int, optIn string, locked []string) (skip bool, msg string) {
	if uid != 0 || len(locked) == 0 {
		return false, ""
	}
	if optIn == "1" {
		return true, "running as root: mode 000 does not block reads (" + RootOptIn + "=1)"
	}
	first := locked[0]
	for _, l := range locked {
		first = min(first, l)
	}
	return false, "running as root: " + first + " cannot be made unreadable; run as a normal user or set " + RootOptIn + "=1 to skip"
}

// entry is one tree section: a file (with its bytes), a directory or a symlink (to its target).
type entry struct {
	kind, body string
	mode       os.FileMode
}

// parseEntry reads a tree section name and body: the path, then " (dir)", " (symlink)" and " (mode NNN)"
// markers in any order (golden.Read already took off the content markers).
func parseEntry(name, body string) (string, entry, error) {
	e := entry{kind: "file", body: body, mode: 0o644}
	explicit := false
	for {
		i := strings.LastIndex(name, " (")
		if i < 0 || !strings.HasSuffix(name, ")") {
			break
		}
		switch m := name[i+2 : len(name)-1]; {
		case m == "dir":
			e.kind, e.body = "dir", ""
		case m == "symlink":
			e.kind, e.body = "symlink", strings.TrimSuffix(body, "\n")
		case strings.HasPrefix(m, "mode "):
			v, err := strconv.ParseUint(strings.TrimPrefix(m, "mode "), 8, 32)
			if err != nil || v > 0o777 {
				return "", e, fmt.Errorf("%s: bad mode", name)
			}
			e.mode, explicit = os.FileMode(v), true
		default:
			return name, e, nil // not a marker: part of the path
		}
		name = name[:i]
	}
	if e.kind == "dir" && !explicit {
		e.mode = 0o755
	}
	return name, e, nil
}

// split sorts a case's sections: @env lines, the tree before the run, the tree it must leave (before
// with every "@after <path>" set and every "@gone <path>" removed) and the @kind text.
func split(c Case, sub *strings.Replacer) (env map[string]string, before, after map[string]entry, kind string, err error) {
	env, before, after = map[string]string{}, map[string]entry{}, map[string]entry{}
	gone := map[string]bool{}
	sections := map[string]string{}
	for k, v := range c.Files {
		sections[k] = sub.Replace(v)
	}
	for _, d := range c.Dirs {
		sections[d+" (dir)"] = ""
	}
	for key, body := range sections {
		var target map[string]entry
		switch rest, isAfter := strings.CutPrefix(key, "@after "); {
		case key == "@kind":
			kind = c.Files[key]
		case key == "@env":
			for _, kv := range strings.Split(strings.TrimSuffix(body, "\n"), "\n") {
				if k, v, ok := strings.Cut(kv, "="); ok {
					env[k] = v
				}
			}
		case strings.HasPrefix(key, "@gone "):
			gone[strings.TrimPrefix(key, "@gone ")] = true
		case isAfter:
			target, key = after, rest
		case strings.HasPrefix(key, "@"):
			return nil, nil, nil, "", fmt.Errorf("unknown section %q", key)
		default:
			target = before
		}
		if target != nil {
			p, e, perr := parseEntry(key, body)
			if perr != nil {
				return nil, nil, nil, "", perr
			}
			target[p] = e
		}
	}
	for k, v := range defaultEnv {
		if _, ok := env[k]; !ok {
			env[k] = sub.Replace(v)
		}
	}
	merged := map[string]entry{}
	for p, e := range before {
		if !gone[p] {
			merged[p] = e
		}
	}
	for p, e := range after {
		merged[p] = e
	}
	return env, before, merged, kind, nil
}

// readRecordEnv reads dir/record.env (KEY=VALUE lines, # comments); nil when there is none.
func readRecordEnv(dir string) (map[string]string, error) {
	data, err := os.ReadFile(filepath.Join(dir, "record.env"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	conf := map[string]string{}
	for _, ln := range strings.Split(string(data), "\n") {
		if k, v, ok := strings.Cut(ln, "="); ok && !strings.HasPrefix(ln, "#") {
			conf[k] = strings.Trim(v, `'"`)
		}
	}
	return conf, err
}

// kindProblem checks @kind against the SOURCE, COMMIT and SHA256 of record.env: a recorded case names the pinned source and its tools, a
// truth-* case (a declared difference from the source) and only it starts "truth: ".
func kindProblem(conf map[string]string, name, kind string) string {
	if conf["SOURCE"] == "" {
		return ""
	}
	want := fmt.Sprintf("recorded from %s at %s (sha256 %s)\n", conf["SOURCE"], conf["COMMIT"], conf["SHA256"])
	if strings.HasPrefix(name, "truth-") {
		if !strings.HasPrefix(kind, "truth: ") {
			return fmt.Sprintf("@kind %q: a truth-* case says why with \"truth: \"", kind)
		}
		return ""
	}
	if !strings.HasPrefix(kind, want) || !strings.Contains(kind, "\ntools: ") {
		return fmt.Sprintf("@kind %q, want %q then a tools: line (record/record.sh)", kind, want)
	}
	return ""
}

// replay runs one case and says what differs from its recording; skip is set when root may not run it.
func (r Runner) replay(t *testing.T, path string) (problems []string, skip string) {
	t.Helper()
	c, err := Read(path)
	if err != nil {
		return []string{err.Error()}, ""
	}
	conf, err := readRecordEnv(filepath.Dir(path))
	if err != nil {
		return []string{err.Error()}, ""
	}
	dir, want := t.TempDir(), t.TempDir()
	t.Cleanup(func() { snapshot(dir, nil); snapshot(want, nil) }) // unlock so TempDir can remove them
	pairs := []string{"@DIR@", dir}
	keys := make([]string, 0, len(r.Subst))
	for k := range r.Subst {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		pairs = append(pairs, k, r.Subst[k])
	}
	sub := strings.NewReplacer(pairs...)
	env, before, after, kind, err := split(c, sub)
	if err != nil {
		return []string{err.Error()}, ""
	}
	if msg := kindProblem(conf, strings.TrimSuffix(filepath.Base(path), ".txt"), kind); msg != "" {
		return []string{msg}, ""
	}
	var locked []string
	for _, tree := range []map[string]entry{before, after} {
		for p, e := range tree {
			if e.mode&0o400 == 0 && e.kind != "symlink" {
				locked = append(locked, p)
			}
		}
	}
	if skip, msg := rootVerdict(os.Geteuid(), os.Getenv(RootOptIn), locked); skip {
		return nil, msg
	} else if msg != "" {
		return []string{msg}, ""
	}
	for _, m := range []struct {
		root string
		tree map[string]entry
	}{{dir, before}, {want, after}} {
		if err := materialize(m.root, m.tree); err != nil {
			return []string{err.Error()}, ""
		}
	}
	var skipPaths []string
	if r.Setup != nil {
		skipPaths = r.Setup(t, dir, env)
	}
	args := make([]string, len(c.Args))
	for i, a := range c.Args {
		args[i] = sub.Replace(a)
	}
	environ := make([]string, 0, len(env))
	for k, v := range env {
		environ = append(environ, k+"="+v)
	}
	sort.Strings(environ)
	if conf["CHDIR"] == "1" {
		t.Chdir(dir)
	}
	var o, e bytes.Buffer
	code := r.Main(environ, args, cli.Stdio{Out: &o, Err: &e, Getenv: func(k string) string { return env[k] }})
	back := make([]string, 0, len(pairs))
	for i := 0; i < len(pairs); i += 2 {
		back = append(back, pairs[i+1], pairs[i])
	}
	unsub := strings.NewReplacer(back...)
	got, stderr := unsub.Replace(fmt.Sprintf("exit %d\n%s", code, o.String())), unsub.Replace(e.String())
	if got != c.Want || stderr != c.Stderr {
		problems = append(problems, fmt.Sprintf("got\n%s-- stderr --\n%s\nwant\n%s-- stderr --\n%s", got, stderr, c.Want, c.Stderr))
	}
	left, err := snapshot(dir, skipPaths)
	if err != nil {
		return append(problems, err.Error()), ""
	}
	wanted, err := snapshot(want, nil)
	if err != nil {
		return append(problems, err.Error()), ""
	}
	return append(problems, treeProblems(unsub, left, wanted)...), ""
}

// materialize writes tree under root: parents as 0755 directories, then the explicit modes, deepest first,
// so a locked directory is locked after its contents are written.
func materialize(root string, tree map[string]entry) error {
	paths := make([]string, 0, len(tree))
	for p := range tree {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		e, full := tree[p], filepath.Join(root, p)
		err := os.MkdirAll(filepath.Dir(full), 0o755)
		if err == nil {
			switch e.kind {
			case "dir":
				err = os.MkdirAll(full, 0o755)
			case "symlink":
				err = os.Symlink(e.body, full)
			default:
				err = os.WriteFile(full, []byte(e.body), 0o644)
			}
		}
		if err != nil {
			return err
		}
	}
	for i := len(paths) - 1; i >= 0; i-- {
		if e := tree[paths[i]]; e.kind != "symlink" {
			if err := os.Chmod(filepath.Join(root, paths[i]), e.mode); err != nil {
				return err
			}
		}
	}
	return nil
}

// snapshot is every entry under root by slash path, spelled "<kind> <mode> <content|target>", except the
// skip entries. It unlocks what it reads (u+r, and u+rx on a directory), so a snapshot dir stays removable.
func snapshot(root string, skip []string) (map[string]string, error) {
	skipped := map[string]bool{}
	for _, s := range skip {
		skipped[s] = true
	}
	out := map[string]string{}
	var walk func(rel string) error
	walk = func(rel string) error {
		d := filepath.Join(root, rel)
		if fi, err := os.Lstat(d); err != nil {
			return err
		} else if fi.Mode().Perm()&0o700 != 0o700 {
			if err := os.Chmod(d, fi.Mode().Perm()|0o700); err != nil {
				return err
			}
		}
		names, err := os.ReadDir(d)
		if err != nil {
			return err
		}
		for _, n := range names {
			p := filepath.ToSlash(filepath.Join(rel, n.Name()))
			if skipped[p] || skipped[p+"/"] {
				continue
			}
			full := filepath.Join(root, p)
			fi, err := os.Lstat(full)
			if err != nil {
				return err
			}
			mode := fi.Mode().Perm()
			switch {
			case fi.Mode()&os.ModeSymlink != 0:
				target, err := os.Readlink(full)
				if err != nil {
					return err
				}
				out[p] = "symlink -> " + target
			case fi.IsDir():
				out[p] = fmt.Sprintf("dir %04o", mode)
				if err := walk(p); err != nil {
					return err
				}
			default:
				if mode&0o400 == 0 {
					if err := os.Chmod(full, mode|0o400); err != nil {
						return err
					}
				}
				b, err := os.ReadFile(full)
				if err != nil {
					return err
				}
				out[p] = fmt.Sprintf("file %04o %q", mode, b)
			}
		}
		return nil
	}
	return out, walk("")
}

// treeProblems lists, in path order, every entry missing from, extra in or changed in the tree left.
func treeProblems(unsub *strings.Replacer, left, want map[string]string) []string {
	paths := []string{}
	for p := range left {
		paths = append(paths, p)
	}
	for p := range want {
		if _, ok := left[p]; !ok {
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)
	var out []string
	for _, p := range paths {
		got, inLeft := left[p]
		w, inWant := want[p]
		switch {
		case !inWant:
			out = append(out, fmt.Sprintf("extra %s after the run: %s", p, unsub.Replace(got)))
		case !inLeft:
			out = append(out, fmt.Sprintf("missing %s after the run: want %s", p, unsub.Replace(w)))
		case got != w:
			out = append(out, fmt.Sprintf("changed %s after the run: %s, want %s", p, unsub.Replace(got), unsub.Replace(w)))
		}
	}
	return out
}
