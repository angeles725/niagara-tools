package boxstation

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// handleFrame answers one BOX frame: one reply per message, an error reply for a bad one.
func (st *Station) handleFrame(frame map[string]any) map[string]any {
	msgs, _ := frame["m"].([]any)
	replies := []any{}
	for _, raw := range msgs {
		m, _ := raw.(map[string]any)
		channel, _ := m["c"].(string)
		key, _ := m["k"].(string)
		body, _ := m["b"].(map[string]any)
		out, err := st.dispatch(channel, key, body)
		if err != nil {
			replies = append(replies, map[string]any{"c": m["c"], "k": m["k"], "r": m["r"], "t": "e",
				"b": map[string]any{"isErr": true, "m": err.Error()}})
			continue
		}
		replies = append(replies, map[string]any{"c": m["c"], "k": m["k"], "r": m["r"], "b": out, "t": "rp"})
	}
	return map[string]any{"v": "2.3", "p": "box", "n": frame["n"], "m": replies}
}

func (st *Station) dispatch(channel, key string, body map[string]any) (any, error) {
	if channel == "reg" && key == "loadTypes" {
		return st.loadTypes(body), nil
	}
	if channel != "ssession" {
		return nil, fmt.Errorf("ValueError: unknown channel %s", channel)
	}
	switch key {
	case "make":
		st.sessions["sess-1"] = true
		return "sess-1", nil
	case "makessc":
		return map[string]any{"isReadonly": false}, nil
	case "pollchgs":
		events := st.events
		st.events = nil
		if events == nil {
			events = []any{}
		}
		return events, nil
	case "del":
		id, _ := body["id"].(string)
		delete(st.sessions, id)
		return nil, nil
	case "callssc":
		sck, _ := body["sck"].(string)
		if sck == "impliedTags" || sck == "impliedRelations" {
			return st.impliedCall(sck, body["scarg"])
		}
		arg, _ := body["scarg"].(map[string]any)
		return st.ssc(sck, arg)
	}
	return nil, fmt.Errorf("ValueError: unknown key %s", key)
}

func (st *Station) ssc(key string, arg map[string]any) (any, error) {
	switch key {
	case "loadRoot":
		return map[string]any{"h": "2", "t": "baja:Station"}, nil
	case "loadSlots":
		ord, _ := arg["o"].(string)
		n, err := st.resolveComponent(ord)
		if err != nil {
			return nil, err
		}
		depth, _ := arg["d"].(float64)
		tree := serialize(n, int(depth), true)
		st.events = append(st.events, map[string]any{"evs": map[string]any{"nm": "sync", "ver": "1.0",
			"ops": []any{map[string]any{"nm": "l", "h": n.Handle, "b": tree}}}})
		return nil, nil
	case "syncTo":
		ops, _ := arg["ops"].([]any)
		if len(ops) != 1 {
			return nil, fmt.Errorf("ValueError: exactly one op per syncTo")
		}
		op, _ := ops[0].(map[string]any)
		return st.sync(op)
	case "checkLinks":
		return []any{st.checkLink(arg)}, nil
	case "invokeAction":
		return st.invoke(arg)
	}
	return nil, fmt.Errorf("ValueError: unknown ssc key %s", key)
}

func (st *Station) checkLink(arg map[string]any) any {
	th, _ := arg["t"].(string)
	target := st.byHandle[th]
	name := st.unique(target, "Link")
	if c, _ := arg["c"].(bool); c {
		sh, _ := arg["s"].(string)
		link := &Node{Name: name, Type: "baja:Link"}
		for _, kv := range [][2]any{{"sourceOrd", "h:" + st.byHandle[sh].Handle}, {"sourceSlotName", arg["ss"]},
			{"targetSlotName", arg["ts"]}} {
			link.Children = append(link.Children, &Node{Name: kv[0].(string), Type: "baja:String", Value: kv[1]})
		}
		target.Children = append(target.Children, link)
	}
	return []any{map[string]any{"v": true, "r": nil, "s": name}}
}

func (st *Station) invoke(arg map[string]any) (any, error) {
	h, _ := arg["h"].(string)
	n := st.byHandle[h]
	if n == nil {
		return nil, fmt.Errorf("KeyError: '%s'", h)
	}
	action, _ := arg["a"].(string)
	var reply any
	switch {
	case action == "save" && n == st.root:
		st.saves++
		if st.ConfigPath != "" {
			if err := st.rewriteConfig(); err != nil {
				return nil, err
			}
		}
	case action == "set" && strings.HasSuffix(n.Type, "NumericWritable"):
		b, _ := arg["b"].(map[string]any)
		whole := map[string]any{"t": "baja:StatusNumeric", "s": []any{
			map[string]any{"n": "value", "t": "baja:Double", "v": b["v"]},
			map[string]any{"n": "status", "t": "baja:Status", "v": "0"}}}
		leaf := n.Child("fallback")
		if leaf == nil {
			leaf = &Node{Name: "fallback", Type: "baja:StatusNumeric"}
			n.Children = append(n.Children, leaf)
		}
		st.fill(leaf, whole)
	case action == "active" || action == "inactive" || action == "auto" || action == "ping":
	default:
		answered := false
		if st.Respond != nil {
			reply, answered = st.Respond(st, action, n)
		}
		if !answered {
			return nil, fmt.Errorf("ValueError: unsupported action %s", action)
		}
	}
	if st.AfterInvoke != nil {
		st.AfterInvoke(st, action, n)
	}
	return reply, nil
}

// rewriteConfig appends to config.bog and moves its mtime 5 s ahead, like a station persisting.
func (st *Station) rewriteConfig() error {
	f, err := os.OpenFile(st.ConfigPath, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(f, "saved%d", st.saves)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	info, err := os.Stat(st.ConfigPath)
	if err != nil {
		return err
	}
	mtime := info.ModTime().Add(5 * time.Second)
	return os.Chtimes(st.ConfigPath, mtime, mtime)
}
