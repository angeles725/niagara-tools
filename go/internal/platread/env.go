package platread

import (
	"fmt"
	"os"
	"strings"
)

type cred struct{ host, port, user, password string }

// loadEnv reads KEY=VALUE lines (a value may be quoted; # starts a comment). It refuses a file anyone but
// its owner may read, a missing key, and a value still wrapped in <...> from a template; a message names
// the key, never the value.
func loadEnv(path string) (cred, error) {
	info, err := os.Stat(path)
	if err != nil {
		return cred{}, fmt.Errorf("env file %s: %v (set NIAGARA_PLAT_ENV or pass --env)", path, err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return cred{}, fmt.Errorf("env file %s is mode %04o: make it 0600 (chmod 600)", path, info.Mode().Perm())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cred{}, fmt.Errorf("env file %s: %v", path, err)
	}
	kv := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		k, v, ok := strings.Cut(line, "=")
		if !ok || strings.HasPrefix(line, "#") {
			continue
		}
		v = strings.TrimSpace(v)
		if len(v) >= 2 && (v[0] == '\'' || v[0] == '"') && v[len(v)-1] == v[0] {
			v = v[1 : len(v)-1]
		}
		kv[strings.TrimPrefix(strings.TrimSpace(k), "export ")] = v
	}
	c := cred{host: kv["NIAGARA_PLAT_HOST"], port: kv["NIAGARA_PLAT_PORT"], user: kv["NIAGARA_PLAT_USER"],
		password: kv["NIAGARA_PLAT_PASSWORD"]}
	if c.port == "" {
		c.port = "5011"
	}
	for _, kv := range [][2]string{{"NIAGARA_PLAT_HOST", c.host}, {"NIAGARA_PLAT_USER", c.user},
		{"NIAGARA_PLAT_PASSWORD", c.password}, {"NIAGARA_PLAT_PORT", c.port}} {
		k, v := kv[0], kv[1]
		switch {
		case v == "":
			return cred{}, fmt.Errorf("env file %s has no %s", path, k)
		case strings.HasPrefix(v, "<") && strings.HasSuffix(v, ">"):
			return cred{}, fmt.Errorf("env file %s: %s is still wrapped in <...>; write the value without the brackets",
				path, k)
		}
	}
	return c, nil
}
