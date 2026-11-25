package writeops

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/capacity"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The refusals of n4_register_bacnet_spyder that come before a token.
const (
	// spyderQuery reads every BACnet device's slot path and instance (columns by position: the
	// header of a dotted select is the station's display text).
	spyderQuery = "select slotPath, config.deviceObject.objectId from bacnet:BacnetDevice"

	msgNetworkOff   = "%s is disabled (enabled false or the disabled status flag): a Spyder registered under it is not polled and its writeDeviceInstance is not answered; enable the network, then redo the dry run"
	msgNetworkFault = "%s is in fault (%s): a Spyder registered under it cannot start; clear the fault, then redo the dry run"
	msgNameInUse    = "a component named %s is already in use under %s (display name %s): the station would rename the new one; " +
		"to look it up with n4_bql_query write the escaped name, name = '%s' ($24 is a literal $)"
	msgInstanceUsed   = "device:%d is already used by %s: two devices at one instance make the station rebind the first one's address on every I-Am (B1165); choose a free instance"
	msgInstanceUnread = "the station's BACnet devices could not be read (%s): a duplicate instance elsewhere on the station cannot be ruled out, " +
		"and writeDeviceInstance has no inverse while two devices at one instance make the station rebind addresses (B1165); " +
		"refusing before the token: fix the read, then redo the dry run"
)

// networkUsable refuses a target that is not a BACnet network, or that is disabled or in fault.
func networkUsable(c plan.Client, network string) error {
	_, nodes, err := plan.Handle(c, network, 1)
	if err != nil {
		return err
	}
	if t := rootType(nodes); t != bacnetNetworkTyp {
		return plan.ToolError(network + " is " + pyjson.Repr(t) + ", not a " + bacnetNetworkTyp)
	}
	if disabledSlots(nodes) {
		return plan.ToolError(fmt.Sprintf(msgNetworkOff, network))
	}
	if cause, faulted := faultOf(nodes); faulted {
		return plan.ToolError(fmt.Sprintf(msgNetworkFault, network, cause))
	}
	return nil
}

// spyderFree refuses a name already under the network and an instance another BACnet device holds
// (one BQL over every device); a lookup that cannot be read is a refusal too: writeDeviceInstance has no inverse.
func spyderFree(c plan.Client, d createData, a *spyderArgs, stored string) error {
	if d.children[stored] {
		return plan.ToolError(fmt.Sprintf(msgNameInUse, stored, a.network, pyjson.StrRepr(a.display),
			strings.ReplaceAll(stored, "$", "$24")))
	}
	g, ok := c.(capacity.OrdGetter)
	if !ok {
		return plan.ToolError(fmt.Sprintf(msgInstanceUnread, "this client cannot run BQL"))
	}
	rows, why := spyderRows(g)
	if why != "" {
		return plan.ToolError(fmt.Sprintf(msgInstanceUnread, why))
	}
	want := fmt.Sprintf("device:%d", a.instance)
	for _, r := range rows {
		if r[1] == want {
			return plan.ToolError(fmt.Sprintf(msgInstanceUsed, a.instance, r[0]))
		}
	}
	return nil
}

// spyderRows is each BACnet device as {slot path, objectId}, or why they could not be read.
func spyderRows(g capacity.OrdGetter) ([][2]string, string) {
	ord, err := bql.ComposeOrd("station:|slot:/", spyderQuery)
	if err != nil {
		return nil, err.Error()
	}
	text, cut, err := g.GetOrd(ord, licenceTimeout, bql.MaxResponseBytes)
	if err != nil {
		return nil, err.Error()
	}
	columns, rows, capped, err := bql.ParseCSV(text, bql.MaxRowsLimit, nil)
	switch {
	case cut:
		return nil, fmt.Sprintf("the answer was cut at %d bytes", bql.MaxResponseBytes)
	case err != nil:
		return nil, "the answer could not be parsed: " + err.Error()
	case capped:
		return nil, fmt.Sprintf("the answer has more than %d rows", bql.MaxRowsLimit)
	case len(columns) < 2:
		return nil, "the answer has no two columns"
	}
	out := make([][2]string, 0, len(rows))
	for _, row := range rows {
		cell := func(i int) string { v, _ := row.Get(columns[i]); return strings.TrimSpace(pyjson.Str(v)) }
		out = append(out, [2]string{cell(0), cell(1)})
	}
	return out, ""
}
