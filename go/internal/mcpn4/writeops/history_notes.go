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

// capacityNote is how long the planned history lasts: records × the sampling interval.
func capacityNote(s historySpec) string {
	ms, _ := strconv.ParseFloat(s.interval, 64)
	return fmt.Sprintf(noteHistoryDefaultCapacity, readableDuration(defaultHistoryCapacity*ms), readableDuration(ms))
}
