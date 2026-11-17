package bql

// The station reads of n4_schedule_audit (station-audits S12). schedule:ControlSchedule is the
// abstract parent of the boolean, numeric, enum and string schedules and of the calendar
// (schedule-rt module.xml; BWeeklySchedule and BCalendarSchedule extend BControlSchedule), so one
// query lists every schedule with an out slot; the trigger schedule extends
// BCompositeSchedule and is not listed.

// ScheduleAuditQuery reads, by position, each schedule's slot path and how many links read it:
// BComponent.getKnobCount, the outbound links of the component (a knob is a link's reflection
// on its source), which BQL lists as a field like every public no-argument getter.
const ScheduleAuditQuery = "select slotPath, knobCount from schedule:ControlSchedule"

// ScheduleListQuery reads the slot paths only: the fallback when a station refuses knobCount.
const ScheduleListQuery = "select slotPath from schedule:ControlSchedule"
