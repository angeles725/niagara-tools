package writeops_test

import "strings"

const calcOrd = "station:|slot:/Folder/Calc"

// calcNodes are the loads of Calc the set_slot cases plan on and read back: well-formed slots of
// every value type, odd and missing values, odd roots and handles.
var calcNodes = []string{
	`{"": {"h": "10", "t": "kitControl:NumericConst"}, "out": {"t": "baja:StatusNumeric"},
	  "out/value": {"t": "baja:Double", "v": "1.0"}, "out/status": {"t": "baja:Status", "v": "0"},
	  "d": {"t": "baja:Double", "v": "2.0"}, "b": {"t": "baja:Boolean", "v": "true"},
	  "s": {"t": "baja:String", "v": "old"}, "flag": {"t": "baja:StatusBoolean"},
	  "flag/value": {"t": "baja:Boolean", "v": "true"}, "grp": {"t": "baja:Struct"},
	  "grp/sp": {"t": "baja:StatusNumeric"}, "grp/sp/value": {"t": "baja:Double", "v": "7.5"},
	  "zero": {"t": "baja:Double"}, "e": {"t": "baja:String"}, "n": {"t": null}, "x": {}}`,
	`{"": {"h": "10"}, "out": {"t": "baja:StatusNumeric"}, "out/value": {"v": "x y"},
	  "out/status": {"v": 5}, "d": {"t": "baja:Double", "v": 7}, "b": {"t": "baja:Boolean", "v": true},
	  "s": {"t": "baja:String", "v": 5}, "flag": {"t": "baja:StatusBoolean"}, "flag/value": {"v": "TRUE"},
	  "grp/sp": {"t": "baja:StatusNumeric"}, "grp/sp/value": {"v": null}}`,
	`{"": {"h": 16}, "out": {"t": "baja:StatusNumeric"}, "out/value": {"v": "１.５"},
	  "out/status": {"v": "8;facets"}, "d": {"t": "baja:Double", "v": " 1_0 "}, "b": {"t": "baja:Boolean", "v": "false"},
	  "s": {"t": "baja:String", "v": ""}, "flag": {"t": "baja:StatusBoolean"}, "grp/sp": {"t": "baja:StatusNumeric"},
	  "grp/sp/value": {"v": [1]}}`,
	`{"": {"h": "10"}, "out": {"t": "baja:StatusNumeric"}, "out/value": {"v": "nan"}, "d": {"t": "baja:Double", "v": "1e400"},
	  "s": {"t": "baja:String", "v": null}, "grp/sp": {"t": "baja:StatusNumeric"}, "grp/sp/value": {"v": 1e400}}`,
	`{"": {"h": ""}, "out": {"t": "baja:StatusNumeric"}}`,
	`{"": {"t": "kitControl:NumericConst"}, "out": {"t": "baja:StatusNumeric"}}`,
	`{"out": {"t": "baja:StatusNumeric"}}`,
}

