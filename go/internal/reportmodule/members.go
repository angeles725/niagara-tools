package reportmodule

// input is the directory a member reads.
type input int

const (
	inSrc      input = iota // <artifact>/src; a SKIP "no src/" row when it is not a directory
	inArtifact              // <artifact> itself (the module root for a module member), ungated
	inRc                    // <artifact>/src/rc; a SKIP "no src/rc/" row when it is not a directory
)

// verdict is the row a lint member adds after its relayed rows.
type verdict int

const (
	none     verdict = iota // no verdict row: the member's own rows speak
	rows                    // PASS "clean" when no row was relayed
	failRows                // PASS "clean" when no FAIL row was relayed (WARN rows do not block)
)

// member is one step of the report. A table member runs <toolbelt>/<script>.sh on its input and relays the
// rows filter f selects, then its verdict; an advisory one relays its WARN rows with a PASS verdict row and turns
// any non-zero exit into an ERROR row; a module member with a noSource text SKIPs on its own "no ... sources"
// exit 3; a table member's rows carry its script name as the check. A special member (special*.go) does its own
// rows and names its own checks (plano, vendor-floor, eslint, dup-keys, ...).
type member struct {
	script   string // toolbelt member, without .sh ("" for ESLint, which is not a toolbelt script)
	suffix   string // the artifact name must end with it ("" any artifact)
	in       input
	f        filter
	v        verdict
	advisory bool
	// modeAware: an advisory member whose rows FAIL in a `modularity: strict` module (lint-size), so its exit 1
	// with a FAIL row is relayed as FAIL; any other advisory member's exit 1 is an env fault.
	modeAware bool
	noSource  string
	// floor: a floor check (BUILD-LOOP §5): its FAIL rows can never be baselined (baseline.go).
	floor bool
	// home: the member takes the installation, `--niagara-home <dir>` before its input when report-module
	// was given one; without it the member reads NIAGARA_HOME from the environment it inherits.
	home    bool
	special func(r *report, a artifact)
}

// ntplScript is the template lint: a module member that needs the installation to know the stock modules.
const ntplScript = "lint-ntpl-manifest-deps"

// artifact is what a member runs on: a profile artifact (its directory name labels the rows) or, for the
// once-per-module members, the module root labelled "(module)".
type artifact struct{ name, dir string }

var (
	failWarn = filter{sel: "FAIL|WARN"}
	warnOnly = filter{sel: "WARN"}
	failOnly = filter{sel: "FAIL"}
)

// perArtifact is every per-artifact step, in report order (the bash §1-§6).
var perArtifact = []member{
	{script: "verify-module", special: (*report).verifyModule, floor: true},
	{script: "slot-coverage", special: (*report).slotCoverage},
	{script: "lint-timers", special: (*report).lintTimers},
	{script: "verify-module", special: (*report).plano},
	{script: "lint-lexicon-ascii", special: (*report).lexiconASCII},
	// kit-backlog B11.6: the lexicon-ascii companion; a clean or lexicon-less artifact prints nothing.
	{script: "lint-lexicon-apostrophe", in: inArtifact, f: warnOnly},
	{script: "lint-delays", f: failWarn},
	{script: "lint-demand-scope", f: warnOnly},
	{script: "lint-silent-protection", f: filter{sel: "FAIL|WARN|ADVISORY"}},
	{script: "lint-link-target-flags", special: (*report).linkTargetFlags},
	{script: "lint-ext-writable-shape", f: failWarn},
	{script: "lint-no-system-out", f: failOnly, v: rows},
	{script: "lint-slotomatic-markers", f: failWarn, v: rows},
	{script: "lint-instance-type-order", f: failWarn},
	{script: "lint-clock-zero-floor", f: warnOnly},
	{script: "lint-null-context-write", f: warnOnly},
	{script: "lint-bql-string-concat", f: warnOnly},
	{script: "lint-arbitrary-ord", f: warnOnly},
	{script: "lint-persist-hot-write", f: warnOnly},
	{script: "lint-changed-hot-write", f: warnOnly},
	{script: "lint-inert-coordination", f: warnOnly},
	{script: "lint-session-store-lazy-evict", f: warnOnly},
	{script: "lint-bformat-call", f: warnOnly},
	// kit-backlog B10.8: the course-coverage source lints (corpus B1417, B1419). No verdict row, as lint-service-types.
	{script: "lint-charset-default", f: warnOnly},
	{script: "lint-slot-default-type", f: failOnly},
	// kit-backlog B11.6: the course-coverage run B source lints (corpus B1242 K3, K4), advisory WARN rows.
	{script: "lint-lifecycle-guard", f: warnOnly},
	{script: "lint-intrinsic-lock", f: warnOnly},
	{script: "lint-service-types", f: failWarn},
	{script: "lint-driver-scaffold", f: failWarn},
	{script: "lint-se-display", suffix: "-se", f: failOnly, v: rows},
	{script: "lint-jasmine-ux", suffix: "-ux", in: inArtifact, f: filter{sel: "WARN", mode: nosite}},
	{script: "lint-wb-external-ord-value", suffix: "-wb", f: warnOnly},
	{script: "lint-spa-poll-no-recovery", suffix: "-ux", in: inRc, f: failWarn},
	{script: "lint-subscribe-without-unsubscribe", suffix: "-rt", f: warnOnly},
	{script: "lint-recovery-path", suffix: "-rt", f: failOnly, v: rows},
	{script: "lint-config-sanity", suffix: "-rt", f: failWarn, v: failRows},
	{script: "lint-status-parity", suffix: "-rt", f: warnOnly},
	{script: "lint-servlet", suffix: "-ux", special: (*report).servlet},
	// retro 2026-11-10-mcpn4-history-batch-tools Δ6: the write guards of any artifact's web code (a servlet can live in -rt).
	{script: "lint-servlet-write-guards", f: failWarn},
	{script: "rc-scan", suffix: "-ux", special: (*report).rcScan},
	{suffix: "-ux", special: (*report).eslint},
	{script: "lint-vendor-floor", suffix: "-ux", special: (*report).vendorFloor},
	// retro kit-gzip-bundle-writer-briefs Δ1: a dist/*.js over 300 KB ships its .gz twin (native, no toolbelt script; WARN only).
	{suffix: "-ux", special: (*report).bundleGzip},
	{script: "lint-wb-threading", suffix: "-wb", f: filter{mode: swap, keep: "WARN"}},
	{script: "lint-no-md5-credential-digest", f: warnOnly, v: rows},
	{script: "lint-wb-file-chooser", suffix: "-wb", f: warnOnly},
	{script: "lint-size", advisory: true, modeAware: true},
	{script: "lint-license-isoperational-gate", advisory: true},
	{script: "lint-set-null-ord", advisory: true},
	{script: "lint-ws-placement", advisory: true},
	{script: "schema-risk", special: (*report).schemaRisk, floor: true},
}

