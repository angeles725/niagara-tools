package writeops_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

func apPlan(t *testing.T) (*typed, []any) {
	t.Helper()
	c := apStation(apGolden())
	p, err := writeops.Apply.Plan(c, apArgs(
		apRow("R1", "setpoint", pyjson.FloatNumber(25), "label", "Zone A"),
		apRow("R2", "gain", pyjson.FloatNumber(2.5), "setpoint", pyjson.FloatNumber(30))))
	if err != nil {
		t.Fatal(err)
	}
	return c, p.Ops
}

// S5: each copy's bindings name the copy's own components by slot ORD, never the golden's handle.
func TestApplyAddsOneCopyPerRowWithEveryBindingPointedAtItsOwnTarget(t *testing.T) {
	_, ops := apPlan(t)
	for i, row := range []string{"R1", "R2"} {
		want := map[string]any{"B1": apParent + "/" + row + "/Pt", "B2": apParent + "/" + row + "/Pt",
			"B3": apParent + "/" + row + "/Pt/Inner"}
		if got := apBindingOrds(ops[i]); fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("%s bindings = %v, want %v", row, got, want)
		}
		text := js(t, ops[i])
		if strings.Contains(text, `"h:`) || !strings.Contains(text, `"nm": "a", "h": "5", "n": "`+row+`"`) {
			t.Errorf("%s add holds a handle ORD or is not an add under the parent: %s", row, text)
		}
	}
}

func TestApplySetsEachConfigOnTheCopyAfterTheAdds(t *testing.T) {
	_, ops := apPlan(t)
	var got []string
	for _, op := range ops {
		o := op.(*pyjson.Object)
		got = append(got, fmt.Sprint(get(o, "nm"), " ", get(o, "n")))
	}
	want := "[a R1 a R2 s R1/templateConfig/setpoint s R1/templateConfig/label s R2/templateConfig/gain s R2/templateConfig/setpoint]"
	if fmt.Sprint(got) != want {
		t.Errorf("ops = %v, want %s", got, want)
	}
	if text := js(t, ops[2]); text != `{"nm": "s", "h": "5", "n": "R1/templateConfig/setpoint", "b": {"nm": "p", "t": "baja:Double", "v": "25.0"}}` {
		t.Errorf("set op = %s", text)
	}
	if text := js(t, ops[3]); !strings.Contains(text, `"b": {"nm": "p", "t": "baja:String", "v": "Zone A"}`) {
		t.Errorf("string set op = %s", text)
	}
}

func TestApplyInverseRemovesEachCopyAndTheNotesSayWhatHoldsAfterwards(t *testing.T) {
	c := apStation(apGolden())
	p, err := writeops.Apply.Plan(c, apArgs(apRow("R1"), apRow("R2", "setpoint", pyjson.FloatNumber(1))))
	if err != nil {
		t.Fatal(err)
	}
	if got := js(t, p.Inverse); got != `[{"nm": "v", "h": "5", "n": "R1"}, {"nm": "v", "h": "5", "n": "R2"}]` {
		t.Errorf("inverse = %s", got)
	}
	notes := js(t, p.Notes)
	for _, want := range []string{"slot ORD", "rename", "decoding"} {
		if !strings.Contains(notes, want) {
			t.Errorf("notes lack %q: %s", want, notes)
		}
	}
	if p.Writes != 3 {
		t.Errorf("writes = %d, want 3 (2 adds, 1 set)", p.Writes)
	}
}

func apRefusal(t *testing.T, c *typed, args *pyjson.Object, want ...string) {
	t.Helper()
	_, err := writeops.Apply.Plan(c, args)
	if err == nil {
		t.Errorf("%s: not refused", js(t, args))
		return
	}
	for _, w := range want {
		if !strings.Contains(err.Error(), w) {
			t.Errorf("err = %q, want it to hold %q", err, w)
		}
	}
}

