package inlinesiteliteral

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: the wrapper contract, written by hand from the deferred G42 spec (no bash lint to record).
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 5) }

// run runs the rule alone on a module root holding files; flags may name @ROOT@. It returns exit, stdout and
// stderr with the root cut.
func run(t *testing.T, files map[string]string, flags ...string) (int, string, string) {
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
	args := []string{ID}
	for _, f := range flags {
		args = append(args, strings.ReplaceAll(f, "@ROOT@", dir))
	}
	var o, e bytes.Buffer
	code := lint.Command(Spec())(append(args, dir), cli.Stdio{Out: &o, Err: &e})
	return code, strings.ReplaceAll(o.String(), dir+"/", ""), strings.ReplaceAll(e.String(), dir, "@ROOT@")
}

func warn(path string, line int, kind, lit string) string {
	return "WARN  " + ID + "  " + path + ":" + strconv.Itoa(line) + "  " + kind + " literal outside the site table: \"" + lit + "\"\n"
}

const (
	logic   = "Mod/Mod-rt/src/com/x/BRoomControl.java"
	servlet = "Mod/Mod-ux/src/com/x/ux/BRoomServlet.java"
	app     = "Mod/Mod-ux/src/rc/app.js"
	table   = "Mod/Mod-rt/src/com/x/Site.java"
)

func class(name, body string) string { return "package com.x;\nclass " + name + " {\n" + body + "}\n" }

const ord = `  BOrd room = BOrd.make("station:|slot:/Rooms/Room1");` + "\n"

