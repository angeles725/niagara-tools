package writeops_test

import "strings"

// spNodes are the loads of a writable for the invoke cases: numeric and boolean writables with
// and without a fallback, odd fallback types and values, and roots that tell nothing.
var spNodes = []string{
	`{"": {"h": "11", "t": "control:NumericWritable"}}`,
	`{"": {"h": "11", "t": "control:NumericWritable"}, "fallback": {"t": "baja:StatusNumeric"},
	  "fallback/value": {"v": "3.5"}, "fallback/status": {"v": "8"}}`,
	`{"": {"h": "11", "t": "control:BooleanWritable"}, "fallback": {"t": "baja:StatusBoolean"},
	  "fallback/value": {"v": "true"}}`,
	`{"": {"h": "11", "t": "control:BooleanWritable"}}`,
	`{"": {"h": "11", "t": "x:y:NumericWritable"}}`,
	`{"": {"h": "11", "t": "kitControl:NumericConst"}}`,
	`{"": {"h": "11", "t": null}}`,
	`{"": {"h": "11", "t": 5}}`,
	`{"": {"h": "11"}}`,
	`{"": {"h": "11", "t": "control:NumericWritable"}, "fallback": {"t": ["l"]}}`,
	`{"": {"h": "11", "t": "control:NumericWritable"}, "fallback": {"t": {"d": 1}}}`,
	`{"": {"h": "11", "t": "control:NumericWritable"}, "fallback": {"t": null}}`,
	`{"": {"h": "11", "t": "control:NumericWritable"}, "fallback": {"t": "baja:StatusNumeric"},
	  "fallback/value": {"v": "x"}}`,
	`{"": {"h": "11", "t": "control:NumericWritable"}, "fallback": {"t": "baja:StatusNumeric"},
	  "fallback/value": {"v": "7.0"}}`,
	`{"": {"h": null, "t": "control:NumericWritable"}}`,
}

// invokeArgs are n4_invoke_action arguments (ord is Sp).
var invokeArgs = []string{
	`"action": "set", "arg": 7.0, "arg_type": "baja:Double"`,
	`"action": "set", "arg": 7, "arg_type": "baja:Double"`,
	`"action": "set", "arg": 3.5, "arg_type": "baja:Double"`,
	`"action": "set", "arg": true, "arg_type": "baja:Boolean"`,
	`"action": "set", "arg": true, "arg_type": "baja:Double"`,
	`"action": "set", "arg": 1e400, "arg_type": "baja:Double"`,
	`"action": "set", "arg": 1` + strings.Repeat("0", 400) + `, "arg_type": "baja:Double"`,
	`"action": "set", "arg": "x", "arg_type": "baja:Boolean"`,
	`"action": "set", "arg": 1.0`,
	`"action": "set", "arg_type": "baja:Double"`,
	`"action": "set", "arg": null, "arg_type": "baja:Double"`,
	`"action": "set"`,
	`"action": "active"`, `"action": "inactive"`, `"action": "auto"`,
	`"action": "auto", "arg": 1.0`, `"action": "auto", "arg_type": "baja:Double"`,
	`"action": "save"`, `"action": "restart"`, `"action": "Set"`, `"action": "set\n"`, `"action": ""`,
}

// invokeCases are the n4_invoke_action cases: each argument set on each load, read back from
// the unchanged load, from no load, and from loads with a different or no fallback.
func invokeCases() []string {
	var out []string
	const spOrd = "station:|slot:/Folder/Sp"
	for _, args := range invokeArgs {
		for _, n := range spNodes {
			before := `{"` + spOrd + `": ` + n + `}`
			out = addCase(out, "n4_invoke_action", `"ord": "`+spOrd+`", `+args, before, before, `{}`,
				`{"`+spOrd+`": `+spNodes[1]+`}`, `{"`+spOrd+`": `+spNodes[13]+`}`, `{"`+spOrd+`": {"": {"h": "11"}}}`)
		}
	}
	return out
}
