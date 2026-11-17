package mstpplan

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Existing holds the MACs and device IDs already on the trunk.
type Existing struct {
	MACs, IDs map[int]bool
}

// ReadExisting reads a CSV with a `mac,device_id` header (either column may be empty); blank lines
// and lines starting with # are skipped. A BQL export reduced to those two columns fits.
func ReadExisting(path string) (Existing, error) {
	e := Existing{MACs: map[int]bool{}, IDs: map[int]bool{}}
	if path == "" {
		return e, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return e, fmt.Errorf("cannot read %s: %v", path, err)
	}
	header := false
	for n, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		cells := strings.Split(line, ",")
		if !header {
			if len(cells) != 2 || strings.TrimSpace(cells[0]) != "mac" || strings.TrimSpace(cells[1]) != "device_id" {
				return e, fmt.Errorf("%s:%d: header must be mac,device_id", path, n+1)
			}
			header = true
			continue
		}
		if len(cells) != 2 {
			return e, fmt.Errorf("%s:%d: want 2 cells, got %d", path, n+1, len(cells))
		}
		for i, set := range []map[int]bool{e.MACs, e.IDs} {
			c := strings.TrimSpace(cells[i])
			if c == "" {
				continue
			}
			v, err := strconv.Atoi(c)
			if err != nil {
				return e, fmt.Errorf("%s:%d: %q is not an integer", path, n+1, c)
			}
			set[v] = true
		}
	}
	return e, nil
}
