package cli

import "testing"

func TestStdioEnvUsesTheInjectedGetenv(t *testing.T) {
	t.Parallel()
	s := Stdio{Getenv: func(name string) string { return "value of " + name }}
	if got := s.Env("HOME"); got != "value of HOME" {
		t.Errorf("Env = %q, want the injected value", got)
	}
}

func TestStdioEnvZeroValueReadsTheProcess(t *testing.T) {
	t.Setenv("NIAGARA_TOOLS_CLI_TEST", "from the process")
	if got := (Stdio{}).Env("NIAGARA_TOOLS_CLI_TEST"); got != "from the process" {
		t.Errorf("zero-value Env = %q, want the process value", got)
	}
}
