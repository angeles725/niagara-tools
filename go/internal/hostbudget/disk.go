package hostbudget

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// diskRows checks "/" and every mount once, in order. With a free-space floor (DiskMinFree > 0) the
// floor alone decides each row: FAIL below it, PASS otherwise. Without one the percent rule applies.
// The floor is the user's rule of 2026-11-16 (first 40 GB free, then "ahora tu cuida que no sobrepasen de los 101",
// which scripts/ci-local.sh passes as 101G): a 1 TB C: at 90%
// used still has 90 GB free. (retro 2026-11-15 coordination-stretch T2)
func diskRows(c Config, p Probe) []Row {
	seen := map[string]bool{}
	var rows []Row
	for _, m := range append([]string{"/"}, c.Mounts...) {
		if seen[m] {
			continue
		}
		seen[m] = true
		res := "disk:" + m
		d, err := p.Statfs(m)
		if err != nil {
			rows = append(rows, Row{Warn, res, "?", "?", "(statfs: " + err.Error() + ")"})
			continue
		}
		size := d.Used + d.Avail
		if c.DiskMinFree > 0 {
			st := Pass
			if d.Avail < c.DiskMinFree {
				st = Fail
			}
			rows = append(rows, Row{st, res, gbString(d.Used), gbString(size),
				fmt.Sprintf("(%s free = %d bytes, fail below %s free)", gbString(d.Avail), d.Avail, gbString(c.DiskMinFree))})
			continue
		}
		st := classify(d.Used, size, c.DiskPct, WarnDiskPct)
		rows = append(rows, Row{st, res, gbString(d.Used), gbString(size),
			fmt.Sprintf("(%d%% used, fail above %d%%)", pct(d.Used, size), c.DiskPct)})
	}
	return rows
}

// sizeUnits are the decimal suffixes parseSize reads (an optional trailing B is allowed): the same
// decimal GB the rows print.
var sizeUnits = map[string]float64{"": 1, "K": 1e3, "M": 1e6, "G": 1e9, "T": 1e12}

// parseSize reads a byte count such as 40G, 40GB, 1.5G, 500M or 123 (case-insensitive).
func parseSize(s string) (uint64, error) {
	u := strings.ToUpper(s)
	u = strings.TrimSuffix(u, "B")
	num := strings.TrimRight(u, "KMGT")
	unit := u[len(num):]
	mult, ok := sizeUnits[unit]
	if !ok || num == "" {
		return 0, errors.New("not a size (examples: 40G, 500M, 123)")
	}
	v, err := strconv.ParseFloat(num, 64)
	if err != nil || v < 0 || strings.ContainsAny(num, "+-eEnN ") {
		return 0, errors.New("not a size (examples: 40G, 500M, 123)")
	}
	return uint64(v * mult), nil
}
