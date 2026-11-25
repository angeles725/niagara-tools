package box

import (
	"fmt"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

func TestLoadManyOfNOrdsIsCeilNOverCapLoadFrames(t *testing.T) {
	for _, n := range []int{1, 2, 7, 50, 51, 120} {
		ls := newLoadStation(t)
		c := openClient(t, ls)
		ords := ordsOf(n)
		rs, err := c.LoadMany(ords, 1, LoadOptions{})
		if err != nil {
			t.Fatal(err)
		}
		checkNodes(t, ords, rs)
		if want := (n + MaxFrameMessages - 1) / MaxFrameMessages; ls.loadFrames != want || ls.loadMsgs != n {
			t.Errorf("%d ords: %d loadSlots frames carrying %d messages, want %d and %d",
				n, ls.loadFrames, ls.loadMsgs, want, n)
		}
		if c.LoadsOutstanding() != 0 {
			t.Errorf("%d ords: %d loads left outstanding", n, c.LoadsOutstanding())
		}
	}
}

func TestLoadManyTakesFewerRoundTripsThanLoadTreeEach(t *testing.T) {
	ords := ordsOf(10)
	one, many := newLoadStation(t), newLoadStation(t)
	c1, c2 := openClient(t, one), openClient(t, many)
	base1, base2 := c1.Frames(), c2.Frames()
	for _, o := range ords {
		if _, err := c1.LoadTree(o, 1, LoadOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := c2.LoadMany(ords, 1, LoadOptions{}); err != nil {
		t.Fatal(err)
	}
	if a, b := c1.Frames()-base1, c2.Frames()-base2; a != 30 || b != 3 {
		t.Fatalf("10 ords: LoadTree each %d frames, LoadMany %d; want 30 and 3 (poll, loadSlots, poll)", a, b)
	}
}

func TestLoadManyOneRefusedOrdIsItsOwnErrorAndTheSiblingsLoad(t *testing.T) {
	ls := newLoadStation(t)
	ords := ordsOf(3)
	ls.fail[ords[1]] = true
	c := openClient(t, ls)
	rs, err := c.LoadMany(ords, 1, LoadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	checkNodes(t, []string{ords[0]}, rs[:1])
	checkNodes(t, []string{ords[2]}, rs[2:])
	if e := boxErr(t, rs[1].Err, KindBox); !e.Station || e.Msg != "denied "+ords[1] || rs[1].Nodes != nil {
		t.Fatalf("refused entry: %+v, nodes %v", e, rs[1].Nodes)
	}
	if c.LoadsOutstanding() != 0 {
		t.Fatalf("%d loads outstanding", c.LoadsOutstanding())
	}
}

func TestLoadManyMatchesNoLoadToTheWrongOrd(t *testing.T) {
	ls := newLoadStation(t)
	ls.reorder = func(ops []any) []any {
		if len(ops) != 3 {
			return ops
		}
		return []any{ops[2], ops[0], ops[1]}
	}
	ords := ordsOf(3)
	c := openClient(t, ls)
	for _, o := range ords { // the handles are known: a load of another handle is told apart
		c.SetHandle(o, "h:"+o)
	}
	rs, err := c.LoadMany(ords, 1, LoadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	checkNodes(t, ords, rs)
}

func TestLoadManyOrdWithoutALoadEventFailsAsLoadTreeDoes(t *testing.T) {
	ls := newLoadStation(t)
	ords := ordsOf(2)
	ls.silent[ords[1]] = true
	c := openClient(t, ls)
	rs, err := c.LoadMany(ords, 1, LoadOptions{Attempts: 2})
	if err != nil {
		t.Fatal(err)
	}
	checkNodes(t, ords[:1], rs[:1])
	want := fmt.Sprintf("no load event for %s after 2 polls", ords[1])
	if e := boxErr(t, rs[1].Err, KindBox); !strings.HasSuffix(e.Msg, want) {
		t.Fatalf("second result: %v, want %q", rs[1].Err, want)
	}
}

func TestLoadManyRemembersTheHandlesLikeLoadTree(t *testing.T) {
	ls := newLoadStation(t)
	c := openClient(t, ls)
	ords := ordsOf(2)
	if _, err := c.LoadMany(ords, 1, LoadOptions{}); err != nil {
		t.Fatal(err)
	}
	if got := pyjson.Repr(c.Handles()); !strings.Contains(got, "'"+ords[1]+"': 'h:"+ords[1]+"'") {
		t.Fatalf("handles %s", got)
	}
}
