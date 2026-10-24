package box

import (
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

func TestPollDelaysAreAPureBoundedSchedule(t *testing.T) {
	for _, tt := range []struct {
		first, limit, total float64
		attempts            int
		want                []float64
	}{
		{0.1, 0.8, 3.0, 20, []float64{0.1, 0.2, 0.4, 0.8, 0.8, 0.7}},
		{0.1, 0.8, 3.0, 3, []float64{0.1, 0.2}},
		{0.1, 0.8, 3.0, 1, nil},
		{0.1, 0.8, 3.0, 0, nil},
		{0.25, 0.8, 3.0, 6, []float64{0.25, 0.5, 0.8, 0.8, 0.65}},
		{0.1, 0.8, 10.0, 17, []float64{0.1, 0.2, 0.4, 0.8, 0.8, 0.8, 0.8, 0.8, 0.8, 0.8, 0.8, 0.8, 0.8, 0.8, 0.5}}, // python3: box.poll_delays
		{0.1, 0.8, 0.0, 5, nil},
	} {
		if got := PollDelays(tt.first, tt.limit, tt.total, tt.attempts); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("PollDelays(%v, %v, %v, %d) = %v, want %v", tt.first, tt.limit, tt.total, tt.attempts, got, tt.want)
		}
	}
}

func TestDroppedEventsStartsAtZero(t *testing.T) {
	if c := newClient(t, "http://127.0.0.1:1", nil); c.DroppedEvents() != 0 || len(c.PendingEvents()) != 0 {
		t.Fatal("a new client holds events")
	}
}

func TestFlattenRefusesMalformedTrees(t *testing.T) {
	for _, tt := range []struct{ name, tree, want string }{
		{"children that are null", `{"s": null}`, "slot (root) has children that are null, not a list"},
		{"children that are a string", `{"s": ""}`, "slot (root) has children that are a string, not a list"},
		{"a child that is a string", `{"s": ["x"]}`, "slot (root) has a child that is a string, not an object"},
		{"a child without a name", `{"s": [{"t": "x"}]}`, "slot (root) has a child without a slot name"},
		{"a numeric name", `{"s": [{"n": 5}]}`, "slot (root) has a child without a slot name"},
		{"an empty name", `{"s": [{"n": ""}]}`, "slot (root) has a child without a slot name"},
		{"a nested numeric name", `{"s": [{"n": "a", "s": [{"n": 5}]}]}`, "slot a has a child without a slot name"},
		{"nested children that are a number", `{"s": [{"n": "a", "s": [{"n": "b", "s": 1}]}]}`,
			"slot a/b has children that are a number, not a list"},
		{"a well-formed tree", `{"s": [{"n": "a", "s": [{"n": "b"}]}]}`, ""},
	} {
		err := flatten(mustObject(t, tt.tree), "", pyjson.NewObject())
		got := ""
		if err != nil {
			e := err.(*Error)
			if e.Kind != KindBox || e.Channel != Channel || e.Key != "loadSlots" {
				t.Errorf("%s: %#v", tt.name, e)
			}
			got = strings.TrimPrefix(e.Msg, "malformed BOX reply: ")
		}
		if got != tt.want {
			t.Errorf("%s: %q, want %q", tt.name, got, tt.want)
		}
	}
}
