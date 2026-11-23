// Package retrogh is the GitHub side of retro deltas: the hidden marker that ties an issue to a retro row, listing the
// issues that carry the retro-delta label, and creating, commenting and labelling. Every gh call goes through a
// Runner so tests inject a fake.
package retrogh

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Label is the label every retro-delta issue carries.
const Label = "retro-delta"

// ListLimit caps one issue listing; a listing that reaches it may hide a marker and is an error.
const ListLimit = 1000

// Runner runs `gh <args>` with stdin and returns its stdout.
type Runner func(args []string, stdin string) (string, error)

// Exec is the Runner that runs the real gh.
func Exec(args []string, stdin string) (string, error) {
	cmd := exec.Command("gh", args...)
	cmd.Stdin = strings.NewReader(stdin)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errb.String())
		if i := strings.IndexByte(msg, '\n'); i >= 0 {
			msg = msg[:i]
		}
		return out.String(), fmt.Errorf("gh %s: %v: %s", args[0], err, msg)
	}
	return out.String(), nil
}

// Issue is one GitHub issue as `gh issue list --json` reports it.
type Issue struct {
	Number   int
	Title    string
	State    string // OPEN or CLOSED
	Body     string
	Comments []string
	Labels   []string
}

// Closed reports whether the issue is closed.
func (i Issue) Closed() bool { return strings.EqualFold(i.State, "CLOSED") }

const fields = "number,title,state,body,comments,labels"

type rawIssue struct {
	Number   int    `json:"number"`
	Title    string `json:"title"`
	State    string `json:"state"`
	Body     string `json:"body"`
	Comments []struct {
		Body string `json:"body"`
	} `json:"comments"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

func parseIssues(out string) ([]Issue, error) {
	var raw []rawIssue
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		return nil, fmt.Errorf("gh issue list: unreadable JSON: %v", err)
	}
	issues := make([]Issue, len(raw))
	for i, r := range raw {
		issues[i] = Issue{Number: r.Number, Title: r.Title, State: r.State, Body: r.Body}
		for _, c := range r.Comments {
			issues[i].Comments = append(issues[i].Comments, c.Body)
		}
		for _, l := range r.Labels {
			issues[i].Labels = append(issues[i].Labels, l.Name)
		}
	}
	return issues, nil
}

// ListLabeled lists every issue, open and closed, that carries label. A listing of ListLimit issues is refused.
func ListLabeled(run Runner, label string) ([]Issue, error) {
	out, err := run([]string{"issue", "list", "--state", "all", "--label", label, "--limit", strconv.Itoa(ListLimit), "--json", fields}, "")
	if err != nil {
		return nil, err
	}
	issues, err := parseIssues(out)
	if err == nil && len(issues) >= ListLimit {
		return nil, fmt.Errorf("gh issue list: %d issues carry the %s label, the listing limit: a marker may be hidden", len(issues), label)
	}
	return issues, err
}

// Search lists up to 30 issues, open and closed, that match a GitHub search query.
func Search(run Runner, query string) ([]Issue, error) {
	out, err := run([]string{"issue", "list", "--state", "all", "--search=" + query, "--limit", "30", "--json", fields}, "")
	if err != nil {
		return nil, err
	}
	return parseIssues(out)
}

// Marker is the hidden line that ties an issue (or a comment on it) to a retro row.
func Marker(src, id string) string { return "<!-- retro-delta: " + src + " " + id + " -->" }

var markerRe = regexp.MustCompile(`<!-- retro-delta: (.+?) -->`)

// Markers maps "<src> <id>" to the issue whose body or comments carry its marker.
func Markers(issues []Issue) map[string]Issue {
	m := map[string]Issue{}
	for _, is := range issues {
		for _, text := range append([]string{is.Body}, is.Comments...) {
			for _, f := range markerRe.FindAllStringSubmatch(text, -1) {
				if _, dup := m[f[1]]; !dup {
					m[f[1]] = is
				}
			}
		}
	}
	return m
}
