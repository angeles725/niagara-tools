package writeops

import (
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// historyModes maps the mode argument to the word of the extension types: interval samples on
// a period (B<Kind>IntervalHistoryExt, with an interval slot), cov records each change of value
// (B<Kind>CovHistoryExt, history-rt javax.baja.history.ext, no interval slot).
var historyModes = map[string]string{"interval": "Interval", "cov": "Cov"}

// historyMode is the mode the arguments ask for: the mode argument, else the mode of a given
// ext_type, else interval. A mode the ext_type contradicts is refused.
func historyMode(args *pyjson.Object, typ string) (string, error) {
	typMode := ""
	if typ != "" {
		typMode = "interval"
		if strings.HasSuffix(typ, "CovHistoryExt") {
			typMode = "cov"
		}
	}
	raw, given := args.Get("mode")
	if !given {
		if typMode == "" {
			typMode = "interval"
		}
		return historyModes[typMode], nil
	}
	mode, _ := raw.(string) // interval or cov: the schema's enum
	if typMode != "" && typMode != mode {
		return "", plan.ToolError(fmt.Sprintf(msgHistoryModeType, typ, typMode, mode))
	}
	return historyModes[mode], nil
}

// historyInterval is the interval slot value for mode: required (at least 1 ms) for Interval,
// refused for Cov, which has no interval slot.
func historyInterval(args *pyjson.Object, mode string) (string, error) {
	interval, given := args.Get("interval_ms")
	if mode == "Cov" {
		if given {
			return "", plan.ToolError(msgHistoryCovInterval)
		}
		return "", nil
	}
	if !given {
		return "", plan.ToolError(msgHistoryNoInterval)
	}
	iv, err := plan.Coerce("baja:RelTime", interval, "0")
	if err != nil {
		return "", err
	}
	if pyjson.Str(iv) == "0" {
		return "", plan.ToolError("interval_ms must be at least 1")
	}
	return pyjson.Str(iv), nil
}

// historyCapacity is the capacity argument as history:Capacity encodes it, "" when not given:
// BCapacity.encodeToString is "<restrictBy>:<max>", restrictBy 1 by record count
// (makeByRecordCount(int)) and 0 unlimited (UNLIMITED = new BCapacity(0, 0L)). records is the
// count for the notes ("" when unlimited).
func historyCapacity(args *pyjson.Object) (encoded, records string, err error) {
	raw, given := args.Get("capacity")
	if !given {
		return "", "", nil
	}
	if raw == "unlimited" {
		return "0:0", "", nil
	}
	n, ok := raw.(pyjson.Number)
	i, isInt := n.BigInt()
	if !ok || !isInt || i.Sign() <= 0 || i.Cmp(big.NewInt(math.MaxInt32)) > 0 {
		return "", "", plan.ToolError(fmt.Sprintf(msgHistoryCapacity, pyjson.Repr(raw)))
	}
	return "1:" + i.String(), i.String(), nil
}

// capacityOp sets the capacity of the extension name under the point with handle h, in the same
// batch right after its add: historyConfig (history-rt BHistoryConfig) is a component, and a
// station refuses an add body that nests a component; a set op walks a slot path
// (javax.baja.sync.SetOp.getTarget).
func capacityOp(h any, name, encoded string) *pyjson.Object {
	return obj("nm", "s", "h", h, "n", name+"/historyConfig/capacity",
		"b", obj("nm", "p", "t", "history:Capacity", "v", encoded))
}
