package golden

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

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
