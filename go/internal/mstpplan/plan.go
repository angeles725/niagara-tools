// Package mstpplan plans BACnet MS/TP addresses for a run of controllers set by DIP switches:
// each device's MAC, the switches to turn ON and its device ID, refusing a MAC that is reserved,
// above maxMaster, beyond what the switches can set, or already used (field retro 2026-11-16 U9:
// the table was computed by hand three times in one job).
package mstpplan

import "fmt"

// Options is one plan request.
type Options struct {
	FirstMAC, Count, Dip, IDBase, MaxMaster, LabelStart, ReservedMax int
	Label                                                            string
	Existing                                                         Existing
}

// Row is one planned device.
type Row struct {
	Label    string
	MAC      int
	Switches []int // switch numbers ON, highest first (switch n has weight 2^(n-1))
	DeviceID int
}

// Build returns the rows and every conflict found (all of them, so one run lists every fix).
func Build(o Options) ([]Row, []string) {
	var rows []Row
	var conflicts []string
	maxDip := 1<<o.Dip - 1
	for i := 0; i < o.Count; i++ {
		mac := o.FirstMAC + i
		id := o.IDBase + mac
		switch {
		case mac <= o.ReservedMax:
			conflicts = append(conflicts, fmt.Sprintf("MAC %d is reserved (0-%d: the JACE and spare masters)", mac, o.ReservedMax))
		case mac > o.MaxMaster:
			conflicts = append(conflicts, fmt.Sprintf("MAC %d is above maxMaster %d (raise maxMaster on the trunk or use a lower MAC)", mac, o.MaxMaster))
		case mac > maxDip:
			conflicts = append(conflicts, fmt.Sprintf("MAC %d needs more than %d DIP switches (max %d)", mac, o.Dip, maxDip))
		}
		if o.Existing.MACs[mac] {
			conflicts = append(conflicts, fmt.Sprintf("MAC %d is already used by an existing device", mac))
		}
		if o.Existing.IDs[id] {
			conflicts = append(conflicts, fmt.Sprintf("device ID %d is already used by an existing device", id))
		}
		rows = append(rows, Row{fmt.Sprintf("%s %d", o.Label, o.LabelStart+i), mac, switchesOn(mac, o.Dip), id})
	}
	return rows, conflicts
}

// switchesOn lists the switches set for mac, highest first.
func switchesOn(mac, dip int) []int {
	var on []int
	for n := dip; n >= 1; n-- {
		if mac&(1<<(n-1)) != 0 {
			on = append(on, n)
		}
	}
	return on
}
