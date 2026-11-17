package writeops

import "testing"

// A rollback of a rename reads restored only when the old name is back and the new one gone.
func TestRenamedBackChecksBothNames(t *testing.T) {
	op := obj("nm", "r", "h", "10", "o", "Power", "n", "Numeric$20Writable40001")
	for _, tc := range []struct {
		names []string
		want  bool
	}{
		{[]string{"Numeric$20Writable40001"}, true},
		{[]string{"Power"}, false},
		{[]string{"Power", "Numeric$20Writable40001"}, false},
	} {
		kv := []any{"", obj("h", "10")}
		for _, n := range tc.names {
			kv = append(kv, n, obj("h", "40"))
		}
		got, err := renamedBack(obj(kv...), op)
		if err != nil || got != tc.want {
			t.Errorf("%v: got %v, %v; want %v", tc.names, got, err, tc.want)
		}
	}
}