// setArgs are n4_set_slot arguments (ord is Calc unless given).
var setArgs = []string{
	`"slot": "out", "value": 1.0, "value_type": "baja:StatusNumeric"`,
	`"slot": "out", "value": 5.5, "value_type": "baja:StatusNumeric", "status": "0"`,
	`"slot": "out", "value": 5, "value_type": "baja:StatusNumeric", "status": "a0;f"`,
	`"slot": "out", "value": 1.0, "value_type": "baja:StatusNumeric", "status": "0\n"`,
	`"slot": "out", "value": 1.0, "value_type": "baja:StatusNumeric", "status": "0\n\n"`,
	`"slot": "out", "value": 1.0, "value_type": "baja:StatusNumeric", "status": "0;a\nb"`,
	`"slot": "out", "value": 1.0, "value_type": "baja:StatusNumeric", "status": "zz"`,
	`"slot": "out", "value": 1.0, "value_type": "baja:StatusNumeric", "status": ""`,
	`"slot": "out", "value": 1.0, "value_type": "baja:StatusNumeric", "status": 7`,
	`"slot": "out", "value": 1.0, "value_type": "baja:StatusNumeric", "status": null`,
	`"slot": "out", "value": true, "value_type": "baja:StatusNumeric"`,
	`"slot": "out", "value": NaN, "value_type": "baja:StatusNumeric"`,
	`"slot": "out", "value": -Infinity, "value_type": "baja:Double"`,
	`"slot": "out", "value": 1e400, "value_type": "baja:Double"`,
	`"slot": "out", "value": 1` + strings.Repeat("0", 400) + `, "value_type": "baja:Double"`,
	`"slot": "out", "value": 9007199254740993, "value_type": "baja:StatusNumeric"`,
	`"slot": "out", "value": "1.0", "value_type": "baja:StatusNumeric"`,
	`"slot": "out", "value": [1], "value_type": "baja:StatusNumeric"`,
	`"slot": "out", "value": true, "value_type": "baja:StatusBoolean"`,
	`"slot": "out", "value": 1.0, "value_type": "baja:Int"`,
	`"slot": "out", "value": 1.0, "value_type": "baja:Int", "status": "0"`,
	`"slot": "out/value", "value": 1.0, "value_type": "baja:Double"`,
	`"slot": "grp/out/status", "value": 1.0, "value_type": "baja:Double"`,
	`"slot": "grp/sp", "value": 9.0, "value_type": "baja:StatusNumeric"`,
	`"slot": "grp/sp", "value": 7.5, "value_type": "baja:StatusNumeric"`,
	`"slot": "d", "value": 2.0, "value_type": "baja:Double"`,
	`"slot": "d", "value": 3, "value_type": "baja:Double", "status": "0"`,
	`"slot": "d", "value": -0.0, "value_type": "baja:Double"`,
	`"slot": "d", "value": 1e-05, "value_type": "baja:Double"`,
	`"slot": "b", "value": true, "value_type": "baja:Boolean"`,
	`"slot": "b", "value": 1, "value_type": "baja:Boolean"`,
	`"slot": "s", "value": "old", "value_type": "baja:String"`,
	`"slot": "s", "value": "é\ud800", "value_type": "baja:String"`,
	`"slot": "s", "value": 1, "value_type": "baja:String"`,
	`"slot": "flag", "value": true, "value_type": "baja:StatusBoolean"`,
	`"slot": "flag", "value": false, "value_type": "baja:StatusBoolean", "status": "40"`,
	`"slot": "zero", "value": 1.0, "value_type": "baja:Double"`,
	`"slot": "e", "value": "x", "value_type": "baja:String"`,
	`"slot": "n", "value": "x", "value_type": "baja:String"`,
	`"slot": "x", "value": "x", "value_type": "baja:String"`,
	`"slot": "nope", "value": 1.0, "value_type": "baja:Double"`,
	`"slot": "out\n", "value": 1.0, "value_type": "baja:StatusNumeric"`,
	`"slot": "a b", "value": 1.0, "value_type": "baja:Double"`,
	`"slot": "", "value": 1.0, "value_type": "baja:Double"`,
	`"slot": "a//b", "value": 1.0, "value_type": "baja:Double"`,
	`"slot": "1x", "value": 1.0, "value_type": "baja:Double"`,
	`"slot": "é", "value": 1.0, "value_type": "baja:Double"`,
	`"slot": "out", "value": 1.0, "value_type": "baja:StatusNumeric", "ord": "station:|slot:/Folder/Missing"`,
}

// corpus is every case as one JSON line: each argument set on each load, read back from the
// unchanged load, from a load that lost the slot or the component, and (set_slot) from a
// tampered one.
func corpus() []string {
	var out []string
	for _, args := range setArgs {
		if !strings.Contains(args, `"ord"`) {
			args = `"ord": "` + calcOrd + `", ` + args
		}
		for _, n := range calcNodes {
			before := `{"` + calcOrd + `": ` + n + `}`
			out = addCase(out, "n4_set_slot", args, before, before, `{}`, `{"`+calcOrd+`": `+calcNodes[1]+`}`,
				`{"`+calcOrd+`": {"": {"h": "10"}}}`)
		}
	}
	return append(out, invokeCases()...)
}

// addCase appends one case line per after load.
func addCase(out []string, tool, args, before string, afters ...string) []string {
	for _, after := range afters {
		out = append(out, strings.ReplaceAll(`{"tool": "`+tool+`", "args": {`+args+`}, "before": `+before+
			`, "after": `+after+`}`, "\n\t  ", " "))
	}
	return out
}
