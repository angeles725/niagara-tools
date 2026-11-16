package golden

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// entry is one tree section: a file (with its bytes), a directory or a symlink (to its target).
type entry struct {
	kind, body string
	mode       os.FileMode
}

// parseEntry reads a tree section name and body: the path, then " (dir)", " (symlink)" and " (mode NNN)"
// markers in any order (golden.Read already took off the content markers).
func parseEntry(name, body string) (string, entry, error) {
	e := entry{kind: "file", body: body, mode: 0o644}
	explicit := false
	for {
		i := strings.LastIndex(name, " (")
		if i < 0 || !strings.HasSuffix(name, ")") {
			break
		}
		switch m := name[i+2 : len(name)-1]; {
		case m == "dir":
			e.kind, e.body = "dir", ""
		case m == "symlink":
			e.kind, e.body = "symlink", strings.TrimSuffix(body, "\n")
		case strings.HasPrefix(m, "mode "):
			v, err := strconv.ParseUint(strings.TrimPrefix(m, "mode "), 8, 32)
			if err != nil || v > 0o777 {
				return "", e, fmt.Errorf("%s: bad mode", name)
			}
			e.mode, explicit = os.FileMode(v), true
		default:
			return name, e, nil // not a marker: part of the path
		}
		name = name[:i]
	}
	if e.kind == "dir" && !explicit {
		e.mode = 0o755
	}
	return name, e, nil
}

// split sorts a case's sections: @env lines, the tree before the run, the tree it must leave (before
// with every "@after <path>" set and every "@gone <path>" removed) and the @kind text.
func split(c Case, sub *strings.Replacer) (env map[string]string, before, after map[string]entry, kind string, err error) {
	env, before, after = map[string]string{}, map[string]entry{}, map[string]entry{}
	gone := map[string]bool{}
	sections := map[string]string{}
	for k, v := range c.Files {
		sections[k] = sub.Replace(v)
	}
	for _, d := range c.Dirs {
		sections[d+" (dir)"] = ""
	}
	for key, body := range sections {
		var target map[string]entry
		switch rest, isAfter := strings.CutPrefix(key, "@after "); {
		case key == "@kind":
			kind = c.Files[key]
		case key == "@env":
			for _, kv := range strings.Split(strings.TrimSuffix(body, "\n"), "\n") {
				if k, v, ok := strings.Cut(kv, "="); ok {
					env[k] = v
				}
			}
		case strings.HasPrefix(key, "@gone "):
			gone[strings.TrimPrefix(key, "@gone ")] = true
		case isAfter:
			target, key = after, rest
		case strings.HasPrefix(key, "@"):
			return nil, nil, nil, "", fmt.Errorf("unknown section %q", key)
		default:
			target = before
		}
		if target != nil {
			p, e, perr := parseEntry(key, body)
			if perr != nil {
				return nil, nil, nil, "", perr
			}
			target[p] = e
		}
	}
	for k, v := range defaultEnv {
		if _, ok := env[k]; !ok {
			env[k] = sub.Replace(v)
		}
	}
	merged := map[string]entry{}
	for p, e := range before {
		if !gone[p] {
			merged[p] = e
		}
	}
	for p, e := range after {
		merged[p] = e
	}
	return env, before, merged, kind, nil
}

// readRecordEnv reads dir/record.env (KEY=VALUE lines, # comments); nil when there is none.
func readRecordEnv(dir string) (map[string]string, error) {
	data, err := os.ReadFile(filepath.Join(dir, "record.env"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	conf := map[string]string{}
	for _, ln := range strings.Split(string(data), "\n") {
		if k, v, ok := strings.Cut(ln, "="); ok && !strings.HasPrefix(ln, "#") {
			conf[k] = strings.Trim(v, `'"`)
		}
	}
	return conf, err
}

// kindProblem checks @kind against the SOURCE, COMMIT and SHA256 of record.env: a recorded case names the pinned source and its tools, a
// truth-* case (a declared difference from the source) and only it starts "truth: ".
func kindProblem(conf map[string]string, name, kind string) string {
	if conf["SOURCE"] == "" {
		return ""
	}
	want := fmt.Sprintf("recorded from %s at %s (sha256 %s)\n", conf["SOURCE"], conf["COMMIT"], conf["SHA256"])
	if strings.HasPrefix(name, "truth-") {
		if !strings.HasPrefix(kind, "truth: ") {
			return fmt.Sprintf("@kind %q: a truth-* case says why with \"truth: \"", kind)
		}
		return ""
	}
	if !strings.HasPrefix(kind, want) || !strings.Contains(kind, "\ntools: ") {
		return fmt.Sprintf("@kind %q, want %q then a tools: line (record/record.sh)", kind, want)
	}
	return ""
}
