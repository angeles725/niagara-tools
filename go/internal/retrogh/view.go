package retrogh

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// State is the state ("OPEN" or "CLOSED") of issue n.
func State(run Runner, n int) (string, error) {
	out, err := run([]string{"issue", "view", strconv.Itoa(n), "--json", "state"}, "")
	if err != nil {
		return "", err
	}
	var v struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		return "", fmt.Errorf("gh issue view: unreadable JSON: %v", err)
	}
	return v.State, nil
}
