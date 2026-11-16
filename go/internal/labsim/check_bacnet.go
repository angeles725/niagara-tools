package labsim

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/angeles725/niagara-tools/go/internal/labsim/bacnet"
	"github.com/angeles725/niagara-tools/go/internal/labsim/bacobj"
	"github.com/angeles725/niagara-tools/go/internal/labsim/scenario"
)

// printObjects prints, for a scenario with BACnet devices, what an integrator
// sees when a driver learns them: each device and its object list.
func printObjects(w io.Writer, f *scenario.File) {
	if !f.HasBACnet() {
		return
	}
	fmt.Fprintln(w)
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, checkObjectHeader)
	for i := range f.Devices {
		d := &f.Devices[i]
		b := d.BACnet
		if b == nil {
			continue
		}
		cov := textOff
		if b.COV {
			cov = textOn
		}
		fmt.Fprintf(w, checkBACnetDevice, d.Name, *b.DeviceInstance, b.Listen, b.VendorID, b.VendorName, b.Model, b.MaxAPDU, cov)
		row := func(cells ...string) { fmt.Fprintln(tw, strings.Join(append([]string{d.Name}, cells...), "\t")) }
		row(bacnet.Device.String(), strconv.Itoa(*b.DeviceInstance), d.Name, "-", "-", accessRead)
		for j := range d.Points {
			p := &d.Points[j]
			m := p.BACnet
			if m == nil {
				continue
			}
			id := p.ObjectID()
			units, commandable, access := "-", textNo, accessRead
			if kind, _ := bacobj.KindOf(id.Type); kind == bacobj.Analog {
				number, _ := bacobj.ParseUnits(m.Units)
				units = bacobj.UnitsName(number)
			}
			if m.Commandable {
				commandable = textYes
			}
			if m.Commandable || (p.Writable && !bacobj.IsInput(id.Type)) {
				access = accessWrite
			}
			row(m.ObjectType, strconv.Itoa(*m.Instance), p.Name, units, commandable, access)
		}
	}
	tw.Flush()
}
