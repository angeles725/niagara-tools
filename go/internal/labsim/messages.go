package labsim

// Exit codes.
const (
	exitOK     = 0
	exitFailed = 1
	exitUsage  = 2
)

// Every text lab-sim prints besides the register map and JSON.
const (
	msgNoCommand      = "lab-sim: no command (run, check, get, set, status, stop, probe, bacnet-probe; see lab-sim help)\n"
	msgUnknownCommand = "lab-sim: unknown command %q (see lab-sim help)\n"
	msgUsage          = "lab-sim %s: %v (see lab-sim help)\n"
	msgArgCount       = "wrong number of arguments"
	msgUnknownFlag    = "unknown option %s"
	msgFlagValue      = "option %s needs a value"

	msgCannotRead = "lab-sim: cannot read the scenario: %v\n"
	msgFault      = "lab-sim: %s: %s\n"
	msgFaultCount = "lab-sim: %s: %d faults, nothing served\n"
	msgFailed     = "lab-sim: %v\n"

	msgExposed   = "listen address %v; the scenario asks for it, but only an explicit --listen %s binds outside this machine"
	msgBadListen = "--listen %q is not host:port"
	msgNotNumber = "value %q is not a number"

	msgProbeFunction = "--fn must be one of 1, 2, 3, 4, 5, 6, 15, 16"
	msgProbeNumber   = "--%s %q is not a whole number in %d-%d"
	msgProbeCount    = "function %d reads: it needs --count and takes no --values"
	msgProbeValues   = "function %d writes: it needs --values and takes no --count"
	msgProbeOne      = "function %d writes exactly one value"
	msgProbeRequest  = "the request is not one a Modbus device accepts (exception %d): check the count and the address"
	msgProbeAnswer   = "no usable answer from %s within %d ms: %v"
)

// The rendering of a scenario by check.
const (
	checkHeader   = "DEVICE\tUNIT\tPOINT\tTABLE\tADDRESS\tREFERENCE\tTYPE\tORDER\tACCESS\tINITIAL\tBEHAVIOUR"
	checkScenario = "scenario: %s\n"
	checkTop      = "listen: %s  control: %s  unknown unit: %s\n"
	accessWrite   = "read/write"
	accessRead    = "read-only"
	textConstant  = "constant"
	textRamp      = "ramp %s..%s / %d ms"
	textSine      = "sine +/-%s / %d ms"
	textWalk      = "random_walk +/-%s / %d ms within %s..%s, seed %d"
	textFailAfter = "fails after %d reads"
	textDelayAll  = "every response held %d ms"
	textDelay     = "held %d ms"
	textOffline   = "offline %d-%d ms"
	textDevice    = "%s (unit %d)"
	textPoint     = "%s (unit %d), point %s"
)

// The texts of the BACnet front end.
const (
	msgBadBroadcast    = "--bacnet-broadcast %q is not an IPv4 address and a port"
	msgBadBACnetListen = "--bacnet-listen address %q is not an IPv4 address and a port"
	msgBACnetWhich     = "--bacnet-listen %q names no device and the scenario has %d BACnet devices: write device=host:port"
	msgBACnetNoDevice  = "--bacnet-listen names %q, which is not a BACnet device of the scenario"
	msgBACnetTwice     = "--bacnet-listen names %q twice"
	msgBACnetExposed   = "BACnet device %q: listen address %v; the scenario asks for it, but only an explicit --bacnet-listen %s=%s binds outside this machine"
	msgBACnetBind      = "BACnet device %q: %v"
)

// The rendering of the BACnet devices by check.
const (
	checkBACnetDevice = "BACnet device %s: instance %d, listen %s, vendor %d %q, model %q, max APDU %d, COV %s\n"
	checkObjectHeader = "DEVICE\tOBJECT TYPE\tINSTANCE\tNAME\tUNITS\tCOMMANDABLE\tACCESS"
	textOn            = "on"
	textOff           = "off"
	textYes           = "yes"
	textNo            = "no"
)

// The texts of bacnet-probe.
const (
	msgProbeAction    = "the action %q is none of who-is, read, read-multiple, write, subscribe"
	msgProbeArgs      = "%s takes %s"
	msgProbeObject    = "object %q is not type,instance with a type of %s and an instance of 0-4194303"
	msgProbeProperty  = "property %q is not a property name (present-value, object-list, units, ...; all, required and optional in read-multiple)"
	msgProbeIndex     = "array index %q is not a whole number from 0"
	msgProbeRange     = "the range %s-%s is not two device instances, low then high"
	msgProbeValue     = "value %q cannot be written to %s: %s"
	msgProbePointType = "write needs an analog, binary or multi-state object: its type gives the type of the value"
	msgProbeMode      = "--mode %q is none of unconfirmed, confirmed, cancel"
	msgProbeMaxAPDU   = "--max-apdu %q is none of 50, 128, 206, 480, 1024, 1476"
	msgProbeNoAnswer  = "no answer from %s within %d ms: %v"
	wantAnalog        = "an analog value is a number or null"
	wantBinary        = "a binary value is 0, 1 or null"
	wantMultiState    = "a multi-state value is a state number from 1 or null"
	argsWhoIs         = "no argument, or the lowest and highest device instance"
	argsRead          = "an object, a property and optionally an array index"
	argsReadMultiple  = "an object and a list of properties separated by commas"
	argsWrite         = "an object and a value"
	argsSubscribe     = "an object"
)
