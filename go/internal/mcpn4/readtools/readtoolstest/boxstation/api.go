package boxstation

import (
	"fmt"
	"sort"
)

// This file is the arranging API black-box tests use to set station state while it serves:
// each call locks the station and goes through the same paths a BOX frame takes.

// SetVersion sets what GET /obix/about/ reports; "" makes the endpoint answer 404.
func (st *Station) SetVersion(v string) { st.Do(func(*Station) { st.version = v }) }

// Sessions are the BOX session ids that are open (made and not yet deleted), sorted.
func (st *Station) Sessions() []string {
	out := []string{}
	st.Do(func(*Station) {
		for id := range st.sessions {
			out = append(out, id)
		}
	})
	sort.Strings(out)
	return out
}

// ByHandle is the node with handle h, nil when absent (lock with Do while the station serves).
func (st *Station) ByHandle(h string) *Node { return st.byHandle[h] }

// Add creates a component of type typ under the component with handle parent, as a syncTo add
// does: the name is made unique. It returns the name given and the new handle.
func (st *Station) Add(parent, name, typ string) (nn, handle string, err error) {
	st.Do(func(*Station) {
		var out any
		out, err = st.sync(map[string]any{"nm": "a", "h": parent, "n": name, "b": map[string]any{"t": typ}})
		if err != nil {
			return
		}
		nn = out.([]any)[0].(map[string]any)["nn"].(string)
		handle = st.byHandle[parent].Child(nn).Handle
	})
	return nn, handle, err
}

// Set writes body (a BOX body) to slot (a/b path) of the component with handle h, as a syncTo
// set does.
func (st *Station) Set(h, slot string, body map[string]any) (err error) {
	st.Do(func(*Station) { _, err = st.sync(map[string]any{"nm": "s", "h": h, "n": slot, "b": body}) })
	return err
}

// Link adds a link from slot srcSlot of the component with handle src to slot tgtSlot of the
// component with handle tgt, as checkLinks does; it returns the link's name.
func (st *Station) Link(src, srcSlot, tgt, tgtSlot string) (name string, err error) {
	st.Do(func(*Station) {
		if st.byHandle[src] == nil || st.byHandle[tgt] == nil {
			err = fmt.Errorf("KeyError: '%s' or '%s'", src, tgt)
			return
		}
		res := st.checkLink(map[string]any{"s": src, "ss": srcSlot, "t": tgt, "ts": tgtSlot, "c": true})
		name = res.([]any)[0].(map[string]any)["s"].(string)
	})
	return name, err
}
