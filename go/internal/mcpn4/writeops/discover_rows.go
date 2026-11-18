package writeops

import (
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// discoverReadback waits for the discovery job the station returned (its ORD is the invoke
// reply, {"t":"baja:Ord","v":"slot:/Services/JobService/BacnetDiscoverDevicesJob<N>"}), then
// reads its rows and compares them with the devices under the network. A job that ended success
// is verified, failed or canceled a mismatch, anything else unverified with why.
func discoverReadback(c plan.Client, _ *pyjson.Object, p *plan.Planned, replies, _ []any) (any, any, any, string, error) {
	d := p.Data.(discoverData)
	requested := obj("action", discoverAction, "network_ord", d.network)
	job := jobOrd(replies[0])
	if job == "" {
		return requested, replies[0], obj("hint", "the station returned no job ORD (a network in fatal fault returns none)"), "unverified", nil
	}
	state, nodes, err := waitJob(c, job, d)
	if err != nil {
		return nil, nil, nil, "", err
	}
	observed := obj("job", job, "job_state", state)
	switch {
	case state == "success":
	case jobComplete(state):
		return requested, replies[0], observed, "mismatch", nil
	default:
		observed.Set("hint", "the discovery job is still "+pyjson.Str(state)+" after "+strconv.Itoa(d.wait)+" s plus "+
			strconv.Itoa(int(discoverSlack))+" s; read it later under "+job)
		return requested, replies[0], observed, "unverified", nil
	}
	known, err := knownDevices(c, d.network)
	if err != nil {
		return nil, nil, nil, "", err
	}
	rows, answered := []any{}, map[string]bool{}
	for _, name := range box.Children(nodes, "").Keys() {
		if !strings.Contains(pyjson.Str(get(box.Node(nodes, name), "t")), "DiscoveryDevice") {
			continue
		}
		row := discoveredRow(nodes, name, known)
		answered[pyjson.Str(get(row, "in_database"))] = true
		rows = append(rows, row)
	}
	silent := []any{}
	for _, k := range known {
		if !answered[k.name] {
			silent = append(silent, obj("name", k.name, "instance", k.instance, "address", k.address))
		}
	}
	observed.Set("found", len(rows))
	observed.Set("devices", rows)
	observed.Set("not_answering", silent)
	return requested, replies[0], observed, "verified", nil
}

// jobOrd is the station ORD of the reply's job ("" when the reply holds none).
func jobOrd(reply any) string {
	v := reply
	if o, ok := reply.(*pyjson.Object); ok {
		v = get(o, "v")
	}
	s, _ := v.(string)
	switch {
	case strings.HasPrefix(s, "station:|"):
		return s
	case strings.HasPrefix(s, "slot:/"):
		return "station:|" + s
	}
	return ""
}

// waitJob polls the job's state until it completes or waitResponseTime plus discoverSlack pass,
// then loads its rows (a DiscoveryDevice is a struct: its address is two levels down).
func waitJob(c plan.Client, job string, d discoverData) (any, *pyjson.Object, error) {
	deadline := d.w.Monotonic() + float64(d.wait) + discoverSlack
	for {
		nodes, err := c.LoadTree(job, 3, box.LoadOptions{})
		if err != nil {
			return nil, nil, err
		}
		state := box.ChildValue(box.Children(nodes, ""), "jobState", "unknown")
		if jobComplete(state) || d.w.Monotonic() >= deadline {
			return state, nodes, nil
		}
		d.w.Sleep(d.w.SaveInterval)
	}
}

// knownDevice is one device registered under the network.
type knownDevice struct {
	name, address string
	instance      any
}

// knownDevices is every child of the network that has a BacnetAddress, with its objectId.
func knownDevices(c plan.Client, network string) ([]knownDevice, error) {
	nodes, err := c.LoadTree(network, 2, box.LoadOptions{})
	if err != nil {
		return nil, err
	}
	var out []knownDevice
	for _, name := range box.Children(nodes, "").Keys() {
		addr := box.Node(nodes, name+"/address")
		if addr == nil || get(addr, "t") != "bacnet:BacnetAddress" {
			continue
		}
		k := knownDevice{name: name, address: pyjson.Str(get(addr, "d"))}
		if obj, err := c.LoadTree(network+"/"+name+"/config/deviceObject", 1, box.LoadOptions{}); err == nil {
			k.instance = instance(box.ChildValue(box.Children(obj, ""), "objectId", nil))
		}
		out = append(out, k)
	}
	return out, nil
}

// discoveredRow is one DiscoveryDevice child as a row, matched against the registered devices by
// instance: registered (same address), address_differs (an I-Am rebind risk) or new.
func discoveredRow(nodes *pyjson.Object, name string, known []knownDevice) *pyjson.Object {
	val := func(slot string) any { return box.ChildValue(box.Children(nodes, name), slot, nil) }
	sub := func(slot string) any { return box.ChildValue(box.Children(nodes, name+"/address"), slot, nil) }
	inst := instance(val("objectId"))
	network, _ := wholeNumberText(sub("networkNumber"))
	mac := pyjson.Str(sub("macAddress"))
	vendor, _ := wholeNumberText(val("vendorId"))
	row := obj("instance", inst, "network", network, "mac", mac, "mac_display", MacDisplay(strOr(sub("addressType")), mac),
		"name", unescapeSlot(strOr(val("deviceName"))), "vendor_id", vendor, "vendor", strOr(val("vendorName")),
		"model", strOr(val("modelName")), "objects", objectCount(val("listSize")))
	row.Set("in_database", nil)
	row.Set("status", "new")
	address := strconv.Itoa(network) + ":" + mac
	for _, k := range known {
		if inst == nil || k.instance != inst {
			continue
		}
		row.Set("in_database", k.name)
		row.Set("status", "registered")
		if !strings.EqualFold(k.address, address) {
			row.Set("status", "address_differs")
			row.Set("database_address", k.address)
		}
	}
	if dup := val("duplicate"); dup == "true" {
		row.Set("duplicate", true)
	}
	return row
}

// instance is the number of a "device:2004" object identifier (nil when it is not one).
func instance(v any) any {
	s := pyjson.Str(v)
	if !strings.HasPrefix(s, "device:") {
		return nil
	}
	n, err := strconv.Atoi(s[len("device:"):])
	if err != nil {
		return nil
	}
	return n
}

func wholeNumberText(v any) (int, bool) {
	n, err := strconv.Atoi(pyjson.Str(v))
	return n, err == nil
}

func strOr(v any) string {
	if v == nil {
		return ""
	}
	return pyjson.Str(v)
}
