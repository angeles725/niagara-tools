package readtools

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/retro"
)

// SessionRetroDraft is n4_session_retro_draft: the session retro draft of the server's state
// dir, reads only. The window starts at the server start unless since says otherwise, so a
// draft covers this session and not the whole history of the state directory.
func (t *Tools) SessionRetroDraft(args *pyjson.Object) (*pyjson.Object, error) {
	station := "unknown"
	if t.Ctx.Session != nil {
		station = t.Ctx.Session.StationName
	}
	since, _ := lookup(args, "since").(string) // validated against the schema: a string
	if since == "" {
		since = t.Ctx.StartedAt
	}
	return retro.Draft(t.stateDir(), station, since, t.Ctx.Observations)
}
