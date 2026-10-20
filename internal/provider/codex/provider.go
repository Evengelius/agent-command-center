package codex

import (
	"context"
	"errors"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/Evengelius/agent-command-center/internal/proc"
	"github.com/Evengelius/agent-command-center/internal/provider"
)

// Name is the adapter's display name.
const Name = "OpenAI Codex"

// TestedVersion is the last Codex CLI version probed; it matches the
// "Last probed" line of doc/research/probes/README.md (doc/evaluation.md,
// After a CLI update).
const TestedVersion = "0.163.0"

// VersionTimeout bounds `codex --version`. No source gives one; it is
// inferred, within the 10 s detection bound (ASM-21).
const VersionTimeout = 5 * time.Second

// versionRe matches "codex-cli 0.145.0" (CX-2, "Version").
var versionRe = regexp.MustCompile(`([0-9]+\.[0-9]+\.[0-9]+)`)

// Provider is the Codex adapter (architecture.md, Codex adapter).
type Provider struct {
	Host    provider.Host
	Spawner proc.Spawner // proc.Exec when nil
	// Args are added after app-server.
	Args      []string
	CloseStep time.Duration    // CloseStep when 0
	Now       func() time.Time // time.Now when nil

	mu    sync.Mutex
	model string // the fallback model, kept for a resume after a crash
}

func (p *Provider) ID() string          { return ProviderID }
func (p *Provider) DisplayName() string { return Name }

// Detect finds codex.exe on the PATH, else in the standalone installer's
// %LOCALAPPDATA%\Programs\OpenAI\Codex\bin, and reads its version. The npm
// shims (codex.cmd, codex.ps1) are never run; resolving the native binary
// inside an npm install is not built in the MVP and is reported as such
// (CX-2, "Recommendation"). Only the version is read: the session's
// account/rateLimits/read tells a ChatGPT sign-in from an API key
// (architecture.md, Codex adapter, Detect; plan 21, item 5.2).
func (p *Provider) Detect(ctx context.Context) (d provider.Detection) {
	start := time.Now()
	h := p.Host
	d = provider.Detection{ID: ProviderID, Name: Name, Tested: TestedVersion}
	defer func() { d.ElapsedTime = time.Since(start) }()

	names := []string{"codex"}
	if h.Windows {
		names = []string{"codex.exe"}
	}
	exe := h.Find(names...)
	if exe == "" && h.Windows && h.LocalAppData != "" {
		if p := filepath.Join(h.LocalAppData, "Programs", "OpenAI", "Codex", "bin", "codex.exe"); provider.IsFile(p) {
			exe = p
		}
	}
	if exe == "" {
		if h.Windows && h.Find("codex.cmd", "codex.ps1") != "" {
			d.Problem = "npm install found (codex.cmd), native codex.exe needed"
		} else {
			d.Problem = "codex not found on PATH"
		}
		return d
	}
	d.Path = exe
	d.Version, d.Problem = h.Version(ctx, exe, VersionTimeout, versionRe, "--version")
	d.Installed = d.Problem == ""
	return d
}

// Start launches `codex app-server` in the project directory and sends
// initialize; initialized and thread/start (or thread/resume) follow its
// response, and session.ready the thread's. The process belongs to a job
// object, so it never outlives the dashboard (ASM-19), and cancelling ctx
// kills its whole tree.
func (p *Provider) Start(ctx context.Context, sc provider.SessionContext) (provider.Session, error) {
	if sc.Executable == "" {
		return nil, errors.New("no Codex executable")
	}
	if sc.ProjectPath == "" {
		return nil, errors.New("no project directory")
	}
	s, err := p.start(ctx, sc)
	if err != nil {
		return nil, err // not a typed nil
	}
	return s, nil
}

// fallbackModel is the model a refusal fell back to in this run, "" before.
func (p *Provider) fallbackModel() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.model
}

func (p *Provider) setFallbackModel(m string) {
	p.mu.Lock()
	p.model = m
	p.mu.Unlock()
}
