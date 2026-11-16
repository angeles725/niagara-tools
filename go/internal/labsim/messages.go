package labsim

// Exit codes.
const (
	exitOK     = 0
	exitFailed = 1
	exitUsage  = 2
)

// Every text lab-sim prints besides the register map and JSON.
const (
	msgNoCommand      = "lab-sim: no command (run, check, get, set, status, stop, probe; see lab-sim help)\n"
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
