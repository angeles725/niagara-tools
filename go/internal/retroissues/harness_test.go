package retroissues

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// fakeGH is an injectable gh: it answers listings from labeled / searched JSON and records every call.
type fakeGH struct {
	labeled, searched string // JSON arrays of issues
	listErr           error
	createErr         error
	commentErr        error // gh issue comment
	editErr           error // gh issue edit (the label)
	next              int
	calls             []string
	stdins            []string
}

func (g *fakeGH) run(args []string, stdin string) (string, error) {
	line := strings.Join(args, " ")
	g.calls = append(g.calls, line)
	g.stdins = append(g.stdins, stdin)
	switch {
	case strings.HasPrefix(line, "issue list") && strings.Contains(line, "--search="):
		if g.listErr != nil {
			return "", g.listErr
		}
		return orEmpty(g.searched), nil
	case strings.HasPrefix(line, "issue list"):
		if g.listErr != nil {
			return "", g.listErr
		}
		return orEmpty(g.labeled), nil
	case strings.HasPrefix(line, "label list"):
		return `[{"name":"retro-delta"}]`, nil
	case strings.HasPrefix(line, "issue comment") && g.commentErr != nil:
		return "", g.commentErr
	case strings.HasPrefix(line, "issue edit") && g.editErr != nil:
		return "", g.editErr
	case strings.HasPrefix(line, "issue create"):
		if g.createErr != nil {
			return "", g.createErr
		}
		g.next++
		return fmt.Sprintf("https://github.com/o/r/issues/%d\n", 100+g.next), nil
	}
	return "", nil
}

func orEmpty(s string) string {
	if s == "" {
		return "[]"
	}
	return s
}

// called counts the recorded gh calls that start with prefix.
func (g *fakeGH) called(prefix string) int {
	n := 0
	for _, c := range g.calls {
		if strings.HasPrefix(c, prefix) {
			n++
		}
	}
	return n
}

// env is a scratch kit root, an outside directory and a denylist.
type env struct {
	t        *testing.T
	kit, out string
	denylist string
	gh       *fakeGH
}

func newEnv(t *testing.T) *env {
	t.Helper()
	root := t.TempDir()
	e := &env{t: t, kit: filepath.Join(root, "kit"), out: filepath.Join(root, "outside"), gh: &fakeGH{}}
	for _, d := range []string{filepath.Join(e.kit, "retros"), e.out} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	e.denylist = filepath.Join(root, "denylist.txt")
	e.write(e.denylist, "# test denylist\n\\bsecretword\\b\n")
	return e
}

func (e *env) write(path, text string) string {
	e.t.Helper()
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		e.t.Fatal(err)
	}
	return path
}

func (e *env) kitRetro(name, text string) string {
	return e.write(filepath.Join(e.kit, "retros", name), text)
}
func (e *env) outRetro(name, text string) string { return e.write(filepath.Join(e.out, name), text) }

func (e *env) read(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		e.t.Fatal(err)
	}
	return string(b)
}

// run executes retro-issues with the scratch kit root and denylist.
func (e *env) run(args ...string) (code int, out, errs string) {
	var o, er bytes.Buffer
	s := cli.Stdio{Out: &o, Err: &er, Getenv: func(k string) string {
		switch k {
		case "NIAGARA_CLIENT_DENYLIST":
			return e.denylist
		case "RETRO_ISSUES_KIT_ROOT":
			return e.kit
		}
		return ""
	}}
	code = Command(e.gh.run)(args, s)
	return code, o.String(), er.String()
}

var errGH = errors.New("gh: not logged in")

const alphaRetro = `<!-- review-status: pending -->
# 2026-02-01 · kit · alpha

## Proposed kit deltas (propose-never-apply)
| Δ | Delta | Target file / § | Token |
|---|---|---|---|
| Δ1 | APPLIED 2026-02-01. already shipped | ` + "`go/internal/x`" + ` | ` + "`[ev: retro alpha Δ1]`" + ` |
| Δ2 | **n4_read_slots drops a key.** The filter loses the key when the slot name is long | ` + "`go/internal/mcpn4/slots.go`" + ` | ` + "`[ev: retro alpha Δ2]`" + ` |
| Δ3 | A lint for the module descriptor | ` + "`toolbelt/lint-descriptor.sh`" + ` | ` + "`[ev: retro alpha Δ3]`" + ` |
| Δ4 | Something with no area at all | nowhere | none |
`
