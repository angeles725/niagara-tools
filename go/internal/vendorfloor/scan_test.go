package vendorfloor

import (
	"os"
	"testing"
)

// The tests below run the real driver: they skip, with the reason, when node or acorn is missing.

const (
	needsReplaceAll = "api: .replaceAll( needs Chrome 85" + guarded + "\n"
	notAtFloor      = "syntax: does not parse at ecmaVersion 2020 (Chromium 83 floor): "
)

// Mutation: VF5 -- parse at ecmaVersion latest so the ??= library parses and the FAIL row vanishes
func TestSyntaxAboveTheFloorFails(t *testing.T) {
	t.Parallel()
	vars := needAcorn(t)
	d := rc(t, map[string]string{"bad-2.0.0.min.js": "var cfg = {};\ncfg.timeout ??= 5000;\n"})
	code, out, errs := call(vars, d)
	check(t, "??=", code, out, errs, ExitFail, "FAIL  vendor-floor  vendor/bad-2.0.0.min.js:2  "+notAtFloor+"Unexpected token\n", "")
}

func TestModuleOnlyLibraryWarns(t *testing.T) {
	t.Parallel()
	vars := needAcorn(t)
	d := rc(t, map[string]string{"esm.mjs": "import x from \"./y.js\";\nexport default x;\n"})
	want := `vendor-floor  vendor/esm.mjs:1  syntax: parses only as an ES module ('import' and 'export' may appear only with 'sourceType: module' (1:0)) — load with type="module" or vendor the UMD/global build` + "\n"
	code, out, errs := call(vars, d)
	check(t, "module only", code, out, errs, ExitOK, "WARN  "+want, "")
	code, out, errs = call(vars, d, "--strict")
	check(t, "module only --strict", code, out, errs, ExitFail, "FAIL  "+want, "")
}

// Mutation: VF6 -- drop the API table so the rows vanish
func TestEachAPIFamilyWarns(t *testing.T) {
	t.Parallel()
	vars := needAcorn(t)
	d := rc(t, map[string]string{"api.js": "var a = s.replaceAll(\"-\", \"_\");\nvar b = Object.hasOwn(o, \"k\");\nvar c = structuredClone(o);\nvar d = new WeakRef(o);\n"})
	row := func(line, call string) string {
		return "  vendor-floor  vendor/api.js:" + line + "  api: " + call + guarded + "\n"
	}
	body := []string{row("1", ".replaceAll( needs Chrome 85"), row("2", "Object.hasOwn( needs Chrome 93"),
		row("3", "structuredClone( needs Chrome 98"), row("4", "new WeakRef needs Chrome 84")}
	var warn, fail string
	for _, r := range body {
		warn, fail = warn+"WARN"+r, fail+"FAIL"+r
	}
	code, out, errs := call(vars, d)
	check(t, "api", code, out, errs, ExitOK, warn, "")
	code, out, errs = call(vars, "--strict", d)
	check(t, "api --strict", code, out, errs, ExitFail, fail, "")
}

// Tokens, not text: an API name in a comment, a string or a template is never flagged.
func TestNameInACommentOrStringIsNotFlagged(t *testing.T) {
	t.Parallel()
	vars := needAcorn(t)
	d := rc(t, map[string]string{"lib-1.0.0.min.js": "/* s.replaceAll( and structuredClone( are not calls */\n// new WeakRef(x)\n" +
		"var msg = \"call .replaceAll( later\", tpl = `Object.hasOwn(`;\nfunction get(o) { return o?.a?.b ?? 0; }\nvar big = 10n;\n"})
	code, out, errs := call(vars, d, "--strict")
	check(t, "comment and string", code, out, errs, ExitOK, "", "")
}

// Mutation: VF9 -- tokenize as a module only so the API call after a legacy octal vanishes
func TestSloppyScriptKeepsItsRowsAfterALegacyOctal(t *testing.T) {
	t.Parallel()
	vars := needAcorn(t)
	d := rc(t, map[string]string{"sloppy.js": "var MODE = 010;\nfunction fmt(s) { return s.replaceAll(\"_\", \" \"); }\n"})
	code, out, errs := call(vars, d)
	check(t, "sloppy", code, out, errs, ExitOK, "WARN  vendor-floor  vendor/sloppy.js:2  "+needsReplaceAll, "")
}

func TestFilesAreReportedInPathOrder(t *testing.T) {
	t.Parallel()
	vars := needAcorn(t)
	d := rc(t, map[string]string{"z.js": "x ??= 1;\n", "a/lib.cjs": "s.replaceAll(a, b);\n", "ok.js": "var a = 1;\n"})
	code, out, errs := call(vars, d+"/")
	check(t, "order", code, out, errs, ExitFail,
		"WARN  vendor-floor  vendor/a/lib.cjs:1  "+needsReplaceAll+"FAIL  vendor-floor  vendor/z.js:1  "+notAtFloor+"Unexpected token\n", "")
}

// A file the driver cannot read ends the run as an env fault after the rows of the files before it.
func TestUnreadableFileIsAnEnvFault(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	vars := needAcorn(t)
	d := rc(t, map[string]string{"a.js": "s.replaceAll(a, b);\n", "b.js": "var b = 1;\n", "c.js": "x ??= 1;\n"})
	if err := os.Chmod(d+"/vendor/b.js", 0); err != nil {
		t.Fatal(err)
	}
	code, out, errs := call(vars, d)
	check(t, "unreadable", code, out, errs, ExitEnv, "WARN  vendor-floor  vendor/a.js:1  "+needsReplaceAll,
		"lint-vendor-floor: env fault: EACCES: permission denied, open '"+d+"/vendor/b.js'\n")
}
