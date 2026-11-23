package retroissues

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/retrodup"
	"github.com/angeles725/niagara-tools/go/internal/retrorows"
)

const usage = `usage: retro-issues.sh [--apply [--write-back]] [--area mcp-n4|build-n4-module] [--alias <stem>=<ALIAS>]...
                       [--only <id>[,<id>...]]... [--new <id>[,<id>...]]... [--dup-threshold <0..1>] <retro.md>...
       retro-issues.sh --check
`

var aliasRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

type options struct {
	apply, writeBack, check bool
	area                    string
	aliases                 map[string]string
	only, renew             map[string]bool // normalised row ids ("Δ1", "ΔB2") or "<src>:<id>"
	threshold               float64
	kitRoot                 string
	files                   []string
}

// normID turns "2", "B2" or "Δ2" into "Δ2"; "<src>:<id>" keeps its source.
func normID(s string) string {
	src := ""
	if i := strings.LastIndex(s, ":"); i >= 0 {
		src, s = s[:i]+":", s[i+1:]
	}
	return src + "Δ" + strings.TrimPrefix(s, "Δ")
}

func addIDs(set map[string]bool, list string) {
	for _, id := range strings.Split(list, ",") {
		if id = strings.TrimSpace(id); id != "" {
			set[normID(id)] = true
		}
	}
}

// parseArgs reads the command line; a non-empty message is a usage error.
func parseArgs(args []string, kitRoot string) (options, string) {
	o := options{aliases: map[string]string{}, only: map[string]bool{}, renew: map[string]bool{}, threshold: retrodup.DefaultThreshold, kitRoot: kitRoot}
	for i := 0; i < len(args); i++ {
		a := args[i]
		val := func() (string, bool) {
			if i+1 >= len(args) {
				return "", false
			}
			i++
			return args[i], true
		}
		switch a {
		case "--apply":
			o.apply = true
		case "--write-back":
			o.writeBack = true
		case "--check":
			o.check = true
		case "--area", "--alias", "--only", "--new", "--dup-threshold", "--kit-root":
			v, ok := val()
			if !ok {
				return o, a + " needs a value\n" + usage
			}
			if msg := o.set(a, v); msg != "" {
				return o, msg + "\n" + usage
			}
		default:
			if strings.HasPrefix(a, "-") {
				return o, "unknown flag: " + a + "\n" + usage
			}
			o.files = append(o.files, a)
		}
	}
	switch {
	case o.check && (len(o.files) > 0 || o.apply):
		return o, "--check takes no retro and no --apply\n" + usage
	case !o.check && len(o.files) == 0:
		return o, usage
	case o.writeBack && !o.apply:
		return o, "--write-back needs --apply\n" + usage
	}
	return o, ""
}

func (o *options) set(flag, v string) string {
	switch flag {
	case "--area":
		if v != retrorows.AreaMCP && v != retrorows.AreaBuild {
			return "--area must be " + retrorows.AreaMCP + " or " + retrorows.AreaBuild
		}
		o.area = v
	case "--alias":
		stem, alias, ok := strings.Cut(v, "=")
		if !ok || stem == "" || !aliasRe.MatchString(alias) {
			return "--alias must be <stem>=<ALIAS> with an ALIAS of letters, digits, . _ -"
		}
		o.aliases[stem] = alias
	case "--only":
		addIDs(o.only, v)
	case "--new":
		addIDs(o.renew, v)
	case "--dup-threshold":
		f, err := strconv.ParseFloat(v, 64)
		if err != nil || !(f > 0 && f <= 1) { // also refuses NaN
			return fmt.Sprintf("--dup-threshold must be a number in (0, 1], not %q", v)
		}
		o.threshold = f
	case "--kit-root":
		o.kitRoot = v
	}
	return ""
}
