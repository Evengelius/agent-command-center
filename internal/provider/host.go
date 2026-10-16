package provider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Host is what detection reads from the system (architecture.md, adapters,
// Detect). Tests replace PATH, the folders and Run, so detection is checked
// without the real CLIs.
type Host struct {
	PATH         string // the PATH value, searched in order
	Home         string // %USERPROFILE% or $HOME
	LocalAppData string // %LOCALAPPDATA%; "" outside Windows
	Windows      bool
	// Run runs exe with args, stdin empty, and returns what it printed and
	// its exit code; err is set when it could not run or timed out.
	Run func(ctx context.Context, exe string, args ...string) (Output, error)
}

// Output is what one run printed.
type Output struct {
	Stdout, Stderr string
	ExitCode       int
}

// SystemHost reads the real environment.
func SystemHost() Host {
	home, _ := os.UserHomeDir()
	return Host{
		PATH:         os.Getenv("PATH"),
		Home:         home,
		LocalAppData: os.Getenv("LOCALAPPDATA"),
		Windows:      filepath.Separator == '\\',
		Run:          run,
	}
}

// run runs a detection command. A version check reads nothing from the
// CLI's configuration and makes no model call (proposal step 1, item 3).
func run(ctx context.Context, exe string, args ...string) (Output, error) {
	cmd := exec.CommandContext(ctx, exe, args...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	cmd.WaitDelay = time.Second // a grandchild holding the pipes cannot block detection
	err := cmd.Run()
	o := Output{Stdout: out.String(), Stderr: errb.String(), ExitCode: cmd.ProcessState.ExitCode()}
	var exit *exec.ExitError
	if errors.As(err, &exit) && ctx.Err() == nil {
		err = nil // a non-zero exit is an answer, reported by its code
	}
	return o, err
}

// Find looks for the first of names in each PATH folder in turn; "" when
// none exists as a regular file. It runs nothing.
func (h Host) Find(names ...string) string {
	for _, dir := range filepath.SplitList(h.PATH) {
		if dir == "" {
			continue
		}
		for _, n := range names {
			if p := filepath.Join(dir, n); IsFile(p) {
				return p
			}
		}
	}
	return ""
}

// IsFile reports whether p is an existing regular file.
func IsFile(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.Mode().IsRegular()
}

// Version runs exe with args within timeout and returns the first match of
// re's first group in its standard output, or the problem shown on the
// startup screen ("version unreadable: …", ui-design.md, Questions).
func (h Host) Version(ctx context.Context, exe string, timeout time.Duration, re *regexp.Regexp, args ...string) (string, string) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	o, err := h.Run(ctx, exe, args...)
	switch {
	case ctx.Err() != nil:
		return "", fmt.Sprintf("version unreadable: no answer within %s", timeout)
	case err != nil:
		return "", "version unreadable: " + err.Error()
	case o.ExitCode != 0:
		msg := fmt.Sprintf("version unreadable: exit %d", o.ExitCode)
		if l := firstLine(o.Stderr); l != "" {
			msg += ": " + l
		}
		return "", msg
	}
	m := re.FindStringSubmatch(o.Stdout)
	if m == nil {
		return "", "version unreadable: " + strings.TrimSpace(firstLine(o.Stdout))
	}
	return m[1], ""
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i]
	}
	if len(s) > 120 {
		s = s[:120] + "…"
	}
	return s
}
