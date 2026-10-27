// Command agent-command-center is the terminal mission control for Claude
// Code and OpenAI Codex CLI (doc/architecture.md, How it starts).
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/Evengelius/agent-command-center/internal/app"
	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/provider/codex"
)

func main() {
	cfg, err := app.ParseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "agent-command-center:", err)
		os.Exit(app.ExitError)
	}
	// Both adapters, Claude Code's and Codex's, detect their CLI and start its
	// sessions, reading the real environment through host.
	host := provider.SystemHost()
	// The application's own configuration file: the editor's send and newline
	// keys, read once, here (plan 21, item 5.1). A problem is shown on the
	// first screen and the default keys apply.
	keys, problem := input.DefaultKeys(), ""
	// The same file names the proxy Codex is started through (plan 22,
	// amendment 19), read next to the keys: without a configuration folder the
	// keys' problem already says so, and Codex gets no proxy.
	proxy, proxyProblem := "", ""
	if path, err := input.ConfigFile(); err != nil {
		problem = "config.json: no configuration folder (" + err.Error() + "); default keys used"
	} else {
		keys, problem = input.LoadKeys(path)
		proxy, proxyProblem = input.LoadCodexProxy(path)
	}
	m := app.New(app.Options{
		Keys:              keys,
		KeysProblem:       problem,
		CodexProxy:        proxy,
		CodexProxyProblem: proxyProblem,
		Config:            cfg,
		Providers:         []provider.Provider{&claude.Provider{Host: host}, &codex.Provider{Host: host, Proxy: proxy}},
		Now:               time.Now,
		Clipboard:         input.System{},
		MockSpeed:         1,
		Drafts:            &input.Drafts{},
		Projects:          &input.Projects{},
		PickImages:        input.PickImages,
		PickFolder:        input.PickFolder,
		Git:               input.GitCLI{},
		Sessions:          &input.Sessions{},
	})
	code, out := app.Run(m)
	for _, l := range out {
		fmt.Fprintln(os.Stderr, l)
	}
	os.Exit(code)
}
