package buildmodule

import (
	"os"
	"strings"
)

// extractBaselines is --schema-baseline-ref <ref>: what the schema-risk SKIP row tells the operator to run, done
// for the run. Each artifact report-module walks is extracted as it is at <ref> (`git archive <ref> .` in the
// artifact directory) into a temporary directory, and <artifact>/.deploy-baseline is a link to it; cleanup
// removes both, whatever the exit. A link left by a killed run dangles, so report-module reads no baseline from
// it (a SKIP, never a stale baseline). It runs before gradle, so a wrong ref costs no build.
// [ev: retro mcpn4-history-batch-tools Δ4]
func (b *build) extractBaselines() int {
	ref := b.opt.baselineRef
	if ref == "" {
		return proceed
	}
	source := b.root + "/" + b.opt.mod
	if code := b.gitQuiet(source, "rev-parse", "--verify", "--quiet", ref+"^{commit}"); code != 0 {
		return b.refRefusal(code, msgBaselineRefNotCommit, ref, source)
	}
	if code := b.gitQuiet(source, "rev-parse", "--verify", "--quiet", ref+":./"); code != 0 {
		return b.refRefusal(code, msgBaselineRefNoModule, ref, source)
	}
	artifacts := walkedArtifacts(source)
	for _, a := range artifacts {
		if b.baselineTaken(a + "/.deploy-baseline") {
			b.warn(msgBaselineRefExists, a+"/.deploy-baseline")
			return ExitUsage
		}
	}
	for _, a := range artifacts {
		name := a[strings.LastIndex(a, "/")+1:]
		if b.gitQuiet(a, "rev-parse", "--verify", "--quiet", ref+":./") != 0 {
			b.warn(msgBaselineRefNewArtifact, name, ref)
			continue
		}
		if code := b.extractBaseline(a, ref); code != proceed {
			return code
		}
		b.say(msgBaselineRefDone, name, ref)
	}
	return proceed
}

// extractBaseline archives artifact a at ref into a temporary directory and links <a>/.deploy-baseline to it.
func (b *build) extractBaseline(a, ref string) int {
	tmp, code := b.makeTemp(true)
	if code != 0 {
		return ExitFault
	}
	tree := tmp + "/baseline"
	if _, code = b.run("git", "-C", a, "archive", "-o", tmp+"/b.tar", ref, "."); code == 0 {
		if _, code = b.run("mkdir", tree); code == 0 {
			_, code = b.run("tar", "-x", "-f", tmp+"/b.tar", "-C", tree)
		}
	}
	if code != 0 {
		b.warn(msgBaselineRefFailed, ref, a, code)
		return ExitFault
	}
	link := a + "/.deploy-baseline"
	// Made and registered under tempsMu, which the signal handler's cleanup takes: an interrupt cannot fall
	// between the two, and a path this run did not make (Symlink failed) is never registered for removal.
	b.tempsMu.Lock()
	defer b.tempsMu.Unlock()
	if err := os.Symlink(tree, link); err != nil {
		b.warn(msgBaselineRefLink, link, err)
		return ExitFault
	}
	b.temps = append(b.temps, link)
	return proceed
}

// gitQuiet runs git in dir with its output dropped and returns its status as bash reports it.
func (b *build) gitQuiet(dir string, args ...string) int {
	cmd := b.command("git", append([]string{"-C", dir}, args...)...)
	return b.status("git", cmd.Run())
}

// refRefusal maps a failed git check: git that cannot be started is an environment error (exit 10), any other
// failure is the operator's ref or module (a usage error, exit 2, after the message).
func (b *build) refRefusal(code int, format string, a ...any) int {
	if code == notFound || code == notExecutable {
		return ExitEnv
	}
	b.warn(format, a...)
	return ExitUsage
}

// walkedArtifacts are the artifact directories report-module walks under source: directories that are not
// symlinks and not dot-directories (baselineMissing reads the same set).
func walkedArtifacts(source string) []string {
	entries, _ := os.ReadDir(source)
	var dirs []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			dirs = append(dirs, source+"/"+e.Name())
		}
	}
	return dirs
}

// baselineTaken reports whether link is a .deploy-baseline the run must not replace. A dangling symlink is the
// leftover of a killed run (its extraction is gone, and report-module reads it as no baseline): it is removed,
// with one INFO line, so a retry does not need a manual cleanup.
func (b *build) baselineTaken(link string) bool {
	st, err := os.Lstat(link)
	if err != nil {
		return false
	}
	if st.Mode()&os.ModeSymlink != 0 {
		if _, err := os.Stat(link); err != nil && os.Remove(link) == nil {
			b.warn(msgBaselineRefDangling, link)
			return false
		}
	}
	return true
}
