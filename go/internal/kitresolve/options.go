package kitresolve

// options is the kit-resolve command line.
type options struct{ refresh, offlineOK, help bool }

// parseArgs reads the arguments left to right; ok is false on an unknown one. --help ends the reading
// where it stands, as the shell loop did.
func parseArgs(args []string) (o options, ok bool) {
	for _, arg := range args {
		switch arg {
		case "--refresh":
			o.refresh = true
		case "--offline-ok":
			o.offlineOK = true
		case "-h", "--help":
			o.help = true
			return o, true
		default:
			return o, false
		}
	}
	return o, true
}
