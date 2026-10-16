# agent-dashboard (bjornjee/agent-dashboard)

Plan 1, step 6. Read-only study of a shallow clone; nothing was built, installed
or run. Citations `path:line` refer to the repository at the commit below.
Web pages were accessed 2026-10-15.

## 1. Source and commit

- URL: https://github.com/bjornjee/agent-dashboard
- Commit: `b3c04cf7f0aa01ddfa1d215dd6f25a04400396df`, committed
  2026-07-08T15:54:31+08:00 (`git log -1`).
- Version in the tree: `0.35.0` (`VERSION`); CHANGELOG.md:3 dates 0.35.0 to
  2026-07-05; releases are cut by release-please (`release-please-config.json`)
  and goreleaser (`.goreleaser.yml`).
- Activity (GitHub page, 2026-10-15): 25 stars, 3 forks, 533 commits on
  `main`. Releases page: latest v0.35.0 (05 Jul 2026), 10 releases on the first
  page (v0.35.0 to v0.29.0), 5 pages in total. No commit after 2026-07-08 at the
  inspected head: about three months idle at the access date. One main author
  (`.github/CODEOWNERS`), small audience.

## 2. License

- File `LICENSE`, first line: `MIT License`; line 3:
  `Copyright (c) 2025-present bjornjee`. SPDX: **MIT**. README.md ("License"
  section) and both plugin manifests agree
  (`adapters/claude-code/.claude-plugin/plugin.json`,
  `adapters/codex/.codex-plugin/plugin.json`: `"license": "MIT"`).
- For a proprietary project: forking, copying, modifying and linking are all
  allowed in closed source, provided the copyright notice and permission text
  are kept in every copy or substantial portion. Per AGENTS.md, anything taken
  is recorded with source, commit and license next to it.
- Embedded third-party code with its own license: `internal/diagrams/mermaid.min.js`
  (Mermaid bundle, MIT upstream, not verified in the file);
  `internal/usage/codex.go:17` says its pricing table is "Ported from CodexBar's
  CostUsagePricing" (origin license not recorded in the repo).

## 3. Stack and dependencies

- Go `1.26.4` (`go.mod:3`) for the TUI, the web server and tools; Node.js
  (CommonJS, no npm dependencies: `adapters/claude-code/package.json`,
  `adapters/codex/hooks/package.json` have no `dependencies`) for the hook
  scripts; vanilla JS/CSS PWA in `internal/web/static/`; Playwright for web
  tests (`tests/playwright/package.json`).
- UI: Bubble Tea v2, Bubbles v2, Lip Gloss v2, Glamour v2 (the `charm.land/*`
  v2 module paths).
- Direct Go dependencies (`go.mod:5-26`, non-`// indirect`). Licenses are the
  upstream ones as commonly published, not read from the module cache:

| Module | Version | Use | License |
| --- | --- | --- | --- |
| charm.land/bubbletea/v2 | v2.0.2 | TUI loop | MIT |
| charm.land/bubbles/v2 | v2.1.0 | viewport, textinput | MIT |
| charm.land/lipgloss/v2 | v2.0.2 | styling, layout | MIT |
| charm.land/glamour/v2 | v2.0.0 | Markdown | MIT |
| github.com/alecthomas/chroma/v2 | v2.20.0 | syntax highlighting | MIT |
| github.com/fsnotify/fsnotify | v1.9.0 | file watching | BSD-3-Clause |
| github.com/jmoiron/sqlx | v1.4.0 | SQL helper | MIT |
| modernc.org/sqlite | v1.48.0 | pure-Go SQLite | BSD-3-Clause |
| github.com/BurntSushi/toml | v1.6.0 | settings | MIT |
| github.com/stretchr/testify | v1.11.1 | tests | MIT |
| golang.org/x/oauth2 | v0.36.0 | Google OAuth for the PWA | BSD-3-Clause |
| golang.org/x/sync | v0.20.0 | singleflight | BSD-3-Clause |
| github.com/bluekeyes/go-gitdiff | v0.8.1 | diff parsing | MIT |
| github.com/charmbracelet/x/ansi | v0.11.6 | ANSI width/strip | MIT |

