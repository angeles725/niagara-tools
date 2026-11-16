package blackbox

// n4_commissioning_report, the alarms section.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

const comAlarmSvc = "slot:/Services/AlarmService/"

var comAlarmTables = map[string][]string{
	"points": {"slot:/Logic/Temp,Temp,Temp,control:NumericPoint,0", "slot:/Logic/Fan,Fan,Fan,control:BooleanWritable,0",
		"slot:/Logic/Sp,Sp,Sp,control:NumericWritable,0"},
	"alarms": {"slot:/Logic/Temp/OutOfRange,OutOfRange,alarm:AlarmSourceExt,High",
		"slot:/Logic/Sp/OutOfRange,OutOfRange,alarm:AlarmSourceExt,Critical"},
	"classes": {comAlarmSvc + "defaultAlarmClass,defaultAlarmClass,0,0", comAlarmSvc + "High,High,3,2", comAlarmSvc + "Low,Low,x,0"},
}

func TestCommissioningListsPointsWithoutAnAlarmExtensionOnlyWhereAsked(t *testing.T) {
	r, _ := comRig(t, comAlarmTables)
	comAlarmService(r, "defaultAlarmClass", "High", "Low")
	s := comSection(t, r, "alarms", nil)
	if why, _ := comCheck(t, s, "points_without_alarm")["skipped"].(string); !strings.Contains(why, "expect_alarms") {
		t.Errorf("without expect_alarms: %v", comCheck(t, s, "points_without_alarm"))
	}
	s = comSection(t, r, "alarms", Args{"expect_alarms": Args{}})
	comWant(t, s, "points_without_alarm", 1, "slot:/Logic/Fan")
	s = comSection(t, r, "alarms", Args{"expect_alarms": Args{"kinds": []any{"numeric"}}})
	comWant(t, s, "points_without_alarm", 0)
}

func TestCommissioningListsAlarmExtensionsWhoseClassDoesNotExist(t *testing.T) {
	r, sv := comRig(t, comAlarmTables)
	comAlarmService(r, "defaultAlarmClass", "High", "Low")
	s := comSection(t, r, "alarms", nil)
	ext := "slot:/Logic/Sp/OutOfRange"
	if fact := comFact(comWant(t, s, "unknown_alarm_class", 1, ext), ext); fact != "alarmClass Critical names no alarm class of the station" {
		t.Errorf("fact = %q", fact)
	}
	// without the class list nothing can be said of a class name
	tables := map[string][]string{"alarms": comAlarmTables["alarms"], "classes": {comFailed}}
	comRoutes(sv, tables)
	s = comSection(t, r, "alarms", nil)
	if why, _ := comCheck(t, s, "unknown_alarm_class")["not_checked"].(string); !strings.Contains(why, "404") {
		t.Errorf("unknown_alarm_class = %v", comCheck(t, s, "unknown_alarm_class"))
	}
}

func TestCommissioningListsAlarmClassesWithNoRecipientLinked(t *testing.T) {
	r, _ := comRig(t, comAlarmTables)
	h := comAlarmService(r, "defaultAlarmClass", "High", "Low")
	for _, class := range []string{"defaultAlarmClass", "High"} {
		if _, err := r.Station.Link(h[class], "alarm", h["Console"], "routeAlarm"); err != nil {
			t.Fatal(err)
		}
	}
	s := comSection(t, r, "alarms", nil)
	c := comWant(t, s, "classes_without_recipient", 1, comAlarmSvc+"Low")
	if fact := comFact(c, comAlarmSvc+"Low"); !strings.Contains(fact, "no link from its alarm topic") {
		t.Errorf("fact = %q", fact)
	}
	// a link whose source cannot be resolved may be the missing one: nothing is asserted
	r.Station.Do(func(st *boxstation.Station) {
		st.ByHandle(h["Console"]).Child("Link").Child("sourceOrd").Value = "virtual:/x"
	})
	s = comSection(t, r, "alarms", nil)
	if why, _ := comCheck(t, s, "classes_without_recipient")["not_checked"].(string); !strings.Contains(why, "unresolved") {
		t.Errorf("with an unresolved link: %v", comCheck(t, s, "classes_without_recipient"))
	}
}

func TestCommissioningCountsOpenAndUnacknowledgedAlarmsPerClass(t *testing.T) {
	r, _ := comRig(t, comAlarmTables)
	comAlarmService(r, "defaultAlarmClass", "High", "Low")
	s := comSection(t, r, "alarms", nil)
	open := comWant(t, s, "open_alarms", 1, comAlarmSvc+"High") // one class with open alarms
	unacked := comWant(t, s, "unacked_alarms", 1, comAlarmSvc+"High")
	if comFact(open, comAlarmSvc+"High") != "3 open alarm(s)" || comFact(unacked, comAlarmSvc+"High") != "2 unacknowledged alarm(s)" {
		t.Errorf("open %v unacked %v", open, unacked)
	}
	if open["alarms"] != 3.0 || unacked["alarms"] != 2.0 || open["not_answered"] != 1.0 || unacked["not_answered"] != nil { // Low answered no open count
		t.Errorf("not_answered: open %v unacked %v", open["not_answered"], unacked["not_answered"])
	}
}
