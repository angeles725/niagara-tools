package writeops

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// RegisterSpyder is n4_register_bacnet_spyder: the sequence proven four times on a field job (retro
// R12 delta M9, issue #932) as one plan and one token. The add is the plan's op (create's own plan:
// escaped name, type check, licence device limit, inverse and read-back); the steps that need the
// new component's handle (its address, its objectId, writeDeviceInstance, the point discovery)
// are the plan's follow_up, listed in the plan and hashed into the token, and run in the same batch
// after the add (register_spyder_run.go). The inverse removes the created device; the field write
// has none, and the plan says so.
var RegisterSpyder = plan.Impl{ScopeCheck: registerScope, PlanWith: registerPlan, Run: registerRun,
	Inverse: registerInverse, Readback: registerReadback, Unapplied: registerUnapplied}

const (
	// mstpAddressType is BBacnetAddress's addressType of an MS/TP controller, as set on the Spyders
	// registered live (network 2, MAC 0c); networkNumber and macAddress are its other two children.
	mstpAddressType = 3
	maxMstpMAC      = 254
	maxNetworkNum   = 65534
	// factoryInstance is the instance every new Spyder ships with: a second unit announcing it makes
	// the station rebind the first one's address (B1165).
	factoryInstance = 9999
	objectIDType    = "bacnet:BacnetObjectIdentifier"
	registerWait    = 120
)

const (
	msgNoInverse = "writeDeviceInstance has no inverse: it writes config/deviceObject/objectId into the controller at the " +
		"component's address (field write). The inverse removes the created device only; to undo the field write, " +
		"set objectId back on a component at that address and invoke writeDeviceInstance again"
	msgOneToken = "one token covers the add and every step of follow_up: after the add the tool sets the new device's address " +
		"and objectId, writes the instance into the controller (writeDeviceInstance, refused when the device is in fault) and, " +
		"with discover, asks its points; the read-back reads each back"
	msgDiscoverTraffic = "discover puts ReadProperty traffic on the field network and adds no point to the station"
	msgFactory         = "instance 9999 is the factory instance of a new Spyder: a second unit announcing it makes the station rebind " +
		"the first one's address (B1165); register at the final instance"
)

// spyderArgs are the validated arguments.
type spyderArgs struct {
	network, display, mac, expected string
	netNumber, instance, wait       int
	discover                        bool
}

func registerScope(s *ord.Scope, args *pyjson.Object) error {
	name, _ := args.Get("name")
	stored, err := plan.NewName("name", name)
	if err != nil {
		return err
	}
	parent, _ := args.Get("network_ord")
	_, err = s.CheckChild(parent, stored)
	return err
}

func registerArgs(args *pyjson.Object) (*spyderArgs, error) {
	a := &spyderArgs{network: str(args, "network_ord"), display: str(args, "name"), expected: str(args, "expected_model")}
	var mac int
	for _, f := range []struct {
		name   string
		out    *int
		lo, hi int
	}{{"network_number", &a.netNumber, 1, maxNetworkNum}, {"mac", &mac, 0, maxMstpMAC}, {"instance", &a.instance, 0, discoverMaxInst}} {
		if v, given := args.Get(f.name); !given || v == nil {
			return nil, plan.ToolError(f.name + " is required")
		}
		var err error
		if *f.out, err = intArg(args, f.name, 0, f.lo, f.hi); err != nil {
			return nil, err
		}
	}
	if a.instance == factoryInstance {
		return nil, plan.ToolError(msgFactory)
	}
	a.mac = fmt.Sprintf("%02x", mac)
	var err error
	if a.wait, err = intArg(args, "wait_seconds", registerWait, 1, pointsMaxWait); err != nil {
		return nil, err
	}
	d, given := args.Get("discover")
	a.discover = !given || d == nil || d == true
	if a.expected != "" {
		if !a.discover {
			return nil, plan.ToolError("expected_model is read from the discovery: it needs discover true")
		}
		m, known := spyderModelByName(a.expected)
		if !known {
			return nil, plan.ToolError(fmt.Sprintf("expected_model %s is not a known Spyder model (known: %s)", pyjson.StrRepr(a.expected),
				strings.Join(spyderModelNames(), ", ")))
		}
		a.expected = m.name
	}
	return a, nil
}

// steps are the follow-up steps in order: each a set of a slot of the new device, or an invoke of an
// action of it (target "points": of its points extension).
func (a *spyderArgs) steps() []any {
	set := func(slot, typ, value string) any { return obj("op", "set", "slot", slot, "type", typ, "value", value) }
	steps := []any{set("address/addressType", "baja:Integer", strconv.Itoa(mstpAddressType)),
		set("address/networkNumber", "baja:Integer", strconv.Itoa(a.netNumber)),
		set("address/macAddress", octetStringType, a.mac),
		set("config/deviceObject/objectId", objectIDType, fmt.Sprintf("device:%d", a.instance)),
		obj("op", "invoke", "action", writeDeviceInstance)}
	if a.discover {
		steps = append(steps, obj("op", "invoke", "target", "points", "action", pointsAction))
	}
	return steps
}

// registerData is what the run and the read-back need; Run fills child, failed and discoverReply.
type registerData struct {
	create    createData
	args      *spyderArgs
	network   string
	steps     []any
	w         *plan.State
	child     string
	failed    string
	discovery any
}

func registerPlan(c plan.Client, args *pyjson.Object, w *plan.State, _ *session.Session) (*plan.Planned, error) {
	a, err := registerArgs(args)
	if err != nil {
		return nil, err
	}
	if err := networkUsable(c, a.network); err != nil {
		return nil, err
	}
	p, err := createPlan(c, obj("parent_ord", a.network, "name", a.display, "type", spyderType))
	if err != nil {
		return nil, err
	}
	stored := pyjson.Str(get(p.Ops[0].(*pyjson.Object), "n"))
	if err := spyderFree(c, p.Data.(createData), a, stored); err != nil {
		return nil, err
	}
	steps := a.steps()
	if err := w.CheckRoom(len(steps)+1, "this registration"); err != nil {
		return nil, err
	}
	p.Notes = append(p.Notes, msgOneToken, msgNoInverse)
	if a.discover {
		p.Notes = append(p.Notes, msgDiscoverTraffic)
	}
	p.Keys, p.Writes = obj("follow_up", steps), len(steps)+1
	p.Data = &registerData{create: p.Data.(createData), args: a, network: a.network, steps: steps, w: w}
	return p, nil
}

func registerInverse(p *plan.Planned, replies []any) ([]any, error) {
	return createInverse(&plan.Planned{Ops: p.Ops, Data: p.Data.(*registerData).create}, replies)
}

func registerUnapplied(c plan.Client, _ *pyjson.Object, p *plan.Planned) (bool, error) {
	d := p.Data.(*registerData)
	return createUnapplied(c, obj("parent_ord", d.network), &plan.Planned{Data: d.create})
}
