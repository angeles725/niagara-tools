package retrogh

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
)

var issueURL = regexp.MustCompile(`/issues/([0-9]+)\s*$`)

// Create files an issue and returns its number.
func Create(run Runner, title, body string, labels []string) (int, error) {
	args := []string{"issue", "create", "--title", title, "--body-file", "-"}
	for _, l := range labels {
		args = append(args, "--label", l)
	}
	out, err := run(args, body)
	if err != nil {
		return 0, err
	}
	m := issueURL.FindStringSubmatch(out)
	if m == nil {
		return 0, fmt.Errorf("gh issue create: no issue URL in its answer")
	}
	return strconv.Atoi(m[1])
}

// Comment adds a comment to issue n and gives it the retro-delta label, so the fold audit and the open-issue check
// find it by label.
func Comment(run Runner, n int, body string) error {
	if _, err := run([]string{"issue", "comment", strconv.Itoa(n), "--body-file", "-"}, body); err != nil {
		return err
	}
	_, err := run([]string{"issue", "edit", strconv.Itoa(n), "--add-label", Label}, "")
	return err
}

// EnsureLabels creates every label (name -> description) the repository lacks.
func EnsureLabels(run Runner, want map[string]string) error {
	out, err := run([]string{"label", "list", "--limit", "500", "--json", "name"}, "")
	if err != nil {
		return err
	}
	var have []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(out), &have); err != nil {
		return fmt.Errorf("gh label list: unreadable JSON: %v", err)
	}
	present := map[string]bool{}
	for _, l := range have {
		present[l.Name] = true
	}
	names := make([]string, 0, len(want))
	for n := range want {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		if !present[n] {
			if _, err := run([]string{"label", "create", n, "--description", want[n]}, ""); err != nil {
				return err
			}
		}
	}
	return nil
}
