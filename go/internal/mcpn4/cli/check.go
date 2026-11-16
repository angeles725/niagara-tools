package cli

import (
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The start checks of main() and Server(...): what a parsed command line must hold before the
// server is made.

type startError struct{ msg string }

func (e *startError) Error() string { return e.msg }

// stations is the stations dict of main(): NAME -> URL (the last repeat wins) and the names in
// insertion order.
func stations(cfg Config) (map[string]string, []string, error) {
	urls := map[string]string{}
	var names []string
	for _, item := range cfg.Stations {
		name, url, ok := strings.Cut(item, "=")
		if !ok || name == "" || url == "" {
			return nil, nil, &startError{"--station expects NAME=URL, got " + pyjson.StrRepr(item)}
		}
		if _, seen := urls[name]; !seen {
			names = append(names, name)
		}
		urls[name] = url
	}
	return urls, names, nil
}

// Check makes the start checks of main() and Server(...) in their order: NAME=URL and
// NAME=PATH pairs, https station URLs (http:// only with --allow-http-for-tests), station names
// the per-station flags may name.
func Check(cfg Config) error {
	urls, names, err := stations(cfg)
	if err != nil {
		return err
	}
	for _, homes := range []struct {
		flag  string
		items []string
	}{{"--station-home", cfg.StationHomes}, {"--niagara-home", cfg.NiagaraHomes}} {
		for _, item := range homes.items {
			name, path, ok := strings.Cut(item, "=")
			if !ok || name == "" || path == "" {
				return &startError{homes.flag + " expects NAME=PATH, got " + pyjson.StrRepr(item)}
			}
		}
	}
	for _, name := range names {
		url := urls[name]
		if !strings.HasPrefix(url, "https://") && !(cfg.AllowHTTP && strings.HasPrefix(url, "http://")) {
			return &startError{"station " + name + ": URL must start with https://"}
		}
	}
	for _, f := range []struct {
		flag  string
		names []string
	}{{"--insecure-tls", cfg.InsecureTLS}, {"--allow-tier-b", cfg.AllowTierB}, {"--allow-tier-c", cfg.AllowTierC}} {
		unknown := map[string]bool{}
		for _, n := range f.names {
			if _, ok := urls[n]; !ok {
				unknown[n] = true
			}
		}
		if len(unknown) > 0 {
			list := make([]string, 0, len(unknown))
			for n := range unknown {
				list = append(list, n)
			}
			sort.Strings(list)
			return &startError{f.flag + " names unconfigured station(s): " + strings.Join(list, ", ")}
		}
	}
	return nil
}

// niagaraHomes is the --niagara-home pairs Check accepted as NAME -> PATH (the path keeps every
// "=" after the first, the last repeat wins).
func niagaraHomes(cfg Config) map[string]string {
	homes := map[string]string{}
	for _, item := range cfg.NiagaraHomes {
		name, path, _ := strings.Cut(item, "=")
		homes[name] = path
	}
	return homes
}
