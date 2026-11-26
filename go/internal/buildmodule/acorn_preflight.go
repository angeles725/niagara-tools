package buildmodule

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
)

// The acorn preflight (#1116): report-module's vendor-floor check parses every vendored browser library under
// <mod>-ux/src/rc/vendor with node + acorn and, when either is missing, ends the build with a SKIP row after
// gradle, the gate and the report ran. A module that vendors a library therefore asks for both here, before
// gradle, and a build that cannot have them is exit 10 naming the one command that fixes it. Nothing is
// installed: the kit never runs npm on its own. The resolution is the one lint-vendor-floor uses
// (go/internal/vendorfloor/tools.go): KIT_NODE else `node` on PATH; KIT_ACORN else the toolbelt's own
// `npm install --prefix toolbelt/eslint` else the bare name, which node resolves itself (NODE_PATH included).

const (
	envNode  = "KIT_NODE"
	envAcorn = "KIT_ACORN"

	// bundledAcornDir is where the toolbelt's own npm install puts acorn, under the toolbelt directory.
	bundledAcornDir = "/eslint/node_modules/acorn"

	msgNoNode = "build.sh: FAIL — node not found, and %s holds vendored JS that report-module's vendor-floor " +
		"check parses with it: install Node, or set KIT_NODE=<node binary> (--no-preflight skips this check)"
	msgNoAcorn = "build.sh: FAIL — acorn is not resolvable from '%s', and %s holds vendored JS that report-module's " +
		"vendor-floor check parses with it: run `npm install --prefix %s/eslint`, or set KIT_ACORN=<acorn package dir> " +
		"(--no-preflight skips this check)"
)

// acornPreflight ends the chain with exit 10 when the module vendors JS and node or acorn cannot be resolved;
// --no-preflight skips it with the rest of the environment preflight.
func (b *build) acornPreflight() int {
	if b.opt.skipPreflight {
		return proceed
	}
	vendor := b.vendoredJS()
	if vendor == "" {
		return proceed
	}
	node := b.s.Env(envNode)
	if node == "" {
		node, _ = exec.LookPath("node")
	}
	if st, err := os.Stat(node); node == "" || err != nil || st.IsDir() || st.Mode()&0o111 == 0 {
		b.warn(msgNoNode, vendor)
		return ExitEnv
	}
	spec := b.acornSpec()
	cmd := b.command(node, "-e", "require(process.argv[1])", spec)
	if cmd.Run() != nil {
		b.warn(msgNoAcorn, spec, vendor, b.here)
		return ExitEnv
	}
	return proceed
}

// acornSpec is what node is asked to require(): KIT_ACORN, else the toolbelt's own install when it exists, else
// the bare package name.
func (b *build) acornSpec() string {
	if spec := b.s.Env(envAcorn); spec != "" {
		return spec
	}
	if st, err := os.Stat(b.here + bundledAcornDir); err == nil && st.IsDir() {
		return b.here + bundledAcornDir
	}
	return "acorn"
}

// vendoredJS is the first <mod>-ux/src/rc/vendor directory of the module that holds a .js, .mjs or .cjs file
// (vendor-floor's own list of vendored libraries), "" when none does.
func (b *build) vendoredJS() string {
	dirs, _ := filepath.Glob(b.root + "/" + b.opt.mod + "/*-ux/src/rc/vendor")
	slices.Sort(dirs)
	for _, dir := range dirs {
		if holdsScript(dir) {
			return dir
		}
	}
	return ""
}

// holdsScript says whether dir (a symlink to a directory is followed; one below it is not) holds a script.
func holdsScript(dir string) bool {
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.IsDir() && holdsScript(dir+"/"+e.Name()) {
			return true
		}
		if e.Type().IsRegular() {
			switch filepath.Ext(e.Name()) {
			case ".js", ".mjs", ".cjs":
				return true
			}
		}
	}
	return false
}
