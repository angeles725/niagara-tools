package plan_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/journal"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// refusing is a station whose every send is answered with err and after which its loads fail: the
// read-back of the targets cannot be made.
type refusing struct {
	*station
	err error
}

func (f refusing) SSC(string, any) (any, error) {
	f.station.loadErr = &box.Error{Kind: box.KindBox, Msg: "read-back unavailable"}
	return nil, f.err
}

func (f refusing) Sync(any) (any, error) { return f.SSC("", nil) }

// refuse makes the rig's next send fail with err and its read-back after that fail.
func (r *rig) refuse(err error) {
	r.ctx.Session = session.NewSession(refusing{r.st, err}, "FakeStation", "http://fake.invalid", "2", "4.14.0.162", "")
}

// The station's own text for a link it refused before adding anything (box-rt
// BComponentSpaceSessionHandler.doLinkCheck, :658-659): every check runs before the first link is
// added (checkLinks, :617-640), so the text alone says nothing was applied.
const readonlyTarget = "Could not create link: Target property is readonly."

// A link the station refuses with that text is journaled as refused although the read-back of the
// target cannot be made: not in-doubt, with the station's text in the error.
func TestAStatedLinkRefusalIsRefusedEvenWhenTheReadbackFails(t *testing.T) {
	r := newRig(t)
	a := node("source_ord", calc, "source_slot", "out", "target_ord", sp, "target_slot", "in8")
	r.refuse(&box.Error{Kind: box.KindBox, Msg: readonlyTarget, Station: true})
	tok := r.dry(t, "n4_create_link", a)
	text, isErr, _ := r.execute(t, "n4_create_link", a, tok)
	lines := r.lines(t, "journal.jsonl")
	if len(lines) == 0 {
		t.Fatalf("nothing journaled: %q", text)
	}
	id := get(lines[0], "batch_id").(string)
	if !isErr || strings.Contains(text, "in-doubt") || !strings.Contains(text, "nothing was applied") ||
		!strings.Contains(text, readonlyTarget) || !strings.Contains(text, id) {
		t.Fatalf("%v %q", isErr, text)
	}
	view, err := journal.New(r.dir).Read(id)
	if err != nil || get(view, "state") != "refused" || get(view, "station_error") != readonlyTarget {
		t.Fatalf("view %v err %v", view, err)
	}
}

// Anything else stays in-doubt when the read-back cannot be made: another station text, the same
// words from a transport failure, and the same text on a tool whose op is not a link check.
func TestOnlyTheStationsStatedRefusalSkipsTheReadback(t *testing.T) {
	link := node("source_ord", calc, "source_slot", "out", "target_ord", sp, "target_slot", "in8")
	for name, e := range map[string]*box.Error{
		"another text":  {Kind: box.KindBox, Msg: "station error", Station: true},
		"not a station": {Kind: box.KindBox, Msg: readonlyTarget},
	} {
		r := newRig(t)
		r.refuse(e)
		tok := r.dry(t, "n4_create_link", link)
		text, isErr, _ := r.execute(t, "n4_create_link", link, tok)
		if !isErr || !strings.Contains(text, "in-doubt") {
			t.Errorf("%s: %q", name, text)
		}
	}
	r := newRig(t)
	r.refuse(&box.Error{Kind: box.KindBox, Msg: readonlyTarget, Station: true})
	tok := r.dry(t, "n4_set_slot", setArgs())
	if text, isErr, _ := r.execute(t, "n4_set_slot", setArgs(), tok); !isErr || !strings.Contains(text, "in-doubt") {
		t.Errorf("set_slot: %q", text)
	}
}
