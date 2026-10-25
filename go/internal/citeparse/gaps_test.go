package citeparse

import (
	"reflect"
	"testing"
)

// TestCardPathGaps pins the I3 forms of kit issue #417 the parser did not read.
func TestCardPathGaps(t *testing.T) {
	t.Parallel()
	const legend = "`R=organized/sT/sT-rt` and `V=R/vineflower`, under niagara-research.\n"
	cases := []struct {
		name, text string
		want       []full
	}{
		{"bare file name", "[ev: Flags.java CONFIRM_REQUIRED]", []full{{1, Path, "Flags.java", "", "", "CONFIRM_REQUIRED"}}},
		{"partial package path", "[ev: comm/NCommTimer.java:162]", []full{{1, Path, "comm/NCommTimer.java", "", "", ":162"}}},
		{"an alias defined by another alias", legend + "[CERT] `V/javax/G.java:75-81`",
			[]full{{2, Path, "organized/sT/sT-rt/vineflower/javax/G.java", "", "", ":75-81"}}},
		{"a niagara-help legend alias", "guides (GD =\n  niagara-help 4.15.6.20 `guides-clean/`) `GD/Graphics/aPx.txt:18`",
			[]full{{2, Path, "niagara-help/versions/4.15.6.20/guides-clean/Graphics/aPx.txt", "", "", ":18"}}},
		{"ASCII ellipsis prefix", "[ev: .../rollup/BR.java:12-31]", []full{{1, Path, "…/rollup/BR.java", "", "", ":12-31"}}},
		{"backticked file of a module part", "[ev: code alarm-rt `BAlarmService.java:168` escalateAlarms]",
			[]full{{1, Code, "alarm-rt", "BAlarmService.java", "", ":168"}}},
		{"a module part's named copy", "[ev: code basicDriver-rt vineflower `BBasicNetwork.java:30-66`]",
			[]full{{1, Vineflower, "basicDriver-rt", "BBasicNetwork.java", "", ":30-66"}}},
		{"prose words are notes, not tokens", "[ev: code cloudLink-rt module.xml types list]",
			[]full{{1, Code, "cloudLink-rt", "module.xml", "", ""}}},
		{"tokens stop at the first prose word", "[ev: code cloudLink-rt module.palette SmartTagDictionary entry with tags]",
			[]full{{1, Code, "cloudLink-rt", "module.palette", "", "SmartTagDictionary"}}},
		{"a capitalised prose word is a note", "[ev: CLAUDE.md Model Assignments]", []full{{1, Path, "CLAUDE.md", "", "", ""}}},
		{"a comma after a listed file", "[ev: code basicDriver-rt vineflower `BBasicNetwork.java:30-66,240-278`, `util/B.java:17`]",
			[]full{{1, Vineflower, "basicDriver-rt", "BBasicNetwork.java", "", ":30-66,240-278"}}},
		{"more ranges of the same file", "[ev: code Tuning.java:332-333, :549-552, :629]", []full{{1, Code, "", "Tuning.java", "", ":332-333,549-552,629"}}},
		{"a comma after a devguide file", "[ev: devguide security/security.txt:97, :220-222]", []full{{1, Devguide, "security/security.txt", "", "", ":97,220-222"}}},
		{"`lines a-b` after a file is a line range", "[ev: code themeDistech-ux extracted/hx/theme.css lines 4-16]",
			[]full{{1, Code, "themeDistech-ux", "extracted/hx/theme.css", "", ":4-16"}}},
		{"a section range is its two ends", "`types/a.md` § 11.1-11.3, x", []full{{1, Path, "types/a.md", "", "", "§11.1 §11.3"}}},
		{"a backticked path with spaces", "see `Dir with space/file.md:3` x", []full{{1, Path, "Dir with space/file.md", "", "", ":3"}}},
		{"a label before a path is not one spaced path", "`code x-rt a/B.java:3` and `FMT …/y.sh:1`", nil},
		{"a list of sections", "`types/a.md` § 3, § 5.2, § 7 list", []full{{1, Path, "types/a.md", "", "", "§3 §5.2 §7"}}},
		{"a §§ list of sections", "`types/a.md` §§ 3, 5 list", []full{{1, Path, "types/a.md", "", "", "§3 §5"}}},
		{"a list of sections in a tag", "[ev: types/a.md § 3, § 5]", []full{{1, Path, "types/a.md", "", "", "§3 §5"}}},
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
