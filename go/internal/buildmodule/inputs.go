package buildmodule

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/findwalk"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/modjar"
)

// The listing of gradle roots below a repo root looks for a gradlew file 2 or 3 levels down.
const belowMinDepth, belowMaxDepth = 2, 3

// resolveInputs checks what the chain needs before anything runs: the module root and its gradle root,
// JDK 8, a niagara_home that is a mirror (or the opt-in), and the gate next to build.sh.
func (b *build) resolveInputs() int {
	o := b.opt
	if !lint.IsDir(o.root) {
		b.warn(msgRootNotFound, o.root)
		return ExitEnv
	}
	root, ok := absolute(o.root)
	if !ok {
		b.warn(msgRootUnresolved, o.root)
		return ExitEnv
	}
	b.root = root
	if code := b.findGradleRoot(); code != proceed {
		return code
	}
	b.java8 = b.s.Env("JAVA8")
	if b.java8 == "" {
		b.java8 = "/usr/lib/jvm/java-8-openjdk-amd64"
	}
	b.home = o.home
	gate := b.here + "/" + gateScript
	switch {
	case !lint.IsDir(b.java8):
		b.warn(msgNoJava8, b.java8)
	case b.home == "":
		b.warn(msgNoHome)
	case !lint.IsDir(b.home + "/" + m2Repository):
		b.warn(msgNotAHome, b.home)
	case !b.isMirror() && !o.intoHome:
		// Fail closed: gradle's :jar step replaces <niagara_home>/modules/<jar>, so only a mirror is
		// written by default. [ev: retro r10-buildsh-mirror-guard]
		b.warn(msgRealHome, b.home)
	case !isExecutable(gate):
		b.warn(msgNoGate, gate)
	default:
		// Messages above name niagara_home as given; gradle runs in the gradle root, where a relative
		// path names another directory, so it gets the absolute one (bench Δ9).
		if abs, err := filepath.Abs(b.home); err == nil {
			b.home = abs
		}
		return proceed
	}
	return ExitEnv
}

// isMirror reports a niagara_home made by mirror-niagara-home.sh.
func (b *build) isMirror() bool { return modjar.IsMirror(b.home) }

// absolute is the directory as `cd -- <dir> && pwd` spells it: joined to the working directory and
// cleaned, symlinks kept; ok is false when it cannot be entered.
func absolute(dir string) (abs string, ok bool) {
	if !filepath.IsAbs(dir) {
		wd, err := os.Getwd() // $PWD when it names the working directory, as bash keeps it
		if err != nil {
			return "", false
		}
		dir = wd + "/" + dir
	}
	abs = filepath.Clean(dir)
	return abs, isExecutable(abs)
}

// findGradleRoot walks up from the module root to the directory holding an executable ./gradlew (a client
// multi-project layout passes the module dir). A gradlew that exists but is not executable fails fast
// instead of being walked past (a worktree checkout of a 100644 gradlew).
func (b *build) findGradleRoot() int {
	g := b.root
	for g != "/" && !isExecutable(g+"/gradlew") {
		if lint.IsFile(g + "/gradlew") {
			b.warn(msgGradlewNotExec, g)
			return ExitEnv
		}
		g = filepath.Dir(g)
	}
	if !isExecutable(g + "/gradlew") {
		b.warn(msgNoGradlew, b.root)
		// The opposite mistake: the REPO root was passed, one or more levels ABOVE the gradle root
		// (a client repo nests several gradle groups as siblings). List them.
		// [ev: client A retro "phase 2 amps alarms" Δ2]
		if below := gradleRootsBelow(b.root); len(below) > 0 {
			b.warn(msgRootsBelow, b.root)
			for _, dir := range below {
				b.warn(msgRootBelow, dir)
			}
		}
		b.warn(msgTryChmod, b.root)
		return ExitEnv
	}
	b.gradleRoot = g
	return proceed
}

// gradleRootsBelow lists the directories 1 or 2 levels below root that hold a regular gradlew file, in
// bytewise order of the gradlew path. The root is followed when it is a symlink (find -H, issue #329); dot
// directories, node_modules and build are not entered from the second level down.
func gradleRootsBelow(root string) []string {
	var wrappers []string
	findwalk.Walk(root, true, belowMaxDepth, func(e findwalk.Entry) bool {
		switch {
		case e.Depth < belowMinDepth:
		case e.Dir:
			return !strings.HasPrefix(e.Name, ".") && !prunedBelow[e.Name]
		case e.Name == "gradlew" && isRegular(e.Path):
			wrappers = append(wrappers, e.Path)
		}
		return true
	}, nil)
	sort.Strings(wrappers)
	for i, w := range wrappers {
		wrappers[i] = filepath.Dir(w)
	}
	return wrappers
}
