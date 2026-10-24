package citeparse

import (
	"reflect"
	"testing"
)

// TestCardKinds pins the I4 kinds of kit issue #418: course, bitácora, kit paths, external retros, module
// file references and memories named before the word.
func TestCardKinds(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, text string
		want       []full
	}{
		{"course pdf page", "[ev: course Labs/Lab Instructions.pdf p81]", []full{{1, Course, "Labs/Lab Instructions.pdf", "", "", "p81"}}},
		{"course pdf page range", "[ev: course Slides/9 Multithreaded Development.pdf p26-28]",
			[]full{{1, Course, "Slides/9 Multithreaded Development.pdf", "", "", "p26-28"}}},
		{"course file lines", "[ev: course Slide Code/gradle.properties:14-25]", []full{{1, Course, "Slide Code/gradle.properties", "", "", ":14-25"}}},
		{"course directory with prose", "[ev: course Review Assignment 2 build]", []full{{1, Course, "Review Assignment 2 build", "", "", ""}}},
		{"bitácora section", "[ev: bitácora 5cuartos §11]", []full{{1, Bitacora, "5cuartos", "", "", "§11"}}},
		{"bitacora section range", "[ev: bitacora 5cuartos §7-§8]", []full{{1, Bitacora, "5cuartos", "", "", "§7 §8"}}},
		{"kit path", "[ev: kit types/logic-authoring.md:62-70]", []full{{1, Path, "KIT/types/logic-authoring.md", "", "", ":62-70"}}},
		{"kit cross-reference", "[ev: kit BUILD-LOOP.md §4.b]", []full{{1, Path, "KIT/BUILD-LOOP.md", "", "", "§4.b"}}},
		{"kit retro", "[ev: kit retro campaign8-slot-per-slot (A1/B792)]", []full{{1, Retro, "campaign8-slot-per-slot", "", "", ""}}},
		{"kit commit is unverifiable", "[ev: kit 3793ff2]", []full{{1, Unverifiable, "3793ff2", "", "", ""}}},
		{"external retro by path", "[ev: external retro niagara-research/retros/2026-10-06-p-hmi.md Δ1, Δ4]",
			[]full{{1, Retro, "2026-10-06-p-hmi", "", "", "Δ1"}, {1, Retro, "2026-10-06-p-hmi", "", "", "Δ4"}}},
		{"module.xml of a module part", "[ev: cloudLink-rt module.xml dep `cloudLink-rt`]", []full{{1, Code, "cloudLink-rt", "module.xml", "", ""}}},
		{"module.xml of a lowercase module", "[ev: nmodsreflow module.xml]", []full{{1, Code, "nmodsreflow", "module.xml", "", ""}}},
		{"client module code", "[ev: Apillm BApillmImporterManager.java rowFor()]", []full{{1, Code, "Apillm", "BApillmImporterManager.java", "", ""}}},
		{"docSource path", "[ev: docSource baja/javax/baja/sys/Sys.java:443-446]", []full{{1, Code, "docSource", "baja/javax/baja/sys/Sys.java", "", ":443-446"}}},
		{"a lowercase word before a file is not a module", "[ev: see foo.java]", []full{{1, Unparsed, "see foo.java", "", "", ""}}},
		{"vineflower path with no module, spaced lines", "[ev: vineflower annotation/Requires.java :11-19]",
			[]full{{1, Vineflower, "", "annotation/Requires.java", "", ":11-19"}}},
		{"memory named before the word", "[ev: client-module-version-key memory]", []full{{1, Mem, "client-module-version-key", "", "", ""}}},
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
