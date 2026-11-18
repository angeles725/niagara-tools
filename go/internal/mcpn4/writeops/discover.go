package writeops

import (
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// DiscoverDevices is n4_discover_devices: the BACnet device discovery Workbench's Device Manager
// runs, submitted on the station. BBacnetNetwork.submitDeviceManagerJob (hidden, admin invoke)
// takes a bacnet:DeviceDiscoveryConfig and submits a TRANSIENT BacnetDiscoverDevicesJob under
// Services/JobService that sends Who-Is (to the global broadcast address when every network is
// asked, the Workbench "Send Global") and keeps one DiscoveryDevice child per I-Am
// (BBacnetNetwork.java:359-374, BBacnetDiscoverDevicesJob.java:68-149). No device is added, but
// it is a write: field traffic, and an I-Am from a registered instance at another address
// rewrites that device's address (B1165). The read-back reads the job's rows (discover_rows.go).
var DiscoverDevices = plan.Impl{ScopeOrds: func(args *pyjson.Object) []any { return []any{str(args, "network_ord")} },
	PlanWith: discoverPlan, Readback: discoverReadback}

const (
	discoverAction   = "submitDeviceManagerJob"
	discoverMaxInst  = 4194302 // BDeviceDiscoveryConfig deviceHighLimit default: every instance
	discoverMaxNet   = 65534   // BACnet network numbers 1..65534 (65535 is the broadcast network)
	discoverMaxWait  = 60
	discoverDefWait  = 10
	discoverSlack    = 30.0 // seconds past waitResponseTime the read-back waits for the job
	bacnetNetworkTyp = "bacnet:BacnetNetwork"
)

// discoverData is what the read-back needs: the network, the wait and the write state's clock.
type discoverData struct {
	network string
	wait    int
	w       *plan.State
}

func discoverPlan(c plan.Client, args *pyjson.Object, w *plan.State, _ *session.Session) (*plan.Planned, error) {
	low, err := intArg(args, "low_instance", 0, 0, discoverMaxInst)
	if err != nil {
		return nil, err
	}
	high, err := intArg(args, "high_instance", discoverMaxInst, 0, discoverMaxInst)
	if err != nil {
		return nil, err
	}
	if low > high {
		return nil, plan.ToolError("low_instance " + strconv.Itoa(low) + " is above high_instance " + strconv.Itoa(high))
	}
	wait, err := intArg(args, "wait_seconds", discoverDefWait, 1, discoverMaxWait)
	if err != nil {
		return nil, err
	}
	networks, label, err := discoverNetworks(args)
	if err != nil {
		return nil, err
	}
	network := str(args, "network_ord")
	h, nodes, err := plan.Handle(c, network, 1)
	if err != nil {
		return nil, err
	}
	if t := rootType(nodes); t != bacnetNetworkTyp {
		return nil, plan.ToolError(network + " is " + pyjson.Repr(t) + ", not a " + bacnetNetworkTyp)
	}
	cfg := obj("nm", "p", "t", "bacnet:DeviceDiscoveryConfig", "s", []any{
		discoverInt("deviceLowLimit", low), discoverInt("deviceHighLimit", high),
		obj("nm", "p", "n", "networks", "t", "bacnet:DiscoveryNetworks", "v", networks),
		discoverInt("waitResponseTime", wait)})
	notes := []any{
		"sends Who-Is " + label + " for instances " + strconv.Itoa(low) + ".." + strconv.Itoa(high) +
			" and waits " + strconv.Itoa(wait) + " s for I-Am replies; each replying device is then read (about 8 ReadProperty requests)",
		"no device is added to the station database: the rows are children of a transient job under Services/JobService",
		"an I-Am from a registered instance at another address rewrites that device's address (B1165); the read-back flags it as address_differs",
		"the action is hidden and needs admin invoke rights on the network",
		"no inverse: field traffic cannot be undone",
	}
	return &plan.Planned{Ops: []any{invokeOp(obj("h", h, "a", discoverAction, "b", cfg))}, Notes: notes,
		Data: discoverData{network: network, wait: wait, w: w}}, nil
}

func discoverInt(name string, v int) *pyjson.Object {
	return obj("nm", "p", "n", name, "t", "baja:Integer", "v", strconv.Itoa(v))
}

// discoverNetworks is the networks argument as BDiscoveryNetworks.encodeToString writes it:
// "all;" when omitted or empty (global broadcast), else "-;" and the numbers comma-separated.
func discoverNetworks(args *pyjson.Object) (encoded, label string, err error) {
	raw, given := args.Get("networks")
	if !given || raw == nil {
		return "all;", "to the global broadcast address (every network)", nil
	}
	list, ok := raw.([]any)
	if !ok {
		return "", "", plan.ToolError("networks must be a list of BACnet network numbers, got " + pyjson.Repr(raw))
	}
	if len(list) == 0 {
		return "all;", "to the global broadcast address (every network)", nil
	}
	nums := make([]string, 0, len(list))
	for _, v := range list {
		n, ok := wholeNumber(v)
		if !ok || n < 1 || n > discoverMaxNet {
			return "", "", plan.ToolError("networks holds " + pyjson.Repr(v) + "; a BACnet network number is 1.." + strconv.Itoa(discoverMaxNet))
		}
		nums = append(nums, strconv.Itoa(n))
	}
	return "-;" + strings.Join(nums, ","), "on network(s) " + strings.Join(nums, ", "), nil
}

// intArg is the integer argument name within [lo, hi], def when it is absent.
func intArg(args *pyjson.Object, name string, def, lo, hi int) (int, error) {
	raw, given := args.Get(name)
	if !given || raw == nil {
		return def, nil
	}
	n, ok := wholeNumber(raw)
	if !ok || n < lo || n > hi {
		return 0, plan.ToolError(name + " must be a whole number in " + strconv.Itoa(lo) + ".." + strconv.Itoa(hi) +
			", got " + pyjson.Repr(raw))
	}
	return n, nil
}

// wholeNumber is v as an int when it is a JSON integer (never a bool or a fraction).
func wholeNumber(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int64:
		return int(x), true
	case float64:
		if x == float64(int(x)) {
			return int(x), true
		}
	case pyjson.Number:
		if b, ok := x.BigInt(); ok && b.IsInt64() {
			return int(b.Int64()), true
		}
	}
	return 0, false
}
