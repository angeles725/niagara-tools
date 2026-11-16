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
