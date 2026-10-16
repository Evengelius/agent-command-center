# Summary: fork or build, and the first agent CLI

Plan 1, step 7. The decision plan 1 owes plan 2, with the reasons, drawn from
the other nine research files. Each reason names the file and section that
holds its evidence.

## Decision

1. **Build from scratch.** None of the four reference projects is a fork
   candidate. Reuse is limited to permissively licensed pieces, copied with
   their notice and a record of source, commit and license.
2. **Claude Code first, then Codex CLI.** Claude Code can be driven properly
   through its supported interfaces, so the brief's default stands (plan 1,
   step 7: "Claude Code unless steps 1-3 show it cannot be driven properly").

## Why build from scratch

The four references share the same mismatch with the brief: they observe
agent CLIs, they do not converse with them.

| Project (commit) | License | Why not a fork | Worth studying |
| - | - | - | - |
| agent-dashboard (`b3c04cf7f0aa01ddfa1d215dd6f25a04400396df`) | MIT | Observes TUIs in tmux and controls them by `send-keys` and pane scraping; no streaming, no hosted conversation, no native approvals; needs tmux and a Unix build; reads `~/.claude/.credentials.json` for quota and installs hooks | transcript and subagent parsing (undocumented fallback), quota bar and subagent tree presentation ([`agent-dashboard.md`](agent-dashboard.md), section 13) |
| claude-dashboard (`711650f9cd210edb851494b1a6bc388f53dbcb70`) | MIT | tmux and screen scraping; no conversation, quota, tokens, cost or subagents; no native Windows; rewrites `~/.tmux.conf` | confirms Go with Bubble Tea and Lip Gloss fits; a counter-example for status guessing ([`claude-dashboard.md`](claude-dashboard.md), section 13) |
| ccdash (`5263e726304d1296dc468af5aed49f2f5175e685`) | MIT | observe-only, polling, Linux and tmux; no conversation, subagent tree or quota; no Windows build; its default launch rewrites `~/.claude/settings*.json` | usage parsers and price tables from transcripts, an undocumented fallback ([`ccdash.md`](ccdash.md), section 13) |
| jev-pilot (`5d27135b34edf81b40c0bf1df01ae401f6b72ea7`) | MIT | a Claude Code plugin that steers the agent, not a dashboard; no Go code; its optional parts change Claude Code's configuration, proxy its traffic and read its credential file | Claude Code mods as a possible data source, to weigh in plan 2; how to detect JEV read-only ([`jev-pilot.md`](jev-pilot.md), section 14) |

Three of them break hard constraints of AGENTS.md as they ship (credential
read, configuration rewrites), so even a fork would start by removing code.
The core the brief asks for, a real streamed conversation with approvals and
live subagents, is absent from all four, and the research shows it comes from
the CLIs' own protocols ([`integration.md`](integration.md), questions 1 to
4), which none of them uses.

What can be reused, all under permissive licenses:
- The Codex app-server JSON Schema (Apache-2.0, `openai/codex` at
  `rust-v0.145.0`), to generate Go types; the installed binary's schema is
  identical ([`codex-cli.md`](codex-cli.md), section 1).
- The Claude Agent SDK for Python (MIT) as the reference for the control
  protocol; small enough to reimplement in Go, so nothing needs copying
  ([`claude-code.md`](claude-code.md), section 1).
- For the input editor, ideas or code from the Bubbles textarea and tview's
  TextArea (MIT), and a clipboard library (MIT or BSD-3-Clause)
  ([`terminal-input.md`](terminal-input.md), section 11).

## Why Claude Code first

Steps 1 and 3 show Claude Code can be driven properly, from a client written
without its SDK:

- **Conversation:** one long-lived `claude -p` process with stream-json in
  and out carried three turns, token deltas, an interrupt and a context
  request (P2; [`claude-code.md`](claude-code.md), sections 4 and 10).
- **Approvals:** `can_use_tool` requests arrived and an allow ran the tool,
  for the orchestrator and for a subagent (P2, P3b).
- **Subagents:** the task events showed a backgrounded subagent's lifecycle
  (P3, P3b).
- **Zone D:** quota (`rate_limit_event`), tokens, context, API-equivalent
  cost, durations and errors all have a source
  ([`telemetry-sources.md`](telemetry-sources.md)).

Its weak points are known and have fallbacks: the control protocol's wire
form is documented only in SDK code; quota arrives on transitions, not every
turn; subagents do not stream tokens; resume does not replay history, so the
dashboard keeps its own record ([`integration.md`](integration.md), questions
1, 2, 5 and 8).

Codex follows as the second provider and is in good shape for it: its
app-server carried turns, deltas, approvals, an interrupt and two parallel
subagents from a plain Node driver (P6b, P7), and gives quota on request
without a turn. Two points need care before it is built: the app-server is
labelled experimental, and `thread/start` with a project `cwd` and a write
sandbox writes trust into `config.toml`, which must be avoided
([`codex-cli.md`](codex-cli.md), sections 3.1 and 10).

## What plan 2 inherits

- **Architecture:** one long-lived CLI process per session, driven over
  pipes; one provider interface for both CLIs (the concept table in
  [`integration.md`](integration.md), "Common provider shape"); events update
  the state, the state is rendered.
- **Interface:** Bubble Tea v2 and Lip Gloss v2; an application-owned
  multiline editor; native clipboard first, OSC 52 as well; mouse capture
  with in-app selection ([`terminal-input.md`](terminal-input.md), section
  11).
- **tmux is not needed to drive either CLI:** both run over pipes on native
  Windows (P1 to P4, P6b, P7). The brief's preferred stack includes tmux;
  plan 2 decides whether it has any role (it runs only under WSL here).
- **Startup paths:** the forms, translations and validation steps in
  [`integration.md`](integration.md), "Path forms for the startup screen".
- **Open probe questions:** listed in [`claude-code.md`](claude-code.md)
  section 10 and [`codex-cli.md`](codex-cli.md) section 10; the ones that
  affect the MVP are gathered at the end of
  [`integration.md`](integration.md) and
  [`telemetry-sources.md`](telemetry-sources.md).
- **Installation:** Go is not installed, and no dependency may be added
  before the owner approves the proposal (plan 3; AGENTS.md).
