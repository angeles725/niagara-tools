package rdbmsdialect

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: the command contract (usage, --strict, the empty-root error), written by hand from the G88 spec
// (build-n4-module-kit/ledger.d/2026-10-23-r8-u9-c68-c71.md): there is no bash lint to record.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 3) }

// rows runs the rule on a module root holding files (path -> source) and returns its exit code and stdout, root cut.
func rows(t *testing.T, files map[string]string) (int, string) {
	t.Helper()
	dir := t.TempDir()
	for p, src := range files {
		full := filepath.Join(dir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var o, e bytes.Buffer
	code := lint.Command(Spec())([]string{ID, dir}, cli.Stdio{Out: &o, Err: &e})
	if e.Len() != 0 {
		t.Fatalf("stderr %q", e.String())
	}
	return code, strings.ReplaceAll(o.String(), dir+"/", "")
}

const (
	exp   = "Mod/Mod-rt/src/com/x/BExport.java"
	other = "Mod/Mod-rt/src/com/x/BOther.java"
)

func class(name, members string) string {
	return "package com.x;\npublic class " + name + " {\n" + members + "}\n"
}

func warn(path string, line int, msg string) string {
	return "WARN  " + ID + "  " + path + ":" + string(rune('0'+line)) + "  " + msg + "\n"
}

const (
	fixedMsg = "hard-coded dialect descriptor: take it from getHistories().getExportDescriptorType().getInstance() (types/cloud-connector.md § 11)"
	pgMsg    = "postgres jdbc url: Postgres has no shipped dialect, ship a BRdbms subclass (types/cloud-connector.md § 11)"
)

// TestSpecCases: the six cases of the G88 spec and the edges around them.
func TestSpecCases(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files map[string]string
		want  string
	}{
		{"new BSqlServerHistoryExport()",
			map[string]string{exp: class("BExport", "  Object e = new BSqlServerHistoryExport();\n")}, warn(exp, 3, fixedMsg)},
		{"new BMySQLHistoryExport()",
			map[string]string{exp: class("BExport", "  Object e = new BMySQLHistoryExport();\n")}, warn(exp, 3, fixedMsg)},
		{"Oracle and HsqlDb are dialects too, with space before the parenthesis",
			map[string]string{exp: class("BExport", "  Object a = new BOracleHistoryExport ();\n  Object b = new BHsqlDbHistoryExport(\n  );\n")},
			warn(exp, 3, fixedMsg) + warn(exp, 4, fixedMsg)},
		{"a postgres jdbc url in a module with no BRdbms subclass",
			map[string]string{exp: class("BExport", "  String u = \"jdbc:postgresql://db/x\";\n")}, warn(exp, 3, pgMsg)},
		{"the descriptor read from the database is clean",
			map[string]string{exp: class("BExport", "  Object e = db.getHistories().getExportDescriptorType().getInstance();\n")}, ""},
		{"the same new in a comment, a javadoc or a string is clean",
			map[string]string{exp: class("BExport", "  // new BSqlServerHistoryExport()\n  /** new BMySQLHistoryExport() */\n  String s = \"new BOracleHistoryExport()\";\n")}, ""},
		{"a postgres url in a comment is clean",
			map[string]string{exp: class("BExport", "  // jdbc:postgresql://db/x\n")}, ""},
		{"a postgres url in a module that ships its own BRdbms dialect is clean",
			map[string]string{exp: class("BExport", "  String u = \"jdbc:postgresql://db/x\";\n"), other: "package com.x;\npublic class BPgRdbms extends BRdbms {\n}\n"}, ""},
		{"a class that only mentions BRdbms in a comment is no dialect",
			map[string]string{exp: class("BExport", "  String u = \"jdbc:postgresql://db/x\";\n"), other: class("BOther", "  // extends BRdbms\n")}, warn(exp, 3, pgMsg)},
		{"the fixed dialect still WARNs when the module ships a BRdbms dialect",
			map[string]string{exp: class("BExport", "  Object e = new BMySQLHistoryExport();\n"), other: "package com.x;\npublic class BPgRdbms extends BRdbms {\n}\n"}, warn(exp, 3, fixedMsg)},
		{"srcTest and a part that is not -rt are not read",
			map[string]string{exp: class("BExport", "  int x;\n"), "Mod/Mod-rt/srcTest/com/x/T.java": class("T", "  Object e = new BMySQLHistoryExport();\n"),
				"Mod/Mod-wb/src/com/x/W.java": class("W", "  Object e = new BMySQLHistoryExport();\n")}, ""},
	} {
		if code, got := rows(t, tc.files); got != tc.want || code != 0 {
			t.Errorf("%s: exit %d, got\n%s\nwant\n%s", tc.name, code, got, tc.want)
		}
	}
}
