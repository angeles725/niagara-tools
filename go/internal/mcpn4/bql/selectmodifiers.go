package bql

import "regexp"

// A BQL select reads select [TOP n] [DISTINCT | ALL] <columns> (bql-rt SelectParser.parse: top(),
// quantifier(), then projection(); keywords are case-insensitive). The modifiers are no column,
// so the select list drops them before it is split. A modifier word with no column after it
// (select top from x) is a column name and stays.
var selectModifiersRe = regexp.MustCompile(`^(?is)(?:top` + space + `+[0-9]+` + space + `+)?(?:(?:distinct|all)` + space + `+)?`)

// dropSelectModifiers is list without its leading TOP n and DISTINCT or ALL.
func dropSelectModifiers(list string) string {
	return list[len(selectModifiersRe.FindString(list)):]
}
