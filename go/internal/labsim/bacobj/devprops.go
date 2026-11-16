package bacobj

import "github.com/angeles725/niagara-tools/go/internal/labsim/bacnet"

// What the Device object states about a simulator that never retries and has
// one fixed configuration.
const (
	apduTimeoutMS    = 3000
	apduRetries      = 0
	databaseRevision = 1
)

func text(pick func(DeviceSpec) string) func(reading) []bacnet.Value {
	return func(r reading) []bacnet.Value { return one(bacnet.Text(pick(r.d.spec))) }
}

func number(n uint32) func(reading) []bacnet.Value {
	return func(reading) []bacnet.Value { return one(bacnet.Unsigned(n)) }
}

// deviceTable are the properties of the Device object, in the order ALL gives
// them.
var deviceTable = []property{
	{id: bacnet.PropObjectIdentifier, required: true, get: func(r reading) []bacnet.Value { return one(bacnet.OID(r.d.id)) }},
	{id: bacnet.PropObjectName, required: true, get: text(func(s DeviceSpec) string { return s.Name })},
	{id: bacnet.PropObjectType, required: true, get: func(reading) []bacnet.Value { return one(bacnet.Enum(uint32(bacnet.Device))) }},
	{id: bacnet.PropSystemStatus, required: true, get: func(reading) []bacnet.Value { return one(bacnet.Enum(bacnet.StatusOperational)) }},
	{id: bacnet.PropVendorName, required: true, get: text(func(s DeviceSpec) string { return s.VendorName })},
	{id: bacnet.PropVendorIdentifier, required: true, get: func(r reading) []bacnet.Value { return one(bacnet.Unsigned(r.d.spec.VendorID)) }},
	{id: bacnet.PropModelName, required: true, get: text(func(s DeviceSpec) string { return s.Model })},
	{id: bacnet.PropFirmwareRevision, required: true, get: text(func(s DeviceSpec) string { return s.Firmware })},
	{id: bacnet.PropApplicationVersion, required: true, get: text(func(s DeviceSpec) string { return s.Application })},
	{id: bacnet.PropProtocolVersion, required: true, get: number(bacnet.ProtocolVersion)},
	{id: bacnet.PropProtocolRevision, required: true, get: number(bacnet.ProtocolRevision)},
	{id: bacnet.PropServicesSupported, required: true, get: func(r reading) []bacnet.Value {
		bits := make([]bool, bacnet.ServicesSupportedBits)
		for _, bit := range r.d.supported {
			bits[bit] = true
		}
		return one(bacnet.Bits(bits...))
	}},
	{id: bacnet.PropObjectTypesSupported, required: true, get: func(reading) []bacnet.Value { return one(bacnet.Bits(supportedTypes()...)) }},
	{id: bacnet.PropObjectList, required: true, array: true, get: func(r reading) []bacnet.Value {
		out := make([]bacnet.Value, 0, len(r.d.order))
		for _, id := range r.d.order {
			out = append(out, bacnet.OID(id))
		}
		return out
	}},
	{id: bacnet.PropMaxAPDULength, required: true, get: func(r reading) []bacnet.Value { return one(bacnet.Unsigned(uint32(r.d.spec.MaxAPDU))) }},
	{id: bacnet.PropSegmentationSupported, required: true, get: func(reading) []bacnet.Value { return one(bacnet.Enum(bacnet.SegmentationNone)) }},
	{id: bacnet.PropAPDUTimeout, required: true, get: number(apduTimeoutMS)},
	{id: bacnet.PropNumberOfAPDURetries, required: true, get: number(apduRetries)},
	{id: bacnet.PropDeviceAddressBinding, required: true, get: func(reading) []bacnet.Value { return nil }}, // an empty list
	{id: bacnet.PropDatabaseRevision, required: true, get: number(databaseRevision)},
	{id: bacnet.PropDescription, get: text(func(s DeviceSpec) string { return s.Description })},
}
