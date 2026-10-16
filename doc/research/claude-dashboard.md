# claude-dashboard (seunggabi)

Plan 1, step 6: reference project review. Read only; nothing built, installed
or run. Sources: `R` = github.com/seunggabi/claude-dashboard @ `711650f`
(paths below are relative to the repository root); web pages accessed
2026-10-15.

## 1. Source and commit

- URL: https://github.com/seunggabi/claude-dashboard
- Commit: `711650f9cd210edb851494b1a6bc388f53dbcb70`, 2026-03-27T12:20:42+09:00
  (shallow clone, `git log -1`).
- Activity (GitHub page, 2026-10-15): 38 stars, 5 forks, 40 commits, 2 open
  issues. Latest release v0.13.1, 2026-03-27, the same day as the commit; then
  v0.13.0 (2026-03-24), v0.12.0 (2026-02-23), v0.11.0 and v0.10.0
  (2026-02-20) (github.com/seunggabi/claude-dashboard/tags). No commit for
  about 6.5 months at the access date: small, single-author, dormant.

## 2. License

- `LICENSE`, first line: `MIT License`; line 3: `Copyright (c) 2025 seunggabi`
  (R `LICENSE:1-3`). SPDX: **MIT**. The README badge and the Homebrew formula
  say the same (R `README.md:5`, `.goreleaser.yml:48`).
- For a proprietary project: forking, copying and modifying are allowed,
  including in closed-source code, provided the copyright notice and the
  permission notice are kept with the copied parts. Per AGENTS.md, record
  source, commit and license next to anything taken.

## 3. Stack and dependencies

Go (`go 1.24.2`, R `go.mod:3`; the README says Go 1.25, R `README.md:7,236`),
Bubble Tea + Bubbles + Lip Gloss, tmux as the session backend. Direct
dependencies (R `go.mod:5-10`):

| Module | Version | License |
| --- | --- | --- |
| github.com/charmbracelet/bubbletea | v1.3.10 | MIT |
| github.com/charmbracelet/bubbles | v0.21.1 | MIT |
| github.com/charmbracelet/lipgloss | v1.1.0 | MIT |
| gopkg.in/yaml.v3 | v3.0.1 | MIT and Apache-2.0 |

21 indirect modules (R `go.mod:12-33`), the usual Charm tree (x/ansi,
x/cellbuf, termenv, go-runewidth, uniseg, atotto/clipboard,
golang.org/x/sys...). The licenses in the table are the projects' published
ones, not checked file by file for this review. External programs called at
run time: `tmux`, `ps`, `lsof`, `bash`, `curl` (sections 4, 5, 8).

## 4. Architecture

Bubble Tea application in Elm style, one `Model`, five views (dashboard
table, logs, detail, create form, help) (R `internal/app/app.go:31-70`).

| Package | Role |
| --- | --- |
| `cmd/claude-dashboard/main.go` | CLI: `new`, `attach`, `setup`, TUI by default; runs auto-setup first (`:22-31`, `:138-149`) |
| `internal/app` | model, `Update`, `View`; attach loop (`app.go:672-717`) |
| `internal/session` | discovery (`detector.go`), create/kill/logs (`manager.go`), data model and status symbols (`session.go:53-66`) |
| `internal/tmux` | wrapper over the tmux CLI (`client.go`), `list-sessions` parser (`parser.go:20`) |
| `internal/conversation` | reads the latest `~/.claude/projects/<dir>/*.jsonl` (`reader.go`) |
| `internal/monitor` | CPU/memory from `ps` with a BFS over the process tree (`process.go:30-90`); tick (`ticker.go:14`) |
| `internal/ui` | string renderers: table (`dashboard.go:29`), log viewport (`logs.go:20-44`), detail, create form, help, status bar (`statusbar.go:11-56`) |
| `internal/styles` | Lip Gloss palette and styles (`styles.go:7-80`) |
| `internal/setup` | embeds three bash scripts, installs them, rewrites `~/.tmux.conf` (`setup.go`) |
| `internal/config` | `~/.claude-dashboard/config.yaml` (`config.go:39-82`) |

