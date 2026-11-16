package modulemap

import "strings"

// opts are the parsed arguments of one subcommand.
type opts struct {
	cmd, target, root, part string
	adapter                 bool
}

func parse(args []string) (opts, error) {
	var o opts
	if len(args) == 0 {
		return o, fail(2, "%s", usage)
	}
	o.cmd, args = args[0], args[1:]
	switch o.cmd {
	case "add-concern", "where", "tests":
	default:
		return o, fail(2, "module-map: unknown subcommand %q\n%s", o.cmd, usage)
	}
	for i := 0; i < len(args); i++ {
		switch a := args[i]; a {
		case "--root", "--part":
			if i+1 >= len(args) {
				return o, fail(2, "module-map: %s needs a value\n%s", a, usage)
			}
			i++
			if a == "--root" {
				o.root = args[i]
			} else {
				o.part = args[i]
			}
		case "--adapter":
			o.adapter = true
		default:
			if strings.HasPrefix(a, "--") || o.target != "" {
				return o, fail(2, "module-map: unexpected argument %q\n%s", a, usage)
			}
			o.target = a
		}
	}
	if o.target == "" || o.root == "" {
		return o, fail(2, "%s", usage)
	}
	if o.cmd != "add-concern" && (o.part != "" || o.adapter) {
		return o, fail(2, "module-map: --part and --adapter belong to add-concern\n%s", usage)
	}
	return o, nil
}
