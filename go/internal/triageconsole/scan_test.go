package triageconsole

import "testing"

// TestLevelFieldTwoWord: the header parser reads MUY DETALLADO and MÁS DETALLADO (UTF-8 and Latin-1) as one level
// token followed by the "[" timestamp field, where a first-field split would see only MUY or MÁS.
func TestLevelFieldTwoWord(t *testing.T) {
	for _, c := range []struct{ line, lv, next string }{
		{"MUY DETALLADO [10:00:00 01-Oct-26 CST][sys] m", "MUY DETALLADO", "[10:00:00"},
		{"MÁS DETALLADO [10:00:00 01-Oct-26 CST][sys] m", "MÁS DETALLADO", "[10:00:00"},
		{"M\xc1S DETALLADO [10:00:00 01-Oct-26 CST][sys] m", "M\xc1S DETALLADO", "[10:00:00"},
		{"DETALLADO [10:00:00 01-Oct-26 CST][sys] m", "DETALLADO", "[10:00:00"},
		{"MUY [10:00:00 01-Oct-26 CST][sys] m", "MUY", "[10:00:00"},
		{"MUY DETALLADOS [10:00:00 01-Oct-26 CST][sys] m", "MUY", "DETALLADOS"},
		{"MUY  DETALLADO [10:00:00 01-Oct-26 CST][sys] m", "MUY", "DETALLADO"},
		{"MAS DETALLADO [10:00:00 01-Oct-26 CST][sys] m", "MAS", "DETALLADO"},
	} {
		lv, next := levelField(c.line)
		if lv != c.lv || next != c.next {
			t.Errorf("levelField(%q) = %q, %q; want %q, %q", c.line, lv, next, c.lv, c.next)
		}
	}
	for _, lv := range []string{"MUY DETALLADO", "MÁS DETALLADO", "M\xc1S DETALLADO", "CONFIGURAR", "DETALLADO"} {
		if !isLevel(lv) {
			t.Errorf("isLevel(%q) = false, want true", lv)
		}
	}
	for _, lv := range []string{"MUY", "MÁS", "DETALLADOS", "MAS DETALLADO", "Muy Detallado"} {
		if isLevel(lv) {
			t.Errorf("isLevel(%q) = true, want false", lv)
		}
	}
}
