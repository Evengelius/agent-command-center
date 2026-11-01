// Command wirelog is the test-only logging wrapper for a live run of the
// dashboard (doc/evaluation.md, After a CLI update). Built as codex.exe or
// claude.exe and placed first on the PATH of the dashboard's window, it runs
// the real CLI:
//   - any call that is not a session (the dashboard's --version) is passed
//     through unchanged;
//   - for a session, app-server for Codex or stream-json for Claude Code, the
//     arguments are passed through unchanged (no override), and stdin, stdout
//     and stderr are copied byte for byte into in.jsonl, out.jsonl and err.txt
//     in the log folder, the dashboard's side written first. in-times.txt and
//     out-times.txt give, for each line, "<unix ms> <line number from 1>
//     <bytes>", the time its newline arrived.
//
// wrap.json, next to this executable: {"real": "...", "log": "..."}
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sync"
	"time"
)

type conf struct {
	Real string `json:"real"`
	Log  string `json:"log"`
}

// lineTimes writes one row for each newline it sees: the time, the line's
// number and its length in bytes.
type lineTimes struct {
	f    *os.File
	n    int
	size int
}

func (l *lineTimes) Write(p []byte) (int, error) {
	now := time.Now().UnixMilli()
	for _, b := range p {
		l.size++
		if b == '\n' {
			l.n++
			fmt.Fprintf(l.f, "%d %d %d\n", now, l.n, l.size)
			l.size = 0
		}
	}
	return len(p), nil
}

func main() {
	self, _ := os.Executable()
	b, err := os.ReadFile(filepath.Join(filepath.Dir(self), "wrap.json"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "wrap:", err)
		os.Exit(2)
	}
	var c conf
	if err := json.Unmarshal(b, &c); err != nil {
		fmt.Fprintln(os.Stderr, "wrap:", err)
		os.Exit(2)
	}
	args := os.Args[1:]
	if !slices.Contains(args, "app-server") && !slices.Contains(args, "stream-json") {
		cmd := exec.Command(c.Real, args...)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		os.Exit(code(cmd.Run()))
	}
	_ = os.MkdirAll(c.Log, 0o755)
	open := func(name string) *os.File {
		f, err := os.Create(filepath.Join(c.Log, name))
		if err != nil {
			fmt.Fprintln(os.Stderr, "wrap:", err)
			os.Exit(2)
		}
		return f
	}
	start := time.Now()
	in, out, errf := open("in.jsonl"), open("out.jsonl"), open("err.txt")
	inT, outT := &lineTimes{f: open("in-times.txt")}, &lineTimes{f: open("out-times.txt")}
	cmd := exec.Command(c.Real, args...)
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "wrap:", err)
		os.Exit(2)
	}
	meta, _ := json.Marshal(map[string]any{"argv": append([]string{c.Real}, args...), "start_ms": start.UnixMilli(), "wrapper_pid": os.Getpid(), "cli_pid": cmd.Process.Pid})
	_ = os.WriteFile(filepath.Join(c.Log, "argv.json"), meta, 0o644)
	go func() {
		_, _ = io.Copy(io.MultiWriter(stdin, in, inT), os.Stdin)
		_ = stdin.Close()
	}()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); _, _ = io.Copy(io.MultiWriter(os.Stdout, out, outT), stdout) }()
	go func() { defer wg.Done(); _, _ = io.Copy(io.MultiWriter(os.Stderr, errf), stderr) }()
	wg.Wait()
	rc := code(cmd.Wait())
	for _, f := range []*os.File{in, out, errf, inT.f, outT.f} {
		_ = f.Close()
	}
	end, _ := json.Marshal(map[string]any{"end_ms": time.Now().UnixMilli(), "exit": rc})
	_ = os.WriteFile(filepath.Join(c.Log, "end.json"), end, 0o644)
	os.Exit(rc)
}

func code(err error) int {
	if e, ok := err.(*exec.ExitError); ok {
		return e.ExitCode()
	}
	if err != nil {
		return 2
	}
	return 0
}
