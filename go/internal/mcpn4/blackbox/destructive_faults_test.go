package blackbox

// The fault hook of the destructive tests: the Go form of the kit's patched FakeStation methods
// (_sync, _check_link, hook). It answers every frame itself, from Station.Answer, so it can
// record the syncTo ops the server sends, replace one message's answer, and render a node
// without a type as "t": null the way the kit's fake (and a live station's frozen children) do.

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// override answers one callssc message in the station's place; ok false leaves it to the station.
type override func(sck string, arg map[string]any) (body any, err error, ok bool)

type faults struct {
	mu        sync.Mutex
	recording bool
	sent      []map[string]any                         // syncTo ops seen while recording
	replace   override                                 // nil: the station answers
	before    func(sck string)                         // a spy run before the station answers
	raw       func(frame map[string]any) (int, []byte) // a non-zero HTTP status replaces the answer
}

// install makes f the station's hook; under the station lock, so the serving goroutines see it.
func (f *faults) install(st *boxstation.Station) {
	st.Do(func(*boxstation.Station) {
		st.Hook = func(frame map[string]any) (int, []byte) { return f.answer(st, frame) }
	})
}

func (f *faults) set(fn func(f *faults)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fn(f)
}

// record starts recording the syncTo ops and returns what was sent so far each time it is called.
func (f *faults) record() func() []map[string]any {
	f.set(func(f *faults) { f.recording, f.sent = true, nil })
	return func() []map[string]any {
		f.mu.Lock()
		defer f.mu.Unlock()
		return append([]map[string]any(nil), f.sent...)
	}
}

func (f *faults) answer(st *boxstation.Station, frame map[string]any) (int, []byte) {
	f.mu.Lock()
	replace, before, raw := f.replace, f.before, f.raw
	f.mu.Unlock()
	if raw != nil {
		if status, body := raw(frame); status != 0 {
			return status, body
		}
	}
	replies := []any{}
	for _, m := range objects(frame["m"]) {
		body, _ := m["b"].(map[string]any)
		sck, _ := body["sck"].(string)
		arg, _ := body["scarg"].(map[string]any)
		if sck == "syncTo" {
			f.seen(arg)
		}
		if before != nil && sck != "" {
			before(sck)
		}
		if replace != nil {
			if out, err, ok := replace(sck, arg); ok {
				replies = append(replies, envelope(m, out, err))
				continue
			}
		}
		one := st.Answer(map[string]any{"v": frame["v"], "p": frame["p"], "n": frame["n"], "m": []any{m}})
		replies = append(replies, objects(one["m"])[0])
	}
	reply := map[string]any{"v": "2.3", "p": "box", "n": frame["n"], "m": replies}
	nullTypes(reply)
	data, err := json.Marshal(reply)
	if err != nil {
		return 500, []byte(err.Error())
	}
	return 200, data
}

func (f *faults) seen(arg map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.recording {
		return
	}
	for _, op := range objects(arg["ops"]) {
		f.sent = append(f.sent, op)
	}
}

// envelope is the station's reply to message m: body, or an error reply for err.
func envelope(m map[string]any, body any, err error) map[string]any {
	if err != nil {
		return map[string]any{"c": m["c"], "k": m["k"], "r": 0, "t": "e",
			"b": map[string]any{"isErr": true, "m": err.Error()}}
	}
	return map[string]any{"c": m["c"], "k": m["k"], "r": 0, "b": body, "t": "rp"}
}

// nullTypes renders a BOX node without a type ("t": "") as "t": null, as the kit's fake does.
func nullTypes(v any) {
	switch x := v.(type) {
	case map[string]any:
		if t, ok := x["t"]; ok && t == "" && x["nm"] == "p" {
			x["t"] = nil
		}
		for _, sub := range x {
			nullTypes(sub)
		}
	case []any:
		for _, sub := range x {
			nullTypes(sub)
		}
	}
}

// syncAdd is the add op of a syncTo argument named name, nil when it is something else.
func syncAdd(sck string, arg map[string]any, name string) map[string]any {
	if sck != "syncTo" {
		return nil
	}
	ops := objects(arg["ops"])
	if len(ops) == 1 && ops[0]["nm"] == "a" && (name == "" || ops[0]["n"] == name) {
		return ops[0]
	}
	return nil
}

// refusal is the error a station answers for an op it cannot process.
var refusal = fmt.Errorf("ValueError: Unable to process request.")

// adds are the add ops among ops, in order.
func adds(ops []map[string]any) []map[string]any {
	out := []map[string]any{}
	for _, op := range ops {
		if op["nm"] == "a" {
			out = append(out, op)
		}
	}
	return out
}

// names are the "n" of each op.
func names(ops []map[string]any) []any {
	out := []any{}
	for _, op := range ops {
		out = append(out, op["n"])
	}
	return out
}
