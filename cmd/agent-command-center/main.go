// Command agent-command-center is the terminal mission control for Claude
// Code and OpenAI Codex CLI (doc/architecture.md, How it starts).
package main

import (
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

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
		os.Exit(2)
	}
	// Both adapters detect their CLI; their sessions arrive in steps 2
	// (Claude Code) and 5 (Codex), and until then Start reports it.
	host := provider.SystemHost()
	m := app.New(app.Options{
		Config:    cfg,
		Providers: []provider.Provider{&claude.Provider{Host: host}, &codex.Provider{Host: host}},
		Now:       time.Now,
		Clipboard: input.System{},
		MockSpeed: 1,
	})
	_, err = tea.NewProgram(m).Run()
	m.Shutdown()
	if err != nil {
		fmt.Fprintln(os.Stderr, "agent-command-center:", err)
		os.Exit(1)
	}
}
