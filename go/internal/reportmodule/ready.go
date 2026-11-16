package reportmodule

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// report-module --ready (retro row P4, a client A module 2026-11-16): one "ready to load" report per built jar
// of the module root, with everything checked by hand before an operator loads a module on a production
// station: the module and its version, the target Niagara (the baja floor of META-INF/module.xml and the
// Niagara of the home it was compiled against), the signer, the verify-module gate, schema-risk against the
// deployed baseline, the jar's sha256 and whether the mirror holds the deployed jar again. No lint member runs
// and the FAIL baseline (.report-baseline) is not read: this is a load decision, not a punch-list.

// readyUsage is the --ready paragraph of the usage text.
const readyUsage = "  --ready [--niagara-home <dir>] [--expect-niagara <ver>] [--expect-signer <alias>] [--json]\n" +
	"    the ready-to-load report of each built jar (<artifact>/build/libs): module, niagara-target, niagara-home,\n" +
	"    signer, verify-module, schema-risk, sha256 and mirror rows, then \"READY TO LOAD: <MOD> <version> for\n" +
	"    Niagara <x> (sha256 <12>)\" and the post-load read-back, or \"NOT READY\" with the failing rows; exit 0 ready,\n" +
	"    1 not ready, 3 env (an ERROR row or no built jar). --target-version and --first-deploy apply; it refuses\n" +
	"    every lint-run option (--lint-only, --write-baseline, --no-baseline, --console-dir, --profile, --legacy,\n" +
	"    --wiring-map), and --json/--expect-* need --ready.\n"

// readyMain runs the ready report over the jars of o.root (a directory) and returns its exit code.
func readyMain(o options, s cli.Stdio, environ []string) int {
	artifacts, _ := discover(o.root)
	tb := toolbelt(s)
	if !isDir(tb) {
		fmt.Fprintf(s.Err, "report-module: toolbelt not found: %s\n", tb)
		return ExitEnv
	}
	x := runner{tb: tb, env: memberEnv(environ)}
	var jars []readyJar
	for _, a := range artifacts {
		if jar := firstJar(a.dir + "/build/libs"); jar != "" {
			jars = append(jars, assess(o, x, a, jar))
		}
	}
	if len(jars) == 0 {
		fmt.Fprintf(s.Err, "report-module: --ready: no built jar under %s/*/build/libs\n", o.root)
		return ExitEnv
	}
	if o.jsonOut {
		if err := writeJSON(s.Out, o.root, jars); err != nil {
			fmt.Fprintf(s.Err, "report-module: --json: %v\n", err)
			return ExitEnv
		}
	} else {
		writeText(s.Out, jars)
	}
	code := ExitClean
	for _, j := range jars {
		for _, r := range j.Rows {
			switch {
			case r.Status == errored:
				return ExitEnv
			case r.Status == failed:
				code = ExitFail
			}
		}
	}
	return code
}

// assess is the ready rows of one artifact's jar, in report order, and its verdict.
func assess(o options, x runner, a artifact, jar string) readyJar {
	info := readJar(jar)
	j := readyJar{Artifact: a.name, Jar: jar, Module: info.name, Version: info.version, SHA256: info.sum}
	add := func(st, check, detail string) { j.Rows = append(j.Rows, readyRow{st, check, detail}) }
	if info.why != "" {
		add(failed, "module", info.why)
	} else {
		add(pass, "module", info.name+" "+info.version)
	}
	add(niagaraTarget(info.bajaFloor, o.expectNiagara))
	home := homeNiagara(o.niagaraHome)
	add(niagaraHome(o.niagaraHome, home, o.expectNiagara))
	add(signerRow(jar, o.expectSigner))
	add(verifyGate(x, o, jar))
	st, detail := schemaRiskVerdict(x, a.dir, o.firstDeploy)
	if st == skip && !o.firstDeploy {
		st = failed // no deployed baseline: the load would not be checked against what the station runs
	}
	add(st, schemaRiskCheck, detail)
	if info.sum == "" {
		add(errored, "sha256", "the jar cannot be read")
	} else {
		add(pass, "sha256", info.sum)
	}
	add(mirrorRow(x.tb, o.niagaraHome, jar, info.sum))
	switch {
	case o.expectNiagara != "":
		j.Niagara = o.expectNiagara
	case home != "":
		j.Niagara = home
	default:
		j.Niagara = ">= " + shownOr(info.bajaFloor, "?")
	}
	j.conclude()
	return j
}
