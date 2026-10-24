package lint

import (
	"reflect"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/xmlsrc"
)

// A Doc's warnings are Warn findings at their line, the column in the message.
func TestXMLWarnings(t *testing.T) {
	doc, err := xmlsrc.Parse("c.xml", []byte("<bajaObjectGraph>\n <p v='&#x0;'/></bajaObjectGraph>"))
	if err != nil {
		t.Fatal(err)
	}
	want := []Finding{{Severity: Warn, Line: 2, Msg: "col 8: U+0000 in 1 place (Niagara reads it; XML 1.0 forbids it)"}}
	if got := XMLWarnings(doc); !reflect.DeepEqual(got, want) {
		t.Errorf("XMLWarnings = %+v\n want %+v", got, want)
	}
	clean, _ := xmlsrc.Parse("c.xml", []byte("<bajaObjectGraph/>"))
	if got := XMLWarnings(clean); got != nil {
		t.Errorf("XMLWarnings(clean) = %+v, want nil", got)
	}
}
