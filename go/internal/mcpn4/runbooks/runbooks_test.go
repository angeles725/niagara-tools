package runbooks

import (
	"regexp"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/registry"
)

func TestAllLoadsTheShippedRunbooksInNameOrder(t *testing.T) {
	books := All()
	var names []string
	for _, b := range books {
		names = append(names, b.Name)
		if b.Title == "" || b.Description == "" || strings.TrimSpace(b.Body) == "" {
			t.Errorf("%s: empty title, description or body", b.Name)
		}
	}
	if got := strings.Join(names, ","); got != "alarm-triage,commissioning,post-deploy-check" {
		t.Fatalf("names %s", got)
	}
}

var toolName = regexp.MustCompile(`\bn4_[a-z_]+`)
var placeholder = regexp.MustCompile(`\{\{([a-z_]+)\}\}`)

// A runbook names only tools the server registers, so a renamed tool breaks this test, not a session.
func TestRunbooksNameOnlyRegisteredTools(t *testing.T) {
	reg := registry.Default()
	known := map[string]bool{}
	for _, tl := range append(append([]registry.Tool{}, reg.Read...), reg.Write...) {
		known[tl.Name] = true
	}
	for _, b := range All() {
		for _, n := range toolName.FindAllString(b.Body+b.Description, -1) {
			if !known[n] {
				t.Errorf("%s names %s, which the registry does not hold", b.Name, n)
			}
		}
	}
}

// Every placeholder is a declared argument and every declared argument is used.
func TestPlaceholdersMatchDeclaredArguments(t *testing.T) {
	for _, b := range All() {
		declared := map[string]bool{}
		for _, a := range b.Args {
			declared[a.Name] = true
			if !a.Required && a.Default == "" {
				t.Errorf("%s: optional argument %s has no default", b.Name, a.Name)
			}
		}
		used := map[string]bool{}
		for _, m := range placeholder.FindAllStringSubmatch(b.Body, -1) {
			used[m[1]] = true
			if !declared[m[1]] {
				t.Errorf("%s: placeholder {{%s}} is not a declared argument", b.Name, m[1])
			}
		}
		for n := range declared {
			if !used[n] {
				t.Errorf("%s: argument %s is never used", b.Name, n)
			}
		}
	}
}

func TestParseReadsFrontMatterArgumentsAndDefaults(t *testing.T) {
	b, err := parse("x.md", "---\ntitle: T\ndescription: D\narg station!: The station\narg scope: A scope = station:|slot:/\n---\nBody {{station}}\n")
	if err != nil {
		t.Fatal(err)
	}
	if b.Name != "x" || b.Title != "T" || b.Description != "D" || b.Body != "Body {{station}}\n" || len(b.Args) != 2 {
		t.Fatalf("%+v", b)
	}
	if a := b.Args[0]; a.Name != "station" || !a.Required || a.Description != "The station" || a.Default != "" {
		t.Fatalf("arg0 %+v", a)
	}
	if a := b.Args[1]; a.Name != "scope" || a.Required || a.Description != "A scope" || a.Default != "station:|slot:/" {
		t.Fatalf("arg1 %+v", a)
	}
}

func TestParseRefusesMalformedRunbooks(t *testing.T) {
	for _, c := range []struct{ text, want string }{
		{"no front matter", "x.md: missing front matter"},
		{"---\ntitle: T\n", "x.md: unterminated front matter"},
		{"---\ntitle: T\nbogus\n---\nB", `x.md: front matter line "bogus" is not key: value`},
		{"---\ntitle: T\ncolour: red\n---\nB", `x.md: unknown front matter key "colour"`},
		{"---\ndescription: D\n---\nB", "x.md: needs title and description"},
	} {
		if _, err := parse("x.md", c.text); err == nil || err.Error() != c.want {
			t.Errorf("%q: err %v, want %s", c.text, err, c.want)
		}
	}
}
