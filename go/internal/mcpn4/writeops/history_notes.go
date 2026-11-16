package writeops

import (
	"fmt"
	"math"
	"strconv"
)

// defaultHistoryCapacity is the record count of a new extension's history (history-rt
// BHistoryConfig.capacity default: BCapacity.makeByRecordCount(500)).
const defaultHistoryCapacity = 500

// durationUnits are the readable units, largest first, with their length in milliseconds.
var durationUnits = []struct {
	name string
	ms   float64
}{{"d", 86400000}, {"h", 3600000}, {"min", 60000}, {"s", 1000}, {"ms", 1}}

// readableDuration is ms in the largest unit it fills at least twice (so 41.7 h, not 1.7 d),
// rounded to one decimal: 100 ms, 50 s, 5 min, 41.7 h, 70 d.
func readableDuration(ms float64) string {
	for _, u := range durationUnits {
		if ms >= 2*u.ms || u.name == "ms" {
			v := math.Round(ms/u.ms*10) / 10
			return strconv.FormatFloat(v, 'f', -1, 64) + " " + u.name
		}
	}
	return ""
}

// capacityNote is how long the planned history lasts: records × the sampling interval for an
// interval extension; a change-of-value history lasts as long as its changes take.
func capacityNote(s historySpec) string {
	records := s.records
	if records == "" {
		records = strconv.Itoa(defaultHistoryCapacity)
	}
	switch {
	case s.capacity == "0:0":
		return noteHistoryUnlimited
	case s.mode == "Cov" && s.capacity == "":
		return fmt.Sprintf(noteHistoryCovCapacity, noteHistoryCovDefault, records)
	case s.mode == "Cov":
		return fmt.Sprintf(noteHistoryCovCapacity, "Capacity "+records+" records (Roll)", records)
	}
	ms, _ := strconv.ParseFloat(s.interval, 64)
	n, _ := strconv.ParseFloat(records, 64)
	if s.capacity == "" {
		return fmt.Sprintf(noteHistoryDefaultCapacity, readableDuration(n*ms), readableDuration(ms))
	}
	return fmt.Sprintf(noteHistoryCapacity, records, readableDuration(n*ms), readableDuration(ms))
}
