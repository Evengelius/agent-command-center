package codex

import (
	"context"
	"errors"
	"path/filepath"
	"regexp"
	"time"

	"github.com/Evengelius/agent-command-center/internal/provider"
)

// Name is the adapter's display name.
const Name = "OpenAI Codex"

// VersionTimeout bounds `codex --version`. No source gives one; it is
// inferred, within the 10 s detection bound (ASM-21).
const VersionTimeout = 5 * time.Second

// versionRe matches "codex-cli 0.145.0" (CX-2, "Version").
var versionRe = regexp.MustCompile(`([0-9]+\.[0-9]+\.[0-9]+)`)

// Provider is the Codex adapter (architecture.md, Codex adapter).
type Provider struct {
	Host provider.Host
}

func (p *Provider) ID() string          { return ProviderID }
func (p *Provider) DisplayName() string { return Name }

// Detect finds codex.exe on the PATH, else in the standalone installer's
// %LOCALAPPDATA%\Programs\OpenAI\Codex\bin, and reads its version. The npm
// shims (codex.cmd, codex.ps1) are never run; resolving the native binary
// inside an npm install is not built in the MVP and is reported as such
// (CX-2, "Recommendation"). The sign-in kind is not read yet.
func (p *Provider) Detect(ctx context.Context) (d provider.Detection) {
	start := time.Now()
	h := p.Host
	d = provider.Detection{ID: ProviderID, Name: Name}
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

// Start arrives with the adapter's session in step 5.
func (p *Provider) Start(context.Context, provider.SessionContext) (provider.Session, error) {
	return nil, errors.New("the Codex session is not built yet (plan 4, step 5)")
}
