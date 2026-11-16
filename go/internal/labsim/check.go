package labsim

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/labsim/regmap"
	"github.com/angeles725/niagara-tools/go/internal/labsim/scenario"
)

// load reads a scenario; a file that cannot be read or has faults is reported
// on stderr, one line per fault, and ok is false.
func load(path string, s cli.Stdio) (f *scenario.File, ok bool) {
	f, faults, err := scenario.Load(path)
	if err != nil {
		fmt.Fprintf(s.Err, msgCannotRead, err)
		return nil, false
	}
	for _, fault := range faults {
		fmt.Fprintf(s.Err, msgFault, path, fault)
	}
	if len(faults) > 0 {
		fmt.Fprintf(s.Err, msgFaultCount, path, len(faults))
		return nil, false
	}
	return f, true
}

// checkCommand validates a scenario and prints its register map.
func checkCommand(p parsed, s cli.Stdio) (int, error) {
	f, ok := load(p.rest[0], s)
	if !ok {
		return exitFailed, nil
	}
	printMap(s.Out, f)
	return exitOK, nil
}

// printMap prints what an integrator types into a driver: one row per point.
func printMap(w io.Writer, f *scenario.File) {
	if f.Name != "" {
		fmt.Fprintf(w, checkScenario, f.Name)
	}
	fmt.Fprintf(w, checkTop, f.Listen, f.Control, f.UnknownUnit)
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, checkHeader)
	var notes []string
	for _, d := range f.Devices {
		if d.Faults != nil {
			notes = addNote(notes, fmt.Sprintf(textDevice, d.Name, d.Unit), deviceFaults(d.Faults))
		}
		for _, p := range d.Points {
			table, _ := regmap.ParseTable(p.Table)
			words := regmap.Type(p.Type).Words()
			order, access := p.Order, accessRead
			if table.Bits() {
				order = "-"
			}
			if p.Writable {
				access = accessWrite
			}
			fmt.Fprintln(tw, strings.Join([]string{
				d.Name, strconv.Itoa(d.Unit), p.Name, p.Table,
				span(p.Address, words, strconv.Itoa), span(p.Address, words, table.ReferenceOf),
				p.Type, order, access, num(p.Initial()), behaviourText(p.Behaviour),
			}, "\t"))
			if p.Faults != nil {
				notes = addNote(notes, fmt.Sprintf(textPoint, d.Name, d.Unit, p.Name), counters(p.Faults.FailAfterReads, p.Faults.DelayMS, textDelay))
			}
		}
	}
	tw.Flush()
	for _, note := range notes {
		fmt.Fprintln(w, note)
	}
}

// span renders the addresses of a point: one, or first-last.
func span(address, words int, render func(int) string) string {
	if words == 1 {
		return render(address)
	}
	return render(address) + "-" + render(address+words-1)
}

func num(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) }

func behaviourText(b *scenario.Behaviour) string {
	if b == nil {
		return textConstant
	}
	switch b.Kind {
	case scenario.KindRamp:
		return fmt.Sprintf(textRamp, num(*b.Min), num(*b.Max), b.PeriodMS)
	case scenario.KindSine:
		return fmt.Sprintf(textSine, num(b.Amplitude), b.PeriodMS)
	case scenario.KindRandomWalk:
		return fmt.Sprintf(textWalk, num(b.Step), b.IntervalMS, num(*b.Min), num(*b.Max), *b.Seed)
	}
	return textConstant
}

func addNote(notes []string, where string, parts []string) []string {
	if len(parts) == 0 {
		return notes
	}
	return append(notes, where+": "+strings.Join(parts, "; "))
}

func deviceFaults(f *scenario.Faults) []string {
	parts := counters(f.FailAfterReads, f.DelayMS, textDelayAll)
	for _, w := range f.Offline {
		parts = append(parts, fmt.Sprintf(textOffline, w.FromMS, w.ToMS))
	}
	return parts
}

func counters(failAfter *int, delayMS int, delayText string) []string {
	var parts []string
	if failAfter != nil {
		parts = append(parts, fmt.Sprintf(textFailAfter, *failAfter))
	}
	if delayMS > 0 {
		parts = append(parts, fmt.Sprintf(delayText, delayMS))
	}
	return parts
}
