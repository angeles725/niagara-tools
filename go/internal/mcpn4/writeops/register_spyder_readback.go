package writeops

import (
	"fmt"
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// registerReadback reads the device back: it exists with the requested address and objectId, the
// controller reports the same instance (writeInstanceReadback) and the discovery is summarized
// (its rows are n4_discover_points'; the summary keeps the counts and the hardware). The verdict
// is the worst of the parts: a difference is a mismatch; a step that did not run or an answer that
// could not be read leaves it unverified. When a follow_up step failed the device is half
// configured, so it is not compared with the request: the verdict is unverified, never a mismatch.
func registerReadback(c plan.Client, _ *pyjson.Object, p *plan.Planned, replies, inverse []any) (any, any, any, string, error) {
	d := p.Data.(*registerData)
	stored := pyjson.Str(get(inverse[0].(*pyjson.Object), "n"))
	child, err := ord.ChildOrd(d.network, stored)
	if err != nil {
		return nil, nil, nil, "", err
	}
	h, nodes, err := plan.Handle(c, child, 4)
	if err != nil {
		return nil, nil, nil, "", err
	}
	a := d.args
	id := fmt.Sprintf("device:%d", a.instance)
	requested := obj("name", stored, "type", spyderType, "object_id", id, "address", obj("address_type", strconv.Itoa(mstpAddressType),
		"network_number", strconv.Itoa(a.netNumber), "mac", a.mac))
	addr := box.Children(nodes, "address")
	value := func(kids *pyjson.Object, name string) any { return box.ChildValue(kids, name, nil) }
	got := obj("address_type", value(addr, "addressType"), "network_number", value(addr, "networkNumber"), "mac", value(addr, "macAddress"))
	observedID := value(box.Children(nodes, "config/deviceObject"), "objectId")
	observed := obj("name", stored, "type", rootType(nodes), "object_id", observedID, "address", got)
	verdict := "verified"
	worst := func(v string) {
		if v == "mismatch" || v == "unverified" && verdict == "verified" {
			verdict = v
		}
	}
	want := get(requested, "address").(*pyjson.Object)
	if d.failed == "" && (rootType(nodes) != spyderType || observedID != id || pyjson.Str(get(got, "address_type")) != pyjson.Str(get(want, "address_type")) ||
		pyjson.Str(get(got, "network_number")) != pyjson.Str(get(want, "network_number")) || !sameOctets(get(got, "mac"), a.mac)) {
		worst("mismatch")
	}
	if d.failed != "" {
		observed.Set("failed_step", d.failed)
		worst("unverified")
	} else {
		_, _, asked, v, err := writeInstanceReadback(c, child, h, nil, nil)
		if err != nil {
			return nil, nil, nil, "", err
		}
		for _, k := range asked.(*pyjson.Object).Keys() {
			observed.Set(k, get(asked.(*pyjson.Object), k))
		}
		worst(v)
		if a.discover {
			worst(discoverSummary(c, d, observed))
		}
	}
	accepted, err := first(replies[0])
	return requested, accepted, observed, verdict, err
}

// discoverSummary adds the summary of the point discovery to observed (and the model check when
// asked for) and returns its verdict.
func discoverSummary(c plan.Client, d *registerData, observed *pyjson.Object) string {
	shadow := &plan.Planned{Data: pointsData{ext: d.child + "/points", wait: d.args.wait, w: d.w}}
	_, _, rows, verdict, err := pointsReadback(c, nil, shadow, []any{d.discovery}, nil)
	if err != nil {
		observed.Set("discover", obj("hint", "the discovery job could not be read: "+plan.PyStr(err)))
		return "unverified"
	}
	summary := pyjson.NewObject()
	for _, k := range rows.(*pyjson.Object).Keys() {
		if k != "points" {
			summary.Set(k, get(rows.(*pyjson.Object), k))
		}
	}
	observed.Set("discover", summary)
	if expected := d.args.expected; expected != "" && verdict == "verified" {
		model, _ := get(asObj(get(summary, "hardware")), "model").(string)
		switch {
		case model == "":
			observed.Set("hint", "the discovery read no known hardware id (analogValue:0), so the model "+expected+" was not checked")
			return "unverified"
		case model != expected:
			observed.Set("hint", fmt.Sprintf("the controller is a %s, not the expected %s: run n4_invoke_action modelChange with arg %s "+
				"before any program work (a download is refused on a model mismatch)", model, expected, model))
			return "mismatch"
		}
		observed.Set("model_check", model)
	}
	return verdict
}
