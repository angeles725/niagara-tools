package capacity

import (
	"reflect"
	"testing"
)

func n(v int) *int { return &v }

func TestHeadroomIsLimitLessUsedAndNullWhenEitherIsUnknown(t *testing.T) {
	cases := []struct {
		u    Usage
		want *int
	}{
		{Usage{Rule: "devices", Used: n(20), Limit: n(26)}, n(6)},
		{Usage{Rule: "devices", Used: n(38), Limit: n(26)}, n(-12)},
		{Usage{Rule: "devices", Used: nil, Limit: n(26)}, nil},
		{Usage{Rule: "devices", Used: n(3), Unlimited: true}, nil},
	}
	for _, c := range cases {
		if got := c.u.Headroom(); !reflect.DeepEqual(got, c.want) {
			t.Errorf("Headroom(%+v) = %v, want %v", c.u, got, c.want)
		}
	}
}

func TestCheckFitsWhenEveryNamedRuleStaysWithinItsLimit(t *testing.T) {
	usages := []Usage{{Rule: "devices", Used: n(20), Limit: n(26)}, {Rule: "points", Used: n(400), Limit: n(500)}}
	got := Check(usages, map[string]int{"devices": 6, "points": 100})
	if got.Fits == nil || !*got.Fits || len(got.Over) != 0 {
		t.Fatalf("Check = %+v, want fits true, no over", got)
	}
}

func TestCheckIsOverByTheExcessOfEachRule(t *testing.T) {
	usages := []Usage{{Rule: "devices", Used: n(20), Limit: n(26)}, {Rule: "points", Used: n(400), Limit: n(500)}}
	got := Check(usages, map[string]int{"devices": 18, "points": 100})
	if got.Fits == nil || *got.Fits || !reflect.DeepEqual(got.Over, []Over{{Rule: "devices", By: 12}}) {
		t.Fatalf("Check = %+v, want fits false, devices over by 12", got)
	}
}

func TestCheckNeverFailsOnAnUnlimitedRule(t *testing.T) {
	got := Check([]Usage{{Rule: "devices", Used: n(5000), Unlimited: true}}, map[string]int{"devices": 1000})
	if got.Fits == nil || !*got.Fits {
		t.Fatalf("Check = %+v, want fits true", got)
	}
}

func TestCheckIsNullWhenACountIsUnknownAndNothingIsSurelyOver(t *testing.T) {
	usages := []Usage{{Rule: "devices", Used: nil, Limit: n(26), Note: "x"}, {Rule: "points", Used: n(1), Limit: n(5)}}
	if got := Check(usages, map[string]int{"devices": 1, "points": 1}); got.Fits != nil {
		t.Fatalf("Check = %+v, want fits null", got)
	}
	// a rule surely over decides false even beside an unknown one
	got := Check(usages, map[string]int{"devices": 1, "points": 9})
	if got.Fits == nil || *got.Fits || !reflect.DeepEqual(got.Over, []Over{{Rule: "points", By: 5}}) {
		t.Fatalf("Check = %+v, want fits false, points over by 5", got)
	}
}

func TestCheckIgnoresRulesThePlanDoesNotName(t *testing.T) {
	usages := []Usage{{Rule: "devices", Used: n(30), Limit: n(26)}, {Rule: "points", Used: n(1), Limit: n(5)}}
	got := Check(usages, map[string]int{"points": 2})
	if got.Fits == nil || !*got.Fits {
		t.Fatalf("Check = %+v, want fits true (devices not in the plan)", got)
	}
}
