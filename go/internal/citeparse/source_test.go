package citeparse

import (
	"reflect"
	"testing"
)

// full is the part of a Pointer a source-grammar case pins.
type full struct {
	Line                         int
	Kind                         Kind
	Target, File, Version, Locat string
}

func fulls(ps []Pointer) []full {
	var out []full
	for _, p := range ps {
		out = append(out, full{p.Line, p.Kind, p.Target, p.File, p.Version, p.Locator()})
	}
	return out
}

func TestCardSources(t *testing.T) {
	t.Parallel()
	const vAlias = "Code: `V=organized/baja/baja/vineflower/com/tridium` (4.14).\n"
	cases := []struct {
		name, text string
		want       []full
	}{
		{"code in a module part", "[ev: code kitControl-rt BFoo.java:10-12]", []full{{1, Code, "kitControl-rt", "BFoo.java", "", ":10-12"}}},
		{"code path with no module", "[ev: code javax/baja/sys/BA.java:87-90,148]", []full{{1, Code, "", "javax/baja/sys/BA.java", "", ":87-90,148"}}},
		{"code jar entry is unparsed", "[ev: code devkit-wb.jar!gradle/b.vm:18]", []full{{1, Unparsed, "code devkit-wb.jar!gradle/b.vm:18", "", "", ""}}},
		{"devguide", "[ev: devguide lex_lex.js.txt:441-455]", []full{{1, Devguide, "lex_lex.js.txt", "", "", ":441-455"}}},
		{"guide", "[ev: guide Platform/aSoftwareManager.txt:18]", []full{{1, Guide, "Platform/aSoftwareManager.txt", "", "", ":18"}}},
		{"niagara-help with a version", "[ev: niagara-help 4.15.6.20 guides-clean/T/U.txt:25]", []full{{1, Help, "guides-clean/T/U.txt", "", "4.15.6.20", ":25"}}},
		{"mem and memory", "[ev: mem shared-checkout] [ev: memory a-b.md]", []full{{1, Mem, "shared-checkout", "", "", ""}, {1, Mem, "a-b", "", "", ""}}},
		{"repo path with a test name", "[ev: tests/awk.bats AWB1]", []full{{1, Path, "tests/awk.bats", "", "", "AWB1"}}},
		{"kit file with lines", "[ev: BUILD-LOOP.md:131,171]", []full{{1, Path, "BUILD-LOOP.md", "", "", ":131,171"}}},
		{"kit cross-reference in a tag", "[ev: types/platform.md § 1]", []full{{1, Path, "types/platform.md", "", "", "§1"}}},
		{"ellipsis prefix", "[ev: …/Templates/D.txt:33]", []full{{1, Path, "…/Templates/D.txt", "", "", ":33"}}},
		{"alias in prose", vAlias + "FINE (`V/sys/Nre.java:730`).", []full{{2, Path, "organized/baja/baja/vineflower/com/tridium/sys/Nre.java", "", "", ":730"}}},
		{"directory alias", "Guides: `G/` = `niagara-help/g/Platform/` (x).\n| a | `G/aSoft.txt:18` |", []full{{2, Path, "niagara-help/g/Platform/aSoft.txt", "", "", ":18"}}},
		{"cited-as alias", "(`niagara-help/d/ui/uiChangelog.txt`, cited `CL:` below)\n| [CERT-doc] `CL:297-336` |", []full{{2, Path, "niagara-help/d/ui/uiChangelog.txt", "", "", ":297-336"}}},
		{"inline devguide", "x (devguide `virtualComponents.txt:252-254`).", []full{{1, Devguide, "virtualComponents.txt", "", "", ":252-254"}}},
		{"inline CERT organized path", "[CERT] organized/baja/baja/decompiled/N.java:455 y", []full{{1, Path, "organized/baja/baja/decompiled/N.java", "", "", ":455"}}},
		{"inline vineflower, module first", "vineflower driver-rt 4.14.0.162 `javax/B.java:463-469`", []full{{1, Vineflower, "driver-rt", "javax/B.java", "4.14.0.162", ":463-469"}}},
		{"inline vineflower, version first", "devkit-wb 4.14.0.162 vineflower `com/J.java:270-272`", []full{{1, Vineflower, "devkit-wb", "com/J.java", "4.14.0.162", ":270-272"}}},
		{"inline javap", "javap of `httpClient-rt.jar` 4.15.6.20 shows", []full{{1, Javap, "httpClient-rt.jar", "", "4.15.6.20", ""}}},
		{"inline cross-reference, number", "see `types/observability.md` § 1, and", []full{{1, Path, "types/observability.md", "", "", "§1"}}},
		{"inline cross-reference, title", "(`METHODOLOGY.md` § Domain correctness) does", []full{{1, Path, "METHODOLOGY.md", "", "", "§Domain correctness"}}},
		{"inline cross-reference, quoted title", "`CONTRIBUTING.md` § 2.2 \"Fail closed on I/O\".", []full{{1, Path, "CONTRIBUTING.md", "", "", "§2.2"}}},
		{"a backticked path without lines is not a citation", "run `toolbelt/build.sh` now", nil},
		{"a tag is not read again as prose", "`[ev: code types/x.md:3]`", []full{{1, Code, "", "types/x.md", "", ":3"}}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := fulls(Card(tt.text)); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got  %+v\nwant %+v", got, tt.want)
			}
		})
	}
}
