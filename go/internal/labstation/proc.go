package labstation

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// running reads the pid file and says whether that process is alive (a zombie counts as gone).
func (l *lab) running() (int, bool) {
	b, err := os.ReadFile(l.path("pid"))
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || pid < 1 || syscall.Kill(pid, 0) != nil {
		return pid, false
	}
	if st, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid)); err == nil {
		if f := strings.Fields(string(st[strings.LastIndexByte(string(st), ')')+1:])); len(f) > 0 && f[0] == "Z" {
			return pid, false
		}
	}
	return pid, true
}

func (l *lab) status() int {
	if pid, ok := l.running(); ok {
		fmt.Fprintf(l.s.Out, "lab-station: %s running (pid %d)\n", l.name, pid)
		return 0
	}
	fmt.Fprintf(l.s.Out, "lab-station: %s stopped\n", l.name)
	return exitFail
}

func (l *lab) start() int {
	if pid, ok := l.running(); ok {
		fmt.Fprintf(l.s.Err, "lab-station: %s is already running (pid %d)\n", l.name, pid)
		return exitFail
	}
	if err := l.prepare(); err != nil {
		fmt.Fprintf(l.s.Err, "lab-station: %v\n", err)
		return exitFail
	}
	in, err := os.OpenFile(l.path("in"), os.O_RDWR, 0) // the station holds both ends: no end of file, no holder
	if err != nil {
		fmt.Fprintf(l.s.Err, "lab-station: %v\n", err)
		return exitFail
	}
	defer in.Close()
	log, err := os.Create(l.path("out.log"))
	if err != nil {
		fmt.Fprintf(l.s.Err, "lab-station: %v\n", err)
		return exitFail
	}
	defer log.Close()
	c := exec.Command(l.exe, l.name)
	c.Dir, c.Stdin, c.Stdout, c.Stderr = filepath.Dir(l.exe), in, log, log
	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := c.Start(); err != nil {
		fmt.Fprintf(l.s.Err, "lab-station: %s did not start: %v\n", l.name, err)
		return exitFail
	}
	exited := make(chan struct{})
	go func() { _ = c.Wait(); close(exited) }() // reaps it while this process lives
	pid := c.Process.Pid
	if err := os.WriteFile(l.path("pid"), []byte(strconv.Itoa(pid)+"\n"), 0o600); err != nil {
		fmt.Fprintf(l.s.Err, "lab-station: %v\n", err)
		return exitFail
	}
	line, ok, _ := l.waitLine(0, func(s string) bool { return strings.Contains(s, startedMark) }, exited)
	if !ok {
		fmt.Fprintf(l.s.Err, "lab-station: %s did not start: %s\n", l.name, line)
		return exitFail
	}
	fmt.Fprintf(l.s.Out, "lab-station: %s started (pid %d): %s\n", l.name, pid, line)
	return 0
}

// prepare makes the state dir and the console pipe, and keeps the previous log as out.log.1.
func (l *lab) prepare() error {
	if err := os.MkdirAll(l.dir, 0o700); err != nil {
		return err
	}
	if fi, err := os.Stat(l.path("in")); err != nil || fi.Mode()&os.ModeNamedPipe == 0 {
		_ = os.Remove(l.path("in"))
		if err := syscall.Mkfifo(l.path("in"), 0o600); err != nil {
			return fmt.Errorf("console pipe %s: %v", l.path("in"), err)
		}
	}
	if fileExists(l.path("out.log")) {
		_ = os.Rename(l.path("out.log"), l.path("out.log.1"))
	}
	return nil
}

// send writes one console line to the pipe; it fails when no station reads it.
func (l *lab) send(verb string) error {
	f, err := os.OpenFile(l.path("in"), os.O_WRONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(verb + "\r\n")
	return err
}

func (l *lab) stop() int {
	_, ok := l.running()
	if !ok || l.send("quit") != nil {
		fmt.Fprintf(l.s.Err, "lab-station: %s is not running under lab-station (no console pipe): stop it another way\n",
			l.name)
		return exitFail
	}
	gone := make(chan struct{})
	go func() {
		for {
			if _, ok := l.running(); !ok {
				close(gone)
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()
	// the whole log of this run: a shutdown line only ever ends a run, and start keeps the previous run apart
	line, ok, exited := l.waitLine(0, func(s string) bool { return strings.Contains(s, shutdownMark) }, gone)
	if !ok && !exited {
		fmt.Fprintf(l.s.Err, "lab-station: %s did not stop: %s\n", l.name, line)
		return exitFail
	}
	if !ok {
		line = "the process exited without " + shutdownMark
	}
	_ = os.Remove(l.path("pid"))
	fmt.Fprintf(l.s.Out, "lab-station: %s stopped: %s\n", l.name, line)
	return 0
}

func (l *lab) console(verb string) int {
	if _, ok := l.running(); !ok {
		fmt.Fprintf(l.s.Err, "lab-station: %s is not running under lab-station\n", l.name)
		return exitFail
	}
	fi, err := os.Stat(l.path("out.log"))
	if err != nil {
		fmt.Fprintf(l.s.Err, "lab-station: %v\n", err)
		return exitFail
	}
	if err := l.send(verb); err != nil {
		fmt.Fprintf(l.s.Err, "lab-station: %v\n", err)
		return exitFail
	}
	text, ok := l.waitText(fi.Size(), "niagara>")
	if !ok {
		fmt.Fprintf(l.s.Err, "lab-station: no console prompt after %s in %s\n", verb, l.timeout)
		return exitFail
	}
	text = strings.TrimRight(strings.TrimLeft(text[:strings.LastIndex(text, "niagara>")], "\n"), "\n")
	fmt.Fprintln(l.s.Out, text)
	return 0
}
