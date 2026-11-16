package cisteps

import (
	"reflect"
	"testing"
)

func TestSourceRefsReadTheLiteralTailAfterTheLastExpansion(t *testing.T) {
	script := "#!/bin/bash\n" +
		". \"$GO_DIR/go-tool.sh\" || refuse \"cannot load $GO_DIR/go-tool.sh\"\n" +
		"  . \"$ROOT/scripts/lib/sha256.sh\" || exit 3\n" +
		"source \"${HERE}/lib/a.sh\"; echo done\n" +
		". \"$BATS_TEST_DIRNAME/../go/go-tool.sh\"\n" +
		". ./lib/plain.sh\n" +
		"# shellcheck source=/dev/null\n" +
		". \"$1\"\n" +
		"# shellcheck source=SCRIPTDIR/named.sh\n" +
		". \"$x\"\n" +
		"echo \". not/a/source.sh\"\n" +
		"cat > f <<'SH'\n" +
		". \"${BASH_SOURCE%/*}/b.sh\"\n" +
		"SH\n"
	got := sourceRefs("tests/s.bats", []byte(script))
	want := []sourceRef{
		{2, "go-tool.sh", false}, {3, "scripts/lib/sha256.sh", false}, {4, "lib/a.sh", false},
		{5, "go/go-tool.sh", false}, {6, "lib/plain.sh", false}, {10, "tests/named.sh", true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("refs\n%v\nwant\n%v", got, want)
	}
}

func TestSourceRefsMarkAPathTheyCannotReadAsUnresolved(t *testing.T) {
	for _, line := range []string{`. "$1"`, `. "$f"`, "source `pick`", `. "$(cd "$d" && pwd)"`, `. "${LIB}"`} {
		got := sourceRefs("s.sh", []byte(line+"\n"))
		if len(got) != 1 || got[0].tail != "" {
			t.Errorf("%s: refs %v, want one unresolved", line, got)
		}
	}
}

func TestShellcheckClosureFollowsSourcesBothWaysAndFallsBackOnAnUnresolvedOne(t *testing.T) {
	files := map[string]string{
		"scripts/run.sh":          ". \"$ROOT/scripts/lib/a.sh\"\n",
		"scripts/lib/a.sh":        ". \"$ROOT/go/go-tool.sh\"\n",
		"go/go-tool.sh":           "go_tool_bin() { :; }\n",
		"scripts/other.sh":        "echo unrelated\n",
		"tools/wrap.sh":           ". \"$GO_DIR/go-tool.sh\"\n",
		"scripts/dynamic.sh":      ". \"$1\"\n",
		"scripts/uses-dynamic.sh": ". \"$ROOT/scripts/dynamic.sh\"\n",
		"scripts/missing-dep.sh":  ". \"$ROOT/scripts/lib/gone.sh\"\n",
	}
	set := []string{"go/go-tool.sh", "scripts/dynamic.sh", "scripts/lib/a.sh", "scripts/missing-dep.sh",
		"scripts/other.sh", "scripts/run.sh", "scripts/uses-dynamic.sh", "tools/wrap.sh"}
	read := func(p string) ([]byte, error) { return []byte(files[p]), nil }
	for _, c := range []struct {
		changed []string
		want    []string
		reason  string
	}{
		// forward: a changed script takes what it sources, transitively
		{[]string{"scripts/run.sh"}, []string{"go/go-tool.sh", "scripts/lib/a.sh", "scripts/run.sh"}, ""},
		// backward: a changed lib takes its sourcers (and what they source)
		{[]string{"go/go-tool.sh"}, []string{"go/go-tool.sh", "scripts/lib/a.sh", "scripts/run.sh", "tools/wrap.sh"}, ""},
		{[]string{"scripts/other.sh"}, []string{"scripts/other.sh"}, ""},
		{[]string{"scripts/uses-dynamic.sh"}, nil, "scripts/dynamic.sh:1"},
		{[]string{"scripts/missing-dep.sh"}, nil, "scripts/missing-dep.sh:1"},
	} {
		got, reason, err := shellcheckClosure(c.changed, set, read)
		if err != nil || reason != c.reason || (c.reason == "" && !reflect.DeepEqual(got, c.want)) {
			t.Errorf("%v: closure %v reason %q err %v, want %v reason %q", c.changed, got, reason, err, c.want, c.reason)
		}
	}
}
