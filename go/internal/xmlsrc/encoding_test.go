package xmlsrc

import "testing"

func TestDeclaredEncodings(t *testing.T) {
	for _, c := range []struct{ name, text, attr string }{
		{"ISO-8859-1", "<?xml version='1.0' encoding='ISO-8859-1'?>\n<r v='caf\xe9'><a/></r>", "café"},
		{"lower-case latin1", "<?xml version=\"1.0\" encoding=\"iso-8859-1\"?><r v='\xb0C'/>", "°C"},
		{"US-ASCII", "<?xml version='1.0' encoding='US-ASCII'?><r v='x'/>", "x"},
		{"utf-8 any case", "<?xml version='1.0' encoding='utf-8'?><r v='\xc3\xa9'/>", "é"},
	} {
		d, err := Parse("e.xml", []byte(c.text))
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if v, _ := d.Elements()[0].Attr("v"); v != c.attr {
			t.Errorf("%s: v = %q, want %q", c.name, v, c.attr)
		}
	}
	d, err := Parse("e.xml", []byte("<?xml version='1.0' encoding='ISO-8859-1'?>\n<r>\xe9\xe9<a/></r>"))
	if err != nil || d.Text(0) != "éé" || d.Elements()[1].Line != 2 || d.Elements()[1].Col != 8 {
		t.Errorf("latin1 text/position: %v %+v", err, d)
	}
}

func TestDeclaredEncodingErrors(t *testing.T) {
	for _, c := range []struct{ text, want string }{
		{"<?xml version='1.0' encoding='US-ASCII'?>\n<r v='caf\xe9'/>", "e.xml:2:10: byte 0xe9 is not US-ASCII"},
		{"<?xml version='1.0' encoding='UTF-16'?><r/>", `e.xml:1:1: encoding "UTF-16" is not supported (UTF-8, ISO-8859-1 and US-ASCII are)`},
	} {
		if _, err := Parse("e.xml", []byte(c.text)); err == nil || err.Error() != c.want {
			t.Errorf("err = %v\n want %s", err, c.want)
		}
	}
}
