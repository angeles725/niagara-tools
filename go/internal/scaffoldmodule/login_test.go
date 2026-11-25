package scaffoldmodule

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// loginKitFiles is the dashboard's part of the small kit: the plumbing files the login fragments are put into,
// and templates/dashboard-login (files copied, fragments inserted), each carrying the tokens its rules rewrite.
func loginKitFiles() map[string]string {
	fx := "fixtures/MinimalDash/MinimalDash/"
	rt := fx + "MinimalDash-rt/"
	tp := "templates/dashboard-login/"
	return map[string]string{
		fx + "MODULE-MAP.md":                                          "| control | rt | x |\n| plumbing | rt | y |\n",
		rt + "module-include.xml":                                     "<types>\n  <type class=\"com.angeles.MinimalDash.BMinimalDash\"/>\n</types>\n",
		rt + "module.palette":                                         "<p>\n  <p n=\"MinimalDash\"/>\n</p>\n</bajaObjectGraph>\n",
		rt + "MinimalDash-rt.gradle.kts":                              "deps {\n  api(\":baja\")\n}\nbajadoc {\n  includePackage(\"com.angeles.MinimalDash\")\n}",
		rt + "module.lexicon":                                         "MinimalDash.displayName=X\n",
		fx + "MinimalDash-ux/Servlet.java":                            "getServletName() { return \"minimaldash\"; }\n",
		tp + "files/src/com/example/ModuleA/login/LoginRedirect.java": "package com.example.ModuleA.login; // Example ModuleA \"/modulea/\"\n",
		tp + "files/src/rc/modulea-login/login.css":                   "/* modulea-login */\n",
		tp + "fragments/module-include.xml":                           "  <type class=\"com.example.ModuleA.login.BModuleALoginTemplate\"/>\n",
		tp + "fragments/module.palette":                               "  <p n=\"ModuleAService\" t=\"@SYM@:ModuleAService\"/>\n",
		tp + "fragments/gradle-deps.txt":                              "  api(\":web-rt\")\n",
		tp + "fragments/gradle-bajadoc.txt":                           "  includePackage(\"com.example.ModuleA.login\")\n",
		tp + "fragments/gradle-jar.txt":                               "jar { include(\"modulea-login/**\") }\n",
		tp + "fragments/module.lexicon":                               "applyLogin=A\n",
		tp + "fragments/module-map-row.txt":                           "| login | rt | ModuleA-rt/src/com/example/ModuleA/login/LoginRedirect.java |\n",
	}
}

// The dashboard gets the login overlay in -rt (files with the module's names, the lower-case name in the asset
// directory) and the fragments in the fixture's plumbing, each at its anchor; the dashboard servlet's mount
// follows the module name.
func TestLoginOverlay(t *testing.T) {
	out := t.TempDir()
	r := run(t, newKit(t), "ColdBox", out, "--type", "dashboard", "--vendor", "Acme")
	if r.code != 0 || r.err != "" {
		t.Fatalf("code=%d err=%q", r.code, r.err)
	}
	got := tree(t, filepath.Join(out, "ColdBox"))
	rt := "ColdBox/ColdBox-rt/"
	wantFile(t, got, rt+"src/com/acme/ColdBox/login/LoginRedirect.java", "package com.acme.ColdBox.login; // Acme ColdBox \"/coldbox/\"\n")
	wantFile(t, got, rt+"src/rc/coldbox-login/login.css", "/* coldbox-login */\n")
	wantFile(t, got, "ColdBox/MODULE-MAP.md",
		"| control | rt | x |\n| login | rt | ColdBox-rt/src/com/acme/ColdBox/login/LoginRedirect.java |\n| plumbing | rt | y |\n")
	wantFile(t, got, rt+"module-include.xml",
		"<types>\n  <type class=\"com.acme.ColdBox.BColdBox\"/>\n  <type class=\"com.acme.ColdBox.login.BColdBoxLoginTemplate\"/>\n</types>\n")
	wantFile(t, got, rt+"module.palette", "<p>\n  <p n=\"ColdBox\"/>\n  <p n=\"ColdBoxService\" t=\"cb:ColdBoxService\"/>\n</p>\n</bajaObjectGraph>\n")
	wantFile(t, got, rt+"ColdBox-rt.gradle.kts", "deps {\n  api(\":baja\")\n  api(\":web-rt\")\n}\nbajadoc {\n  includePackage(\"com.acme.ColdBox\")\n"+
		"  includePackage(\"com.acme.ColdBox.login\")\n}\njar { include(\"coldbox-login/**\") }\n") // a file with no final newline gets one
	wantFile(t, got, rt+"module.lexicon", "ColdBox.displayName=X\napplyLogin=A\n")
	wantFile(t, got, "ColdBox/ColdBox-ux/Servlet.java", "getServletName() { return \"coldbox\"; }\n")
}

// No other type gets a login file, and none of their plumbing changes.
func TestLoginOverlayIsDashboardOnly(t *testing.T) {
	tb := newKit(t)
	for _, typ := range []string{"logic", "driver"} {
		out := t.TempDir()
		r := run(t, tb, "ColdBox", out, "--type", typ, "--vendor", "Acme")
		if r.code != 0 || r.err != "" {
			t.Fatalf("%s: code=%d err=%q", typ, r.code, r.err)
		}
		for rel := range tree(t, filepath.Join(out, "ColdBox")) {
			if strings.Contains(rel, "login") || strings.HasSuffix(rel, "Service.java") {
				t.Errorf("%s: a login file: %s", typ, rel)
			}
		}
	}
}

// A kit without templates/dashboard-login is exit 3 naming it and removing the partial tree; only the
// dashboard reads it. A fixture file the fragment needs an anchor in, changed, is exit 3 naming file and anchor.
func TestLoginFaults(t *testing.T) {
	t.Run("template missing", func(t *testing.T) {
		tb := newKit(t, "templates/dashboard-login")
		out := t.TempDir()
		r := run(t, tb, "Foo", out, "--type", "dashboard", "--vendor", "Angeles")
		want := "scaffold-module: dashboard-login template not found: " + tb + "/../templates/dashboard-login\n" +
			"scaffold-module: removed the partial " + out + "/Foo\n"
		if r.code != 3 || r.out != "" || r.err != want {
			t.Errorf("code=%d out=%q err=%q", r.code, r.out, r.err)
		}
		if r := run(t, tb, "Bar", out, "--type", "driver", "--vendor", "Angeles"); r.code != 0 {
			t.Errorf("driver: code=%d err=%q", r.code, r.err)
		}
	})
	t.Run("anchor missing", func(t *testing.T) {
		tb := newKit(t)
		path := filepath.Join(tb, "..", "fixtures/MinimalDash/MinimalDash/MODULE-MAP.md")
		if err := os.WriteFile(path, []byte("| control | rt | x |\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		out := t.TempDir()
		r := run(t, tb, "Foo", out, "--type", "dashboard", "--vendor", "Angeles")
		want := "scaffold-module: cannot add the login to " + out + "/Foo/Foo/MODULE-MAP.md: \"| plumbing |\" not found\n" +
			"scaffold-module: removed the partial " + out + "/Foo\n"
		if r.code != 3 || r.out != "" || r.err != want {
			t.Errorf("code=%d out=%q err=%q", r.code, r.out, r.err)
		}
	})
}
