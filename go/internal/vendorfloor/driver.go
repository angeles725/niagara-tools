package vendorfloor

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// driverJS is the acorn driver node runs from stdin; its header documents the protocol decoded here.
//
//go:embed driver.js
var driverJS string

// request is what the driver is asked to do (argv[2], JSON).
type request struct {
	Acorn       string   `json:"acorn"`       // the require() spec of acorn
	ECMAVersion int      `json:"ecmaVersion"` // the floor the files are parsed at
	Names       []string `json:"names"`       // the name tokens to report
}

// Record types of the driver's output.
const (
	recFile        = "file"
	recUnavailable = "unavailable" // terminal: acorn could not be loaded
	recFault       = "fault"       // terminal: an exception ended the run
	recDone        = "done"        // terminal: every file was reported
)

// record is one output line of the driver.
type record struct {
	Type    string       `json:"type"`
	Message string       `json:"message"` // recFault
	Index   int          `json:"index"`   // recFile: the position of the file in the arguments
	Syntax  *syntaxError `json:"syntax"`  // recFile: nil when the file is a classic script at the floor
	Names   []nameUse    `json:"names"`   // recFile
}

// syntaxError is why a file does not parse as a classic script at the floor.
type syntaxError struct {
	Line    int    `json:"line"`    // 0 when acorn gave no location
	Message string `json:"message"` // acorn's text, "(line:col)" suffix included
	Module  bool   `json:"module"`  // it parses at the floor as an ES module
}

// nameUse is one requested name token with the text of its neighbours ("" where there is none).
type nameUse struct {
	Name  string `json:"name"`
	Prev  string `json:"prev"`
	Prev2 string `json:"prev2"`
	Next  string `json:"next"`
	Line  int    `json:"line"`
}

// drive runs the driver over files and hands each file record to visit as it arrives. It returns the
// terminal record, or an error when node could not run or ended without one: an incomplete run is
// never a clean one. node's own stderr goes to stderr.
func drive(node string, req request, files []string, stderr io.Writer, visit func(record)) (record, error) {
	arg, err := json.Marshal(req)
	if err != nil {
		return record{}, err
	}
	cmd, stdout, err := start(node, append([]string{"-", string(arg)}, files...), stderr)
	if err != nil {
		return record{}, err
	}
	end, decodeErr := decode(stdout, visit)
	_, _ = io.Copy(io.Discard, stdout) // let node finish writing before Wait closes the pipe
	waitErr := cmd.Wait()
	if decodeErr == nil {
		return end, nil
	}
	if waitErr != nil {
		return record{}, fmt.Errorf("%s ended without a result (%v)", node, waitErr)
	}
	return record{}, fmt.Errorf("%s ended without a result (%v)", node, decodeErr)
}

// decode reads records until a terminal one, which it returns.
func decode(r io.Reader, visit func(record)) (record, error) {
	dec := json.NewDecoder(r)
	for {
		var rec record
		if err := dec.Decode(&rec); err != nil {
			if errors.Is(err, io.EOF) {
				return record{}, errors.New("no terminal record")
			}
			return record{}, err
		}
		switch rec.Type {
		case recFile:
			visit(rec)
		case recUnavailable, recFault, recDone:
			return rec, nil
		default:
			return record{}, fmt.Errorf("unknown record type %q", rec.Type)
		}
	}
}

// start starts node with driverJS on stdin. A parallel fork that still holds node open for writing
// makes exec fail with ETXTBSY until its own exec: that transient refusal is retried a few times
// (as reportmodule.run does), any other start failure is returned at once.
func start(node string, args []string, stderr io.Writer) (*exec.Cmd, io.ReadCloser, error) {
	for try := 0; ; try++ {
		cmd := exec.Command(node, args...)
		cmd.Stdin = strings.NewReader(driverJS)
		cmd.Stderr = stderr
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return nil, nil, err
		}
		err = cmd.Start()
		if err == nil {
			return cmd, stdout, nil
		}
		if !errors.Is(err, syscall.ETXTBSY) || try == startTries-1 {
			return nil, nil, fmt.Errorf("cannot run %s: %w", node, err)
		}
		time.Sleep(time.Duration(try+1) * 20 * time.Millisecond)
	}
}

// startTries bounds the ETXTBSY retries of start (about 1 s in all).
const startTries = 10
