package writeops

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// What the read-back of ping reads (#931, retro R12 delta M8). ping asks the driver to check a
// device (a network: every device under it); the station answers nothing, so the only proof is the
// health the check moves: BDevice.health is a BPingHealth with lastOkTime, lastFailTime and
// lastFailCause, and the device status holds down (a failed check) or disabled (the device, or its
// network, is off: the driver then does not check it). A device or a point device ext is read from
// its component; a network by one BQL over its devices.

const (
	pingDevice  = "device"
	pingNetwork = "network"
	// pingQuery reads each device under the network ORD; status.bits is the status as a number, which
	// no locale translates; the columns are read by position.
	pingQuery   = "select slotPath, status.bits, health.lastOkTime, health.lastFailTime, health.lastFailCause from driver:Device"
	pingTimeout = 60
)

var pingFields = []string{"path", "bits", "last_ok", "last_fail", "last_fail_cause"}

// pingRow is one device as the read-back sees it; the times and the cause are text or nil.
type pingRow struct {
	path                    string
	status                  []string
	lastOk, lastFail, cause any
}

// pingRead is the health read of the target at one moment: its rows, whether its network is
// disabled, or why nothing could be read.
type pingRead struct {
	kind            string
	rows            []pingRow
	networkDisabled bool
	err             string
}

// pingKind is network for a driver network, device for a device or a point device ext.
func pingKind(nodes *pyjson.Object) string {
	if strings.HasSuffix(rootType(nodes), "Network") {
		return pingNetwork
	}
	return pingDevice
}

// readPing reads the health of the ping target at ordText from its depth-2 load.
func readPing(c plan.Client, ordText string, nodes *pyjson.Object) *pingRead {
	r := &pingRead{kind: pingKind(nodes)}
	if r.kind == pingNetwork {
		r.networkDisabled = disabledSlots(nodes)
		r.rows, r.err = pingNetworkRows(c, ordText)
		return r
	}
	device := ordText
	if strings.HasSuffix(rootType(nodes), "DeviceExt") { // the ext checks its device, its parent
		device = parentOrd(ordText)
		if device == "" {
			r.err = "the device ext has no parent device"
			return r
		}
		if _, n, err := plan.Handle(c, device, 2); err != nil {
			r.err = err.Error()
			return r
		} else {
			nodes = n
		}
	}
	row, err := pingDeviceRow(device, nodes)
	if err != nil {
		r.err = err.Error()
		return r
	}
	r.rows = []pingRow{row}
	if parent := parentOrd(device); parent != "" { // best effort: a parent that cannot be read is no network
		if _, pn, err := plan.Handle(c, parent, 1); err == nil {
			r.networkDisabled = disabledSlots(pn)
		}
	}
	return r
}

// disabledSlots is whether the loaded component is off: enabled false or the disabled status flag.
func disabledSlots(nodes *pyjson.Object) bool {
	kids := box.Children(nodes, "")
	if v := box.ChildValue(kids, "enabled", nil); v != nil && strings.EqualFold(pyjson.Str(v), "false") {
		return true
	}
	flags, _ := box.StatusFlags(box.ChildValue(kids, "status", "0"))
	return hasFlag(flags, "disabled")
}

func hasFlag(flags []string, name string) bool {
	for _, f := range flags {
		if f == name {
			return true
		}
	}
	return false
}

// parentOrd is the ORD of the component above ordText ("" at a root slot).
func parentOrd(ordText string) string {
	const root = "station:|slot:/"
	i := strings.LastIndexByte(ordText, '/')
	if !strings.HasPrefix(ordText, root) || i < len(root) {
		return ""
	}
	return ordText[:i]
}

// pingDeviceRow is the device's row from its load: its status and health slots.
func pingDeviceRow(ordText string, nodes *pyjson.Object) (pingRow, error) {
	if box.Node(nodes, "health") == nil {
		return pingRow{}, fmt.Errorf("%s has no health slot", ordText)
	}
	flags, err := box.StatusFlags(box.ChildValue(box.Children(nodes, ""), "status", "0"))
	if err != nil {
		return pingRow{}, err
	}
	kids := box.Children(nodes, "health")
	return pingRow{path: strings.TrimPrefix(ordText, "station:|"), status: flags,
		lastOk: cellValue(box.ChildValue(kids, "lastOkTime", nil)), lastFail: cellValue(box.ChildValue(kids, "lastFailTime", nil)),
		cause: cellValue(box.ChildValue(kids, "lastFailCause", nil))}, nil
}

// cellValue is a slot or BQL cell as text; an absent, empty or "null" cell is nil.
func cellValue(v any) any {
	s := strings.TrimSpace(bql.StripControl(pyjson.Str(v)))
	if v == nil || s == "" || s == "null" {
		return nil
	}
	return s
}

// pingNetworkRows reads every device under the network ORD with one BQL.
func pingNetworkRows(c plan.Client, ordText string) ([]pingRow, string) {
	g, ok := c.(ordGetter)
	if !ok {
		return nil, "this client cannot run BQL"
	}
	ord, err := bql.ComposeOrd(ordText, pingQuery)
	if err != nil {
		return nil, err.Error()
	}
	text, cut, err := g.GetOrd(ord, pingTimeout, bql.MaxResponseBytes)
	if err != nil {
		return nil, err.Error()
	}
	if cut {
		text = bql.CompleteRecords(text)
	}
	columns, table, _, err := bql.ParseCSV(text, bql.MaxRowsLimit, nil)
	if err == nil {
		table, err = bql.ByPosition(columns, table, pingFields)
	}
	if err != nil {
		return nil, err.Error()
	}
	rows := []pingRow{}
	for _, t := range table {
		row := pingRow{path: pyjson.Str(get(t, "path")), status: []string{}, lastOk: cellValue(get(t, "last_ok")),
			lastFail: cellValue(get(t, "last_fail")), cause: cellValue(get(t, "last_fail_cause"))}
		if bits, ok := new(big.Int).SetString(strings.TrimSpace(pyjson.Str(get(t, "bits"))), 10); ok {
			row.status, _ = box.StatusFlags(bits.Text(16))
		}
		rows = append(rows, row)
	}
	return rows, ""
}
