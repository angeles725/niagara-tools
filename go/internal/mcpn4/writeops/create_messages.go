package writeops

// The dry-run refusals of n4_create_component that come before a token (bench retro 2026-11-15
// delta 4).
const (
	// msgUnknownType names a type spec the station's registry does not describe, and the route
	// when the module is loaded anyway: a module jar removed and re-added under the same name can
	// leave the registry "Up-to-date" without its types (client A retro 2026-11-16 rows M12, M14).
	msgUnknownType = "type %s is not known to the station: its type registry does not describe it " +
		"(check the module and type names, and that the module is installed on the station). " +
		"If the module is installed and running, the station's registry may be stale: stop the station, " +
		"touch the module jar, start it until its log shows \"Rebuilding registry\", " +
		"then reconnect with n4_connect"
	// msgNameTaken is the first dry-run note when the name is taken under the parent: not a
	// refusal, the station renames on collision (a pinned contract) and the inverse follows it.
	msgNameTaken = "name %s is already taken under %s: the station will assign another name; " +
		"the read-back and the inverse follow it"
)
