package plan_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// failingLoad is a client whose every load fails with err.
type failingLoad struct{ err error }

func (f failingLoad) LoadTree(string, int, box.LoadOptions) (*pyjson.Object, error) {
	return nil, f.err
}
func (failingLoad) SSC(string, any) (any, error) { return nil, nil }
func (failingLoad) Sync(any) (any, error)        { return nil, nil }
func (failingLoad) ClearHandles()                {}

const underProgram = "station:|slot:/Drivers/BacnetNetwork/$32012/ControlProgram/Calc"

// Retro mcpn4-spyder-commissioning-over-box Δ4: a Spyder's ControlProgram unloads on its own and the
// station then answers a path below it with the bare path as the error; a write under it names that.
func TestAPathUnderASpyderControlProgramNamesTheUnloadedProgram(t *testing.T) {
	station := &box.Error{Kind: box.KindBox, Msg: underProgram, Station: true}
	_, _, err := plan.Handle(failingLoad{err: station}, underProgram, 1)
	var be *box.Error
	if !errors.As(err, &be) || !strings.HasPrefix(be.Msg, underProgram) || !be.Station {
		t.Fatalf("the station error is not kept: %#v", err)
	}
	for _, part := range []string{"ControlProgram", "unloads on its own", "load it again"} {
		if !strings.Contains(be.Msg, part) {
			t.Errorf("missing %q: %s", part, be.Msg)
		}
	}
}

// Any other failing path keeps the station's error as it was.
func TestAFailureElsewhereKeepsTheStationError(t *testing.T) {
	station := &box.Error{Kind: box.KindBox, Msg: "no such component", Station: true}
	for _, ord := range []string{"station:|slot:/Folder/Pump", "station:|slot:/Drivers/ControlProgramX/Pump"} {
		_, _, err := plan.Handle(failingLoad{err: station}, ord, 1)
		if err != error(station) {
			t.Errorf("%s: error = %v, want the station's own", ord, err)
		}
	}
	transport := &box.Error{Kind: box.KindBox, Msg: "connection refused"}
	if _, _, err := plan.Handle(failingLoad{err: transport}, underProgram, 1); err != error(transport) {
		t.Errorf("a transport failure was rewritten: %v", err)
	}
}
