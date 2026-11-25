package writeops

import "github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"

// defaultOverride is the argument active and inactive take on a control:BooleanWritable (control-rt
// BBooleanWritable.java:62, :90-91: parameterType BOverride, default new BOverride()). doActive and
// doInactive read override.getDuration() (:280-286), so a call with no argument reaches the station as
// a null pointer and comes back as a bare "station error"; Workbench always sends one. The argument is
// a control:Override with duration 0: WritableSupport.override (:237-253) turns a duration of 0 into the
// point's maxOverrideDuration (no expiry when that is 0 too), the same duration the station's own
// default (getActionParameterDefault, :321-329) carries. auto takes no argument (:92, :288).
func defaultOverride() *pyjson.Object {
	return obj("nm", "p", "t", "control:Override", "s", []any{
		obj("nm", "p", "n", "duration", "t", "baja:RelTime", "v", "0")})
}

// takesOverride is true for the state actions that need that argument.
func takesOverride(action string) bool { return action == "active" || action == "inactive" }
