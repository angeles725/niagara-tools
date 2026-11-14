package livesmoke

import (
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// expectOutputs polls Compare.out and Cooling.out until both equal want or the settle time ends.
func (s *Scenario) expectOutputs(want bool) stepFn {
	return func(*pyjson.Object) (any, any, error) {
		forever := !(s.settle < 1e9) // inf and nan: Python's deadline is never reached
		deadline := time.Now()
		if !forever {
			deadline = deadline.Add(time.Duration(s.settle * float64(time.Second)))
		}
		var seen *pyjson.Object
		for {
			seen = pyjson.NewObject()
			for _, c := range []string{"Compare", "Cooling"} {
				v, err := s.slotValue(c)
				if err != nil {
					return nil, nil, err
				}
				seen.Set(c, equalsTrue(v) || v == "true")
			}
			if allEqual(seen, want) || (!forever && !time.Now().Before(deadline)) {
				break
			}
			time.Sleep(250 * time.Millisecond)
		}
		verdict := "mismatch"
		if allEqual(seen, want) {
			verdict = "verified"
		}
		return verdict, kwargs("expected", want, "observed", seen), nil
	}
}

func allEqual(seen *pyjson.Object, want bool) bool {
	for _, k := range seen.Keys() {
		if v, _ := seen.Get(k); v != want {
			return false
		}
	}
	return true
}
