package writeops

// The dry-run refusals of n4_create_component that come before a token (bench retro 2026-11-15
// delta 4).
const (
	// msgUnknownType names a type spec the station's registry does not describe.
	msgUnknownType = "type %s is not known to the station: its type registry does not describe it " +
		"(check the module and type names, and that the module is installed on the station)"
)
