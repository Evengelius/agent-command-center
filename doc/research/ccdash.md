# ccdash (jedarden/ccdash)

Plan 1, step 6. Read-only study of a shallow clone; nothing was built, installed
or run. Citations `path:line` refer to the repository at the commit below.
Web pages were accessed 2026-10-15.

## 1. Source and commit

- URL: https://github.com/jedarden/ccdash (README.md:401 says this is a
  read-only mirror of `git.ardenone.com/jedarden/ccdash`).
- Commit: `5263e726304d1296dc468af5aed49f2f5175e685`, committed
  2026-10-11T10:01:00-04:00 (`git log -1`).
- Version in the tree: `1.2.5` (`VERSION`); CHANGELOG.md:10 dates 1.2.5 to
  2026-10-11; CHANGELOG.md has 58 `## ` sections.
- Activity (GitHub page, 2026-10-15): 16 stars, 1 fork, 0 watching, 410
  commits. Releases page: latest v1.2.5 (11 Oct 2026), v1.2.4 and v1.2.3 (both
  07 Oct 2026); 11 assets on v1.2.5 (names not shown). Actively maintained by
  one author, small audience.

## 2. License

- File `LICENSE`, first line: `MIT License`; line 3:
  `Copyright (c) 2026 Jed Arden`. SPDX: **MIT**. README.md:395 agrees.
- For a proprietary project: forking, copying and modifying are allowed, in
  closed source, provided the copyright notice and permission text are kept
  with the copied parts (record source, commit and license next to them, as
  AGENTS.md requires). Linking as a Go module is also allowed, but every
  package is under `internal/`, so Go forbids importing it from another
  module: reuse means copying files, not depending on the module.

## 3. Stack and dependencies

Go 1.24.0, toolchain go1.24.11 (go.mod:3-5). TUI: Bubble Tea + Lip Gloss. No
Bubbles components (no textarea, viewport, list). Direct dependencies
(go.mod:7-15); licenses are the upstream projects' known licenses, not read
from the module cache:

| Module | Version | Use | License |
|---|---|---|---|
| github.com/charmbracelet/bubbletea | v1.3.10 | TUI runtime | MIT |
| github.com/charmbracelet/lipgloss | v1.1.0 | styling, layout | MIT |
| github.com/charmbracelet/x/ansi | v0.10.1 | ANSI-aware width/truncation | MIT |
| github.com/shirou/gopsutil/v3 | v3.24.5 | CPU, memory, disk, network | BSD-3-Clause |
| golang.org/x/term | v0.37.0 | `IsTerminal` check | BSD-3-Clause |
| gopkg.in/yaml.v3 | v3.0.1 | `~/.ccdash/config.yaml` | MIT and Apache-2.0 |
| modernc.org/sqlite | v1.40.1 | token cache, OpenCode store (pure Go, no cgo) | BSD-3-Clause |

Indirect: 30 modules (go.mod:17-50), among them `go-osc52`, `termenv`,
`go-runewidth`, `uniseg`, `go-ole`/`wmi` (gopsutil on Windows). None is GPL.

## 4. Architecture

Overview in `docs/architecture.md:1-141`. Packages:

- `cmd/ccdash/main.go` (1118 lines): flags (main.go:28-43), one-shot and
  headless modes (`headless.go`, `watch.go`), `--doctor` (`doctor.go`), hook
  install/uninstall, then `tea.NewProgram(dashboard, tea.WithAltScreen())`
  (main.go:289-292). No mouse option.
- `internal/metrics` (collectors):
  - `SystemCollector` (`system.go`): gopsutil snapshot, rates from the
    previous sample.
  - `TokenCollector` (`tokens.go`, `cache.go`, `source.go`, `codex.go`,
    `opencode.go`): a `Source` interface (source.go:18-25: `Name`,
    `ProjectDirs`, `ParseUsageLine`, `PricingForModel`, `HookInstaller`) with
    three implementations registered at tokens.go:153-158. A background
    goroutine ingests JSONL into SQLite `~/.ccdash/tokens.db` at start and
    every 30 s (tokens.go:215-231, `time.NewTicker(30 * time.Second)` at
    tokens.go:221); the UI only queries the cache. The cache has
    `token_events`, `file_aggregates` (per-file summaries once a file is idle)
    and a `collector_lease` table for leader election between several
    dashboard instances (cache.go:251-256, 1444-1560), WAL mode
    (cache.go:179, 196).
  - `TmuxCollector` (`tmux.go`, `hooks.go`, `agentproc.go`, `needle.go`,
    `worker.go`): merges hook session files, tmux sessions and pane captures,
    and NEEDLE worker registry into one session list; hook state takes
    precedence (architecture.md:28-41).
  - Pricing tables and cost math: `tokens.go` (Claude, GLM; tokens.go:729,
    1041, 1125), `codex.go:232-258` (OpenAI), `session_cost.go` (per-session
    cost from the transcript named after the session id), `projection.go`
    (weekly linear projection).
