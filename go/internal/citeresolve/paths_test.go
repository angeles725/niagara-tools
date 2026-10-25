package citeresolve

import "testing"

func TestFilePointers(t *testing.T) {
	t.Parallel()
	const vAlias = "Code: `V=organized/baja/baja/vineflower/com/tridium` (4.14).\n"
	check(t, []struct {
		name, text string
		want       []want
	}{
		{"versioned devguide range", "[ev: devguide lex_lex.js.txt:441-455]", []want{{Resolved, "lex line 441"}}},
		{"devguide range past EOF", "[ev: devguide lex_lex.js.txt:455-470]", []want{{LineOutOfRange, "file has 460"}}},
		{"unversioned-only devguide notes the mismatch", "[ev: devguide only-old.txt:2]", []want{{Resolved, "version mismatch"}}},
		{"missing devguide", "[ev: devguide nope.txt:1]", []want{{MissingFile, "nope.txt"}}},
		{"guide", "[ev: guide Platform/aSoftwareManager.txt:18]", []want{{Resolved, "guide line 18"}}},
		{"niagara-help with a version", "[ev: niagara-help 4.15.6.20 guides-clean/Platform/aSoftwareManager.txt:3]", []want{{Resolved, "guide line 3"}}},
		{"source path reads the versioned copy first", "[ev: source/baja/javax/baja/util/BWorker.java:12]", []want{{Resolved, "source BWorker line 12"}}},
		{"alias in prose", vAlias + "FINE (`V/sys/Nre.java:730`).", []want{{Resolved, "// Nre line 730"}}},
		{"ellipsis resolves against an earlier hit of the card",
			"[ev: niagara-help 4.15.6.20 guides-clean/Platform/aSoftwareManager.txt:1] [ev: …/Platform/aSoftwareManager.txt:2]",
			[]want{{Resolved, "guide line 1"}, {Resolved, "guide line 2"}}},
		{"ellipsis with no earlier hit", "[ev: …/Platform/aSoftwareManager.txt:2]", []want{{MissingFile, "ellipsis"}}},
		{"inline prose citation", "x (devguide `lex_lex.js.txt:3`) y", []want{{Resolved, "lex line 3"}}},
		{"code in a module part", "[ev: code kitControl-rt BFoo.java:10]", []want{{Resolved, "// BFoo line 10"}}},
		{"code path searched in the decompiles", "[ev: code javax/baja/sys/BA.java:150]", []want{{Resolved, "// BA line 150"}}},
		{"code class name", "[ev: code kitControl-rt BFoo]", []want{{Resolved, "BFoo line 1"}}},
		{"a file in the organized/ listing is not a module", "[ev: code javax/baja/sys/BA.java:150]", []want{{Resolved, "// BA line 150"}}},
		{"code file missing from its module", "[ev: code kitControl-rt BNope.java]", []want{{MissingFile, "kitControl-rt"}}},
		{"vineflower prose citation", "vineflower kitControl-rt 4.14.0.162 `com/tridium/kitControl/BFoo.java:2-3`", []want{{Resolved, "// BFoo line 2"}}},
		{"javap jar found", "javap of `kitControl-rt.jar` 4.15.6.20", []want{{Resolved, "javap output not checked"}}},
		{"javap tag finds the jar", "[ev: javap kitControl-rt.jar 4.15.6.20]", []want{{Resolved, "javap output not checked"}}},
		{"javap jar missing", "javap of `nope-rt.jar` 4.15.6.20", []want{{MissingFile, "nope-rt.jar"}}},
		{"jar entry resolves the jar", "[ev: code kitControl-rt.jar!/com/x/Y.class]", []want{{Resolved, "entry com/x/Y.class not checked"}}},
		{"jar entry of a missing jar", "[ev: code nope-rt.jar!gradle/b.vm:18]", []want{{MissingFile, "nope-rt.jar"}}},
		{"backticked path with spaces", "see `types/with space/note.md:2` x", []want{{Resolved, "note line 2"}}},
		{"spaced path under the course tree", "see `Slide Code/gradle.properties:14` x", []want{{Resolved, "course.prop14=x"}}},
		{"spaced ellipsis path searched in the course tree", "see `…/Lab10 Jobs/BJob.java:1` x", []want{{Resolved, "[read "}}},
		{"spaced infix ellipsis below a course directory", "see `Slide Code/…/gradle.properties:14` x", []want{{Resolved, "course.prop14=x"}}},
		{"kit cross-reference, a section list", "`types/platform.md` § 1, § 2.2 say", []want{{Resolved, "## 1. Profiles | ### 2.2 Go module"}}},
		{"memory by prefix", "[ev: mem shared-checkout]", []want{{Resolved, "shared-checkout-git-discipline.md"}}},
		{"missing memory", "[ev: mem nope]", []want{{MissingFile, "nope"}}},
		{"kit cross-reference by number", "`types/platform.md` § 2, so", []want{{Resolved, "## 2. Software Manager"}}},
		{"kit cross-reference by title", "(`METHODOLOGY.md` § Domain correctness) does", []want{{Resolved, "## Domain correctness"}}},
		{"kit cross-reference, title running into prose", "`types/platform.md` § Domain correctness rules say so", []want{{Resolved, "## Domain correctness rules"}}},
		{"kit cross-reference, heading prefixes the title", "`types/platform.md` § Profiles Mobile rule", []want{{Resolved, "## 1. Profiles"}}},
		{"guide-relative path", "| `Platform/aSoftwareManager.txt:4` |", []want{{Resolved, "guide line 4"}}},
		{"kit cross-reference, title cut at a parenthesis", "`types/platform.md` § Domain correctness (x) and", []want{{Resolved, "## Domain correctness rules"}}},
		{"kit cross-reference, a section range", "`types/platform.md` § 1-2 say", []want{{Resolved, "## 1. Profiles | ## 2. Software Manager"}}},
		{"kit cross-reference, a shared first word is not a match", "`types/platform.md` § Software Factory", []want{{MissingSection, "§Software Factory"}}},
		{"kit cross-reference to a missing heading", "`types/platform.md` § 9, so", []want{{MissingSection, "§9"}}},
		{"repo test name", "[ev: tests/awk.bats AWB1]", []want{{Resolved, "@test \"AWB1"}}},
		{"repo test name missing", "[ev: tests/awk.bats AWB9]", []want{{MissingSection, "AWB9"}}},
		{"package path in prose searched in the decompiles", "see `javax/baja/sys/BA.java:12` there", []want{{Resolved, "// BA line 12"}}},
		{"infix ellipsis below a module part", "[ev: kitControl-rt/…/kitControl/BFoo.java:4]", []want{{Resolved, "// BFoo line 4"}}},
		{"infix ellipsis with no match", "[ev: kitControl-rt/…/other/BFoo.java:4]", []want{{MissingFile, "no other/BFoo.java below kitControl-rt"}}},
		{"organized-relative path", "[ev: kitControl/kitControl-rt/vineflower/com/tridium/kitControl/BFoo.java:5]", []want{{Resolved, "// BFoo line 5"}}},
		{"absolute path is not joined to a root", "[ev: /METHODOLOGY.md]", []want{{MissingFile, "/METHODOLOGY.md"}}},
		{"client module path", "[ev: Cliente/Pan/BPan.java:5]", []want{{Resolved, "// pan 5"}}},
		{"missing path", "[ev: go/internal/nope/x.go:1]", []want{{MissingFile, "go/internal/nope/x.go"}}},
		{"bare file name in no tree", "[ev: BEvaporatorUnit.java:1224]", []want{{MissingFile, "BEvaporatorUnit.java"}}},
	})
}
