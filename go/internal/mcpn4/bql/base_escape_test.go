package bql_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

const baseMsgPrefix = "base must be a plain slot path such as station:|slot:/Drivers, got "

// The station matches a slot path in its escaped form, so a raw space, quote, &, dot, hyphen or
// non-ASCII character names nothing: BaseSlot and ComposeOrd refuse it and say how to write it.
func TestBaseSlotRefusesACharacterTheStationWouldNotMatch(t *testing.T) {
	for _, base := range []string{"station:|slot:/Plant/Cold Room", "/Plant/Cold Room", "slot:/Plant/Cold Room",
		`/Plant/Say"hi"`, "/Plant/it's", "/Plant/A&B", "/Plant/Café", "/Plant/日本", "/Plant/Cold Room",
		"/Plant/a.b", "/Plant/a-b", "/Plant/a%20b", "/Plant/a?b", "/Plant/a#b", "/Plant/../B", "/Plant/a b/"} {
		_, err := bql.BaseSlot(base)
		var ve *pyjson.ValueError
		if !errors.As(err, &ve) {
			t.Errorf("BaseSlot(%q) = %v; want a ValueError", base, err)
			continue
		}
		want := baseMsgPrefix + pyjson.StrRepr(base)
		if !strings.HasPrefix(ve.Msg, want) || !strings.Contains(ve.Msg, "$20") || !strings.Contains(ve.Msg, "escaped form") {
			t.Errorf("BaseSlot(%q) message %q; want prefix %q and the escaped-form hint naming $20", base, ve.Msg, want)
		}
		if ord, err := bql.ComposeOrd(base, "select name from driver:Device"); err == nil || ord != "" {
			t.Errorf("ComposeOrd(%q) = %q, %v; want a refusal and no ORD", base, ord, err)
		}
	}
}

// The escaped spelling, and every base that was valid, is unchanged.
func TestBaseSlotKeepsAnEscapedAndAPlainBase(t *testing.T) {
	for base, want := range map[string]string{"station:|slot:/Plant/Cold$20Room": "/Plant/Cold$20Room",
		"/Plant/Cold$20Room/": "/Plant/Cold$20Room", "slot:/Plant/Caf$u00e9": "/Plant/Caf$u00e9", "/Drivers/N$201": "/Drivers/N$201",
		"/A_b/C9": "/A_b/C9", "station:|slot:/": "/", "/": "/"} {
		if got, err := bql.BaseSlot(base); err != nil || got != want {
			t.Errorf("BaseSlot(%q) = %q, %v; want %q", base, got, err, want)
		}
	}
	q := "select name from driver:Device"
	if got, err := bql.ComposeOrd("station:|slot:/Plant/Cold$20Room", q); err != nil || got != "station:|slot:/Plant/Cold$20Room|bql:"+q+"|view:file:ITableToCsv" {
		t.Errorf("ComposeOrd(escaped) = %q, %v", got, err)
	}
}

// A base the older checks refuse keeps its own message: the frozen Python differential compares it.
func TestBaseSlotKeepsTheOlderRefusalText(t *testing.T) {
	for _, base := range []string{"A", "/A//B", "station:|slot:/A|b", "/A\x7f"} {
		_, err := bql.BaseSlot(base)
		if want := baseMsgPrefix + pyjson.StrRepr(base); err == nil || err.Error() != want {
			t.Errorf("BaseSlot(%q) = %v; want exactly %q", base, err, want)
		}
	}
}
