package commissioningverify

import (
	"reflect"
	"testing"
)

func TestOwedRows(t *testing.T) {
	t.Parallel()
	const h = "| Slot | Owed by | Unit | Safe default | Status |\n|---|---|---|---|---|\n"
	for _, c := range []struct {
		name, text string
		want       []string
	}{
		{"provided", h + "| `a` | x | u | 0 | Filled |\n", nil},
		{"owed", h + "| `a` | x | u | 0 | owed |\n", []string{"a: owed by x, unit u, still at safe default 0"}},
		{"short", h + "| a | x |\n", []string{"malformed values-owed row (needs Slot | Owed by | Unit | Safe default | Status): | a | x |"}},
		{"foreign", h + "\n| Date | Note |\n|:--|--:|\n| d | n |\n", nil},
		{"lone-eof", h + "\n| b | x | u | 1 | owed |", []string{"b: owed by x, unit u, still at safe default 1"}},
		{"no-table", "| Date | Note |\n|---|---|\n", []string{"no values-owed table (header Slot | Owed by | Unit | Safe default | Status) in f.md"}},
		{"prose", "none owed\n", nil},
	} {
		if got := owedRows("f.md", c.text); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}
