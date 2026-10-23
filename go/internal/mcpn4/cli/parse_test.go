package cli

import (
	"strings"
	"testing"
)

func TestMainParsesFlagsWithoutServingAtImport(t *testing.T) {
	t.Parallel()
	cfg, help, err := Parse([]string{"--allow-writes", "--allow-http-for-tests", "--station", "a=https://h1",
		"--station", "b=https://h2:8443", "--insecure-tls", "b", "--credential-env", "LAB"})
	if err != nil || help || !cfg.AllowWrites || !cfg.AllowHTTP || cfg.CredentialEnv != "LAB" ||
		strings.Join(cfg.Stations, " ") != "a=https://h1 b=https://h2:8443" || strings.Join(cfg.InsecureTLS, " ") != "b" {
		t.Errorf("Parse = %+v %v %v", cfg, help, err)
	}
	if cfg, _, _ := Parse(nil); cfg.AllowWrites || cfg.AllowHTTP || cfg.CredentialEnv != "MCP_N4" {
		t.Errorf("defaults = %+v", cfg)
	}
}

func TestFlagsHaveDocumentedDefaultsAndScopeIsRepeatable(t *testing.T) {
	t.Parallel()
	cfg, _, _ := Parse([]string{"--write-scope", "a", "--write-scope", "b"})
	if cfg.TokenTTL != 300 || cfg.MaxWrites != 200 || cfg.AuthCooldown != 30 || cfg.HTTPTimeout != 20 ||
		cfg.LoadWait != 3 || strings.Join(cfg.WriteScopes, ",") != "a,b" || cfg.StateDir != "" {
		t.Errorf("cfg = %+v", cfg)
	}
}

func TestZeroOrNegativeTtlAndBudgetAreRefusedByTheParser(t *testing.T) {
	t.Parallel()
	for _, argv := range [][]string{{"--token-ttl", "0"}, {"--max-writes", "-1"}} {
		if _, _, err := Parse(argv); err == nil || !strings.Contains(err.Msg, "must be >= 1") {
			t.Errorf("%q: err = %v", argv, err)
		}
	}
}

func TestTheCooldownIsConfigurableButNeverBelow30Seconds(t *testing.T) {
	t.Parallel()
	if cfg, _, err := Parse([]string{"--auth-cooldown", "45"}); err != nil || cfg.AuthCooldown != 45 {
		t.Errorf("45: %+v %v", cfg, err)
	}
	if _, _, err := Parse([]string{"--auth-cooldown", "29"}); err == nil || !strings.Contains(err.Msg, ">= 30") {
		t.Errorf("29: err = %v", err)
	}
}
