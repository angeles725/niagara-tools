package reportmodule

import (
	"fmt"
	"strings"
)

// noBaseline is the floor SKIP of a missing baseline: the exact command that extracts <artifact> as it was at
// the deployed commit (`git archive <ref> .` run in the artifact directory writes paths relative to it, so the
// baseline has the artifact's own layout, module-include.xml and src/); %[1]s is the artifact directory as a
// shell word. The operator fills in <deployed-ref>, the tag or commit the station runs. [ev: retro buildsh-mirror-baseline-b6 Δ2]
// (retro row B6, a client A module build 2026-11-16)
const noBaseline = "no .deploy-baseline (floor check not run); create it from the deployed commit: " +
	"mkdir -p %[1]s/.deploy-baseline && git -C %[1]s archive <deployed-ref> . | tar -x -C %[1]s/.deploy-baseline" +
	" — or --first-deploy for a never-deployed module (build-verify.md § Schema-risk deploy workflow)"

// shellWord is s as one POSIX shell word: as is when every byte is safe unquoted, else single-quoted.
func shellWord(s string) string {
	if s != "" && strings.Trim(s, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_./@%+=:,-") == "" {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// schemaRisk is §6: schema-risk <artifact>/.deploy-baseline <artifact>, its output discarded and its exit the
// verdict; no baseline is a floor SKIP unless --first-deploy.
func (r *report) schemaRisk(a artifact) {
	const check = "schema-risk"
	baseline := a.dir + "/.deploy-baseline"
	if !isDir(baseline) {
		if r.o.firstDeploy {
			r.emit(a.name, skip, check, "N/A (--first-deploy: never deployed, no baseline to diff)")
		} else {
			r.emit(a.name, skip, check, fmt.Sprintf(noBaseline, shellWord(a.dir)))
			r.floorSkip = true
		}
		return
	}
	switch code := r.x.quiet(r.x.tb+"/schema-risk.sh", baseline, a.dir); code {
	case 0:
		r.emit(a.name, pass, check, "verdict=SAFE")
	case 1:
		r.emit(a.name, warn, check, "verdict=LOSSY")
	case 2:
		r.emit(a.name, failed, check, "verdict=OUTAGE") // D9a: OUTAGE is a finding, never an env fault
	case 3, 4:
		r.envFault(a.name, check, fmt.Sprintf("env fault (exit %d)", code))
	default:
		r.envFault(a.name, check, fmt.Sprintf("unexpected exit %d", code))
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
