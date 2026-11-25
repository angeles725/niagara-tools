package puretest

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// A pure test never compiles the Baja adapters beside the pure core (src/), so a helper a refactor deleted
// passes the tests and fails only at gradle's compile. With NIAGARA_HOME naming a mirror (or an install),
// the sources of src/ are also compiled, against the jars of <home>/modules, into a directory of their own
// that goes away with the run. Without it one NOTICE says they were not. [ev: retro client-e-dashboard-live-check Δ30]

const (
	adaptersDir    = "adapters"   // under the run's temporary directory: where the adapter classes go
	modulesJars    = "modules/*"  // javac expands a trailing /* class path entry to every jar of the directory
	adapterNoProc  = "-proc:none" // no annotation processing: Slot-o-Matic is gradle's job, not this loop's
	msgNoAdapters  = "NOTICE the Baja adapters under src/ were not compiled (NIAGARA_HOME is not set): a helper deleted by a refactor passes these tests and fails only at gradle — export NIAGARA_HOME=<mirror> to compile them here"
	msgNoHomeJars  = "NIAGARA_HOME %s has no modules/*.jar — point it at a mirror (mirror-niagara-home.sh) or an install"
	msgAdaptersBad = "adapter sources do not compile: %s (javac output above) — a compile error is not a RED"
	msgAdaptersErr = "adapter sources do not compile (javac failed above) — a compile error is not a RED"
)

// diagnosedFile is the file a javac diagnostic line starts with.
var diagnosedFile = regexp.MustCompile(`^(.+\.java):[0-9]+: `)

// checkHome is exit 3 for a NIAGARA_HOME without a jar in modules/; no home is fine (the adapters are skipped).
func (r *run) checkHome(home string) int {
	if home == "" {
		return ExitOK
	}
	if jars, _ := filepath.Glob(home + "/modules/*.jar"); len(jars) == 0 {
		return r.die(ExitEnv, msgNoHomeJars, home)
	}
	return ExitOK
}

// compileAdapters compiles every .java under <rt>/src into <out>/adapters, or says it did not. A module with
// no source there has nothing to compile and nothing to say.
func (r *run) compileAdapters(rt, out, home string) int {
	sources := javaSources(rt + "/" + mainSources)
	if len(sources) == 0 {
		return ExitOK
	}
	if home == "" {
		r.die(ExitOK, msgNoAdapters)
		return ExitOK
	}
	dest := out + "/" + adaptersDir
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return r.die(exitNoChdir, msgNoTemp, err)
	}
	var diagnostics bytes.Buffer
	tee := &run{cli.Stdio{Out: r.s.Out, Err: io.MultiWriter(r.s.Err, &diagnostics), Getenv: r.s.Getenv}}
	args := append(append([]string{}, compileFlags...), adapterNoProc, "-cp", home+"/"+modulesJars, "-d", dest)
	if tee.exec("", javac, append(args, sources...)...) == ExitOK {
		return ExitOK
	}
	if files := diagnosedFiles(diagnostics.String()); len(files) > 0 {
		return r.die(ExitEnv, msgAdaptersBad, strings.Join(files, ", "))
	}
	return r.die(ExitEnv, msgAdaptersErr)
}

// javaSources are the .java files under dir, in name order.
func javaSources(dir string) []string {
	var found []string
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(path, ".java") {
			found = append(found, path)
		}
		return nil
	})
	return found
}

// diagnosedFiles are the distinct files javac's diagnostics name, in order of first mention.
func diagnosedFiles(output string) []string {
	var files []string
	seen := map[string]bool{}
	for _, line := range strings.Split(output, "\n") {
		if m := diagnosedFile.FindStringSubmatch(line); m != nil && !seen[m[1]] {
			seen[m[1]] = true
			files = append(files, m[1])
		}
	}
	return files
}
