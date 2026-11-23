package writeops

import (
	"fmt"
	"slices"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// msgInstanceFaulted is the dry-run refusal of writeDeviceInstance on a Spyder in fault (retro R12 M4:
// a licence capacity fault made the write end in the BOX error "raise() failed. Cannot find vmUuid"
// and the batch in doubt, with the controller untouched).
const (
	msgInstanceFaulted = "writeDeviceInstance refused: %s is in fault (%s). The write would fail with " +
		"\"raise() failed. Cannot find vmUuid\" and the controller would not be written. Clear the fault, then redo " +
		"the dry run"
)

// instanceFaulted is the refusal when the component's own status holds the fault flag, naming its
// faultCause; nil for any other status. A status or faultCause the load left out is the default
// (ok, empty).
func instanceFaulted(ordText string, nodes *pyjson.Object) error {
	kids := box.Children(nodes, "")
	flags, err := box.StatusFlags(box.ChildValue(kids, "status", "0"))
	if err != nil || !slices.Contains(flags, "fault") {
		return nil
	}
	cause := "faultCause is empty"
	if text, _ := box.ChildValue(kids, "faultCause", "").(string); strings.TrimSpace(text) != "" {
		cause = "faultCause: " + bql.StripControl(strings.TrimSpace(text))
	}
	return plan.ToolError(fmt.Sprintf(msgInstanceFaulted, ordText, cause))
}
