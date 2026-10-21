package app

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/proc"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/provider/codex"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 22, amendment 19: Codex through the local proxy, on the startup screen
// (criterion 19.6) and across a resume (criteria 19.3 and 19.7).

const (
	proxyAddr = "http://127.0.0.1:3128"
	proxyLine = "Codex through the proxy " + proxyAddr + " (config.json)"
)

// settingsFrom loads a configuration file's content as main does: the keys,
// then Codex's proxy.
func settingsFrom(t *testing.T, content string) Options {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	keys, problem := input.LoadKeys(p)
	proxy, proxyProblem := input.LoadCodexProxy(p)
	return Options{Keys: keys, KeysProblem: problem, CodexProxy: proxy, CodexProxyProblem: proxyProblem}
}

// startupAt is the startup screen of an application run with opt, at w by h:
// the frame as the terminal receives it, and as text. Called in a bubble.
func startupAt(t *testing.T, opt Options, w, h int) (raw, text string) {
	t.Helper()
	cli := &scriptCLI{id: "claude-code", sess: newScriptSession()}
	opt.Providers, opt.Cwd = []provider.Provider{cli}, t.TempDir()
	hn := newHarness(t, opt)
	hn.send(tea.WindowSizeMsg{Width: w, Height: h})
	raw = hn.m.View().Content
	return raw, plainText(raw)
}

// words are a frame's words in order, whatever the line breaks and the frame
// around them: a message wraps.
func words(text string) string {
	return strings.Join(strings.Fields(strings.NewReplacer("✗", " ", "│", " ").Replace(text)), " ")
}

// sgrSequence matches the one escape sequence that styles a span.
var sgrSequence = regexp.MustCompile(`^\x1b\[[0-9;:]*m$`)

// sgrBefore is the escape sequence that styles text in a raw frame: each span
// is written as its sequence, its text and a reset.
func sgrBefore(t *testing.T, raw, text string) string {
	t.Helper()
	i := strings.Index(raw, text)
	if i < 0 {
		t.Fatalf("%q is not in the frame", text)
	}
	j := strings.LastIndex(raw[:i], "\x1b[")
	if j < 0 || !sgrSequence.MatchString(raw[j:i]) {
		t.Fatalf("%q is not styled by one sequence: %q", text, raw[max(j, 0):i])
	}
	return raw[j:i]
}

// rowsOf are the rows of a frame's text.
func rowsOf(text string) []string { return strings.Split(strings.TrimRight(text, "\n"), "\n") }

// firstRow is the index of the first row holding s, -1 when none does.
func firstRow(rows []string, s string) int {
	for i, r := range rows {
		if strings.Contains(r, s) {
			return i
		}
	}
	return -1
}