- `internal/ui`: `dashboard.go` (model, `Update`, `View`, concurrent
  collection), `layout.go` (two layouts: stacked below 140 columns, three
  columns above, dashboard.go:372-383), `panel_system.go`, `panel_tokens.go`,
  `panel_sessions.go`, `worker_detail.go`, `overlays.go` (lookback picker,
  help), `statusbar.go`, `keys.go`, `styles.go`, `text.go`.
- `internal/notify`: optional webhook when a session waits for a human (off by
  default, README.md:305).
- `internal/updater`: GitHub release check and self-update.
- `internal/config`: `~/.ccdash/config.yaml`.

Update model: **polling**. `Init` starts a 2-second `tea.Tick`
(dashboard.go:385-389); every tick launches the three collectors concurrently
and waits at most 3 s (dashboard.go:506) before sending one `metricsMsg`
(dashboard.go:251-269). Token ingestion polls the filesystem every 30 s
(tokens.go:221). No file watcher, no event stream; hook events are only
seen as files written by the hook scripts and read on the next tick. `View`
re-renders the whole frame and clips it to the terminal (dashboard.go:368).

## 5. Data sources

| Data | Where it is read | How | Documented? |
|---|---|---|---|
| Claude Code tokens and model per assistant message | `~/.claude/projects/**/*.jsonl`, including `<uuid>/subagents/agent-*.jsonl` (tokens.go:89-102, 664-673; parser source.go:38-65, fields `type`, `timestamp`, `message.model`, `message.usage.*`) | filesystem walk every 30 s | **Undocumented**: transcript schema is not a published interface (Claude Code docs only expose `transcript_path` to hooks, https://code.claude.com/docs/en/hooks) |
| Codex tokens and model | `~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl` (codex.go:25-30); `turn_context.payload.model` and `event_msg` with `payload.type == "token_count"`, `info.last_token_usage` (codex.go:67-140) | filesystem walk | **Undocumented**: schema reverse-engineered from a captured rollout (docs/research/codex-rollout-schema.md:1-8) |
| OpenCode tokens | `~/.local/share/opencode/opencode.db` or `OPENCODE_DB`, opened `mode=ro` (opencode.go:42-47, 137-146) | SQLite read | Undocumented internal store |
| Claude/Codex session status (working, asking, stopped), project dir, tmux session, agent PID | `~/.ccdash/sessions/<session-id>.json`, written by ccdash's hook scripts (lifecycle_hooks.go:5-131), read by hooks.go:107-170 | hooks: SessionStart, SessionEnd, UserPromptSubmit, PreToolUse, PostToolUse, Stop, Notification, PermissionRequest (hooks.go:384-393) | **Documented** hook mechanism (Claude: https://code.claude.com/docs/en/hooks; Codex: `developers.openai.com/codex/hooks` per docs/adrs/0007…md:147, not verified here) |
| Status without hooks | `tmux list-sessions` (tmux.go:437), `tmux capture-pane -p -S -15` (tmux.go:530), then string matching on the last 15 lines: "esc to interrupt", "(running)", "⏵⏵ bypass permissions", "❯", "rate limit", "APIError" (tmux.go:640-700) | terminal scraping | **Undocumented** (Claude Code's screen text) |
| Which tmux panes host an agent | `tmux list-panes -a` (agentproc.go:109) + `/proc/<pid>/stat` walk for `claude`, `codex`, `opencode`, `needle*` (agentproc.go:22-54; needle.go:251-300) | process table | OS interface; Linux only (`/proc`) |
| Running Claude processes count | `pgrep -x claude` (tmux.go:762) | process table | Unix only |
| NEEDLE workers (author's own orchestrator) | `~/.needle/state/workers.json`, `~/.needle/state/heartbeats/<id>.json` (needle.go:59-60, 82, 111) | file read | Third-party, n/a for us |
| System metrics | gopsutil (system.go) | library | Documented library |
| Update check | `https://github.com/jedarden/ccdash/releases/latest` (updater.go:22, 124) | HTTP at start for release builds (dashboard.go:192-195) | public web |

What is **not** there:

- **Quota / rate limits: none.** No 5-hour or 7-day subscription data, no
  reset times. `grep` for `rate_limit|five_hour|seven_day|quota` finds only the
  pane-scraping error strings (tmux.go:693-695) and the updater's GitHub API
  comments. The Codex parser reads only `last_token_usage` and ignores any
  other field of `token_count` events (codex.go:54-56, 96-101); whether Codex
  rollouts carry rate-limit data must be checked in our Codex research.
- **Context window: none** displayed. The Codex sample has
  `model_context_window` (codex-rollout-schema.md:36) but the parser ignores it.
- **Cost:** computed locally from bundled per-million price tables (API-list
  equivalent), overridable in config (README.md:328-353). Unknown models are
  priced at **$0** and flagged `?` / `pricing_estimated` (README.md:160-163,
  architecture.md:89-92): flagged, but a zero rather than "N/A".
- **Subagents:** only as token sources (subagent JSONL counted into totals,
  CHANGELOG.md:272-274, tokens.go:664-666). No subagent list, tree or
  lifecycle; `SubagentStart`/`SubagentStop` hooks deliberately not wired
  (docs/adrs/0007…md:88). "Workers" in the UI are separate tmux sessions or
  NEEDLE workers, not Claude Code subagents.
- **Credential files: not read.** A search for `credentials`, `auth.json`,
  `oauth`, `api_key` finds no read of any secret (only a comment at
  notify.go:151 about not logging webhook URLs). No red flag.

## 6. Agent CLIs supported

- **Claude Code**: tokens from transcripts; status from hooks or tmux
  scraping.
- **Codex CLI**: tokens from rollouts; status from hooks written to
  `~/.codex/hooks.json` (codex_hooks.go:33, 105-146); no tmux heuristics
  specific to Codex beyond the process name (agentproc.go:24).
- **OpenCode**: tokens only, from its SQLite store; no hooks
  (opencode.go:77).
- **Gemini CLI: not supported** (no reference in code or docs).
- NEEDLE: the author's own worker orchestrator (needle.go).

## 7. Conversation

**Observe-only.** ccdash never starts, drives or talks to an agent: no
`send-keys`, no PTY, no SDK, no `stream-json`, no app-server. The only
`exec.Command` calls are tmux read commands, `which`, `pgrep` and the updater
(tmux.go:425, 437, 530, 762; agentproc.go:109; hooks.go:269;
updater.go:390-451). Approvals are only *detected* (PermissionRequest /
Notification hooks set `status="waiting"`, lifecycle_hooks.go:112-127) and
optionally pushed to a webhook; they are answered in the agent's own
terminal. Nothing for zone B.

## 8. Configuration side effects (red flag)

- **Launching the dashboard writes Claude Code's configuration without
  asking.** The interactive path calls `setupHooks()` (main.go:258), which
  runs `InstallHooks()` unconditionally (main.go:765-781). That merges eight
  hook entries into **every** `~/.claude/settings*.json` (glob at
  hooks.go:438) plus `settings.json` (hooks.go:444-454), rewriting each file
  with `json.MarshalIndent` of a `map[string]interface{}` (hooks.go:610-621):
  keys are re-sorted, formatting is lost, and the write is a plain
  `os.WriteFile`, not atomic. On exit, `Cleanup()` uninstalls them if no other
  instance is registered (hooks.go:870-877, `UninstallHooks` at
  hooks.go:765-866). It also writes scripts to `~/.ccdash/hooks/`
  (hooks.go:409-424) and PID files to `~/.ccdash/instances/`
  (hooks.go:700-708).
- Codex hooks are written to `~/.codex/hooks.json` only on
  `--install-codex-hooks` (main.go:117-128), atomically
  (codex_hooks.go:173-190); `--uninstall-hooks` removes both
  (main.go:175-180).
- Hook scripts are `bash` + `jq` (+ optional `flock`, `ps`, `tmux`)
  (lifecycle_hooks.go:5-87); a missing `jq` makes them fail, logged to
  `~/.ccdash/<harness>-hook-errors.log` (lifecycle_hooks.go:10-16).
- Self-update (`u` key) writes `/tmp/ccdash-update-script.sh` and runs it with
  `sudo -n` or `pkexec` (updater.go:384-399).
- This violates our hard rule ("never change an agent CLI's configuration
  unless the owner asks"); none of this behaviour may be carried over.

## 9. Platform support

- Built and released for linux and darwin, amd64/arm64 only (Makefile:90-93;
  updater.go:175-180). **No Windows build**; no Windows code path.
- Linux-isms: `/proc` walk (agentproc.go:33-34, needle.go:257, 297), `pgrep`
  and `which` (tmux.go:425, 762), `syscall.Signal(0)` liveness check, which
  does not work on Windows (hooks.go:754-762), bash hook scripts, `/tmp`
  and `sudo` in the updater.
- tmux is optional for token data but central for sessions: hook records are
  displayed only while their tmux session exists (architecture.md:30-32;
  hooks.go:212-286), so without tmux (native Windows) the Sessions panel
  would stay mostly empty.
- The author's target is a headless Linux server over Tailscale
  (docs/adrs/0006…md:29). On our machine it could only run in full under
  WSL 2, observing WSL-side CLIs.

## 10. UI and visual relevance

Screenshot: `docs/images/ccdash.png` (synthetic data, README.md:9-11). Three
bordered panels side by side (System Resources, Token Usage, Sessions) and a
one-line status bar with keys. Rounded borders, `#00aaff` border, `#00ffff`
titles, green/orange/red thresholds (styles.go:18-58).

Close to the owner's wishes:
- dark, dense, colour-coded status rows with emoji badges (ASK, WORK, ACT),
  "needs you (n)" counter (panel_sessions.go);
- threshold-coloured `[|||| 42.1%]` bars (panel_system.go:201-290), 8-level
  sparkline (panel_tokens.go:464-505), per-model cost list sorted by spend,
  budget bar, weekly projection;
- responsive width switch at 140 columns and a final clip to the terminal
  size (dashboard.go:368, 372-383);
- a detail view per worker with `j`/`k` navigation (worker_detail.go).

Far from it: no workflow tree, no agent hierarchy, no conversation, no input
editor, no animation (refresh every 2 s, no spinner), no mouse, no clipboard,
no quota/context gauges, no startup screen. The layout (three equal columns)
does not match the reference zones A-D.

## 11. Quality

- Go: 12,803 lines of production code, 6,413 lines of tests; 33 `_test.go`
  files with 160 `Test*` functions, including golden JSON for `--once`
  (`cmd/ccdash/testdata/once-snapshot-v1.golden.json`). ~6,660 lines of
  Markdown (README, architecture, 7 ADRs, research notes).
- Largest files: `cache.go` 1756, `tokens.go` 1265, `main.go` 1118,
  `hooks.go` 877, `tmux.go` 780.
- CI on the author's Argo Workflows runs gofmt, vet, tests, build
  (architecture.md:143-147). Versioned JSON schema for headless output.
- Mature for its narrow purpose (v1.2.5, careful changelog, explicit
  "estimated" flags), but the UI layer is a single monolithic model with
  hand-drawn panels.

## 12. Reusable parts (MIT: copy with notice)

| Candidate | What it does | Value for us |
|---|---|---|
| `internal/metrics/source.go:38-65` + `ParseUsageLine` pattern | Claude transcript `assistant` line to token event, including `cache_creation.ephemeral_5m/1h` fallback | Useful reference for an *undocumented* fallback token source; label as undocumented |
| `internal/metrics/codex.go:36-140` | Codex rollout `turn_context` + `token_count` parsing, long-context tagging | Same: reference for a Codex fallback, undocumented |
| `internal/metrics/codex.go:232-258`, tokens.go price tables | per-model price tables with provenance comments | Possible starting point for API-equivalent cost; prices go stale and must be re-sourced; change unknown-model handling to "N/A" |
| `internal/metrics/session_cost.go:12-28` | session id from transcript file name | small helper |
| `internal/metrics/projection.go` | weekly linear projection | trivial, re-implement if needed |
| `internal/ui/panel_system.go:201-290`, `panel_tokens.go:464-505` | threshold bar, mini bar, sparkline | small Lip Gloss helpers; easy to rewrite in our style |
| `lifecycle_hooks.go` state machine (asking vs working, questions vs permissions) | rules for which hook events mean "needs a human" | Study only: the *logic* is useful if we ever use hooks, the bash/jq scripts are not |
| `docs/research/codex-rollout-schema.md`, ADR 0007 | Codex format notes | Read as data for our Codex research |

Do not reuse:
- Hook installer (`hooks.go:409-621`, `main.go:765-802`): silent edits to
  `~/.claude/settings*.json`; forbidden by our rules, and non-atomic.
- tmux pane scraping (`tmux.go:525-700`): undocumented, fragile, tmux-only.
- `/proc`, `pgrep`, `syscall.Signal(0)` process logic: not Windows.
- Updater (`internal/updater`): `sudo`/`pkexec`, network at start; not wanted.
- SQLite multi-instance cache and lease (`cache.go`): 1,756 lines solving a
  multi-dashboard problem we do not have; adds `modernc.org/sqlite`.
- `internal/ui` as a whole: three-panel monitor without input, focus,
  viewport or mouse; our zones A-D need a different model.

## 13. Verdict

**Not a fork candidate.** MIT allows it, but ccdash is an observe-only,
polling, Linux/tmux-centric usage monitor: no conversation, no subagent tree,
no quota, no Windows build, and its default launch silently rewrites
`~/.claude/settings*.json`. What it does well (token/cost accounting from
transcripts) is an undocumented fallback for zone D, not our primary source.
Build from scratch; copy at most the Codex/Claude usage parsers and price
tables (with the MIT notice), and keep its hook state rules as a study
reference.
