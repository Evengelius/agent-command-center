package claude

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/Evengelius/agent-command-center/internal/proc"
	"github.com/Evengelius/agent-command-center/internal/provider"
)

// Name is the adapter's display name.
const Name = "Claude Code"

// VersionTimeout is the Python SDK's bound on `claude -v` (CC-2, rule 3).
const VersionTimeout = 2 * time.Second

// versionRe matches "2.1.300 (Claude Code)" (CC-2, rule 3).
var versionRe = regexp.MustCompile(`^\s*([0-9]+\.[0-9]+\.[0-9]+)`)

// Provider is the Claude Code adapter (architecture.md, Claude Code adapter).
type Provider struct {
	Host    provider.Host
	Spawner proc.Spawner // proc.Exec when nil
	// Args are added after the session's own flags; the live gate passes
	// --model haiku.
	Args      []string
	CloseStep time.Duration    // CloseStep when 0
	Now       func() time.Time // time.Now when nil
}

func (p *Provider) ID() string          { return ProviderID }
func (p *Provider) DisplayName() string { return Name }

// Detect finds claude.exe on the PATH, else at the native installer's
// %USERPROFILE%\.local\bin\claude.exe, and reads its version. A .cmd or .bat
// shim is never used: Windows re-parses its arguments through cmd.exe (CC-2,
// rules 1 and 2). The sign-in kind is not read yet.
func (p *Provider) Detect(ctx context.Context) (d provider.Detection) {
	start := time.Now()
	h := p.Host
	d = provider.Detection{ID: ProviderID, Name: Name}
	defer func() { d.ElapsedTime = time.Since(start) }()

	names, local := []string{"claude"}, filepath.Join(h.Home, ".local", "bin", "claude")
	if h.Windows {
		names, local = []string{"claude.exe", "claude.com"}, local+".exe"
	}
	exe := h.Find(names...)
	if exe == "" && h.Home != "" && provider.IsFile(local) {
		exe = local
	}
	if exe == "" {
		if h.Windows && h.Find("claude.cmd", "claude.bat", "claude.ps1") != "" {
			d.Problem = "npm install found, native claude.exe needed"
		} else {
			d.Problem = "claude not found on PATH"
		}
		return d
	}
	d.Path = exe
	d.Version, d.Problem = h.Version(ctx, exe, VersionTimeout, versionRe, "-v")
	d.Installed = d.Problem == ""
	return d
}

// Start launches `claude -p` in the project directory and sends the
// initialize request; session.ready follows its response. The process
// belongs to a job object, so it never outlives the dashboard (ASM-19), and
// cancelling ctx kills its whole tree.
func (p *Provider) Start(ctx context.Context, sc provider.SessionContext) (provider.Session, error) {
	if sc.Executable == "" {
		return nil, errors.New("no Claude Code executable")
	}
	if sc.ProjectPath == "" {
		return nil, errors.New("no project directory")
	}
	s, err := p.start(ctx, sc, childEnv(os.Environ(), p.Host))
	if err != nil {
		return nil, err // not a typed nil
	}
	return s, nil
}
