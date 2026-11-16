package scenario

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/angeles725/niagara-tools/go/internal/labsim/regmap"
)

// Load reads and validates the scenario file at path. The error is only for a
// file that cannot be read; what is wrong with its content is in faults, one
// message each, and the scenario may be used only when there are none.
func Load(path string) (f *File, faults []string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	f, faults = Parse(data)
	return f, faults, nil
}

// Parse reads a scenario, fills the fields it leaves out and validates it. A
// text that is not a scenario at all (bad JSON, an unknown field, a value of the
// wrong kind) is one fault and no File.
func Parse(data []byte) (*File, []string) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	f := &File{}
	if err := dec.Decode(f); err != nil {
		return nil, []string{fmt.Sprintf(msgNotValid, err)}
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, []string{fmt.Sprintf(msgNotValid, msgTrailing)}
	}
	fillDefaults(f)
	return f, Validate(f)
}

// fillDefaults gives every omitted field its documented value.
func fillDefaults(f *File) {
	if f.Listen == "" {
		f.Listen = DefaultListen
	}
	if f.Control == "" {
		f.Control = DefaultControl
	}
	if f.UnknownUnit == "" {
		f.UnknownUnit = UnknownSilent
	}
	if f.MaxConnections == 0 {
		f.MaxConnections = DefaultMaxConnections
	}
	if f.IdleTimeoutMS == 0 {
		f.IdleTimeoutMS = DefaultIdleTimeoutMS
	}
	for d := range f.Devices {
		for p := range f.Devices[d].Points {
			point := &f.Devices[d].Points[p]
			if point.Order == "" {
				point.Order = DefaultOrder
			}
			if point.Type != "" {
				continue
			}
			point.Type = DefaultRegisterType
			if t, ok := regmap.ParseTable(point.Table); ok && t.Bits() {
				point.Type = DefaultBitType
			}
		}
	}
	fillBACnet(f)
}
