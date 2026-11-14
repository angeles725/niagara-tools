package boxstation

import (
	"encoding/json"
	"net/http"
)

// This file lets a black-box write test fault the station the way the kit's write tests patched
// fake_station.py: OnSync is on_sync (it runs after each applied syncTo op) and CheckLink
// replaces _check_link (orig is the station's own). It goes through Hook, so the station's other
// paths are unchanged.

// Faults are the overrides; a nil field keeps the station's own behavior. Both run with the
// station locked.
type Faults struct {
	OnSync    func(st *Station, op map[string]any)
	CheckLink func(st *Station, arg map[string]any, orig func(arg map[string]any) any) any
}

// SetFaults routes every authorized BOX frame through f from now on; it replaces Hook.
func (st *Station) SetFaults(f Faults) {
	st.Do(func(*Station) {
		st.Hook = func(frame map[string]any) (int, []byte) {
			st.mu.Lock()
			defer st.mu.Unlock()
			body, err := json.Marshal(st.faultedFrame(frame, f))
			if err != nil {
				return http.StatusInternalServerError, []byte(err.Error())
			}
			return http.StatusOK, body
		}
	})
}

// faultedFrame is handleFrame with faultedDispatch in place of dispatch.
func (st *Station) faultedFrame(frame map[string]any, f Faults) map[string]any {
	msgs, _ := frame["m"].([]any)
	replies := []any{}
	for _, raw := range msgs {
		m, _ := raw.(map[string]any)
		channel, _ := m["c"].(string)
		key, _ := m["k"].(string)
		body, _ := m["b"].(map[string]any)
		out, err := st.faultedDispatch(channel, key, body, f)
		if err != nil {
			replies = append(replies, map[string]any{"c": m["c"], "k": m["k"], "r": 0, "t": "e",
				"b": map[string]any{"isErr": true, "m": err.Error()}})
			continue
		}
		replies = append(replies, map[string]any{"c": m["c"], "k": m["k"], "r": 0, "b": out, "t": "rp"})
	}
	return map[string]any{"v": "2.3", "p": "box", "n": frame["n"], "m": replies}
}

func (st *Station) faultedDispatch(channel, key string, body map[string]any, f Faults) (any, error) {
	if channel != "ssession" || key != "callssc" {
		return st.dispatch(channel, key, body)
	}
	sck, _ := body["sck"].(string)
	arg, _ := body["scarg"].(map[string]any)
	switch {
	case sck == "checkLinks" && f.CheckLink != nil:
		return []any{f.CheckLink(st, arg, st.checkLink)}, nil
	case sck == "syncTo" && f.OnSync != nil:
		out, err := st.ssc(sck, arg)
		if err == nil { // on_sync runs only after an applied op
			ops, _ := arg["ops"].([]any)
			op, _ := ops[0].(map[string]any)
			f.OnSync(st, op)
		}
		return out, err
	}
	return st.dispatch(channel, key, body)
}
