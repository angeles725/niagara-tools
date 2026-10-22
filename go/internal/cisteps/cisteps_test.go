package cisteps

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const head = "name: CI\non: pull_request\njobs:\n  checks:\n    runs-on: ubuntu-latest\n    steps:\n"

func TestParse(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, yml string
		want      []Step
		err       string // "" = no error; else the message after "w.yml:"
	}{
		{"single-line run", head + "      - name: a\n        run: echo a\n",
			[]Step{{"run", "a", "echo a"}}, ""},
		{"uses step named by its ref", head + "      - uses: actions/checkout@v5\n",
			[]Step{{"uses", "actions/checkout@v5", "actions/checkout@v5"}}, ""},
		{"nameless run named by its first line", head + "      - run: |\n          echo one\n          echo two\n",
			[]Step{{"run", "echo one", "echo one\necho two"}}, ""},
		{"| block dedented, inner indent and blank lines kept, trailing blanks dropped",
			head + "      - name: b\n        run: |\n          if x; then\n            y\n\n          fi\n\n\n      - name: c\n        run: echo c\n",
			[]Step{{"run", "b", "if x; then\n  y\n\nfi"}, {"run", "c", "echo c"}}, ""},
		{"|- block", head + "      - name: b\n        run: |-\n          echo b\n",
			[]Step{{"run", "b", "echo b"}}, ""},
		{"block ends at a dedented line", head + "      - run: |\n          a\n        name: n\n",
			[]Step{{"run", "n", "a"}}, ""},
		{"quoted scalars", head + "      - name: \"dq\"\n        run: 'echo hi'\n",
			[]Step{{"run", "dq", "echo hi"}}, ""},
		{"empty name falls back to the first run line", head + "      - name: \"\"\n        run: echo x\n",
			[]Step{{"run", "echo x", "echo x"}}, ""},
		{"blank first block line falls back to the step line", head + "      - run: |\n\n          x\n",
			[]Step{{"run", "step at line 7", "\nx"}}, ""},
		{"with: and id: are replayable; nested keys are skipped",
			head + "      - uses: a/b@v1\n        id: x\n        with:\n          env: 1\n          run: no\n",
			[]Step{{"uses", "a/b@v1", "a/b@v1"}}, ""},
		{"step-level env/shell/if/working-directory are bad, in order",
			head + "      - name: e\n        env:\n          X: 1\n        shell: bash\n        if: always()\n        working-directory: go\n        run: echo\n",
			[]Step{{"bad", "e", "step-level env: shell: if: working-directory: not replayed"}}, ""},
		{"neither run nor uses", head + "      - id: x\n",
			[]Step{{"bad", "step at line 7", "step-level needs exactly one of run:/uses: not replayed"}}, ""},
		{"both run and uses", head + "      - name: both\n        uses: a/b@v1\n        run: echo\n",
			[]Step{{"bad", "both", "step-level needs exactly one of run:/uses: not replayed"}}, ""},
		{"an empty run: is still a run step", head + "      - name: e\n        run:\n",
			[]Step{{"run", "e", ""}}, ""},
		{"unicode key letters are keys (Python \\w)", head + "      - name: u\n        nämé: x\n        run: echo\n",
			[]Step{{"bad", "u", "step-level nämé: not replayed"}}, ""},
		{"CRLF and CR line ends read as LF", strings.ReplaceAll(head+"      - name: a\n        run: |\n          x\n", "\n", "\r\n"),
			[]Step{{"run", "a", "x"}}, ""},
		{"a second job's steps are read too",
			head + "      - run: a\n  other:\n    steps:\n      - run: b\n",
			[]Step{{"run", "a", "a"}, {"run", "b", "b"}}, ""},
		{"a later duplicate key wins", head + "      - name: a\n        name: b\n        run: x\n",
			[]Step{{"run", "b", "x"}}, ""},
		{"comments and non-key lines are ignored", head + "      # - run: hidden\n      - name: a\n        run: x\n",
			[]Step{{"run", "a", "x"}}, ""},
		{"no steps at all", "name: CI\non: push\n", nil, ""},

		{"folded > block", head + "      - name: f\n        run: >\n          a\n", nil, "8: run: > block style not supported (use | or |-)"},
		{"|+ block", head + "      - name: f\n        run: |+\n          a\n", nil, "8: run: |+ block style not supported (use | or |-)"},
		{"unterminated quote", head + "      - name: \"oops\n        run: x\n", nil, "7: unterminated quoted scalar"},
		{"lone quote", head + "      - name: '\n", nil, "7: unterminated quoted scalar"},
		{"run outside a step", "jobs:\n  a:\n    run: x\n", nil, "3: run: outside a steps: item"},
		{"uses outside a step", "uses: x\n", nil, "1: uses: outside a steps: item"},
		{"workflow-level env", "env:\n  X: 1\n" + head, nil, "1: workflow/job-level env: changes every step; not replayed"},
		{"job-level defaults", "jobs:\n  a:\n    defaults:\n      run:\n", nil, "3: workflow/job-level defaults: changes every step; not replayed"},
		{"job-level env after the steps list", head + "      - run: a\n    env:\n      X: 1\n", nil, "8: workflow/job-level env: changes every step; not replayed"},
	}
	for _, c := range cases {
		got, err := Parse("w.yml", []byte(c.yml))
		if c.err != "" {
			if err == nil || err.Error() != "w.yml:"+c.err {
				t.Errorf("%s: err = %v, want w.yml:%s", c.name, err, c.err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: unexpected error %v", c.name, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
}

func TestParseRejectsInvalidUTF8(t *testing.T) {
	t.Parallel()
	if _, err := Parse("w.yml", []byte("name: \xff\n")); err == nil || !strings.Contains(err.Error(), "UTF-8") {
		t.Errorf("err = %v, want a UTF-8 error", err)
	}
}

// TestParseFixtures pins the shapes tests/ci-local.bats relies on and the real ci.yml.
func TestParseFixtures(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "..", "..")
	read := func(rel string) []Step {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		steps, err := Parse(rel, data)
		if err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
		return steps
	}
	if _, err := Parse("folded.yml", mustRead(t, filepath.Join(root, "tests/fixtures/ci-local/folded.yml"))); err == nil ||
		!strings.HasPrefix(err.Error(), "folded.yml:9: ") {
		t.Errorf("folded.yml: err = %v, want folded.yml:9:", err)
	}
	uses := read("tests/fixtures/ci-local/uncovered-uses.yml")
	want := []Step{
		{"uses", "some/action@v1", "some/action@v1"},
		{"bad", "with env", "step-level env: not replayed"},
		{"bad", "with shell", "step-level shell: not replayed"},
	}
	if !reflect.DeepEqual(uses, want) {
		t.Errorf("uncovered-uses.yml = %q, want %q", uses, want)
	}
	sem := read("tests/fixtures/ci-local/semantics.yml")
	if len(sem) != 6 || sem[3].Body != "false\necho should-not-print" || sem[5].Body != "# a comment line\nX=line-two\necho \"$X-ok\"" {
		t.Errorf("semantics.yml = %q", sem)
	}
	ci := read(".github/workflows/ci.yml")
	if len(ci) < 20 || ci[0] != (Step{"uses", "actions/checkout@v5", "actions/checkout@v5"}) {
		t.Errorf("ci.yml: %d steps, first %q", len(ci), ci[0])
	}
	for _, s := range ci {
		if s.Kind == "bad" {
			t.Errorf("ci.yml step %q is not replayable: %s", s.Name, s.Body)
		}
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