- External programs required at runtime: `tmux` (`internal/tmux/tmux.go:29`,
  `exec.CommandContext(ctx, "tmux", ...)`), `git`, `node` (hooks), optional
  `gh`, `codex`, `z` (README.md "Prerequisites" table).

## 4. Architecture

Components (docs/development/architecture.md, "Internal packages"):

- `cmd/dashboard/main.go`: TUI entry; `cmd/web/main.go`: PWA server (default
  `127.0.0.1:8390`, `cmd/web/main.go:16-17`); helper tools in `cmd/`.
- `adapters/claude-code/`: a Claude Code **plugin** (hooks, skills, agents).
  Its hooks write one JSON state file per session to
  `~/.agent-dashboard/agents/<session_id>.json`
  (`adapters/claude-code/packages/agent-state/index.js:11-12,28-29,141`).
- `adapters/codex/`: the same for Codex (plugin with `hooks/plugin-hooks.json`).
- `internal/state/`: reads the per-agent files (`internal/state/state.go:46`),
  arbitrates state, flags resumable sessions.
- `internal/conversation/`, `internal/codex/conversation/`: incremental JSONL
  readers for Claude and Codex transcripts; subagent discovery.
- `internal/tmux/`: `capture-pane`, `send-keys`, `paste-buffer`, `new-window`,
  `split-window` wrappers (`internal/tmux/tmux.go:105-257,471-520`).
- `internal/harness/`: `domain.Harness` interface (`internal/domain/harness.go:11-34`:
  `Name`, `SpawnCommand`, `Models`, `EffortLevels`, `SessionsDir`, `ConfigDir`)
  with `claude` and `codex` implementations that only build shell command
  strings.
- `internal/usage/`, `internal/db/`: token and cost totals, SQLite cache,
  rate-limit fetch.
- `internal/tui/`: Bubble Tea model/view/keys; `internal/web/`: SSE server.

Data flow (docs/development/architecture.md, "Data flow"): agent CLI hook ->
Node script -> `~/.agent-dashboard/agents/*.json` -> dashboard (fsnotify +
polling) -> TUI, and SSE to the PWA.

Update model: **hybrid, mostly polling**.

- Event-driven part: fsnotify on the agents directory, debounced 50 ms, sends
  `stateUpdatedMsg` (`internal/tui/commands.go:1192-1240`); the web server does
  the same with 300 ms debounce (`internal/web/watcher.go:58-91`).
- Polling part: a 1 s `tea.Tick` (`internal/tui/commands.go:616-620`); each
  tick runs `tmux capture-pane` of the selected agent (20 lines), conversation
  reload and spawn capture (`internal/tui/model.go:1066`); every 5 s subagents,
  plan, diagrams (`model.go:1071-1076`); every 10 s dead-agent pruning and
  token usage (`model.go:1077-1079`); every 60 s (setting
  `rate_limit_poll_seconds`, `internal/config/settings.go:25`) the rate-limit
  HTTP call (`model.go:1080-1082`); every 30 s full state reload and Codex usage
  (`model.go:1083-1086`).
- Animations: a pet sprite at 400 ms (`internal/tui/pet.go:30`) and a dino
  mini-game at 50 ms (`internal/tui/dino.go:52`).

## 5. Data sources

