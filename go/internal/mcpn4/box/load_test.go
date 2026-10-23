package box

import (
	"reflect"
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

func TestFlattenTypeErrors(t *testing.T) {
	for _, tt := range []struct{ name, tree, want string }{
		{"children not iterable", `{"s": null}`, "TypeError"},
		{"a child that is a string", `{"s": ["x"]}`, "TypeError"},
		{"a child without a name", `{"s": [{"t": "x"}]}`, "KeyError"},
		{"a numeric name", `{"s": [{"n": 5}]}`, "TypeError"},
		{"a nested numeric name", `{"s": [{"n": "a", "s": [{"n": 5}]}]}`, "TypeError"},
		{"an empty children string", `{"s": ""}`, ""},
	} {
		node := mustObject(t, tt.tree)
		err := flatten(node, "", pyjson.NewObject())
		got := ""
		if err != nil {
			got = err.(*PyError).Type
		}
		if got != tt.want {
			t.Errorf("%s: %v, want %q", tt.name, err, tt.want)
		}
	}
}