func TestApplyRefusesBadRowsAndNames(t *testing.T) {
	c := apStation(apGolden(), "Taken")
	apRefusal(t, c, apArgs(), "rows must list 1 to 30 entries")
	many := []any{}
	for i := 0; i < 31; i++ {
		many = append(many, apRow(fmt.Sprintf("R%d", i)))
	}
	apRefusal(t, c, apArgs(many...), "rows must list 1 to 30 entries")
	apRefusal(t, c, apArgs(apRow("R1"), apRow("R1")), "a row named 'R1' is given twice")
	apRefusal(t, c, apArgs(apRow("Taken")), "a child named 'Taken' already exists under "+apParent)
	apRefusal(t, c, apArgs(apRow("a/b")), "plain slot name")
	apRefusal(t, c, obj("parent_ord", apParent, "golden", "Ghost", "rows", []any{apRow("R1")}),
		"component 'Ghost' not found under "+apParent)
	apRefusal(t, c, apArgs(obj("name", "R1", "configs", []any{})), "configs of row 'R1' must be an object")
}

func TestApplyRefusesAGoldenThatIsNotADeployedInstance(t *testing.T) {
	g := apGolden()
	tree := g.tree()
	tree.Set("s", []any{g.pt()}) // no templateConfig
	apRefusal(t, apStationOf(tree, g.pt()), apArgs(apRow("R1")), "'Gold'", "templateConfig", "template:TemplateConfig")
	tree = g.tree()
	apKid(tree, "templateConfig").Set("t", "baja:Folder")
	apRefusal(t, apStationOf(tree, g.pt()), apArgs(apRow("R1")), "templateConfig", "template:TemplateConfig")
}

func TestApplyRefusesAConfigThatNamesNoConfigSlot(t *testing.T) {
	apRefusal(t, apStation(apGolden()), apArgs(apRow("R1", "nope", pyjson.FloatNumber(1))),
		"'nope'", "R1", "gain, label, setpoint")
}

func TestApplyRefusesBindingsItCannotPointAtTheCopy(t *testing.T) {
	g := apGolden()
	for name, c := range map[string]struct {
		bind string
		want []string
	}{
		"handle outside the golden": {"h:99", []string{"h:99", "not a handle or slot ORD found in the template instance"}},
		"not a handle":              {"station:|slot:/Lab/Units/Gold/Pt", []string{"station:|slot:/Lab/Units/Gold/Pt", "not a handle"}},
	} {
		s := g
		s.b3 = c.bind // an unconfigured binding too: the copy would keep pointing at the golden
		t.Run(name, func(t *testing.T) {
			apRefusal(t, apStation(s), apArgs(apRow("R1", "setpoint", pyjson.FloatNumber(1))), c.want...)
		})
	}
	tree := g.tree()
	apKid(apKid(apKid(tree, "templateConfig"), "B1"), "targetSlot").Set("v", "#Name")
	apRefusal(t, apStationOf(tree, g.pt()), apArgs(apRow("R1", "setpoint", pyjson.FloatNumber(1))),
		"'setpoint'", "#Name", "n4_rename_component")
}

func TestApplyRefusesAValueThatDoesNotFitTheSlotType(t *testing.T) {
	c := apStation(apGolden())
	apRefusal(t, c, apArgs(apRow("R1", "setpoint", "hot")), "R1", "setpoint", "finite number for baja:Double")
	apRefusal(t, c, apArgs(apRow("R1", "label", pyjson.FloatNumber(3))), "label", "baja:String")
}

// A copy that leaves the templateConfig out (a transient slot) would keep every binding on the golden.
func TestApplyRefusesACopyBodyWithoutTheBindingsItMustRepoint(t *testing.T) {
	c := apStation(apGolden())
	c.types.(*pyjson.Object).Set("baja:Folder", obj("p", "baja:Component", "c", []any{
		obj("st", "p", "n", "templateConfig", "f", "t", "v", obj("nm", "p", "t", "template:TemplateConfig"))}))
	apRefusal(t, c, apArgs(apRow("R1")), "lacks a ConfigBinding", "keep pointing at the golden")
}
