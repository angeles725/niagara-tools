package mstpplan

import (
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// Exit codes of mstp-plan.
const (
	ExitOK       = 0 // the plan has no conflict
	ExitConflict = 1 // a reserved, out-of-range or used MAC / device ID (all listed on stderr)
	ExitUsage    = 2 // bad arguments or an unreadable --existing file
)

const usage = `usage: mstp-plan --first-mac N --count N [--dip 7] [--id-base 2000] [--max-master 127]
       [--reserved-max 3] [--label Device] [--label-start 1] [--existing mac_device_id.csv] [--format md|csv]
Plans MS/TP MACs set by DIP switches (switch n = 2^(n-1)): MAC, switches ON, device ID = id-base + MAC.
`

// Main implements mstp-plan.
func Main(args []string, s cli.Stdio) int {
	fs := flag.NewFlagSet("mstp-plan", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	o := Options{}
	fs.IntVar(&o.FirstMAC, "first-mac", -1, "")
	fs.IntVar(&o.Count, "count", 0, "")
	fs.IntVar(&o.Dip, "dip", 7, "")
	fs.IntVar(&o.IDBase, "id-base", 2000, "")
	fs.IntVar(&o.MaxMaster, "max-master", 127, "")
	fs.IntVar(&o.ReservedMax, "reserved-max", 3, "")
	fs.IntVar(&o.LabelStart, "label-start", 1, "")
	fs.StringVar(&o.Label, "label", "Device", "")
	existing := fs.String("existing", "", "")
	format := fs.String("format", "md", "")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 || o.FirstMAC < 0 || o.Count < 1 || o.Dip < 1 ||
		o.Dip > 8 || (*format != "md" && *format != "csv") {
		fmt.Fprint(s.Err, usage)
		return ExitUsage
	}
	var err error
	if o.Existing, err = ReadExisting(*existing); err != nil {
		fmt.Fprintf(s.Err, "mstp-plan: %v\n", err)
		return ExitUsage
	}
	rows, conflicts := Build(o)
	if len(conflicts) > 0 {
		for _, c := range conflicts {
			fmt.Fprintf(s.Err, "mstp-plan: %s\n", c)
		}
		return ExitConflict
	}
	write(s.Out, rows, *format)
	return ExitOK
}

func write(w io.Writer, rows []Row, format string) {
	if format == "csv" {
		fmt.Fprintln(w, "label,mac,switches_on,device_id")
	} else {
		fmt.Fprint(w, "| Label | MAC | Switches ON | Device ID |\n|---|---|---|---|\n")
	}
	for _, r := range rows {
		on := make([]string, len(r.Switches))
		for i, n := range r.Switches {
			on[i] = strconv.Itoa(n)
		}
		if format == "csv" {
			fmt.Fprintf(w, "%s,%d,%s,%d\n", r.Label, r.MAC, strings.Join(on, " "), r.DeviceID)
		} else {
			fmt.Fprintf(w, "| %s | %d | %s | %d |\n", r.Label, r.MAC, strings.Join(on, ", "), r.DeviceID)
		}
	}
}