// perModule is every once-per-module step on the module root, in report order (the bash §8-§13).
var perModule = []member{
	{script: "lint-agent-on-shape", in: inArtifact, f: failOnly, v: rows},
	{script: "lint-agent-on-include-sync", in: inArtifact, f: failOnly, v: rows},
	{script: "lint-uberjar-api-conflict", in: inArtifact, f: filter{sel: "WARN", mode: base}},
	{script: "lint-structure", in: inArtifact, f: failWarn, v: failRows},
	{script: "lint-module-map", in: inArtifact, f: filter{sel: "FAIL|WARN", mode: base}, v: failRows},
	{special: (*report).modularity},
	{script: "lint-write-path", special: (*report).writePath},
	{script: "lint-bundled-jar-class-version", in: inArtifact, f: filter{sel: "FAIL", mode: base}, v: rows},
	{script: "lint-duplicate-helper", in: inArtifact, f: warnOnly, noSource: "no *.java under a -rt, -ux or -wb part's src/"},
	{script: "lint-machine-default-path", in: inArtifact, f: warnOnly, noSource: "no *.java or *.js or *.properties sources found"},
	{script: "lint-inline-site-literal", in: inArtifact, f: warnOnly, noSource: "no *.java under a -rt or -ux part's src/ and no *.js under an rc/"},
	{script: "lint-private-api-import", in: inArtifact, f: warnOnly, noSource: "no *.java under a -rt, -ux, -wb or -se part's src/"},
	{script: "lint-ndriver-static-collection", in: inArtifact, f: warnOnly, noSource: "no *.java under a -rt part's src/"},
	{script: "lint-ndriver-tagless-correlation", in: inArtifact, f: warnOnly, noSource: "no *.java under a -rt part's src/"},
	// a module with no srcTest/ *.java has nothing to judge: its exit 3 is a SKIP row, never an env fault
	// Mutation: RM-vacuous-nosource -- dropping noSource turns every test-less module's report into an env fault
	{script: "lint-vacuous-tests", in: inArtifact, f: failOnly, v: rows, noSource: "no *.java under a srcTest/"},
	// kit-backlog B5.5: the run-3 module lints. A module without their sources SKIPs, never an env fault.
	// Mutation: RM-new-members-nosource -- dropping noSource turns a module with no permissions file or -rt Java into an env fault
	{script: "lint-permissions-xml", in: inArtifact, f: failWarn, v: failRows, noSource: "no module-permissions.xml and no *.java under a -rt part's src/"},
	{script: "lint-structural-tests", in: inArtifact, f: warnOnly, noSource: "no *.java under a srcTest/"},
	{script: "lint-logger-name", in: inArtifact, f: warnOnly, noSource: "no *.java under a -rt, -ux, -wb or -se part's src/"},
	{script: "lint-adapter-cfg-default", in: inArtifact, f: warnOnly, noSource: "no *.java under a -rt part's src/"},
	// issue #821: a -ux part that writes with the request Context and no station save in the module FAILs; a module
	// with no -ux part has no dashboard writes, so its exit 3 is a SKIP row.
	// Mutation: RM-dashboard-save-nosource -- dropping noSource turns every module without a -ux part into an env fault
	{script: "lint-dashboard-write-save", in: inArtifact, f: failOnly, v: rows, noSource: "no *.java under a -ux part's src/"},
	// retro coordination-stretch-deltas K2/K8: the template lint, with the installation report-module is given.
	// Mutation: RM-ntpl-home -- dropping home runs it without the installation, so a missing module is WARN and never blocks
	{script: ntplScript, in: inArtifact, f: filter{sel: "FAIL|WARN", mode: whole}, v: failRows, home: true, noSource: "no *.gradle.kts and no *.ntpl under the root"},
}

// Scripts lists every toolbelt member the report runs, as script file names (lint-timers.sh, ...), each once
// and in report order; triage-console.sh is the last, run once per report. toolbelt/INDEX.md's Auto column
// reads it (go/internal/lintindex).
func Scripts() []string {
	var out []string
	seen := map[string]bool{}
	for _, list := range [][]member{perArtifact, perModule, {{script: "triage-console"}}} {
		for _, m := range list {
			if m.script != "" && !seen[m.script] {
				seen[m.script] = true
				out = append(out, m.script+".sh")
			}
		}
	}
	return out
}