// TestSpecCases: the six cases of the deferred G42 spec, each with the edge that bounds it.
func TestSpecCases(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files map[string]string
		flags []string
		want  string
	}{
		{"spec 1: an ORD in a logic class", map[string]string{logic: class("BRoomControl", ord)}, nil,
			warn(logic, 3, "ord", "station:|slot:/Rooms/Room1")},
		{"spec 2a: the same literal in a --site-table class", map[string]string{table: class("Site", ord), logic: class("BRoomControl", "  BOrd r = Site.ROOM1;\n")},
			[]string{"--site-table", "@ROOT@/" + table}, ""},
		{"spec 2b: the same literal in a first-line // site-table class", map[string]string{table: "// site-table: the one place for station ORDs\n" + class("Site", ord)}, nil, ""},
		{"a marker below the first line is not a site table", map[string]string{table: "package com.x;\n// site-table\nclass Site {\n" + ord + "}\n"}, nil,
			warn(table, 4, "ord", "station:|slot:/Rooms/Room1")},
		{"spec 3: an IPv4 address in a servlet", map[string]string{servlet: class("BRoomServlet", "  static final String HOST = \"192.168.1.20\";\n")}, nil,
			warn(servlet, 3, "ip", "192.168.1.20")},
		{"spec 4: a site pattern in rc/app.js", map[string]string{app: "var name = \"Room3\";\nvar other = 'Hall';\n"},
			[]string{"--site-pattern", "Room[0-9]+"}, warn(app, 1, "site", "Room3")},
		{"a Java char literal is not a site literal", map[string]string{logic: class("BRoomControl", "  char c = 'R';\n  String s = \"R\";\n")},
			[]string{"--site-pattern", "^R$"}, warn(logic, 4, "site", "R")},
		{"without --site-pattern a room name is not decidable", map[string]string{app: "var name = \"Room3\";\n"}, nil, ""},
		{"spec 5: an ORD in a comment", map[string]string{logic: class("BRoomControl", "  // BOrd.make(\"station:|slot:/Rooms/Room1\")\n  /* \"slot:/Rooms/Room2\" */\n"),
			app: "// fetch('station:|slot:/Rooms/Room1');\n/* \"192.168.1.20\" */\nvar a = 1;\n"}, nil, ""},
		{"spec 6: an ORD in a test source", map[string]string{"Mod/Mod-rt/srcTest/com/x/BRoomTest.java": class("BRoomTest", ord), logic: class("BRoomControl", "  int a;\n")}, nil, ""},
		{"a -wb part and a .js outside rc/ are not read", map[string]string{"Mod/Mod-wb/src/com/x/W.java": class("W", ord), "Mod/Mod-ux/src/tools/gen.js": "var o = 'station:|slot:/A';\n", logic: class("BRoomControl", "  int a;\n")}, nil, ""},
		{"ORD shapes", map[string]string{logic: class("BRoomControl", "  String a = \"slot:/Rooms/Room1\";\n  String b = \"local:|station:|slot:/A\";\n  String c = \"ip:10.1.1.1|fox:|station:|slot:/B\";\n  String d = \"h:2e4f\";\n  String e = \"station:|h:1a\";\n")}, nil,
			warn(logic, 3, "ord", "slot:/Rooms/Room1") + warn(logic, 4, "ord", "local:|station:|slot:/A") + warn(logic, 5, "ord", "ip:10.1.1.1|fox:|station:|slot:/B") + warn(logic, 6, "ord", "h:2e4f") + warn(logic, 7, "ord", "station:|h:1a")},
		// Tuning (2026-10-24 Cliente live roots, 7 of 12 rows false): a query ORD that names no slot below the
		// station root (alarm database, root BQL) is the same on every station, not site data.
		{"query ORDs with no slot path", map[string]string{logic: class("BRoomControl", "  String a = \"station:|alarm:|bql:select * where ackState != 'acked'\";\n  String b = \"station:|slot:/|bql:select * from schedule:NumericSchedule\";\n  String c = \"station:|slot:/\";\n  String d = \"station:|slot:/Drivers|bql:select *\";\n")}, nil,
			warn(logic, 6, "ord", "station:|slot:/Drivers|bql:select *")},
		// Decision 2026-10-24 (retro go-lint-inline-site-literal Δ2): a service ORD, any path under
		// /Services, is module structure identical on every site, not site data.
		{"service ORDs", map[string]string{logic: class("BRoomControl", "  String a = \"station:|slot:/Services/ChiDashboardService\";\n  String b = \"station:|slot:/Services/ChiDashboardService/\";\n  String c = \"slot:/Services/AlarmService/Console\";\n  String d = \"station:|slot:/Rooms/Room1\";\n  String e = \"station:|slot:/ServicesX/A\";\n")}, nil,
			warn(logic, 6, "ord", "station:|slot:/Rooms/Room1") + warn(logic, 7, "ord", "station:|slot:/ServicesX/A")},
		{"not ORDs: a time format, a slot word, a relative path", map[string]string{logic: class("BRoomControl", "  String a = \"h:mm a\";\n  String b = \"slot\";\n  String c = \"file:^data/x\";\n  String d = \"bql:select *\";\n")}, nil, ""},
		{"IPv4 shapes", map[string]string{logic: class("BRoomControl", "  String a = \"http://10.0.0.5:8080/x\";\n  String b = \"999.1.1.1\";\n  String c = \"1.2.3\";\n  String d = \"1.2.3.4.5\";\n")}, nil,
			warn(logic, 3, "ip", "http://10.0.0.5:8080/x")},
		{"JS strings and templates", map[string]string{app: "const base = `station:|slot:/Rooms/${room}`;\nconst ip = '10.0.0.7';\nconst re = /station:\\|slot:\\//;\n"}, nil,
			warn(app, 1, "ord", "station:|slot:/Rooms/${room}") + warn(app, 2, "ip", "10.0.0.7")},
		{"one row per line: its first site literal", map[string]string{logic: class("BRoomControl", "  f(\"10.0.0.1\", \"station:|slot:/A\");\n")}, nil,
			warn(logic, 3, "ip", "10.0.0.1")},
		{"a long literal is cut", map[string]string{logic: class("BRoomControl", "  String a = \"station:|slot:/"+strings.Repeat("x", 100)+"\";\n")}, nil,
			warn(logic, 3, "ord", "station:|slot:/"+strings.Repeat("x", 57)+"...")},
	} {
		code, out, errs := run(t, tc.files, tc.flags...)
		if code != 0 || out != tc.want || errs != "" {
			t.Errorf("%s: exit %d err %q got\n%s\nwant\n%s", tc.name, code, errs, out, tc.want)
		}
	}
}

// TestOptions: --site-table must be a file, --site-pattern a valid regex; --strict turns a row into exit 1.
func TestOptions(t *testing.T) {
	files := map[string]string{logic: class("BRoomControl", ord)}
	if code, _, errs := run(t, files, "--site-table", "@ROOT@/nope.java"); code != 3 || errs != ID+": --site-table file not found: @ROOT@/nope.java\n" {
		t.Errorf("missing table: exit %d err %q", code, errs)
	}
	if code, _, errs := run(t, files, "--site-pattern", "Room[0-9"); code != 3 || !strings.HasPrefix(errs, ID+": bad --site-pattern Room[0-9: ") {
		t.Errorf("bad pattern: exit %d err %q", code, errs)
	}
	if code, out, _ := run(t, files, "--strict"); code != 1 || out != warn(logic, 3, "ord", "station:|slot:/Rooms/Room1") {
		t.Errorf("--strict: exit %d out %q", code, out)
	}
}