// TestCodexProxyOnStartup is criterion 19.6, as TestKeysProblemOnStartup is
// plan 21's: from the file as main reads it, the startup screen shows the codex
// section's problem as a red ✗ line directly under the keys' problem, and the
// proxy in force as a grey line, and neither with no setting. At 120x40 and at
// the smallest size no row is wider than the screen and the Start button and
// the footer stay whole.
func TestCodexProxyOnStartup(t *testing.T) {
	const (
		badKeys   = `"keys": {"send": ["ctrl+s"]}`
		badCodex  = `"codex": {"proxy": "socks5://127.0.0.1:1080"}`
		goodCodex = `"codex": {"proxy": "` + proxyAddr + `"}`
	)
	for _, c := range []struct {
		name, json                       string
		keysProblem, codexProblem, proxy bool   // what the file gives
		hides                            string // a word of the file the screen must not show
	}{
		{"both problems", `{` + badKeys + `, ` + badCodex + `}`, true, true, false, ""},
		{"a codex problem alone", `{` + badCodex + `}`, false, true, false, ""},
		{"the proxy in force", `{` + goodCodex + `}`, false, false, true, ""},
		{"a keys problem and the proxy in force", `{` + badKeys + `, ` + goodCodex + `}`, true, false, true, ""},
		{"a path with the user's name", `{"codex": {"proxy": "C:\\Users\\testuser-zq\\px.exe"}}`, false, true, false, "testuser-zq"},
		{"a problem with the whole file", `{"colours": {}, ` + goodCodex + `}`, true, false, false, ""},
		{"nothing set", `{}`, false, false, false, ""},
		{"keys only", `{"keys": {"send": ["ctrl+enter"], "newline": ["enter"]}}`, false, false, false, ""},
		{"a codex section with no proxy", `{"codex": {}}`, false, false, false, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				opt := settingsFrom(t, c.json)
				if got := opt.KeysProblem != ""; got != c.keysProblem {
					t.Fatalf("keys problem %q", opt.KeysProblem)
				}
				if got := opt.CodexProxyProblem != ""; got != c.codexProblem {
					t.Fatalf("codex problem %q", opt.CodexProxyProblem)
				}
				if got := opt.CodexProxy != ""; got != c.proxy {
					t.Fatalf("proxy %q", opt.CodexProxy)
				}
				if c.codexProblem && (!strings.HasPrefix(opt.CodexProxyProblem, "config.json: ") || !strings.HasSuffix(opt.CodexProxyProblem, "; no proxy for Codex")) {
					t.Fatalf("problem %q", opt.CodexProxyProblem)
				}
				for _, size := range [][2]int{{120, 40}, {80, 24}} {
					w, h := size[0], size[1]
					raw, text := startupAt(t, opt, w, h)
					rows, flat := rowsOf(text), words(text)

					if c.keysProblem && !strings.Contains(flat, opt.KeysProblem) {
						t.Errorf("%dx%d: the screen lacks %q:\n%s", w, h, opt.KeysProblem, text)
					}
					// The screen hides the user's name in a path (plan 15), so a problem that
					// quotes one is compared in its parts.
					if c.codexProblem && c.hides == "" && !strings.Contains(flat, opt.CodexProxyProblem) {
						t.Errorf("%dx%d: the screen lacks %q:\n%s", w, h, opt.CodexProxyProblem, text)
					}
					if c.hides != "" {
						if strings.Contains(text, c.hides) {
							t.Errorf("%dx%d: the screen shows %q:\n%s", w, h, c.hides, text)
						}
						if !strings.Contains(flat, `codex.proxy "C:\Users\…\px.exe" must start with http:// or https://`) {
							t.Errorf("%dx%d: the screen lacks the codex problem with the name hidden:\n%s", w, h, text)
						}
					}
					if !c.codexProblem && strings.Contains(text, "no proxy for Codex") {
						t.Errorf("%dx%d: a codex problem is shown with none:\n%s", w, h, text)
					}
					if n := strings.Count(text, "Codex through the proxy"); (n == 1) != c.proxy || n > 1 {
						t.Errorf("%dx%d: the proxy line is shown %d times, want %v:\n%s", w, h, n, c.proxy, text)
					}
					if c.proxy && !strings.Contains(text, proxyLine) {
						t.Errorf("%dx%d: the screen lacks %q on one row:\n%s", w, h, proxyLine, text)
					}
					want := 0
					if c.keysProblem {
						want++
					}
					if c.codexProblem {
						want++
					}
					if n := strings.Count(text, "✗ config.json"); n != want {
						t.Errorf("%dx%d: %d ✗ config.json lines, want %d:\n%s", w, h, n, want, text)
					}
					if !c.keysProblem && !c.codexProblem && !c.proxy && strings.Contains(text, "config.json") {
						t.Errorf("%dx%d: config.json is named with nothing to say:\n%s", w, h, text)
					}
					for _, r := range rows {
						if n := len([]rune(r)); n > w {
							t.Errorf("%dx%d: row of %d runes: %q", w, h, n, r)
						}
					}
					// The project field's bottom border and the Start button's, and the
					// footer: the blank spacing rows go first.
					if n := strings.Count(text, "╰"); n != 2 {
						t.Errorf("%dx%d: %d bottom borders, want the field's and the button's:\n%s", w, h, n, text)
					}
					if !strings.Contains(rows[len(rows)-1], "Ctrl+Q quit") {
						t.Errorf("%dx%d: the footer is gone:\n%s", w, h, text)
					}

					// The codex problem is directly under the keys' problem: the row above
					// it ends the keys' words. The proxy line follows the problems.
					if c.keysProblem && c.codexProblem {
						k, x := firstRow(rows, `✗ config.json: "ctrl+s"`), firstRow(rows, "✗ config.json: codex.proxy")
						if k < 0 || x <= k || !strings.HasSuffix(strings.TrimSpace(rows[x-1]), "default keys used") {
							t.Errorf("%dx%d: the codex problem is not directly under the keys' problem:\n%s", w, h, text)
						}
					}
					if c.proxy {
						p := firstRow(rows, "Codex through the proxy")
						above := strings.TrimSpace(rows[p-1])
						if c.keysProblem && !strings.HasSuffix(above, "default keys used") || !c.keysProblem && !strings.HasPrefix(above, "Type a full path") {
							t.Errorf("%dx%d: the proxy line follows %q:\n%s", w, h, above, text)
						}
					}

					// Colours: a problem is red, as the required-field marks are; the proxy
					// line is grey, as the hint under the project field is.
					red, grey := sgrBefore(t, raw, "*"), sgrBefore(t, raw, "Type a full path")
					if red == grey {
						t.Fatalf("red and grey are the same sequence %q: the test cannot tell them apart", red)
					}
					if c.codexProblem {
						if got := sgrBefore(t, raw, "✗ config.json: codex.proxy"); got != red {
							t.Errorf("%dx%d: the codex problem is styled %q, want red %q", w, h, got, red)
						}
					}
					if c.keysProblem { // the first ✗ line, the keys' problem
						if got := sgrBefore(t, raw, "✗ config.json: "); got != red {
							t.Errorf("%dx%d: the keys' problem is styled %q, want red %q", w, h, got, red)
						}
					}
					if c.proxy {
						if got := sgrBefore(t, raw, "Codex through the proxy"); got != grey {
							t.Errorf("%dx%d: the proxy line is styled %q, want grey %q", w, h, got, grey)
						}
					}
				}
			})
		})
	}
}

