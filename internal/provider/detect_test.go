package provider_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/provider/codex"
)

// answers is a fake Run: what each executable prints for its version, keyed
// by base name. A missing name fails the test: detection ran something it
// should not have.
type answers map[string]provider.Output

func (a answers) host(t *testing.T, path, home, lad string, ran *[]string) provider.Host {
	return provider.Host{
		PATH: path, Home: home, LocalAppData: lad, Windows: true,
		Run: func(ctx context.Context, exe string, args ...string) (provider.Output, error) {
			*ran = append(*ran, filepath.Base(exe)+" "+strings.Join(args, " "))
			o, ok := a[filepath.Base(exe)]
			if !ok {
				t.Errorf("ran %s %v", exe, args)
				return provider.Output{}, errors.New("unexpected run")
			}
			if o.Stdout == "hang" {
				<-ctx.Done()
				return provider.Output{ExitCode: -1}, ctx.Err()
			}
			return o, nil
		},
	}
}

// touch creates empty files: detection finds a CLI by its file and never
// runs it except through Run.
func touch(t *testing.T, dir string, names ...string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// TestDetect is proposal step 1, item 2: detection with a fake PATH, for
// neither CLI, either one and both, the installers' folders, the npm shims
// that are never run, and a version that cannot be read.
func TestDetect(t *testing.T) {
	ok := answers{
		"claude.exe": {Stdout: "2.1.300 (Claude Code)\n"},
		"codex.exe":  {Stdout: "codex-cli 0.145.0\n"},
	}
	type want struct {
		installed     bool
		version, path string // path relative to the test's root
		problem       string
	}
	cases := []struct {
		name          string
		bin, home     []string // files in <root>/bin (on the PATH) and <root>/home/.local/bin
		lad           []string // files in <root>/lad/Programs/OpenAI/Codex/bin
		answers       answers
		claude, codex want
		ran           string
	}{
		{name: "neither",
			claude: want{problem: "claude not found on PATH"},
			codex:  want{problem: "codex not found on PATH"}},
		{name: "Claude Code only", bin: []string{"claude.exe"}, answers: ok,
			claude: want{installed: true, version: "2.1.300", path: `bin\claude.exe`},
			codex:  want{problem: "codex not found on PATH"},
			ran:    "claude.exe -v"},
		{name: "Codex only", bin: []string{"codex.exe"}, answers: ok,
			claude: want{problem: "claude not found on PATH"},
			codex:  want{installed: true, version: "0.145.0", path: `bin\codex.exe`},
			ran:    "codex.exe --version"},
		{name: "both", bin: []string{"claude.exe", "codex.exe"}, answers: ok,
			claude: want{installed: true, version: "2.1.300", path: `bin\claude.exe`},
			codex:  want{installed: true, version: "0.145.0", path: `bin\codex.exe`},
			ran:    "claude.exe -v,codex.exe --version"},
		{name: "installers' folders off the PATH", home: []string{"claude.exe"}, lad: []string{"codex.exe"}, answers: ok,
			claude: want{installed: true, version: "2.1.300", path: `home\.local\bin\claude.exe`},
			codex:  want{installed: true, version: "0.145.0", path: `lad\Programs\OpenAI\Codex\bin\codex.exe`},
			ran:    "claude.exe -v,codex.exe --version"},
		{name: "npm shims are never run", bin: []string{"claude.cmd", "codex.cmd", "codex.ps1"},
			claude: want{problem: "npm install found, native claude.exe needed"},
			codex:  want{problem: "npm install found (codex.cmd), native codex.exe needed"}},
		{name: "version check fails", bin: []string{"claude.exe", "codex.exe"},
			answers: answers{
				"claude.exe": {Stderr: "error: something broke\nmore\n", ExitCode: 1},
				"codex.exe":  {Stdout: "codex-cli unknown\n"},
			},
			claude: want{path: `bin\claude.exe`, problem: "version unreadable: exit 1: error: something broke"},
			codex:  want{path: `bin\codex.exe`, problem: "version unreadable: codex-cli unknown"},
			ran:    "claude.exe -v,codex.exe --version"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			bin := touch(t, filepath.Join(root, "bin"), c.bin...)
			touch(t, filepath.Join(root, "home", ".local", "bin"), c.home...)
			touch(t, filepath.Join(root, "lad", "Programs", "OpenAI", "Codex", "bin"), c.lad...)
			var ran []string
			h := c.answers.host(t, bin, filepath.Join(root, "home"), filepath.Join(root, "lad"), &ran)
			for _, p := range []struct {
				name string
				p    provider.Provider
				want want
			}{{"claude", &claude.Provider{Host: h}, c.claude}, {"codex", &codex.Provider{Host: h}, c.codex}} {
				d := p.p.Detect(context.Background())
				w := p.want
				if w.path != "" {
					w.path = filepath.Join(root, w.path)
				}
				if d.Installed != w.installed || d.Version != w.version || d.Path != w.path || d.Problem != w.problem {
					t.Errorf("%s: got installed %v, version %q, path %q, problem %q; want %+v",
						p.name, d.Installed, d.Version, d.Path, d.Problem, w)
				}
				if d.ID != p.p.ID() || d.Name != p.p.DisplayName() {
					t.Errorf("%s: id %q name %q", p.name, d.ID, d.Name)
				}
			}
			if got := strings.Join(ran, ","); got != c.ran {
				t.Errorf("ran %q, want %q", got, c.ran)
			}
		})
	}
}

// TestDetectVersionTimeout: a CLI that never answers is reported, within
// its version bound, not waited for.
func TestDetectVersionTimeout(t *testing.T) {
	root := t.TempDir()
	bin := touch(t, filepath.Join(root, "bin"), "claude.exe")
	var ran []string
	h := answers{"claude.exe": {Stdout: "hang"}}.host(t, bin, "", "", &ran)
	start := time.Now()
	d := (&claude.Provider{Host: h}).Detect(context.Background())
	if d.Installed || d.Problem != "version unreadable: no answer within 2s" {
		t.Errorf("installed %v, problem %q", d.Installed, d.Problem)
	}
	if el := time.Since(start); el < claude.VersionTimeout || el > claude.VersionTimeout+time.Second {
		t.Errorf("took %s, want about %s", el, claude.VersionTimeout)
	}
	if d.ElapsedTime < claude.VersionTimeout {
		t.Errorf("ElapsedTime %s", d.ElapsedTime)
	}
}
