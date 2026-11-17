package writeops

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/capacity"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
)

// The device check of n4_create_component (data). A device over the licence's device.limit is
// added but never starts (it sits in fault), so the dry run refuses it before a token.
const (
	deviceType        = "driver:Device"
	licenceTimeout    = 60
	msgDeviceOverMax  = "creating %s would exceed the licence's globalCapacity device.limit %d: the station already counts %d device(s) (modules not counted: %s), so a new one would sit in fault at start; free a device or raise the limit first (n4_license_info shows the usage)"
	noteLicenceUnread = "The licence device limit was not checked (the licence manager page or the device count could not be read); this create is not refused on it."
)

// deviceWithinLicence refuses a device type (a driver:Device subtype, as the type registry
// describes it) whose creation would take the counted devices over the global device.limit.
// A non-device type, or a station whose registry cannot say, is not checked. An unreadable
// licence or count is never a refusal: it is a fixed plan note (no counts in the note, so the
// plan hash does not move between the dry run and the execute).
func deviceWithinLicence(c plan.Client, contracts *plan.Contracts, typ string) (string, error) {
	if contracts == nil {
		return "", nil
	}
	isDevice, err := contracts.Is(typ, deviceType)
	if err != nil || !isDevice {
		return "", nil
	}
	s, _ := c.(capacity.SpyGetter)
	g, _ := c.(capacity.OrdGetter)
	module, _, _ := strings.Cut(typ, ":")
	u, over, why := capacity.DeviceAdd(s, g, module, licenceTimeout)
	switch {
	case over:
		excluded := "none"
		if len(u.Excluded) > 0 {
			excluded = strings.Join(u.Excluded, ", ")
		}
		return "", plan.ToolError(fmt.Sprintf(msgDeviceOverMax, typ, *u.Limit, *u.Used, excluded))
	case u == nil || why != "" && !u.Unlimited && (u.Used == nil || u.Limit == nil):
		return noteLicenceUnread, nil
	}
	return "", nil
}
