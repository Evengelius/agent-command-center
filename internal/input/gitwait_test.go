package input

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestMain lets this test binary play git for TestGitPipesBounded (plan 48,
// design D4): with ACC_FAKE_GIT "git" it writes a line, starts a child that
// keeps its stdout and stderr, writes the child's pid to ACC_FAKE_GIT_PID and
// exits 0; with "child" it sleeps 15 s.
func TestMain(m *testing.M) {
	switch os.Getenv("ACC_FAKE_GIT") {
	case "git":
		fmt.Println("fake git output")
		self, _ := os.Executable()
		c := exec.Command(self)
		c.Env = append(os.Environ(), "ACC_FAKE_GIT=child")
		c.Stdout, c.Stderr = os.Stdout, os.Stderr
		if err := c.Start(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if f := os.Getenv("ACC_FAKE_GIT_PID"); f != "" {
			_ = os.WriteFile(f, []byte(strconv.Itoa(c.Process.Pid)), 0o644)
		}
		os.Exit(0)
	case "child":
		time.Sleep(15 * time.Second)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// TestGitPipesBounded is plan 48's criterion 48.7 (design D4; audit finding
// 3c): git exits 0 while a child it started keeps its stdout and stderr. A
// read, a write and a stream each end within 8 s, WaitDelay after git, with
// git's output and no error, where each waited for the child.
func TestGitPipesBounded(t *testing.T) {
	dir := t.TempDir()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	src, err := os.Open(self)
	if err != nil {
		t.Fatal(err)
	}
	dst, err := os.Create(filepath.Join(bin, "git.exe"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.Copy(dst, src)
	src.Close()
	if cerr := dst.Close(); err != nil || cerr != nil {
		t.Fatal(err, cerr)
	}
	pidFile := filepath.Join(dir, "child.pid")
	t.Setenv("PATH", bin)
	t.Setenv("ACC_FAKE_GIT", "git")
	t.Setenv("ACC_FAKE_GIT_PID", pidFile)
	stopChild := func() {
		if b, err := os.ReadFile(pidFile); err == nil {
			if pid, err := strconv.Atoi(string(b)); err == nil {
				if p, err := os.FindProcess(pid); err == nil {
					_ = p.Kill()
				}
			}
		}
	}
	t.Cleanup(stopChild)
	g := GitCLI{}
	for _, c := range []struct {
		name string
		run  func() (string, error)
	}{
		{"a read", func() (string, error) { return g.run(readLimit, dir, "status") }},
		{"a write", func() (string, error) {
			r := g.exec(dir, nil, nil, "add", []string{"add", "-A"}, nil, nil)
			return r.out, r.err
		}},
		{"a stream", func() (string, error) {
			var first string
			err := g.stream(readLimit, dir, strings.NewReader(""), func(r *bufio.Reader) error {
				first, _ = r.ReadString('\n')
				return nil
			}, "cat-file", "--batch")
			return first, err
		}},
	} {
		start := time.Now()
		out, err := c.run()
		d := time.Since(start)
		stopChild()
		if d > 8*time.Second || err != nil || !strings.Contains(out, "fake git output") {
			t.Errorf("%s: %q, %v, after %v; want git's output and no error within 8 s", c.name, out, err, d.Round(time.Millisecond))
		}
		t.Logf("%s: ended after %v", c.name, d.Round(time.Millisecond))
	}
}
