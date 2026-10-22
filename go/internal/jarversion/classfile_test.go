package jarversion

import (
	"reflect"
	"strconv"
	"testing"
)

func TestMajor(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		head []byte
		want int
	}{
		{"Java 8 header", []byte{0xca, 0xfe, 0xba, 0xbe, 0, 0, 0, 52}, 52},
		{"Java 9 header", []byte{0xca, 0xfe, 0xba, 0xbe, 0, 0, 0, 53}, 53},
		{"two-byte major", []byte{0xca, 0xfe, 0xba, 0xbe, 0, 0, 1, 2}, 258},
		{"magic is not checked (bash read bytes 6-7 only)", []byte{0, 0, 0, 0, 0, 0, 0, 61}, 61},
		{"bytes past the eighth are ignored", []byte{0xca, 0xfe, 0xba, 0xbe, 0, 0, 0, 52, 0xff}, 52},
		{"seven bytes: od printed one value, the seventh byte", []byte{0, 0, 0, 0, 0, 0, 53}, 53},
		{"six bytes: od could not skip, no version", []byte{0, 0, 0, 0, 0, 0}, -1},
		{"empty entry", nil, -1},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Major(tt.head); got != tt.want {
				t.Errorf("Major(% x) = %d, want %d", tt.head, got, tt.want)
			}
		})
	}
}

func TestAboveJava8(t *testing.T) {
	t.Parallel()
	// Mutation: BJCV2 -- dropping the > 52 comparison makes every bundled Java 9+ class pass.
	for major, want := range map[int]bool{-1: false, 0: false, 51: false, 52: false, 53: true, 65: true, 258: true} {
		if got := AboveJava8(major); got != want {
			t.Errorf("AboveJava8(%d) = %v, want %v", major, got, want)
		}
	}
}

func TestSample(t *testing.T) {
	t.Parallel()
	many := make([]string, 25)
	for i := range many {
		many[i] = "p/C" + strconv.Itoa(i) + ".class"
	}
	cases := []struct {
		name  string
		names []string
		want  []string
	}{
		{"only names ending in .class, in archive order", []string{"META-INF/MANIFEST.MF", "b/B.class", "a/", "a/A.class", "x.classes", "y.class/"},
			[]string{"b/B.class", "a/A.class"}},
		{"multi-release META-INF/versions/ entries are skipped", []string{"META-INF/versions/9/module-info.class", "META-INF/versions/11/a/A.class", "a/A.class", "x/META-INF/versions/9/B.class"},
			[]string{"a/A.class", "x/META-INF/versions/9/B.class"}},
		{"at most 20 entries", many, many[:20]},
		{"none", []string{"META-INF/MANIFEST.MF"}, nil},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Sample(tt.names); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Sample() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFailRow(t *testing.T) {
	t.Parallel()
	want := "FAIL  lint-bundled-jar-class-version  m/libs/x.jar  class a/B$1.class is major 53 (> 52 = Java 8) — bundled Java 9+ bytecode throws UnsupportedClassVersionError at load\n"
	if got := FailRow("m/libs/x.jar", "a/B$1.class", 53); got != want {
		t.Errorf("FailRow =\n%q\nwant\n%q", got, want)
	}
}
