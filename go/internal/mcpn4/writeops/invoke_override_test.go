package writeops_test

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

const boolWritable = "station:|slot:/Folder/Sw"

// override is the argument BBooleanWritable.active and .inactive take (control-rt
// BBooleanWritable.java:62, :90-91: parameterType BOverride, default new BOverride(); doActive and
// doInactive read override.getDuration() at :280-286, so a call with no argument fails on the
// station): a control:Override with duration 0, which WritableSupport.override (:237-253) turns into the
// station's maxOverrideDuration, as the default of getActionParameterDefault (:321-329) does.
const override = `"b": {"nm": "p", "t": "control:Override", "s": [{"nm": "p", "n": "duration", "t": "baja:RelTime", "v": "0"}]}`

func boolPlan(t *testing.T, action string, extra ...any) (string, error) {
	t.Helper()
	c := &loads{nodes: obj(boolWritable, obj("", obj("h", "5", "t", "control:BooleanWritable")))}
	args := obj("ord", boolWritable, "action", action)
	for i := 0; i < len(extra); i += 2 {
		args.Set(extra[i].(string), extra[i+1])
	}
	p, err := writeops.InvokeAction.Plan(c, args)
	if err != nil {
		return "", err
	}
	return js(t, p.Ops) + " " + js(t, p.Inverse), nil
}

// active and inactive send the default override; the station refuses them without one.
func TestActiveAndInactiveSendTheDefaultOverrideArgument(t *testing.T) {
	for _, action := range []string{"active", "inactive"} {
		got, err := boolPlan(t, action)
		if err != nil {
			t.Fatal(err)
		}
		want := `[{"ssc": "invokeAction", "arg": {"h": "5", "a": "` + action + `", ` + override + `}}] []`
		if got != want {
			t.Errorf("%s:\n got  %s\n want %s", action, got, want)
		}
	}
}

// auto takes no argument (BBooleanWritable.java:92, :288 doAuto()) and keeps sending none; a
// caller's arg is still refused for the three state actions.
func TestAutoStaysArgumentLessAndAnArgIsStillRefused(t *testing.T) {
	got, err := boolPlan(t, "auto")
	if want := `[{"ssc": "invokeAction", "arg": {"h": "5", "a": "auto"}}] []`; err != nil || got != want {
		t.Errorf("auto: %q, %v", got, err)
	}
	for _, action := range []string{"active", "inactive", "auto"} {
		for _, k := range []string{"arg", "arg_type"} {
			_, err := boolPlan(t, action, k, "x")
			if want := "action '" + action + "' takes no arg"; err == nil || err.Error() != want {
				t.Errorf("%s with %s: err = %v, want %q", action, k, err, want)
			}
		}
	}
}
