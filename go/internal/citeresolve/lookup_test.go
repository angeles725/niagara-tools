package citeresolve

import "testing"

// TestSearchedPaths pins the I3 path-search gaps of kit issue #417: bare base names, partial package paths,
// every copy of a file, and bold-bullet anchors.
func TestSearchedPaths(t *testing.T) {
	t.Parallel()
	check(t, []struct {
		name, text string
		want       []want
	}{
		{"bare file name unique by base name", "[ev: BFoo.java:4]", []want{{Resolved, "read organized/kitControl/kitControl-rt/vineflower"}}},
		{"bare file name in two module parts is ambiguous", "[ev: Dup.java]", []want{{Ambiguous, "organized/alpha/alpha-rt/vineflower/com/a/Dup.java, organized/beta/beta-rt/vineflower/com/b/Dup.java"}}},
		{"bare file name nowhere", "[ev: BNowhere.java:1]", []want{{MissingFile, "BNowhere.java"}}},
		{"partial package path by suffix", "[ev: util/BPollScheduler.java:5]", []want{{Resolved, "vine BPollScheduler line 5"}}},
		{"a range past the first copy is checked in the next", "[ev: util/BPollScheduler.java:25]", []want{{Resolved, "source BPollScheduler line 25"}}},
		{"a range past every copy names the copies", "[ev: util/BPollScheduler.java:31]", []want{{LineOutOfRange, "4 copies"}}},
		{"the docSource copy is a copy of the same file", "[ev: util/BPollScheduler.java:15]", []want{{Resolved, "docSource BPollScheduler line 15"}}},
		{"package path checked in every copy", "[ev: javax/baja/driver/util/BPollScheduler.java:11]", []want{{Resolved, "decompiled BPollScheduler line 11"}}},
		{"ASCII ellipsis searched by suffix", "[ev: .../util/BPollScheduler.java:2]", []want{{Resolved, "vine BPollScheduler line 2"}}},
		{"infix ellipsis checks every copy", "[ev: driver-rt/…/util/BPollScheduler.java:29]", []want{{Resolved, "source BPollScheduler line 29"}}},
		{"client worktrees are copies of one file", "[ev: BPan.java:5]", []want{{Resolved, "// pan 5"}}},
		{"client file by base name", "[ev: BShop.java:8]", []want{{Resolved, "read Cliente/Shop/Shop-rt/src/com/s/BShop.java"}}},
		{"code in a module named without its part", "[ev: code kitControl BFoo.java:10]", []want{{Resolved, "// BFoo line 10"}}},
		{"code module.xml in two parts is ambiguous", "[ev: code gamma module.xml]", []want{{Ambiguous, "organized/gamma/gamma-rt/extracted/META-INF/module.xml"}}},
		{"code in a client module", "[ev: code Shop-rt BShop.java:2]", []want{{Resolved, "shop line 2"}}},
		{"code backticked file in a part", "[ev: code kitControl-rt `BFoo.java:3`]", []want{{Resolved, "// BFoo line 3"}}},
		{"a module part's vineflower copy", "[ev: code driver-rt vineflower `util/BPollScheduler.java:9`]", []want{{Resolved, "vine BPollScheduler line 9"}}},
		{"KIT prefix is the kit root", "[ev: KIT/types/platform.md § 1]", []want{{Resolved, "## 1. Profiles"}}},
		{"bold bullet anchor", "`types/platform.md` § BQL case rules", []want{{Resolved, "BQL case rules"}}},
	})
}
