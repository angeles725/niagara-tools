package citeresolve

import "testing"

func TestPathPointers(t *testing.T) {
	t.Parallel()
	const vAlias = "Code: `V=organized/baja/baja/vineflower/com/tridium` (4.14).\n"
	check(t, []struct {
		name, text string
		want       []want
	}{
		{"alias in prose", vAlias + "FINE (`V/sys/Nre.java:730`).", []want{{Resolved, "// Nre line 730"}}},
		{"ellipsis resolves against an earlier hit of the card",
			"[ev: niagara-help/versions/4.15.6.20/guides-clean/Platform/aSoftwareManager.txt:1] [ev: …/Platform/aSoftwareManager.txt:2]",
			[]want{{Resolved, "guide line 1"}, {Resolved, "guide line 2"}}},
		{"ellipsis with no earlier hit", "[ev: …/Platform/aSoftwareManager.txt:2]", []want{{MissingFile, "ellipsis"}}},
		{"kit cross-reference by number", "`types/platform.md` § 2, so", []want{{Resolved, "## 2. Software Manager"}}},
		{"kit cross-reference by title", "(`METHODOLOGY.md` § Domain correctness) does", []want{{Resolved, "## Domain correctness"}}},
		{"kit cross-reference to a missing heading", "`types/platform.md` § 9, so", []want{{MissingSection, "§9"}}},
		{"repo test name", "[ev: tests/awk.bats AWB1]", []want{{Resolved, "@test \"AWB1"}}},
		{"repo test name missing", "[ev: tests/awk.bats AWB9]", []want{{MissingSection, "AWB9"}}},
		{"client module path", "[ev: Cliente/Pan/BPan.java:5]", []want{{Resolved, "// pan 5"}}},
		{"missing path", "[ev: go/internal/nope/x.go:1]", []want{{MissingFile, "go/internal/nope/x.go"}}},
		{"bare file name with no root", "[ev: BEvaporatorUnit.java:1224]", []want{{Unparsed, "bare file name"}}},
	})
}
