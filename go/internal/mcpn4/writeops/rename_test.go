package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

const renameParent = net + "/Cuby01/points"

// renameRig is a points folder (handle 10) holding two writables named with a station escape and
// one plain point.
func renameRig() *loads {
	folder := obj("", obj("h", "10", "t", "modbusAsync:ModbusAsyncPointDeviceExt"),
		"Numeric$20Writable40001", obj("h", "40", "t", "control:NumericWritable"),
		"Numeric$20Writable40005", obj("h", "41", "t", "control:NumericWritable"),
		"RoomTemp", obj("h", "42", "t", "control:NumericPoint"))
	return &loads{nodes: obj(renameParent, folder)}
}

func renameItem(oldName, newName string) *pyjson.Object {
	return obj("parent_ord", renameParent, "name", oldName, "new_name", newName)
}

func renameArgs(items ...any) *pyjson.Object { return obj("items", items) }

// One rename is the station's RenameOp {nm: r, h: parent, o: old, n: new}; the inverse renames
// back; a batch keeps the given order and its inverse runs in reverse.
func TestRenamePlansTheRenameOpAndItsInverse(t *testing.T) {
	p, err := writeops.RenameComponent.Plan(renameRig(), renameArgs(
		renameItem("Numeric$20Writable40001", "Power"), renameItem("Numeric$20Writable40005", "Setpoint")))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{`{"nm": "r", "h": "10", "o": "Numeric$20Writable40001", "n": "Power"}`,
		`{"nm": "r", "h": "10", "o": "Numeric$20Writable40005", "n": "Setpoint"}`}
	if len(p.Ops) != 2 || js(t, p.Ops[0]) != want[0] || js(t, p.Ops[1]) != want[1] {
		t.Fatalf("ops = %v", p.Ops)
	}
	inv := []string{`{"nm": "r", "h": "10", "o": "Setpoint", "n": "Numeric$20Writable40005"}`,
		`{"nm": "r", "h": "10", "o": "Power", "n": "Numeric$20Writable40001"}`}
	if len(p.Inverse) != 2 || js(t, p.Inverse[0]) != inv[0] || js(t, p.Inverse[1]) != inv[1] {
		t.Fatalf("inverse = %v", p.Inverse)
	}
	if notes := js(t, p.Notes); !strings.Contains(notes, "Numeric$20Writable40001 -> Power") {
		t.Errorf("notes = %s", notes)
	}
}

// Refused before anything is sent: no items, too many, a missing source, a taken or repeated
// target, a malformed new name, an unchanged name.
func TestRenameRefusals(t *testing.T) {
	many := []any{}
	for i := 0; i < 51; i++ {
		many = append(many, renameItem("RoomTemp", "X"))
	}
	for _, tc := range []struct {
		name string
		args *pyjson.Object
		want string
	}{
		{"no items", renameArgs(), "items needs 1 to 50"},
		{"too many", obj("items", many), "items needs 1 to 50"},
		{"missing source", renameArgs(renameItem("Nope", "Power")), "'Nope' not found under"},
		{"taken target", renameArgs(renameItem("Numeric$20Writable40001", "RoomTemp")), "'RoomTemp' already exists under"},
		{"repeated target", renameArgs(renameItem("Numeric$20Writable40001", "Power"),
			renameItem("Numeric$20Writable40005", "Power")), "'Power' is requested twice under"},
		{"repeated source", renameArgs(renameItem("RoomTemp", "A"), renameItem("RoomTemp", "B")),
			"'RoomTemp' is renamed twice under"},
		{"malformed new name", renameArgs(renameItem("RoomTemp", "Room Temp")), "must be a plain slot name"},
		{"unchanged", renameArgs(renameItem("RoomTemp", "RoomTemp")), "already named 'RoomTemp'"},
	} {
		_, err := writeops.RenameComponent.Plan(renameRig(), tc.args)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v, want %q", tc.name, err, tc.want)
		}
	}
}

// The read-back is verified only when every new name is present and every old name is gone.
func TestRenameReadbackChecksBothNames(t *testing.T) {
	args := renameArgs(renameItem("Numeric$20Writable40001", "Power"))
	p, err := writeops.RenameComponent.Plan(renameRig(), args)
	if err != nil {
		t.Fatal(err)
	}
	after := &loads{nodes: obj(renameParent, obj("", obj("h", "10"),
		"Power", obj("h", "40", "t", "control:NumericWritable"),
		"Numeric$20Writable40005", obj("h", "41")))}
	_, _, _, verdict, err := writeops.RenameComponent.Readback(after, args, p, []any{nil}, p.Inverse)
	if err != nil || verdict != "verified" {
		t.Fatalf("verdict = %q, err = %v", verdict, err)
	}
	_, _, observed, verdict, _ := writeops.RenameComponent.Readback(renameRig(), args, p, []any{nil}, p.Inverse)
	if verdict != "mismatch" || !strings.Contains(js(t, observed), "Numeric$20Writable40001") {
		t.Fatalf("unchanged station: verdict = %q observed = %s", verdict, js(t, observed))
	}
}

// Scope covers every parent named by the items.
func TestRenameScopeIsEveryParent(t *testing.T) {
	got := writeops.RenameComponent.ScopeOrds(renameArgs(renameItem("A", "B"),
		obj("parent_ord", net+"/Cuby03/points", "name", "A", "new_name", "B")))
	if len(got) != 2 || got[0] != renameParent || got[1] != net+"/Cuby03/points" {
		t.Errorf("scope = %v", got)
	}
}
