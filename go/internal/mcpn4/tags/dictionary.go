package tags

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// ServiceOrd is where a station keeps its tag dictionaries: "Installed tag dictionaries belong
// under the TagDictionaryService. They are not allowed anywhere else in the station." (Tagging
// guide, tagdictionary-TagDictionaryService).
const ServiceOrd = "station:|slot:/Services/TagDictionaryService"

// Loader is what reading dictionaries needs of a station client.
type Loader interface {
	LoadTree(ord string, depth int, opt box.LoadOptions) (*pyjson.Object, error)
}

// DictionaryQuery lists the installed dictionaries (measured on 4.15: Slot Path, Namespace,
// Enabled, Frozen, Version, Status; the Niagara dictionary's namespace is only in the BQL answer,
// its load omits it).
const DictionaryQuery = "select slotPath, namespace, enabled, frozen, version, status from tagdictionary:TagDictionary"

// DictionaryColumns name DictionaryQuery's columns by position (headers are localized).
var DictionaryColumns = []string{"slot_path", "namespace", "enabled", "frozen", "version", "status"}

// Dictionary is one installed tag dictionary as DictionaryQuery reports it.
type Dictionary struct {
	Name, Ord, Namespace, Version, Status string
	Enabled, Frozen                       bool
}

// FromRow is the dictionary of one DictionaryQuery row keyed by DictionaryColumns.
func FromRow(row *pyjson.Object) Dictionary {
	str := func(k string) string { v, _ := get(row, k).(string); return v }
	path := str("slot_path")
	return Dictionary{Name: bql.UnescapeSlot(path[strings.LastIndex(path, "/")+1:]), Ord: "station:|" + path,
		Namespace: str("namespace"), Version: str("version"), Status: str("status"),
		Enabled: str("enabled") == "true", Frozen: str("frozen") == "true"}
}

// Namespaced is the dictionary whose namespace is ns, ok false when none is installed.
func Namespaced(dicts []Dictionary, ns string) (Dictionary, bool) {
	for _, d := range dicts {
		if d.Namespace == ns {
			return d, true
		}
	}
	return Dictionary{}, false
}

func get(o *pyjson.Object, k string) any {
	if o == nil {
		return nil
	}
	v, _ := o.Get(k)
	return v
}
