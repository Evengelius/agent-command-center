package codex

import (
	"context"
	"os"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/proc"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 22, amendment 19: Codex through the local proxy.

const testProxy = "http://127.0.0.1:3128"

// TestProxyEnv is criterion 19.1: every HTTPS_PROXY and HTTP_PROXY entry is
// removed, in any case, HTTPS_PROXY and HTTP_PROXY are appended with the
// proxy, and every other entry is unchanged and in the same order.
func TestProxyEnv(t *testing.T) {
	base := []string{
		`PATH=C:\Windows\System32`,
		"https_proxy=http://corporate.example:8888",
		"Http_Proxy=http://corporate.example:8889",
		"HOME=/home/me",
		"HTTPS_PROXY=http://corporate.example:8890",
		"NO_PROXY=localhost,127.0.0.1",
		`=C:=C:\work`, // Windows' per-drive entry: a name that is empty
		"ALL_PROXY=socks5://corporate.example:1080",
		"HTTP_PROXY=http://corporate.example:8891",
		"HTTPS_PROXY_EXTRA=kept", // another name that starts the same
		"XHTTP_PROXY=kept",       // and one that ends the same
		"http_proxy=",            // an empty value goes too
		"EMPTY=",
		"hTTpS_pRoxY=http://corporate.example:8892",
		"http_proxy", // no "=": its name is the whole entry
		"BARE",
	}
	before := slices.Clone(base)
	want := []string{
		`PATH=C:\Windows\System32`,
		"HOME=/home/me",
		"NO_PROXY=localhost,127.0.0.1",
		`=C:=C:\work`,
		"ALL_PROXY=socks5://corporate.example:1080",
		"HTTPS_PROXY_EXTRA=kept",
		"XHTTP_PROXY=kept",
		"EMPTY=",
		"BARE",
		"HTTPS_PROXY=" + testProxy,
		"HTTP_PROXY=" + testProxy,
	}
	got := proxyEnv(base, testProxy)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("proxyEnv\n got %q\nwant %q", got, want)
	}
	if !reflect.DeepEqual(base, before) {
		t.Errorf("proxyEnv changed its argument: %q", base)
	}
	// Appending to the result must not reach the argument's backing array.
	_ = append(got, "LATER=1")
	if !reflect.DeepEqual(base, before) {
		t.Errorf("the result shares the argument's array: %q", base)
	}

	// An environment with no proxy, and none at all.
	if got, want := proxyEnv([]string{"PATH=x"}, testProxy), []string{"PATH=x", "HTTPS_PROXY=" + testProxy, "HTTP_PROXY=" + testProxy}; !reflect.DeepEqual(got, want) {
		t.Errorf("no proxy in the environment: %q, want %q", got, want)
	}
	if got, want := proxyEnv(nil, testProxy), []string{"HTTPS_PROXY=" + testProxy, "HTTP_PROXY=" + testProxy}; !reflect.DeepEqual(got, want) {
		t.Errorf("no environment: %q, want %q", got, want)
	}
}

// spawnLog is the real spawner, keeping the spec of each spawn it was asked for.
type spawnLog struct {
	mu    sync.Mutex
	specs []proc.Spec
}

func (s *spawnLog) Spawn(sp proc.Spec) (proc.Process, error) {
	s.mu.Lock()
	s.specs = append(s.specs, sp)
	s.mu.Unlock()
	return proc.Exec{}.Spawn(sp)
}

func (s *spawnLog) all() []proc.Spec {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.specs)
}

// startSilent starts a session of p on the fake app-server that never answers
// (ACC_FAKE=silent, set by the test, once, so the environment is the same for
// every start), resuming resume when it is not empty, and closes it at the end.
func startSilent(t *testing.T, p *Provider, resume string) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	sess, err := p.Start(context.Background(), provider.SessionContext{Executable: exe, ProjectPath: dir, ResumeID: resume})
	if err != nil {
		t.Fatal(err)
	}
	r := &run{t: t, sess: sess, st: state.New()}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = sess.Close(ctx)
		r.until(func(event.Event) bool { return false }, 15*time.Second)
	})
	return exe
}

// environ is os.Environ, sorted: the dashboard's own environment.
func environ() []string {
	env := os.Environ()
	slices.Sort(env)
	return env
}

