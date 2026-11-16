package reportmodule

import (
	"fmt"
	"strings"
)

// noBaseline is the floor SKIP of a missing baseline: the exact command that extracts <artifact> as it was at
// the deployed commit (`git archive <ref> .` run in the artifact directory writes paths relative to it, so the
// baseline has the artifact's own layout, module-include.xml and src/); %[1]s is the artifact directory as a
// shell word. The archive goes to a temporary tar first and .deploy-baseline is made only after it succeeded, so
// a wrong ref, or <deployed-ref> pasted as printed (the shell reads it as a redirection and fails), leaves no
// empty baseline that would read as "no slots". [ev: retro buildsh-mirror-baseline-b6 Δ2]
// (retro row B6, a client A module build 2026-11-16)
const noBaseline = "no .deploy-baseline (floor check not run); create it from the deployed commit" +
	" (put its tag or commit in place of <deployed-ref>): " +
	`T=$(mktemp -d) && git -C %[1]s archive -o "$T/b.tar" <deployed-ref> . && mkdir %[1]s/.deploy-baseline && ` +
	`tar -x -f "$T/b.tar" -C %[1]s/.deploy-baseline; rm -rf "$T"` +
	" — or --first-deploy for a never-deployed module (build-verify.md § Schema-risk deploy workflow)"

// ShellWord is s as one POSIX shell word: as is when every byte is safe unquoted, else single-quoted. build.sh
// quotes the commands it prints with it too.
func ShellWord(s string) string {
	if s != "" && strings.Trim(s, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_./@%+=:,-") == "" {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// schemaRisk is §6: schema-risk <artifact>/.deploy-baseline <artifact>, its output discarded and its exit the
// verdict; no baseline is a floor SKIP unless --first-deploy.
func (r *report) schemaRisk(a artifact) {
	v := schemaRiskVerdict(r.x, a.dir, r.o.firstDeploy)
	switch {
	case v.status == errored:
		r.envFault(a.name, schemaRiskCheck, v.detail)
	case v.missingBaseline:
		r.emit(a.name, v.status, schemaRiskCheck, v.detail)
		r.floorSkip = true
	default:
		r.emit(a.name, v.status, schemaRiskCheck, v.detail)
	}
}

const schemaRiskCheck = "schema-risk"

// schemaVerdict is the schema-risk row of an artifact. missingBaseline is set only when there is no
// .deploy-baseline and no --first-deploy (a SKIP the lint run counts as a floor SKIP and --ready FAILs); the N/A
// SKIP of --first-deploy leaves it unset, so callers never tell the two apart by status and flag.
type schemaVerdict struct {
	status, detail  string
	missingBaseline bool
}

// schemaRiskVerdict is the schema-risk row of an artifact directory: no .deploy-baseline is SKIP (its detail the
// N/A of --first-deploy, else the command that creates the baseline and missingBaseline set), else the verdict of
// schema-risk's exit; an env fault is ERROR. report-module --ready reads it too.
func schemaRiskVerdict(x runner, dir string, firstDeploy bool) schemaVerdict {
	baseline := dir + "/.deploy-baseline"
	if !isDir(baseline) {
		if firstDeploy {
			return schemaVerdict{status: skip, detail: "N/A (--first-deploy: never deployed, no baseline to diff)"}
		}
		return schemaVerdict{status: skip, detail: fmt.Sprintf(noBaseline, ShellWord(dir)), missingBaseline: true}
	}
	switch code := x.quiet(x.tb+"/schema-risk.sh", baseline, dir); code {
	case 0:
		return schemaVerdict{status: pass, detail: "verdict=SAFE"}
	case 1:
		return schemaVerdict{status: warn, detail: "verdict=LOSSY"}
	case 2:
		return schemaVerdict{status: failed, detail: "verdict=OUTAGE"} // D9a: OUTAGE is a finding, never an env fault
	case 3, 4:
		return schemaVerdict{status: errored, detail: fmt.Sprintf("env fault (exit %d)", code)}
	default:
		return schemaVerdict{status: errored, detail: fmt.Sprintf("unexpected exit %d", code)}
	}
}

// writePath is §11 (module root): its no-matrix exit 3 is a SKIP, any other exit 3 an ERROR naming the first
// output line; its FAIL rows are relayed with the STALE/DRIFT advisories as WARN, then PASS when no FAIL* row
// (a non-zero exit with none is a FAIL row).
func (r *report) writePath(a artifact) {
	const check = "lint-write-path"
	out, code, ok := r.x.member(check, a.dir)
	if r.truncated(a.name, check, check+".sh", ok) {
		return
	}
	switch {
	case code == 3 && strings.Contains(out, "no write-path-matrix.md found"):
		r.emit(a.name, skip, check, "no write-path-matrix.md")
	case code == 3:
		r.envFault(a.name, check, "env fault (exit 3): "+cutBefore(out, "\n"))
	default:
		switch rl := r.memberRows(a.name, out, filter{sel: "FAIL", asw: "STALE|DRIFT"}); {
		case rl.failP:
		case code != 0:
			r.exitedNoFail(a.name, check, code)
		default:
			r.emit(a.name, pass, check, "clean")
		}
	}
}

// triageConsole is the once-per-run triage-console --console-dir <dir> (a SKIP row without the flag): its
// non-empty output lines are printed as they are and counted by their leading status.
func (r *report) triageConsole() {
	if r.o.consoleDir == "" {
		fmt.Fprint(r.out, "(run)  SKIP  triage-console  no --console-dir\n")
		r.count(skip)
		return
	}
	out, code, ok := r.x.member("triage-console", "--console-dir", r.o.consoleDir)
	if r.truncated("(run)", "triage-console", "triage-console.sh", ok) {
		return
	}
	if code == 3 {
		fmt.Fprint(r.out, "(run)  ERROR  triage-console  env fault (exit 3)\n")
		r.hadEnv = true
		return
	}
	printed := false
	for _, ln := range strings.Split(out, "\n") {
		if ln == "" {
			continue
		}
		fmt.Fprintln(r.out, ln)
		printed = true
		for _, st := range []string{failed, warn, pass, skip} {
			if strings.HasPrefix(ln, st) {
				r.count(st)
				break
			}
		}
	}
	if !printed {
		fmt.Fprint(r.out, "(run)  PASS  triage-console  0 rows\n")
		r.count(pass)
	}
}
