package spy

import (
	"reflect"
	"testing"
)

func shxInt(n int) *int { return &n }

func TestReadCapacityReadsCountsAndUnlimitedLimits(t *testing.T) {
	c := ReadCapacity(shxParse(t, shxFixture(t, "metrics.html"), 0))
	if c == nil || len(c.Resources) != 6 {
		t.Fatalf("capacity = %+v", c)
	}
	want := Resource{Type: "Points", Used: shxInt(36), UsedText: "36", Limit: nil, LimitText: "none"}
	if !reflect.DeepEqual(c.Resources[2], want) || !NoLimit(c.Resources[2].LimitText) {
		t.Errorf("points = %+v", c.Resources[2])
	}
	if !reflect.DeepEqual(c.Recount, [][2]string{{"recountLastRun", "never"}, {"recountLastFail", "never"},
		{"recountLastFailReason", ""}}) {
		t.Errorf("recount = %q", c.Recount)
	}
}

// The licensed page is the recorded one with limits written in by hand: no licensed station was
// measured, so it pins the reader's rule (grouped digits are a count), not a station's format.
func TestReadCapacityReadsLimitsWithGroupedDigits(t *testing.T) {
	c := ReadCapacity(shxParse(t, shxFixture(t, "metrics-licensed.html"), 0))
	if c == nil {
		t.Fatal("no capacity")
	}
	points, devices := c.Resources[2], c.Resources[1]
	if *points.Limit != 1250 || *points.Used != 1036 || *devices.Limit != 50 || NoLimit(points.LimitText) {
		t.Errorf("points %+v, devices %+v", points, devices)
	}
}

func TestReadCapacityIsNilWithoutACapacityTable(t *testing.T) {
	for _, body := range []string{shxFixture(t, "metrics-no-capacity.html"), shxFixture(t, "index.html"), "",
		"<table><tr><th>Global Capacity</th></tr><tr><th>Type</th><th>Limit</th><th>Used</th></tr></table>",
		"<table><tr><th>Global Capacity</th></tr><tr><th>Name</th><th>Value</th></tr><tr><td>Points</td><td>36</td></tr></table>"} {
		if c := ReadCapacity(shxParse(t, body, 0)); c != nil {
			t.Errorf("capacity of %.60q = %+v", body, c)
		}
	}
}

func TestCountReadsOnlyDigitGroups(t *testing.T) {
	for in, want := range map[string]*int{"36": shxInt(36), "1,250": shxInt(1250), "1.250.000": shxInt(1250000),
		"12 345": shxInt(12345), "0": shxInt(0), "none": nil, "": nil, "1.5": nil, "12,34": nil, "-3": nil,
		"1250,000": nil, "36 points": nil} {
		if got := count(in); !reflect.DeepEqual(got, want) {
			t.Errorf("count(%q) = %v, want %v", in, got, want)
		}
	}
}