// TestStartWithoutProxy is criterion 19.2: with no Proxy the spawn is given no
// environment (nil: the CLI inherits the dashboard's), a start and a resume
// alike, and the rest of the spec is what it always was.
func TestStartWithoutProxy(t *testing.T) {
	t.Setenv("ACC_FAKE", "silent")
	t.Setenv("HTTPS_PROXY", "http://corporate.example:8888") // the owner's own, left alone
	log := &spawnLog{}
	p := &Provider{Spawner: log, CloseStep: time.Second}
	exe := startSilent(t, p, "")
	startSilent(t, p, "th-9")
	specs := log.all()
	if len(specs) != 2 {
		t.Fatalf("%d spawns, want 2", len(specs))
	}
	for i, sp := range specs {
		if sp.Env != nil {
			t.Errorf("spawn %d: Env %q, want nil (inherited)", i, sp.Env)
		}
		if sp.Path != exe || !reflect.DeepEqual(sp.Args, []string{"app-server"}) || sp.Dir == "" {
			t.Errorf("spawn %d: %+v", i, sp)
		}
	}
}

// TestStartWithProxy is criteria 19.1 and 19.7: with a Proxy the spawn's
// environment is the dashboard's, as proxyEnv makes it, and a second start with
// a thread to resume, a resume after a crash, is given the same one. The
// dashboard's own environment is not touched, so nothing else it starts, Claude
// Code included, inherits the proxy (criterion 19.3).
func TestStartWithProxy(t *testing.T) {
	t.Setenv("ACC_FAKE", "silent")
	// The owner's own proxy variables, in the two cases Windows keeps apart only
	// by name, which the setting replaces.
	t.Setenv("HTTPS_PROXY", "http://corporate.example:8888")
	t.Setenv("http_proxy", "http://corporate.example:8889")
	t.Setenv("NO_PROXY", "localhost")
	own := environ()

	log := &spawnLog{}
	p := &Provider{Spawner: log, Proxy: testProxy, CloseStep: time.Second}
	exe := startSilent(t, p, "")
	startSilent(t, p, "th-9") // the resume after a crash
	specs := log.all()
	if len(specs) != 2 {
		t.Fatalf("%d spawns, want 2", len(specs))
	}

	want := proxyEnv(os.Environ(), testProxy)
	for i, sp := range specs {
		if !reflect.DeepEqual(sp.Env, want) {
			t.Errorf("spawn %d: Env differs from proxyEnv(os.Environ(), proxy)\n got %q\nwant %q", i, sp.Env, want)
		}
		if sp.Path != exe || !reflect.DeepEqual(sp.Args, []string{"app-server"}) || sp.Dir == "" {
			t.Errorf("spawn %d: %+v", i, sp)
		}
		// What the CLI is given, read without proxyEnv: exactly one of each
		// variable, with the proxy; none of the owner's; everything else kept.
		var https, http int
		var others []string
		for _, kv := range sp.Env {
			name, value, _ := strings.Cut(kv, "=")
			switch {
			case strings.EqualFold(name, "HTTPS_PROXY"):
				https++
				if value != testProxy {
					t.Errorf("spawn %d: %s", i, kv)
				}
			case strings.EqualFold(name, "HTTP_PROXY"):
				http++
				if value != testProxy {
					t.Errorf("spawn %d: %s", i, kv)
				}
			default:
				others = append(others, kv)
			}
		}
		if https != 1 || http != 1 {
			t.Errorf("spawn %d: %d HTTPS_PROXY and %d HTTP_PROXY entries, want one of each", i, https, http)
		}
		var kept []string
		for _, kv := range own {
			if name, _, _ := strings.Cut(kv, "="); !strings.EqualFold(name, "HTTPS_PROXY") && !strings.EqualFold(name, "HTTP_PROXY") {
				kept = append(kept, kv)
			}
		}
		slices.Sort(others)
		if !slices.Equal(others, kept) {
			t.Errorf("spawn %d: the other variables differ from the dashboard's\n got %q\nwant %q", i, others, kept)
		}
		if !slices.Contains(sp.Env, "NO_PROXY=localhost") {
			t.Errorf("spawn %d: NO_PROXY not inherited: %q", i, sp.Env)
		}
	}
	if !reflect.DeepEqual(specs[0].Env, specs[1].Env) {
		t.Errorf("the resume's environment differs from the first start's\n first %q\nresume %q", specs[0].Env, specs[1].Env)
	}
	if now := environ(); !reflect.DeepEqual(now, own) {
		t.Errorf("starting Codex changed the dashboard's own environment\n was %q\n now %q", own, now)
	}
	if got := os.Getenv("HTTPS_PROXY"); got != "http://corporate.example:8888" {
		t.Errorf("the dashboard's HTTPS_PROXY is %q", got)
	}
}