// TestCodexProxyLinesTogether: the screen shows what it is given, in order, the
// keys' problem, the codex problem, then the proxy line, whatever the
// combination (main gives a problem and a proxy together never).
func TestCodexProxyLinesTogether(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		keysP := `config.json: "ctrl+s" is not a send or newline key (use Enter with Ctrl, Alt or Shift, or Ctrl+J); default keys used`
		codexP := `config.json: codex.proxy "socks5://127.0.0.1:1080" must start with http:// or https:// (such as http://127.0.0.1:3128); no proxy for Codex`
		_, text := startupAt(t, Options{KeysProblem: keysP, CodexProxyProblem: codexP, CodexProxy: proxyAddr}, 120, 40)
		flat := words(text)
		k, x, p := strings.Index(flat, keysP), strings.Index(flat, codexP), strings.Index(flat, proxyLine)
		if k < 0 || x < 0 || p < 0 || !(k < x && x < p) {
			t.Errorf("positions %d, %d, %d: want the keys' problem, the codex problem and the proxy line in that order:\n%s", k, x, p, text)
		}
	})
}

// TestCodexProxyProblemKeepsTheKeys: a problem in the codex section leaves the
// keys of the same file in force (criterion 19.4) and is the only line the
// startup screen adds.
func TestCodexProxyProblemKeepsTheKeys(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		opt := settingsFrom(t, `{"keys": {"send": ["ctrl+enter"], "newline": ["enter"]}, "codex": "x"}`)
		if opt.KeysProblem != "" || opt.CodexProxyProblem == "" || opt.CodexProxy != "" {
			t.Fatalf("keys problem %q, codex problem %q, proxy %q", opt.KeysProblem, opt.CodexProxyProblem, opt.CodexProxy)
		}
		cli := &scriptCLI{id: "claude-code", sess: newScriptSession()}
		opt.Providers, opt.Cwd = []provider.Provider{cli}, t.TempDir()
		h := newHarness(t, opt)
		h.send(tea.WindowSizeMsg{Width: 120, Height: 40})
		if k := h.m.opt.Keys; !k.IsSend("ctrl+enter") || !k.IsNewline("enter") || k.IsSend("enter") {
			t.Errorf("keys in force %v, want the file's", k)
		}
		text := plainText(h.m.View().Content)
		if n := strings.Count(text, "✗ config.json"); n != 1 || !strings.Contains(words(text), opt.CodexProxyProblem) {
			t.Errorf("%d problem lines, want the codex problem alone:\n%s", n, text)
		}
	})
}

