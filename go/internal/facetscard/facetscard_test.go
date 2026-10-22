package facetscard

import (
	"reflect"
	"strings"
	"testing"
)

// cardDoc is a minimal doc whose card section holds rows, followed by the `**Not` stop line and a
// later section whose row must be ignored (the bats card_with helper).
func cardDoc(rows ...string) string {
	lines := append([]string{"# doc", "## BFacets key reference card", "", "| Constant | Key | Value | Purpose |", "|---|---|---|---|"}, rows...)
	lines = append(lines, "", "**Not `BFacets` keys**: `nullText`", "", "## Next section", "| `NOT_IN_CARD` | `\"later\"` | x | y |")
	return strings.Join(lines, "\n") + "\n"
}

func row(name, key string) string { return "| `" + name + "` | `\"" + key + "\"` | a | b |" }

func TestCardKeys(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, text string
		want       []string
	}{
		{"rows in order, rows after the section ignored", cardDoc(row("MIN", "min"), row("NULL_TEXT", "nullText")),
			[]string{"MIN=min", "NULL_TEXT=nullText"}},
		{"no rows", cardDoc(), nil},
		{"empty file", "", nil},
		{"no card heading", "| `MIN` | `\"min\"` | a |\n", nil},
		{"a later ## heading ends the section", "## BFacets key reference card\n" + row("MIN", "min") + "\n## Other\n" + row("LATE", "late") + "\n",
			[]string{"MIN=min"}},
		{"a repeated card heading (prefix match) keeps the section open",
			"## BFacets key reference card (4.15)\n| `MIN` | `\"min\"` | x |\n## BFacets key reference card\n| `MAX` | `\"max\"` |\n",
			[]string{"MIN=min", "MAX=max"}},
		{"malformed rows are skipped",
			"## BFacets key reference card\n|`BAD`|`\"bad\"`|\n| `lower` | `\"x\"` |\n| `NUM` | `\"n1\"` |\n| `OK` | `\"ok\"`|\n| `A` | `\"a\"` |\n",
			[]string{"A=a"}},
		{"CRLF lines and no final newline", "## BFacets key reference card\r\n| `MIN` | `\"min\"` | a |\r\n| `MAX` | `\"max\"` | b |",
			[]string{"MIN=min", "MAX=max"}},
		{"rows before the heading are ignored", row("EARLY", "early") + "\n## BFacets key reference card\n" + row("MIN", "min") + "\n",
			[]string{"MIN=min"}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := CardKeys(tt.text); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CardKeys = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSourceKeys(t *testing.T) {
	t.Parallel()
	java := "public final class BFacets {\n  public static final String MIN = \"min\";\n" +
		"  public static final String ORD_RELATIVIZE= \"ordRelativize\";\n  private static final String[] NUMERIC_KEYS = { MIN };\n}\n"
	cases := []struct {
		name, path, text string
		want             []string
	}{
		{"java String key constants, private arrays ignored", "BFacets.java", java, []string{"MIN=min", "ORD_RELATIVIZE=ordRelativize"}},
		{"java spacing, trailing text, non-matching shapes",
			"x/Odd.java",
			"public static final String MIN = \"min\"; // c\n\tpublic\tstatic  final String MAX=\"max\";x\n" +
				"public static final String LOW = \"a1\";\nstatic final String NOPE = \"nope\";\npublic static final String  B = \"b\" ;\n",
			[]string{"MIN=min", "MAX=max"}},
		{"java last line without a newline is read", "NoEol.java",
			"  public static final String MIN = \"min\";\n  public static final String MAX = \"max\";", []string{"MIN=min", "MAX=max"}},
		{"keys file: comments and malformed lines skipped", "k.keys",
			"# comment MIN=min\nMIN=min\n MAX=max\nMAX=max \nMAX=max\nbad=x\nNUM=n1\nX_Y=\n", []string{"MIN=min", "MAX=max"}},
		{"keys file: duplicates kept", "k.keys", "MIN=min\nMIN=min\nMAX=max\n", []string{"MIN=min", "MIN=min", "MAX=max"}},
		{"keys file: CRLF lines do not match", "k.keys", "MIN=min\r\nMAX=max\r\n", nil},
		{"java-looking lines in a non-.java file do not match", "javalike.txt", "public static final String MIN = \"min\";\n", nil},
		{"empty", "k.keys", "", nil},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := SourceKeys(tt.path, tt.text); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SourceKeys = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCompare(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name            string
		card, src, want []string
	}{
		{"match", []string{"MIN=min", "MAX=max"}, []string{"MIN=min", "MAX=max"}, nil},
		{"missing", []string{"MIN=min"}, []string{"MIN=min", "MAX=max"}, []string{"MISSING MAX=max"}},
		{"extra", []string{"MIN=min", "NULL_TEXT=nullText"}, []string{"MIN=min"}, []string{"EXTRA NULL_TEXT=nullText"}},
		{"missing then extra, no order check", []string{"MIN=min", "ORD=ord"}, []string{"MIN=min", "MAX=max"},
			[]string{"MISSING MAX=max", "EXTRA ORD=ord"}},
		{"order", []string{"MAX=max", "MIN=min"}, []string{"MIN=min", "MAX=max"},
			[]string{"ORDER card #1 MAX=max != source MIN=min", "ORDER card #2 MIN=min != source MAX=max"}},
		{"order skips matching positions", []string{"C=c", "B=b", "A=a"}, []string{"A=a", "B=b", "C=c"},
			[]string{"ORDER card #1 C=c != source A=a", "ORDER card #3 A=a != source C=c"}},
		{"duplicates compare only the shorter length", []string{"MIN=min", "MIN=min", "MAX=max"}, []string{"MIN=min", "MAX=max"},
			[]string{"ORDER card #2 MIN=min != source MAX=max"}},
		{"equal sets with a duplicate match", []string{"MIN=min", "MAX=max"}, []string{"MIN=min", "MAX=max", "MAX=max"}, nil},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Compare(tt.card, tt.src); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Compare = %q, want %q", got, tt.want)
			}
		})
	}
}
