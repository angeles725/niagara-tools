package scenario

// Every fault a scenario can have, as the text lab-sim prints for it. A message
// that concerns a device or a point is prefixed with where it is.
const (
	msgNotValid = "scenario is not valid: %v"
	msgTrailing = "text after the scenario object"

	msgNoDevices      = "no devices: a scenario needs at least one"
	msgListen         = "listen %q is not host:port with a port of 0-65535"
	msgControl        = "control %v: the control port has no authentication and never leaves this machine"
	msgUnknownUnit    = "unknown_unit %q is neither \"silent\" nor \"gateway_exception\""
	msgMaxConnections = "max_connections %d is outside 1-%d"
	msgIdleTimeout    = "idle_timeout_ms %d is negative"

	msgNoName        = "no name"
	msgNameUsed      = "name %q is already used"
	msgUnitRange     = "unit %d is outside %d-%d"
	msgUnitUsed      = "unit %d is already device %q"
	msgNoPoints      = "no points"
	msgFailAfter     = "fail_after_reads %d is negative"
	msgDelay         = "delay_ms %d is negative"
	msgOfflineWindow = "offline window %d-%d must start at 0 or later and end after it starts"

	msgTable         = "table %q is none of coil, discrete_input, input_register, holding_register"
	msgAddress       = "address %d leaves no room for %d register(s) within 0-65535"
	msgType          = "type %q is none of bit, int16, uint16, int32, uint32, float32"
	msgTypeTable     = "type %q does not fit table %q"
	msgOrder         = "order %q is none of ABCD, CDAB, BADC, DCBA"
	msgReadOnlyTable = "writable, but table %q is read-only in the protocol"
	msgValue         = "value %v does not fit %s"
	msgOverlap       = "addresses %d-%d of table %q overlaps %q"

	msgBehaviourKind     = "behaviour %q is none of constant, ramp, sine, random_walk"
	msgBehaviourWritable = "a writable point cannot have behaviour %q: a write would be overwritten"
	msgBehaviourBit      = "a bit cannot have behaviour %q"
	msgBehaviourRange    = "behaviour %s runs over %v to %v, which does not fit %s"
	msgRampBounds        = "ramp needs min below max"
	msgRampPeriod        = "ramp needs period_ms above 0"
	msgSinePeriod        = "sine needs period_ms above 0"
	msgSineAmplitude     = "sine needs amplitude above 0"
	msgWalkSeed          = "random_walk needs a seed"
	msgWalkStep          = "random_walk needs step above 0"
	msgWalkInterval      = "random_walk needs interval_ms above 0"
	msgWalkBounds        = "random_walk needs min below max"
	msgWalkStart         = "random_walk needs value within min-max"
)

// The faults of the BACnet sections; each is prefixed with its place and
// "bacnet".
const (
	msgNoDeviceInstance   = "no device_instance"
	msgDeviceInstance     = "device_instance %d is outside 0-%d"
	msgDeviceInstanceUsed = "device_instance %d is already device %q"
	msgBACnetListen       = "listen %q is not an IPv4 address and a port (BACnet/IP is UDP over IPv4)"
	msgBACnetListenUsed   = "listen %q is already device %q: BACnet/IP has one device per address"
	msgVendorID           = "vendor_id %d is outside 0-%d"
	msgMaxAPDU            = "max_apdu %d is none of 50, 128, 206, 480, 1024, 1476"
	msgBACnetNoDevice     = "the point is mapped but its device has no bacnet section"

	msgObjectType       = "object_type %q is none of %s"
	msgNoInstance       = "no instance"
	msgInstance         = "instance %d is outside 0-%d"
	msgObjectUsed       = "%s %d is already point %q"
	msgCommandableInput = "an input (%s) cannot be commandable"
	msgUnits            = "units %q is neither a unit name (degrees-celsius, percent, ...) nor a number 0-%d"
	msgUnitsKind        = "units belong to analog objects, not to %s"
	msgAnalogOnly       = "min_pres_value, max_pres_value, resolution and cov_increment belong to analog objects, not to %s"
	msgMinAboveMax      = "min_pres_value %v is above max_pres_value %v"
	msgCOVIncrement     = "cov_increment %v is negative"
	msgBinaryTexts      = "active_text and inactive_text belong to binary objects, not to %s"
	msgStatesKind       = "state_texts belong to multi-state objects, not to %s"
	msgNoStates         = "%s needs state_texts: one text per state"
	msgStateValue       = "value %v is not one of the %d states (1-%d)"
)