// envLog wraps a spawner and keeps the environment of each spawn: nil when the
// CLI inherits the dashboard's.
type envLog struct {
	proc.Spawner
	mu   sync.Mutex
	envs [][]string
}

func (e *envLog) Spawn(sp proc.Spec) (proc.Process, error) {
	e.mu.Lock()
	e.envs = append(e.envs, sp.Env)
	e.mu.Unlock()
	return e.Spawner.Spawn(sp)
}

func (e *envLog) all() [][]string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.envs)
}

// proxyEntries are the entries of env that name HTTPS_PROXY or HTTP_PROXY, in
// any case.
func proxyEntries(env []string) (entries []string) {
	for _, kv := range env {
		if name, _, _ := strings.Cut(kv, "="); strings.EqualFold(name, "HTTPS_PROXY") || strings.EqualFold(name, "HTTP_PROXY") {
			entries = append(entries, kv)
		}
	}
	return entries
}

// codexResume runs the dashboard on the real Codex adapter with proxy, crashes
// the session and resumes it with Ctrl+R, and returns the environment of the
// two spawns. Called in a bubble.
func codexResume(t *testing.T, proxy string) [][]string {
	t.Helper()
	w := &wireCodex{thread: "th-proxy", auto: true}
	rec := &envLog{Spawner: w}
	opt := Options{Cwd: t.TempDir(), CodexProxy: proxy}
	opt.Providers = []provider.Provider{installedCodex{&codex.Provider{Spawner: rec, Proxy: proxy, Now: func() time.Time { return testNow }}}}
	h := newHarness(t, opt)
	h.settle()
	h.m.ui.Choice = codex.ProviderID
	h.send(key(tea.KeyEnter, 0))
	h.send(tea.WindowSizeMsg{Width: 160, Height: 60})
	if s := h.m.State(); s.Session.Phase != state.Ready || s.Session.ID != "th-proxy" {
		t.Fatalf("Codex session %q (%q): want Ready", s.Session.Phase, s.Session.ID)
	}
	w.last().exit(3, "Error: something broke\n")
	h.settle()
	if s := h.m.State(); s.Session.Phase != state.Crashed {
		t.Fatalf("session %q after the crash, want Crashed", s.Session.Phase)
	}
	h.send(key('r', tea.ModCtrl))
	if s := h.m.State(); s.Session.Phase != state.Ready || len(w.method("thread/resume")) != 1 {
		t.Fatalf("after Ctrl+R: session %q, %d thread/resume written; want Ready and one", s.Session.Phase, len(w.method("thread/resume")))
	}
	return rec.all()
}

