package kitticket

import (
	"os"
	"strings"
)

const (
	deltasHeader = "| Δ | Delta | Target file / § | Token |"
	labelsLine   = "Labels: `kit`, `from-run`, `campaign-9`\n"
)

// placeholderBody is the ticket with no retro content: the gh-absent body, and the gh body without a readable
// --from file.
func placeholderBody(title, retro string) string {
	return "[kit] " + title + "\n\n" +
		"**Retro**: " + retro + "\n" +
		"**What happened**: <see retro>\n\n" +
		"**Proposed kit deltas**\n" + deltasHeader + "\n|---|---|---|---|\n\n" +
		labelsLine
}

// ghBody is the body the gh path sends: with a --from regular file, its first 5 non-blank "What happened" lines
// and first 20 non-blank "Proposed kit deltas" lines (each read as the bash awk|grep|head did). An unreadable
// file reads as empty after awk's message, once per section, as in the bash.
func (t ticket) ghBody() string {
	title, from := t.req.title, t.req.from
	retro := from
	if retro == "" {
		retro = "<no retro>"
	}
	if from == "" || !isRegular(from) {
		return placeholderBody(title, retro)
	}
	data, err := os.ReadFile(from)
	read := func() []byte {
		if err != nil {
			t.s.Err.Write([]byte("awk: fatal: cannot open file `" + from + "' for reading: " + errText(err) + "\n"))
		}
		return data
	}
	what := section(read(), "## What happened", 5)
	if what == "" {
		what = "<see retro>"
	}
	deltas := section(read(), "## Proposed kit deltas", 20)
	if deltas == "" {
		deltas = deltasHeader
	}
	return "[kit] " + title + "\n\n" +
		"**Retro**: " + retro + "\n" +
		"**What happened**: " + what + "\n\n" +
		"**Proposed kit deltas**\n" + deltas + "\n\n" +
		labelsLine
}

// section is `awk '/^<head>/{f=1;next} f&&/^## /{exit} f{print}' | grep -v '^[[:space:]]*$' | head -<max>` under
// command substitution: the lines after a line starting with head, up to the next "## " line, blank ones dropped.
func section(data []byte, head string, max int) string {
	lines := strings.Split(string(data), "\n")
	if strings.HasSuffix(string(data), "\n") {
		lines = lines[:len(lines)-1]
	}
	var out []string
	in := false
	for _, l := range lines {
		if strings.HasPrefix(l, head) {
			in = true
			continue
		}
		if !in {
			continue
		}
		if strings.HasPrefix(l, "## ") {
			break
		}
		if strings.Trim(l, " \t\n\v\f\r") != "" && len(out) < max {
			out = append(out, l)
		}
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n")
}

func isRegular(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.Mode().IsRegular()
}
