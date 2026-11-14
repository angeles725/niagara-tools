package vendorfloor

import (
	"os"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// env is a cli.Stdio whose environment is vars alone.
func env(vars map[string]string) cli.Stdio {
	return cli.Stdio{Getenv: func(name string) string { return vars[name] }}
}

func TestNodeBinary(t *testing.T) {
	t.Parallel()
	d := t.TempDir()
	node := write(t, d, "bin/node", "#!/bin/sh\n")
	if err := os.Chmod(node, 0o755); err != nil {
		t.Fatal(err)
	}
	plain := write(t, d, "plain/node", "")
	cases := []struct {
		name string
		vars map[string]string
		bin  string
		ok   bool
	}{
		{"KIT_NODE wins over PATH", map[string]string{"KIT_NODE": node, "PATH": "/nonexistent"}, node, true},
		{"KIT_NODE that does not exist", map[string]string{"KIT_NODE": "/nonexistent/node", "PATH": d + "/bin"}, "/nonexistent/node", false},
		{"KIT_NODE without an execute bit", map[string]string{"KIT_NODE": plain}, plain, false},
		{"KIT_NODE that is a directory", map[string]string{"KIT_NODE": d}, d, false},
		{"node on PATH, after a directory without it", map[string]string{"PATH": d + "/plain:" + d + "/bin"}, d + "/bin/node", true},
		{"no node on PATH", map[string]string{"PATH": d}, "", false},
	}
	for _, c := range cases {
		if bin, ok := nodeBinary(env(c.vars)); bin != c.bin || ok != c.ok {
			t.Errorf("%s: nodeBinary = %q, %v; want %q, %v", c.name, bin, ok, c.bin, c.ok)
		}
	}
}

func TestAcornSpec(t *testing.T) {
	t.Parallel()
	d := t.TempDir()
	write(t, d, "with/eslint/node_modules/acorn/package.json", "{}")
	bundled := d + "/with/eslint/node_modules/acorn"
	cases := []struct {
		name string
		vars map[string]string
		want string
	}{
		{"KIT_ACORN wins", map[string]string{"KIT_ACORN": "/opt/acorn", "LINT_VENDOR_FLOOR_TOOLBELT": d + "/with"}, "/opt/acorn"},
		{"the toolbelt install", map[string]string{"LINT_VENDOR_FLOOR_TOOLBELT": d + "/with"}, bundled},
		{"a toolbelt without the install", map[string]string{"LINT_VENDOR_FLOOR_TOOLBELT": d}, "acorn"},
		{"nothing set: node's own resolution", nil, "acorn"},
	}
	for _, c := range cases {
		if got := acornSpec(env(c.vars)); got != c.want {
			t.Errorf("%s: acornSpec = %q, want %q", c.name, got, c.want)
		}
	}
}