// TestCodexProxyKeptOnResume is criteria 19.1, 19.2 and 19.7 through the
// application: the environment Codex is spawned with is the dashboard's with
// the proxy in place of its own, and Ctrl+R after a crash spawns it again with
// the same one; with no setting both spawns inherit (no environment given).
func TestCodexProxyKeptOnResume(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://corporate.example:8888")
	t.Setenv("http_proxy", "http://corporate.example:8889")
	t.Setenv("NO_PROXY", "localhost")
	synctest.Test(t, func(t *testing.T) {
		envs := codexResume(t, proxyAddr)
		if len(envs) != 2 {
			t.Fatalf("%d spawns, want the start and the resume", len(envs))
		}
		for i, env := range envs {
			if env == nil {
				t.Fatalf("spawn %d inherits the environment: the proxy is not given", i)
			}
			if got, want := proxyEntries(env), []string{"HTTPS_PROXY=" + proxyAddr, "HTTP_PROXY=" + proxyAddr}; !slices.Equal(got, want) {
				t.Errorf("spawn %d: proxy entries %q, want %q", i, got, want)
			}
			if !slices.Contains(env, "NO_PROXY=localhost") {
				t.Errorf("spawn %d: NO_PROXY not inherited: %q", i, env)
			}
		}
		if !slices.Equal(envs[0], envs[1]) {
			t.Errorf("the resume's environment differs from the start's\nstart  %q\nresume %q", envs[0], envs[1])
		}
	})
	synctest.Test(t, func(t *testing.T) {
		envs := codexResume(t, "")
		if len(envs) != 2 || envs[0] != nil || envs[1] != nil {
			t.Errorf("with no setting the spawns' environments are %q, want two nil (inherited)", envs)
		}
	})
}

// TestClaudeEnvironmentIgnoresCodexProxy is criterion 19.3: with the setting,
// the environment Claude Code is spawned with is what it is without it, the
// owner's own proxy variables included, and the proxy is nowhere in it; the
// Codex adapter, which has it, is not started. Both proxy variables are
// pinned: since 2026-10-21 the owner's own point at Px, the Codex proxy's
// address, so an inherited one would decide the test (plan 22, the
// corrections' verification, criterion 5.14).
func TestClaudeEnvironmentIgnoresCodexProxy(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://corporate.example:8888")
	t.Setenv("HTTP_PROXY", "http://corporate.example:8080")
	spawn := func(t *testing.T, proxy string) (claudeEnvs, codexEnvs [][]string) {
		t.Helper()
		crec, xrec := &envLog{Spawner: &wireClaude{}}, &envLog{Spawner: &wireCodex{thread: "th-x", auto: true}}
		opt := Options{Cwd: t.TempDir(), CodexProxy: proxy}
		opt.Providers = []provider.Provider{
			installedClaude{&claude.Provider{Spawner: crec, Now: func() time.Time { return testNow }}},
			installedCodex{&codex.Provider{Spawner: xrec, Proxy: proxy, Now: func() time.Time { return testNow }}},
		}
		h := newHarness(t, opt)
		h.settle()
		h.m.ui.Choice = claude.ProviderID
		h.send(key(tea.KeyEnter, 0))
		h.send(tea.WindowSizeMsg{Width: 160, Height: 60})
		return crec.all(), xrec.all()
	}
	var with, without [][]string
	synctest.Test(t, func(t *testing.T) {
		var codexEnvs [][]string
		with, codexEnvs = spawn(t, proxyAddr)
		if len(codexEnvs) != 0 {
			t.Errorf("Codex was spawned %d times in a Claude Code session", len(codexEnvs))
		}
	})
	synctest.Test(t, func(t *testing.T) { without, _ = spawn(t, "") })
	if len(with) != 1 || len(without) != 1 {
		t.Fatalf("%d and %d Claude Code spawns, want one each", len(with), len(without))
	}
	if with[0] == nil || !slices.Equal(with[0], without[0]) {
		t.Errorf("Claude Code's environment differs with the setting\n with    %q\nwithout %q", with[0], without[0])
	}
	if !slices.Contains(with[0], "HTTPS_PROXY=http://corporate.example:8888") {
		t.Errorf("Claude Code's own HTTPS_PROXY is not kept: %q", proxyEntries(with[0]))
	}
	if !slices.Contains(with[0], "HTTP_PROXY=http://corporate.example:8080") {
		t.Errorf("Claude Code's own HTTP_PROXY is not kept: %q", proxyEntries(with[0]))
	}
	for _, kv := range with[0] {
		if strings.Contains(kv, "127.0.0.1:3128") {
			t.Errorf("the Codex proxy is in Claude Code's environment: %q", kv)
		}
	}
}
