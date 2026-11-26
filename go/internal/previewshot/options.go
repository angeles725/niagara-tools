package previewshot

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const usage = `usage: preview-shot <url> <file> [--size WxH] [--budget ms]
       preview-shot --serve <rc-dir> <file> [--path /hmi] [--prefix /mount] [--mock file.json] [--size WxH] [--budget ms]
  defaults: 1280x800, 25000 ms of virtual time, path /hmi`

var sizeRe = regexp.MustCompile(`^([1-9][0-9]*)x([1-9][0-9]*)$`)

type options struct {
	url, serve, out, path, prefix, mock string
	w, h, budget                        int
}

func parse(args []string) (options, error) {
	o := options{path: "/hmi", w: 1280, h: 800, budget: 25000}
	var pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "--") {
			pos = append(pos, a)
			continue
		}
		var v string
		switch a {
		case "--size", "--budget", "--serve", "--path", "--prefix", "--mock":
			if i++; i >= len(args) {
				return o, fmt.Errorf("%s needs a value", a)
			}
			v = args[i]
		default:
			return o, fmt.Errorf("unknown option %s", a)
		}
		if err := o.set(a, v); err != nil {
			return o, err
		}
	}
	return o, o.finish(pos)
}

func (o *options) set(flag, v string) error {
	switch flag {
	case "--size":
		m := sizeRe.FindStringSubmatch(v)
		if m == nil {
			return fmt.Errorf("--size %q is not WxH", v)
		}
		o.w, _ = strconv.Atoi(m[1])
		o.h, _ = strconv.Atoi(m[2])
	case "--budget":
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return fmt.Errorf("--budget %q is not a positive number of milliseconds", v)
		}
		o.budget = n
	case "--serve":
		o.serve = v
	case "--path":
		if !strings.HasPrefix(v, "/") {
			return fmt.Errorf("--path %q must start with /", v)
		}
		o.path = v
	case "--prefix":
		o.prefix = v
	case "--mock":
		o.mock = v
	}
	return nil
}

// finish takes the positional arguments (<url> <file>, or only <file> with --serve) and checks the combination.
func (o *options) finish(pos []string) error {
	if o.serve == "" && (o.prefix != "" || o.mock != "" || o.path != "/hmi") {
		return errors.New("--path, --prefix and --mock go with --serve")
	}
	want := 2
	if o.serve != "" {
		want = 1
	}
	if len(pos) != want {
		return fmt.Errorf("expected %d argument(s) besides the options, got %d", want, len(pos))
	}
	if o.serve == "" {
		if !strings.HasPrefix(pos[0], "http://") && !strings.HasPrefix(pos[0], "https://") && !strings.HasPrefix(pos[0], "file://") {
			return fmt.Errorf("%q is not an http://, https:// or file:// URL", pos[0])
		}
		o.url = pos[0]
	}
	out, err := filepath.Abs(pos[want-1])
	if err != nil {
		return err
	}
	o.out = out
	return nil
}
