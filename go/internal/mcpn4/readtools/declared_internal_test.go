package readtools

// A declared-default lookup that fails keeps the notes the listing gathered before it (review of
// #678); ctfLoader is the registry of contracts_internal_test.go.

import (
	"errors"
	"strings"
	"testing"
)

func TestAResolveFailureKeepsTheNotesAlreadyGathered(t *testing.T) {
	dash := obj("a:Dash", obj("p", "a:Base", "c", []any{obj("st", "p", "n", "setpoint",
		"v", obj("nm", "p", "t", "baja:Double", "v", "0.0"))}))
	f := &ctfLoader{answer: func(n int, _ []string) (any, error) {
		if n > 2 { // the contracts were read; the default lookup fails
			return nil, errors.New("registry gone")
		}
		return dash, nil
	}}
	var gone undescribed
	d := fillDefaults(f, f, &gone, obj("", obj("t", "a:Dash")), []string{""}, func(s string) string { return s })
	note := d.noteText()
	for _, want := range []string{"did not describe type a:Base", "registry gone"} {
		if !strings.Contains(note, want) {
			t.Errorf("note %q lacks %q", note, want)
		}
	}
}
