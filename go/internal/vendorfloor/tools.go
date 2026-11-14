package vendorfloor

import (
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// Environment the tool resolution reads. node itself also reads NODE_PATH (inherited) when the acorn
// spec is the bare package name.
const (
	envNode     = "KIT_NODE"                   // the node binary; unset means `node` on PATH
	envAcorn    = "KIT_ACORN"                  // the acorn package directory (or any require() spec)
	envToolbelt = "LINT_VENDOR_FLOOR_TOOLBELT" // the toolbelt directory, exported by lint-vendor-floor.sh
)

// bundledAcorn is where `npm install --prefix toolbelt/eslint` puts acorn, under the toolbelt directory.
const bundledAcorn = "/eslint/node_modules/acorn"

// nodeBinary resolves node: KIT_NODE, else `node` on PATH. ok is false when nothing executable is there.
func nodeBinary(s cli.Stdio) (bin string, ok bool) {
	bin = s.Env(envNode)
	if bin == "" {
		bin = lookPath("node", s.Env("PATH"))
	}
	return bin, bin != "" && executable(bin)
}

// acornSpec is what the driver passes to require(): KIT_ACORN, else the toolbelt's own install when it
// exists, else the bare package name (node's own resolution, NODE_PATH included).
func acornSpec(s cli.Stdio) string {
	if spec := s.Env(envAcorn); spec != "" {
		return spec
	}
	if toolbelt := s.Env(envToolbelt); toolbelt != "" {
		if st, err := os.Stat(toolbelt + bundledAcorn); err == nil && st.IsDir() {
			return toolbelt + bundledAcorn
		}
	}
	return "acorn"
}

// lookPath is `command -v <name>` over the PATH value given: the first executable file, "" when none.
func lookPath(name, pathEnv string) string {
	for _, dir := range strings.Split(pathEnv, ":") {
		if dir == "" {
			dir = "."
		}
		if p := dir + "/" + name; executable(p) {
			return p
		}
	}
	return ""
}

// executable reports whether p is a file with an execute bit.
func executable(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir() && st.Mode()&0o111 != 0
}