| Source | Where read | Documented? |
| --- | --- | --- |
| Claude Code hooks (SessionStart, PreToolUse, PostToolUse, PermissionRequest, SubagentStart/Stop, Stop, Notification, UserPromptSubmit, StopFailure) via plugin | `adapters/claude-code/hooks/hooks.json` | Documented: https://code.claude.com/docs/en/hooks and plugins https://code.claude.com/docs/en/plugins |
| Claude transcripts `~/.claude/projects/<slug>/<session>.jsonl` (conversation, tool activity, pending questions, plan slug, rate-limit errors) | `internal/conversation/conversation.go:36,170,870,1015,1083,1203,1296,1364,1433` | **Undocumented** file format (hooks only document `transcript_path`) |
| Claude subagent transcripts `<projDir>/<session>/subagents/agent-<id>.jsonl` and `.meta.json` | `conversation.go:677-742` | **Undocumented** |
| Claude session index `~/.claude/sessions/<pid>.json` | `conversation.go:1541-1595` | **Undocumented** |
| Claude plans `~/.claude/plans/` | `conversation.go:1414`, `internal/config/config.go:88` | **Undocumented** |
| `~/.claude/settings.json` (model, effortLevel; read only) | `internal/harness/default_model.go:82`, `default_effort.go:85`, `adapters/claude-code/scripts/hooks/agent-state-reporter.js:25-31` | Documented settings file: https://code.claude.com/docs/en/settings |
| **`~/.claude/.credentials.json` and macOS Keychain "Claude Code-credentials"** (OAuth access token) | **`internal/usage/oauth.go:44-45,58-63,95`** | **Credential file: red flag** |
| `GET https://api.anthropic.com/api/oauth/usage` with that token, header `anthropic-beta: oauth-2025-04-20` and a spoofed `User-Agent: claude-code/2.1.0` | `internal/usage/ratelimit.go:14-19,56-84` | **Undocumented private endpoint** |
| Codex rollouts `$CODEX_HOME/sessions/YYYY/MM/DD/rollout-*.jsonl` (conversation, subagents via `session_meta.source.subagent.thread_spawn`, tokens via `event_msg/token_count`) | `internal/tui/model.go:744-749`, `internal/codex/conversation/locate.go`, `parser.go:1-60`, `subagents.go:16-39`, `internal/usage/codex.go:107-156` | **Undocumented** file format |
| `~/.codex/config.toml` (model, reasoning effort; read only) | `internal/harness/default_model.go:78`, `default_effort.go:78` | Documented config file (see `codex-cli.md`) |
| Codex hooks via plugin | `adapters/codex/hooks/plugin-hooks.json` | Codex hooks; documentation status in `codex-cli.md` |
| tmux pane text (`capture-pane`, ANSI stripped) for live view, state detection, trust-prompt and Codex "tab to queue" detection | `internal/tmux/tmux.go:105-120`, `internal/tmux/trustprompt.go:12`, `internal/harness/codex/input.go:26-28`, `adapters/claude-code/scripts/hooks/agent-state-reporter.js:271`, `adapters/claude-code/packages/agent-state/detect.js:20-26` | **Terminal scraping** |
| api-ninjas quotes (banner), only if `API_NINJAS_KEY` set | `internal/tui/banner.go:50-55` | third-party API |

How it gets each telemetry value:

- **5-hour / 7-day quota and resets**: only from the private OAuth endpoint
  with the credential-file token (`ratelimit.go:91-101`; rendered
  `internal/tui/view.go:849-890`). Hidden when no token (`view.go:852`).
  No Codex quota: `internal/usage/codex.go:83-96` reads only
  `info.total_token_usage`.
- **Tokens**: summed from transcript `message.usage`, de-duplicated by message
  id (`internal/usage/usage.go:55-120`); Codex from the last `token_count`.
- **Cost**: computed from hard-coded price tables (`usage.go:16-20`, Opus at
  15/75 USD per Mtok; `codex.go:18-34`); unknown models **silently fall back**
  to Sonnet (`usage.go:30-31`) or gpt-5.4 pricing (`codex.go:56`).
- **Context window**: not found as a dedicated metric.
- **Subagents**: Claude from `subagents/` transcripts plus
  SubagentStart/Stop hooks; Codex from rollout `thread_spawn` metadata.

## 6. Agent CLIs supported

- **Claude Code**: first-class (README.md "FAQ"); plugin with hooks and skills;
  spawned as `claude [--effort E] [--model M] '<prompt>'` or
  `claude --resume <sid>` inside a tmux pane
  (`internal/harness/claude/claude.go:48-73`).