Data flow and update model: **polling only**. `Init` batches a refresh and a
`tea.Tick` (R `internal/app/app.go:125-130`); each `TickMsg` re-runs
`manager.List` and re-arms the tick (`app.go:143-147`). Interval
`refresh_interval`, default 2 s (R `internal/config/config.go:31`,
`internal/monitor/ticker.go:20`). Each refresh: `tmux list-sessions`, then per
session `capture-pane` (20 lines), `list-panes`, plus `ps -eo` for the whole
process table and `lsof` per non-tmux `claude` process (cached 10 s)
(R `internal/session/detector.go:38-98`, `:107-194`). The log view is loaded
once when opened, not live (`app.go:313-323`, `:620-632`). Attaching quits
Bubble Tea, runs `tmux attach-session` in the foreground, and restarts the TUI
when the user detaches (`app.go:200-208`, `:672-717`). No file watching, no
event stream, no animation.

## 5. Data sources

| Source | Where | Documented? |
| --- | --- | --- |
| `tmux list-sessions -F name\|created\|attached\|windows\|activity\|path` | R `internal/tmux/client.go:46-61`, `parser.go:20` | documented (tmux man page) |
| `tmux capture-pane -p -S -N`: status heuristic and the "logs" view | R `internal/tmux/client.go:93-106`; `internal/session/detector.go:197-244`; `manager.go:110-115` | tmux is documented; **reading Claude Code's TUI from the screen is undocumented terminal scraping** |
| `tmux list-panes -F #{pane_pid}` / `#{pane_current_command}` | R `internal/tmux/client.go:109-122`, `:156-177` | documented (tmux) |
| `ps -eo pid,ppid,%cpu,%mem,args` and `ps -eo pid,ppid,tty,args` | R `internal/monitor/process.go:30`; `internal/session/detector.go:108` | POSIX/BSD `ps`; not on Windows |
| `lsof -a -p PID -d cwd -Fn` (cwd of terminal sessions) | R `internal/session/detector.go:177` | `lsof`; not on Windows |
| `~/.claude/projects/<cwd with / replaced by ->/*.jsonl`, newest by mtime, last 50 `user`/`assistant` text blocks | R `internal/conversation/reader.go:37-48`, `:51-86`, `:158-213`; `internal/app/app.go:629` | **undocumented** (Claude Code's internal transcript files; no schema relied on from the docs) |
| `~/.claude-dashboard/config.yaml` | R `internal/config/config.go:39-53` | own file |
| `https://api.github.com/repos/seunggabi/claude-dashboard/releases/latest` (hourly, from the tmux status bar) | R `internal/setup/scripts/tmux-status-bar.sh:27` | GitHub REST API |

Status heuristic (R `internal/session/detector.go:197-244`): "active" if tmux
`session_activity` is under 2 s old; "waiting" if one of the last 20 screen
lines ends with `?` or contains `y/n`; otherwise "idle" (the "prompt visible"
branch and the default both return idle, `:239-243`). Coarse and guessable
wrong: any output line ending in `?` reads as waiting.

The project-folder mapping replaces only `/` (R
`internal/conversation/reader.go:42`); Claude Code also encodes other
characters (this session's own folder is `c--javadev-projets-agent-command-center`
for `c:\javadev\projets\agent-command-center`), so the lookup misses paths
with `.`, `_`, `\` or `:`. On Windows it never matches.

**Not read at all** (grep for credential, oauth, token, usage, cost, quota,
rate limit, subagent, hooks, statusline over `*.go` and `*.sh`: no hit beyond
CLI usage strings): no quota or rate limits, no tokens, no context, no cost,
no model, no subagents, no hooks. **No credential file is read** anywhere.
"CPU" and "MEM" are `ps` percentages summed over the process tree (R
`internal/monitor/process.go:59-90`).

## 6. Agent CLIs supported

Claude Code only. It launches `claude [args]` in a detached tmux session named
`cd-<name>` (R `internal/session/manager.go:63-98`, `internal/tmux/client.go:64-79`),
detects other tmux sessions whose name contains `claude` or whose process tree
contains `claude` (`internal/session/detector.go:56-60`,
`internal/tmux/client.go:156-234`), and lists bare `claude` processes on a TTY
as read-only "terminal" sessions (`internal/session/detector.go:107-165`).
Codex CLI and Gemini CLI: absent.

## 7. Conversation

**Observe only; no conversation in the dashboard.** The user talks to Claude
Code by attaching the terminal to the tmux session, i.e. Claude Code's own TUI
replaces the dashboard until detach (R `internal/app/app.go:200-208`,
`:700-713`). The dashboard sends no prompt, streams no answer, relays no
approval: there is no SDK, no `-p`/`stream-json`, no PTY of its own.
`tmux.Client.SendKeys` exists (R `internal/tmux/client.go:124-133`) but is
never called. The only keystrokes it injects are Backspaces into the Claude
pane to erase a stray DA1 terminal reply (`?6c`) after attach (R
`internal/app/app.go:636-657`), a terminal-scraping workaround. The "logs"
view is a one-shot snapshot of `capture-pane` text or of the last 50 JSONL
messages, text blocks only, tool calls dropped (R
`internal/conversation/reader.go:198-212`).

## 8. Configuration side effects

- `~/.claude`, `~/.codex`, `~/.gemini`: **never written**; `~/.claude/projects`
  is only read (section 5).
- **`~/.tmux.conf` is rewritten without asking** on first run of any command
  except `--version`, `--help`, `setup` (R `cmd/claude-dashboard/main.go:25-31`,
  `:138-149`; `internal/setup/setup.go:57-141`): it removes earlier
  claude-dashboard blocks and appends global settings: `history-limit 50000`,
  `bind -n F12`, `bind -n C-s` (overrides Ctrl+S in every tmux pane),
  `status-right`, `status-interval 5`, `mouse on`, and
  `terminal-overrides 'xterm*:smcup@:rmcup@'` (replaces any existing override)
  (`setup.go:111-131`), then `tmux source-file` (`:152`).
- Writes three scripts to `~/.local/bin` (R `internal/setup/setup.go:34-54`),
  `~/.cache/claude-dashboard/current-version` on every start
  (`setup.go:162-185`, `cmd/claude-dashboard/main.go:22`); the status-bar
  script calls the GitHub API hourly (section 5). `Ctrl+S` writes pane
  history to `~/Desktop` (R `internal/setup/scripts/tmux-save-history.sh`).
- Sets `mouse on` on the session before each attach (R `internal/app/app.go:702`).
- `install.sh` appends to `~/.zshrc`/`~/.bashrc` and may run
  `sudo apt-get/yum/pacman install tmux` (R `install.sh:37-67`, `:210-240`);
  `alias.sh` appends an alias to the shell rc (R `alias.sh:31-41`).
- Incompatible with our "no silent change" rule as shipped; irrelevant if only
  code is studied.

## 9. Platform support

- Releases are built for linux and darwin only, amd64/arm64
  (R `.goreleaser.yml:12-17`); install paths are Homebrew, `install.sh`,
  `go install` (R `README.md:21-65`).
- **Windows native: no.** Hard dependency on tmux (R
  `internal/app/app.go:101-104`), `ps`, `lsof`, bash scripts, `/`-only paths.
  `DrainStdin` calls `syscall.SetNonblock(int(os.Stdin.Fd()), …)`
  (R `internal/app/app.go:661-669`); on Windows Go's `syscall.SetNonblock`
  takes a `syscall.Handle`, so the package most likely does not compile there
  (inferred from Go's `syscall_windows.go`, not built).
- WSL 2: should work as a Linux program, but it would only see a Linux-side
  `claude`, not the owner's Windows-installed Claude Code 2.1.300.

## 10. UI and visual relevance

- A k9s-style single table (#, name, project, status, uptime, CPU, MEM, path)
  with a cursor, scroll markers, a filter, a one-line status bar and a key-hint
  line (R `internal/ui/dashboard.go:14-118`, `internal/ui/statusbar.go:11-56`).
- No frames or boxes, no panels side by side, no workflow tree, no agent
  hierarchy, no progress bars, no animation, no quota or telemetry zone, no
  conversation zone. Views replace each other full screen.
- Close to the reference only in small details: status symbols with colors
  (`● active` green, `○ idle` grey, `◎ waiting` amber; R
  `internal/session/session.go:53-66`, `internal/ui/dashboard.go:96-107`), a
  dark palette with violet `#7C3AED` as primary, green `#10B981`, amber
  `#F59E0B` (R `internal/styles/styles.go:7-16`), which matches the mock-up's
  green/amber/grey convention. A create form with a directory field
  prefilled with the cwd (R `internal/ui/create.go:20-73`,
  `internal/app/app.go:281-292`) resembles our startup screen's question 2.

## 11. Quality

- Size (`wc -l`): Go 3,055 lines non-test in 19 files, tests 1,781 lines;
  shell 593 lines (the three helper scripts exist twice, `scripts/` and
  `internal/setup/scripts/`, identical by `diff -r`); README 313.
- Tests: 8 `_test.go` files, 149 `func Test…` (config 15, conversation 34,
  detector 9, manager 19, session 16, tmux client 25, parser 13, ui 18). Not
  run. No CI test job: the only workflow is the GoReleaser release
  (R `.github/workflows/release.yml`).
- Docs: README only, partly stale: lists `internal/app/keys.go` and
  `demo/demo.gif`, neither exists (R `README.md:17`, `:267`).
- Smells: `session_prefix` is loaded from config but never used, the prefix
  is the constant `cd-` (R `internal/config/config.go:71-73`,
  `internal/session/session.go:74`); `config.Save` only used in tests;
  `SendKeys`, `GetSessionInfo` unused; errors from probes silently degrade to
  "idle" or empty lists (`detector.go:41-46`, `:207-209`).
- Maturity: early hobby tool, v0.13.1, one author, dormant since 2026-03.

## 12. Reusable parts

All MIT: copying is allowed with the notice kept and the source recorded.

| Candidate | What it does | Worth it? |
| --- | --- | --- |
| `internal/conversation/reader.go:103-154` | ring-buffer tail of a large JSONL file with a 10 MB line buffer | the idea, a few lines; trivial to rewrite. Our transcript path, if any, must use the correct folder encoding (section 5) and be labelled undocumented |
| `internal/monitor/process.go:59-90` | CPU/MEM aggregated over a process tree by BFS | idea only; the `ps` source does not exist on Windows |
| `internal/ui/logs.go` | Bubbles `viewport` wrapper with scroll percent | trivial; standard Bubbles usage |
| `internal/styles/styles.go:7-16` | palette (violet, cyan, green, amber, grey on dark) | colors are not copyrightable in practice; usable as a starting point |
| `internal/ui/statusbar.go` | left/right status bar with a computed gap | trivial pattern |

Not to reuse:
- `internal/setup` and the bash scripts: silent rewrite of user config.
- `internal/tmux`, `internal/session/detector.go`: tmux, `ps`, `lsof`
  dependencies, screen-scraped status; contrary to "drive the CLI through its
  supported interfaces" and unavailable on native Windows.
- `internal/app/app.go` attach loop and `cleanDA1`/`DrainStdin`: hands the
  terminal over to Claude Code's TUI (no zone B), injects keystrokes, Unix-only.

## 13. Verdict

**Not a fork candidate.** MIT allows it, but nothing in it serves our core:
it observes Claude Code through tmux and screen scraping, hosts no
conversation, reads no quota, tokens, cost or subagents, has no multi-zone
layout, does not run on native Windows, and rewrites `~/.tmux.conf` silently.
Value for us: confirmation that Go + Bubble Tea + Lip Gloss fit the project,
and a counter-example for status detection (screen heuristics give
"waiting"/"idle" guesses). Reuse at most the JSONL tail idea, with our own
code.
