package scenario

// The values a BACnet section gets when it leaves a field out.
const (
	DefaultBACnetListen = "127.0.0.1:47809" // not 47808: a station on this machine holds that port
	DefaultVendorID     = 999
	DefaultVendorName   = "lab-sim"
	DefaultModel        = "lab-sim"
	DefaultRevision     = "1"
	DefaultMaxAPDU      = 1476
	DefaultUnits        = "no-units"
)

// Limits of the BACnet fields.
const (
	MaxInstance = 4194302
	MaxVendorID = 65535
)

// BACnetDevice makes a device a BACnet/IP device too, behind its own UDP
// socket: BACnet/IP has one device per address.
type BACnetDevice struct {
	DeviceInstance *int   `json:"device_instance"`
	Listen         string `json:"listen,omitempty"`
	VendorID       int    `json:"vendor_id,omitempty"`
	VendorName     string `json:"vendor_name,omitempty"`
	Model          string `json:"model,omitempty"`
	Firmware       string `json:"firmware,omitempty"`
	Application    string `json:"application,omitempty"`
	Description    string `json:"description,omitempty"`
	MaxAPDU        int    `json:"max_apdu,omitempty"`
	COV            bool   `json:"cov,omitempty"`
}

// BACnetPoint maps a point to a BACnet object of its device.
type BACnetPoint struct {
	ObjectType   string   `json:"object_type"`
	Instance     *int     `json:"instance"`
	Description  string   `json:"description,omitempty"`
	Units        string   `json:"units,omitempty"`
	Commandable  bool     `json:"commandable,omitempty"`
	ActiveText   string   `json:"active_text,omitempty"`
	InactiveText string   `json:"inactive_text,omitempty"`
	StateTexts   []string `json:"state_texts,omitempty"`
	Min          *float64 `json:"min_pres_value,omitempty"`
	Max          *float64 `json:"max_pres_value,omitempty"`
	Resolution   *float64 `json:"resolution,omitempty"`
	COVIncrement float64  `json:"cov_increment,omitempty"`
}

// HasBACnet reports whether any device of the scenario is a BACnet device.
func (f *File) HasBACnet() bool {
	for i := range f.Devices {
		if f.Devices[i].BACnet != nil {
			return true
		}
	}
	return false
}

func orText(value *string, fallback string) {
	if *value == "" {
		*value = fallback
	}
}

// fillBACnet gives the omitted BACnet fields their documented values.
func fillBACnet(f *File) {
	for d := range f.Devices {
		device := &f.Devices[d]
		if b := device.BACnet; b != nil {
			orText(&b.Listen, DefaultBACnetListen)
			orText(&b.VendorName, DefaultVendorName)
			orText(&b.Model, DefaultModel)
			orText(&b.Firmware, DefaultRevision)
			orText(&b.Application, DefaultRevision)
			if b.VendorID == 0 {
				b.VendorID = DefaultVendorID
			}
			if b.MaxAPDU == 0 {
				b.MaxAPDU = DefaultMaxAPDU
			}
		}
		for p := range device.Points {
			if b := device.Points[p].BACnet; b != nil && isAnalog(b.ObjectType) {
				orText(&b.Units, DefaultUnits)
			}
		}
	}
}
