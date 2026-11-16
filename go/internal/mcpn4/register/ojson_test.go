package register

import (
	"strings"
	"testing"
)

func TestSetRoundTripsTwoSpaceJSONByteForByte(t *testing.T) {
	src := "{\n  \"b\": 1,\n  \"a\": {\n    \"x\": \"\\u003c<\\u00e9é\",\n    \"e\": {},\n    \"l\": []\n  },\n  \"n\": 1.50e3\n}"
	out, err := setPath([]byte(src), []string{"a", "x"}, []byte("\"\\u003c<\\u00e9é\""))
	if err != nil || string(out) != src {
		t.Fatalf("round trip =\n%s\n%v\nwant\n%s", out, err, src)
	}
}

func TestSetKeepsAnotherServersInlineArrayByteForByte(t *testing.T) {
	src := "{\n  \"mcpServers\": {\n    \"codegraph\": {\"command\": \"codegraph\", \"args\": [\"serve\", \"--mcp\"]},\n" +
		"    \"tabbed\":\t{ \"command\" : \"x\" }\n  },\n  \"tail\": [1,2]\n}\n"
	out, err := setPath([]byte(src), []string{"mcpServers", "new"}, []byte(`{"command":"c","args":[]}`))
	want := strings.Replace(src, "{ \"command\" : \"x\" }\n", "{ \"command\" : \"x\" },\n"+
		"    \"new\": {\n      \"command\": \"c\",\n      \"args\": []\n    }\n", 1)
	if err != nil || string(out) != want {
		t.Fatalf("set =\n%s\n%v\nwant\n%s", out, err, want)
	}
}

func TestSetReplacesOnlyTheValueOfAnExistingMember(t *testing.T) {
	src := "{\n  \"s\": {\n    \"a\": [1, 2],\n    \"e\": {\"command\": \"old\"},\n    \"z\": true\n  }\n}"
	out, err := setPath([]byte(src), []string{"s", "e"}, []byte(`{"command":"new"}`))
	want := "{\n  \"s\": {\n    \"a\": [1, 2],\n    \"e\": {\n      \"command\": \"new\"\n    },\n    \"z\": true\n  }\n}"
	if err != nil || string(out) != want {
		t.Fatalf("set =\n%s\n%v\nwant\n%s", out, err, want)
	}
}

func TestSetFillsAnEmptyObjectOneLevelDeeperThanItsLine(t *testing.T) {
	src := "{\n  \"p\": {\n    \"mcpServers\": {}\n  }\n}"
	out, err := setPath([]byte(src), []string{"p", "mcpServers", "k"}, []byte(`{"c":1}`))
	want := "{\n  \"p\": {\n    \"mcpServers\": {\n      \"k\": {\n        \"c\": 1\n      }\n    }\n  }\n}"
	if err != nil || string(out) != want {
		t.Fatalf("set =\n%s\n%v\nwant\n%s", out, err, want)
	}
}

func TestSetKeepsAOneLineDocumentOnOneLine(t *testing.T) {
	out, err := setPath([]byte(`{"z": true}`+"\n"), []string{"m", "k"}, []byte(`{"c":"v"}`))
	if want := `{"z": true,"m":{"k":{"c":"v"}}}` + "\n"; err != nil || string(out) != want {
		t.Fatalf("set = %q %v, want %q", out, err, want)
	}
}

func TestSetWritesANewDocumentIndented(t *testing.T) {
	out, err := setPath(nil, []string{"m", "k"}, []byte(`{"c":"v"}`))
	want := "{\n  \"m\": {\n    \"k\": {\n      \"c\": \"v\"\n    }\n  }\n}\n"
	if err != nil || string(out) != want {
		t.Fatalf("set =\n%s\n%v\nwant\n%s", out, err, want)
	}
}

func TestSetRefusesANonObjectOnThePath(t *testing.T) {
	if _, err := setPath([]byte(`{"m":[1]}`), []string{"m", "k"}, []byte(`1`)); err == nil ||
		!strings.Contains(err.Error(), "m") {
		t.Fatalf("err = %v, want a refusal naming m", err)
	}
}

func TestSetRefusesTrailingDataAndASecondDocument(t *testing.T) {
	for _, doc := range []string{"{\"a\": {}} garbage\n", "{\"a\": {}}\n{\"b\": 1}\n", "[1]\n"} {
		if _, err := setPath([]byte(doc), []string{"a", "k"}, []byte(`1`)); err == nil {
			t.Errorf("%q: accepted", doc)
		}
	}
}

func TestSetRefusesADuplicateKeyOnThePath(t *testing.T) {
	doc := "{\n  \"mcpServers\": {\"x\": 1},\n  \"mcpServers\": {\"y\": 2}\n}\n"
	if _, err := setPath([]byte(doc), []string{"mcpServers", "k"}, []byte(`1`)); err == nil ||
		!strings.Contains(err.Error(), `duplicate key "mcpServers"`) {
		t.Fatalf("err = %v, want the duplicate key named", err)
	}
}

func TestSetUsesTheLineEndingOfACRLFFile(t *testing.T) {
	src := "{\r\n  \"s\": {\r\n    \"a\": 1\r\n  }\r\n}\r\n"
	out, err := setPath([]byte(src), []string{"s", "k"}, []byte(`{"c":1}`))
	want := "{\r\n  \"s\": {\r\n    \"a\": 1,\r\n    \"k\": {\r\n      \"c\": 1\r\n    }\r\n  }\r\n}\r\n"
	if err != nil || string(out) != want {
		t.Fatalf("set = %q %v, want %q", out, err, want)
	}
}