- **Codex CLI** 0.130+: plugin (`adapters/codex/`), spawned as
  `codex [-c model_reasoning_effort=E] [--model M] [-a A] [-s S] [PROMPT]` or
  `codex resume <sid>` (`internal/harness/codex/codex.go:1-97`).
- **Gemini CLI**: not supported (the only Go mention is `.gemini` in a
  path deny list, `internal/web/handlers.go:86`).
- Registry knows `claude` and `codex` only (`internal/harness/registry.go:21`).

## 7. Conversation

**Observe and remote-control, not host.** The agent CLI runs in its own TUI in a
tmux pane; the dashboard shows the conversation by re-reading the transcript
JSONL (`internal/tui/model.go:1066`, `conversation.go:169`) and the last 20
pane lines (`commands.go:78-93`). No streaming: text appears when the CLI flushes
it to the transcript, picked up on the next 1 s tick.

Sending is **keystroke injection**: replies go through `tmux send-keys -l` in
512-byte chunks then `Enter` (`internal/tmux/tmux.go:165-201`), or for Codex
`set-buffer` + `paste-buffer -p` then `Enter` or `Tab Enter` chosen by scraping
the pane footer for "tab to queue" (`tmux.go:214-241`,
`internal/harness/codex/input.go:22-36`, `commands.go:644-665`). Approvals
(`y/n`, numbered options, plan approval) are raw keys sent to the pane
(`commands.go:1180-1187`, `internal/tui/keys.go:779-792`). The reply widget is
a single-line `textinput` with `CharLimit = 4096` (`internal/tui/model.go:702-704`).
No SDK, no `stream-json`, no app-server.

## 8. Configuration side effects

- Installing the Claude plugin (`/plugin install`, README.md "Step 2") adds
  hooks that do more than observe: PreToolUse hooks **block** destructive
  shell commands, commits on `main`, and direct `gh pr create`; PostToolUse
  hooks **block** commits unless `make test` passes and enforce conventional
  commits (`adapters/claude-code/hooks/hooks.json`, descriptions of each entry).
  It also ships skills and agents that change how the agent works.
- Codex: the README asks the user to append
  `[plugins."agent-dashboard@agent-dashboard"] enabled = true` to
  `~/.codex/config.toml` (README.md:103-110; `Makefile:60-63` prints it). The
  installer states it no longer writes into `~/.codex` (`install.sh:9`).
- The Go binary itself only reads `~/.claude` and `~/.codex`; it writes its own
  `~/.agent-dashboard/` (state, SQLite, `settings.toml`, `install.sh:236-248`).
- Verdict against our hard rule: the explicit, user-run plugin install is
  consent-based, but its hooks change agent behaviour; we would at most propose
  a passive observation hook, never these gates.

## 9. Platform support

- macOS and Linux only: goreleaser builds `darwin` and `linux`, amd64/arm64
  (`.goreleaser.yml:10-15`); CI on `ubuntu-latest` (`.github/workflows/ci.yml:16,31`).
- **Native Windows unsupported** (README.md:43); tmux mandatory (README.md:29).
  `internal/lock/lock.go:22` uses `syscall.Flock` with no build tag, so the
  package does not compile for Windows. Shell quoting is POSIX
  (`claude.go:95-97`). WSL "should work but is untested" (README.md:43).

## 10. UI and visual relevance

- Layout: banner, a left agent list grouped by state with a subagent tree
  (`├─`/`└─`, `✓` done, `▶` running; `internal/tui/view.go:463-490`), a right
  detail panel (pane capture, conversation history, files, plan, usage), a help
  bar (`view.go:1156-1205`); full-screen diff viewer (`internal/tui/diff_view.go`).
- Quota bars: `█░` progress bars with percent and "resets in 2h 15m"
  (`view.go:809-847,849-890`) — close to zone D of the mock-up.
- Palette: Catppuccin Frappé (`internal/tui/styles.go:19-38`,
  `internal/tui/catppuccin-frappe.json`), dark and pastel, near the reference's
  violet/green/amber.
