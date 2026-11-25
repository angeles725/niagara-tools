package readtools

import (
	"math"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// MaxTagComponents is how many components one n4_read_tags call reads.
const MaxTagComponents = 50

// sessionCaller is a client that can call the station's session handler (box.Client.SSC): the
// route to implied tags and relations. A client without it reports them unavailable.
type sessionCaller interface {
	SSC(key string, arg any) (any, error)
}

const msgImpliedUnavailable = "this station client cannot call the session handler (impliedTags): " +
	"implied tags are not read"

// ReadTags is n4_read_tags: the direct tags, tag groups and relations of each component (one
// load), and its implied tags (one impliedTags call each) unless implied is false.
func (t *Tools) ReadTags(args *pyjson.Object) (*pyjson.Object, error) {
	ords, err := tagOrds(args)
	if err != nil {
		return nil, err
	}
	relations := strArg(args, "relations", "direct") // direct, all or none: the schema's enum
	implied := true
	if v, ok := args.Get("implied"); ok {
		implied = v == true
	}
	start := t.Ctx.Clock()
	comps := []any{}
	loads, err := t.loadTags(ords)
	if err != nil {
		return nil, err
	}
	for i, o := range ords {
		if loads[i].Err != nil {
			return nil, loads[i].Err
		}
		c, err := t.tagEntryOf(o, loads[i].Nodes, implied, relations)
		if err != nil {
			return nil, err
		}
		comps = append(comps, c)
	}
	return obj("components", comps, "count", len(comps),
		"elapsed_ms", int(math.RoundToEven((t.Ctx.Clock()-start)*1000))), nil
}

// tagOrds is the ord argument, or the ords list (1 to MaxTagComponents station ORDs).
func tagOrds(args *pyjson.Object) ([]string, error) {
	var list []any
	if v, ok := args.Get("ords"); ok {
		list, _ = v.([]any)
	}
	if v, ok := args.Get("ord"); ok {
		list = append(list, v)
	}
	if len(list) == 0 || len(list) > MaxTagComponents {
		return nil, toolError("give ord, or ords with 1 to %d component ORDs", MaxTagComponents)
	}
	out := make([]string, len(list))
	for i, v := range list {
		s, ok := v.(string)
		if !ok || !strings.HasPrefix(s, "station:") {
			return nil, toolError("each ORD must be a station ORD such as %s, got %s",
				pyjson.StrRepr(RootOrd), pyjson.Repr(v))
		}
		out[i] = s
	}
	return out, nil
}
