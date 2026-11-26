package structure

import (
	"path/filepath"
	"strings"
	"testing"
)

const (
	servlet  = "package com.x.dash.ux;\npublic class BDashServlet extends BWebServlet {\n}\n"
	template = "package com.x.dash.login;\n\npublic final class BDashLoginTemplate extends BLoginTemplate {\n}\n"
	l18Row   = "L18: login outside the dashboard module (BLoginTemplate subclass in %s, which is not a dashboard -rt: " +
		"a dashboard ships its login in its own -rt, next to the -ux that serves the BWebServlet; " +
		"templates/dashboard-login, types/dashboard.md § The module ships its own login)"
)

// part is the files that keep a profile free of every other row (include file, lexicon, palette) plus its sources.
func part(name string, src map[string]string) map[string]string {
	out := map[string]string{name + "/module-include.xml": "<types/>\n", name + "/module.palette": "<p n=\"A\"/>\n"}
	for p, text := range src {
		out[name+"/src/com/x/"+p] = text
	}
	return out
}

func merge(parts ...map[string]string) map[string]string {
	out := map[string]string{".git/HEAD": "x\n", "M/gradle.properties": "org.gradle.java.installations.auto-detect=false\n"}
	for _, p := range parts {
		for k, v := range p {
			out["M/"+k] = v
		}
	}
	return out
}

// l18Rows runs the lint over a tree and returns its exit code and the L18 rows alone.
func l18Rows(t *testing.T, files map[string]string) (int, string) {
	t.Helper()
	dir := t.TempDir()
	write(t, dir, files)
	code, out, _ := run(filepath.Join(dir, "M"))
	var rows []string
	for _, ln := range strings.SplitAfter(out, "\n") {
		if strings.Contains(ln, "  L18: ") {
			rows = append(rows, strings.ReplaceAll(ln, filepath.Join(dir, "M")+"/", ""))
		}
	}
	return code, strings.Join(rows, "")
}

// TestLoginSeparateModule (L18): a BLoginTemplate subclass in an -rt part that has no BWebServlet of its own and
// no -ux sibling serving one is the retired separate login module: FAIL at the class line, beside a dashboard or alone.
func TestLoginSeparateModule(t *testing.T) {
	dash := merge(part("Dash-rt", map[string]string{"Dash.java": "class Dash {}\n"}), part("Dash-ux", map[string]string{"ux/BDashServlet.java": servlet}),
		part("DashLogin-rt", map[string]string{"login/BDashLoginTemplate.java": template}))
	alone := merge(part("DashLogin-rt", map[string]string{"login/BDashLoginTemplate.java": template}))
	for name, files := range map[string]map[string]string{"beside the dashboard": dash, "alone in its root": alone} {
		code, got := l18Rows(t, files)
		want := "FAIL  lint-structure  DashLogin-rt/src/com/x/login/BDashLoginTemplate.java:3  " + strings.Replace(l18Row, "%s", "DashLogin-rt", 1) + "\n"
		if code != 1 || got != want {
			t.Errorf("%s: exit %d rows\n%s\nwant 1 and\n%s", name, code, got, want)
		}
	}
}

// TestLoginOutsideRt (L18): a login template in a -ux part is outside any dashboard -rt, even where that -ux serves the servlet.
func TestLoginOutsideRt(t *testing.T) {
	files := merge(part("Dash-rt", map[string]string{"Dash.java": "class Dash {}\n"}),
		part("Dash-ux", map[string]string{"ux/BDashServlet.java": servlet, "login/BDashLoginTemplate.java": template}))
	code, got := l18Rows(t, files)
	want := "FAIL  lint-structure  Dash-ux/src/com/x/login/BDashLoginTemplate.java:3  " + strings.Replace(l18Row, "%s", "Dash-ux", 1) + "\n"
	if code != 1 || got != want {
		t.Errorf("exit %d rows\n%s\nwant 1 and\n%s", code, got, want)
	}
}

// TestLoginInDashboardIsClean (L18): the login in the -rt beside the servlet's -ux, a servlet and login in one -rt, a
// dashboard on Niagara's stock login (a servlet and no template: types/dashboard.md allows it) and a template named only
// in a comment each leave L18 silent.
func TestLoginInDashboardIsClean(t *testing.T) {
	ux := part("Dash-ux", map[string]string{"ux/BDashServlet.java": servlet})
	for name, files := range map[string]map[string]string{
		"own login":                    merge(part("Dash-rt", map[string]string{"login/BDashLoginTemplate.java": template}), ux),
		"servlet and login in one -rt": merge(part("Dash-rt", map[string]string{"login/BDashLoginTemplate.java": template, "ux/BDashServlet.java": servlet})),
		"stock login":                  merge(part("Dash-rt", map[string]string{"Dash.java": "class Dash {}\n"}), ux),
		"template in a comment":        merge(part("Dash-rt", map[string]string{"Dash.java": "// class T extends BLoginTemplate {}\nclass Dash {}\n"}), ux),
	} {
		if code, got := l18Rows(t, files); got != "" || code != 0 {
			t.Errorf("%s: exit %d, L18 rows %q; want 0 and none", name, code, got)
		}
	}
}