- Far from the reference: no workflow/phase diagram, no animated connectors, no
  four-zone layout with a resident conversation and input; the subagent tree is
  a list under each session, not a plan tree. Animations are a pet and a game.

## 11. Quality

- Tests: 77 Go `_test.go` files (~28.2k lines) vs ~21.9k lines of non-test Go;
  62 JS test files (`*.test.js` and Playwright `*.spec.js`, ~14.8k lines);
  ~16.0k lines of non-test JS (excluding `mermaid.min.js`), 4.1k lines CSS,
  ~9.0k lines Markdown. Largest files: `internal/tui/view.go` 1,985 lines,
  `model.go` 1,596, `conversation/conversation.go` 1,595, `keys.go` 1,480.
- Mocks via mockery (`.mockery.yaml`, `internal/mocks/`); a `Runner`
  interface wraps tmux and commands for tests (`internal/tmux/tmux.go:28-52`,
  `internal/tui/runner.go`).
- Docs: a Jekyll site (`docs/`), CONTRIBUTING, SECURITY, LEARNINGS, CHANGELOG.
- Maturity: pre-1.0 (0.35.0), frequent releases until July 2026, one author.

## 12. Reusable parts (MIT: copy with notice)

Candidates, each needing the MIT notice and a provenance record:

| Part | What it does | Use for us |
| --- | --- | --- |
| `internal/conversation/conversation.go` (incremental reader, `ReadConversationIncremental` :169, `FindSubagents` :677) | offset-based tailing of Claude JSONL, subagent discovery from `subagents/*.meta.json` | **Study only / fallback**: undocumented format; useful for session history or C-zone fallback, labelled undocumented |
| `internal/usage/usage.go:55-120` + `usage_cache.go` | token sums with message-id de-dup and (size, mtime, offset) cache | small, reusable idea; replace pricing and the silent fallback |
| `internal/codex/conversation/` (parser, subagents via `thread_spawn`, locate) | Codex rollout parsing | study for Codex observation fallback; undocumented |
| `internal/tui/view.go:809-847` | progress bar and reset-duration formatting | trivial; rewrite rather than copy |
| `internal/tui/styles.go` palette | Catppuccin Frappé colours (Catppuccin is MIT) | optional inspiration |
| fsnotify debounce pattern (`commands.go:1192-1240`) | event-driven refresh with coalescing (`singleflight`, `commands.go:614`) | pattern only |
| `internal/domain/harness.go` | provider abstraction | idea only; ours must drive sessions, not build shell strings |

Do not reuse:

- `internal/usage/oauth.go`, `ratelimit.go`: read credential files and call a
  private endpoint with a spoofed User-Agent; forbidden by AGENTS.md.
- `internal/tmux/` send-keys/capture-pane control and
  `internal/harness/codex/input.go`: terminal injection and scraping, contrary
  to "drive each CLI through its supported interfaces"; tmux is WSL-only here.
- The adapters' gating hooks, skills and agents: they change the CLI's
  behaviour and configuration.
- Price tables with silent fallbacks (`usage.go:16-31`, `codex.go:18-56`):
  stale and against "real values or N/A".
- `internal/lock/lock.go` (Unix `Flock`), web/PWA, OAuth, SQLite, dino/pet:
  out of MVP scope or not Windows-portable.

## 13. Verdict

**Not a fork candidate.** It is MIT and on our stack (Go, Bubble Tea v2, Lip
Gloss v2), but its core design is the opposite of ours: it observes agent TUIs
inside tmux and controls them by `send-keys` and pane scraping (no streaming,
no hosted conversation, no native approvals), requires tmux and a Unix build
(no native Windows), gets quota by reading `~/.claude/.credentials.json` and
calling an undocumented endpoint, and installs behaviour-changing hooks.
Keep it as a study source: Claude/Codex transcript and subagent parsing
(labelled undocumented, as a fallback), token de-dup caching, and the quota-bar
and subagent-tree presentation. Anything copied keeps the MIT notice.
