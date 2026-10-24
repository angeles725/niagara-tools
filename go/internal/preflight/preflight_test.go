package preflight

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// fakeHost records every command it is asked to run and answers by the command's path suffix.
type fakeHost struct {
	calls []string
	out   map[string]string
}

func (f *fakeHost) host() Host {
	return Host{Output: func(name string, args []string, combined bool) []byte {
		f.calls = append(f.calls, strings.TrimSpace(filepath.Base(name)+" "+strings.Join(args, " ")))
		for suffix, o := range f.out {
			if strings.HasSuffix(name, suffix) {
				return []byte(o)
			}
		}
		return nil
	}}
}

// pf runs preflight over h with only env visible and returns exit code, stdout and stderr.
func pf(h Host, env map[string]string, args ...string) (int, string, string) {
	var o, e bytes.Buffer
	code := Command(h)(args, cli.Stdio{Out: &o, Err: &e, Getenv: func(k string) string { return env[k] }})
	return code, o.String(), e.String()
}

// mk creates files under root: "p/" a directory, "p" an empty file, "p=text" a file holding text.
func mk(t *testing.T, root string, specs ...string) {
	t.Helper()
	for _, s := range specs {
		name, text, _ := strings.Cut(s, "=")
		p := filepath.Join(root, name)
		err := os.MkdirAll(filepath.Dir(p), 0o755)
		if err == nil && strings.HasSuffix(name, "/") {
			err = os.MkdirAll(p, 0o755)
		} else if err == nil {
			err = os.WriteFile(p, []byte(text), 0o644)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestUsageExits2(t *testing.T) {
	for _, args := range [][]string{nil, {"a"}, {"a", "b", "c"}, {"-x", "a", "b"}, {"-", "b"}, {"--jvm-dir"}, {"--plugin-version"}, {"a", "b", "--sample"}} {
		if code, out, err := pf(Host{}, nil, args...); code != ExitUsage || out != "" || err != usage {
			t.Errorf("%q: exit %d out %q err %q, want 2 and the usage line", args, code, out, err)
		}
	}
}

func TestMissingPathExits3(t *testing.T) {
	d := t.TempDir()
	for _, c := range []struct{ nh, gr, want string }{
		{d + "/no", d, "preflight: niagara_home not found: " + d + "/no\n"},
		{d, d + "/no", "preflight: gradle-root not found: " + d + "/no\n"},
	} {
		if code, out, err := pf(Host{}, nil, c.nh, c.gr); code != ExitEnv || out != "" || err != c.want {
			t.Errorf("exit %d out %q err %q, want 3 and %q", code, out, err, c.want)
		}
	}
}

func TestWindowsPathFailsBeforeThePathChecks(t *testing.T) {
	code, out, _ := pf((&fakeHost{}).host(), nil, "--jvm-dir", t.TempDir(), "--", `C:\n4`, `mod\root`)
	want := "FAIL  win-path      niagara_home is a Windows path — use /mnt/c/... in WSL instead\n" +
		"FAIL  win-path      gradle-root is a Windows path — use /mnt/c/... in WSL instead\n"
	if code != ExitFail || !strings.HasPrefix(out, want) {
		t.Errorf("exit %d out %q, want 1 and the two win-path rows first", code, out)
	}
}
