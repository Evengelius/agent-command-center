# Current Plan

Four plans from `doc/brief.md` and its four amendments, run one at a time and in
order. Each ends with the evaluation summary of `doc/evaluation.md` and waits
for the owner's approval before the next starts. Until plan 3 (the proposal) is
approved: no project code, no dependency installed, no file written outside
`doc/` and this plan; the only exception is plan 1's probes, below.

The application is an interactive workspace (amendment 2): a real conversation
with the orchestrator, beside its workflow tree, its live agents and their
telemetry. It drives Claude Code first; Codex CLI and Gemini CLI as far as
their interfaces allow, each optional (amendment 1). It opens on a startup
screen (agent CLI and project directory) before the dashboard, and its MVP is
fixed by amendment 4: plan 4 below, whose steps plan 3 details.

The order puts the biggest risk first: whether each agent CLI can be driven
from another program decides the architecture, and whether a reference project
is worth forking; reliability is designed with the architecture, not after it.

Active: **Plan 1, not started.**

## Plan 1 - Research

Biggest unknowns first: the agent CLIs and how to drive them, then the
telemetry and terminal input, then the reference projects, then the decision.
Do not assume the reference video, the layout mock-up or the screen images show
a working application. Never read credential files.

1. **The agent CLIs**, one file each: `claude-code.md` (including the workflows
   and statusline pages of the brief), `codex-cli.md`, `gemini-cli.md`, each
   with: its official documentation (pages and date); every supported way to
   drive a session from another program (SDK, headless or streaming modes,
   servers, protocols); how it runs subagents or parallel work; where a program
   can observe it (stream events, hooks, session files, status line, telemetry
   export); how to detect it is installed (and its version), start it in a
   given working directory, and its native permission and sandbox modes; each
   marked documented or undocumented.
2. **`integration.md`**: the 9 research questions of amendment 2, answered for
   each agent CLI with sources: embedding or controlling a session; genuine
   streaming; subagent lifecycle and tool events; approvals and permissions;
   persistence, cancellation and resumption; reusing an existing terminal UI;
   concurrent subagent output; reliable quota; limits on Windows, WSL2, Linux
   and macOS. Each answer with its limitations and a realistic fallback. Also
   the path forms the startup screen accepts and validates (Windows, `~`, WSL
   `/mnt/c/...` and `\\wsl$\...`, Linux, macOS).
3. **Probes**, to settle what documentation alone cannot: throwaway runs of the
   installed CLIs (Claude Code, Codex CLI) in streaming mode with a one-line
   prompt, to record the real events they emit, in a temporary folder outside
   the repository, without changing any configuration. They use some of the
   owner's quota: list the probes and ask the owner's approval before running
   them. Recorded outputs go to `doc/research/probes/`, credentials and
   personal data removed; they become test fixtures later. Gemini CLI, not
   installed, is researched from documentation only.
4. **`telemetry-sources.md`**: for each agent CLI, every way to get zone D's
   values (5-hour and 7-day quota or the CLI's equivalent, reset times, model,
   context, tokens, API-equivalent cost, agent count and concurrency limit,
   duration, errors and retries), documented or not, with its source; what is
   simply unavailable (shown as "N/A").
5. **`terminal-input.md`** (amendment 3): what each candidate UI library offers
   for a multiline editor (selection, cursor movement, very long input,
   bracketed paste); clipboard access per terminal and OS (Windows Terminal,
   PowerShell, WSL2, Linux, macOS; OSC 52 and native clipboards); mouse capture
   versus the terminal's own text selection; tmux copy mode; `$EDITOR`; the
   shortcuts each OS and terminal already takes. Each with its source and its
   limits.
6. **The reference projects**, one file each: `agent-dashboard.md`,
   `claude-dashboard.md`, `ccdash.md`, `jev-pilot.md`, each with: source and
   commit, license, stack and dependencies, architecture, data sources (and
   whether each is documented), which agent CLIs it supports, whether it hosts
   a conversation or only observes, whether its integration matches what steps
   1-3 found to work, what is reusable and under which license terms (the
   project is proprietary: permissive licenses only).
7. **`summary.md`**: fork versus build from scratch, with the reasons; which
   agent CLI to build first (Claude Code unless steps 1-3 show it cannot be
   driven properly).

Checks: the 11 files exist with their sections; every claim has a source; every
license is an SPDX id read at a recorded commit; every reference, every agent
CLI, all 9 questions and every item of amendment 3 are covered; probes run only
after the owner's approval, outside the repository, with their outputs
recorded and cleaned; no credential file read; no CLI configuration changed;
nothing outside `doc/` changed (`git status`).

## Plan 2 - Architecture and reliability

Design the application from the research, with its reliability in the same
pass: fallbacks, cancellation, mock mode and tests shape the event flow and the
state, so they are decided together. Zones A to D of amendment 2 (A:
orchestrator and workflow tree; B: interactive conversation, multiline input,
streaming, Markdown and code blocks, tool activity, clarification and approval
requests, history, follow-ups, interruption; C: live subagents, selectable and
expandable, independent from the conversation; D: usage and telemetry), the
interaction flow of amendment 2 (launch, ask, orchestrate, watch, inspect,
approve, report, continue), the two screens of amendment 4, keyboard navigation
and focus, responsive layouts, and the style of `doc/reference-design.md`.
Bubble Tea, Lip Gloss and tmux unless plan 1 found better; each CLI's own
orchestration reused, not rebuilt.

- `doc/architecture.md`: stack with versions; the five separated parts of
  amendment 2 (CLI execution and session management, event collection and
  normalisation, application state, terminal rendering, input and commands);
  one adapter per agent CLI behind one provider interface (detect, start in a
  directory, converse, observe), with nothing hardcoded around Claude Code; the
  session context (provider, project name, path) shared by every part; the
  integration strategy with Claude Code; how it runs on Windows (tmux needs
  WSL); how it starts.
- `doc/events-and-state.md`: the conceptual events of amendment 2
  (`conversation.message.delta`, `tool.started`, `agent.failed`,
  `approval.requested`, `telemetry.updated`...), each mapped to a real
  documented source per CLI (plan 1's probes as evidence), or marked
  unavailable; the state model; where polling cannot be avoided, and why.
- `doc/ui-design.md`: the two screens of amendment 4 and the transition between
  them: the startup screen (the two questions, validation messages, keys) and
  the dashboard with its project bar (provider, project, path); the revised
  layout at 3 terminal sizes (text mock-ups); the interaction-flow diagram;
  focus and every key binding; scrolling, collapsing, full-screen inspection;
  animations; colors and symbols mapped to the references; every state
  (loading, streaming, waiting for approval, cancelled, no data, error, CLI not
  installed); how AI messages, observed events and inferred labels look
  different; the input editor of amendment 3 (multiline editing and selection,
  paste, clipboard and mouse copy from every panel, configurable Enter-to-send
  and newline keys, `$EDITOR`), with what works where across terminals and OSes.
- `doc/reliability.md` (the brief's phase 3): the parts separated as above
  (rendering reads the state only); what happens when each agent CLI, each
  event source or each telemetry source is missing; approvals, interruption,
  cancellation and resumption without losing the session; concurrent output;
  how every CLI's configuration stays untouched; JEV optional; the mock-data
  mode (mock events through the same pipeline, never shown as real); the test
  plan (event normalisation and telemetry parsing per CLI, from plan 1's
  recorded probes; conversation and agent state transitions: each state and
  transition listed; input: pastes preserved byte for byte, a multiline paste
  never submitting, thousands of lines without freezing, the same behaviour for
  every agent CLI).

Checks: every item of the brief (phases 2 and 3) and of amendments 2 to 4
placed in a section; both screens of amendment 4 designed, and where they
follow or depart from the reference images; every choice traced to a plan 1
finding; every conceptual event mapped or marked unavailable; each source from
plan 1 has a fallback; the state machines list every state and transition, and
fit all three CLIs; mock mode can never be mistaken for a live session;
mock-ups for 3 sizes; no rebuilt workflow engine or terminal protocol.

## Plan 3 - Proposal

- `doc/proposal.md`: the 6 items of the brief and the 7 deliverables of
  amendment 2 (revised layout, interaction-flow diagram, event and state
  architecture, integration strategy, reusable components, limitations and
  risks, updated MVP plan), summarised from plans 1 and 2 with links,
  explaining what amendments 2 to 4 changed in the architecture, and the
  technical limits each CLI puts on the interaction (amendment 4). The MVP
  roadmap fills plan 4 below: step 0 and each priority as ordered steps with
  their own checks, the input editor and clipboard as essential (amendment 3),
  the first agent CLI end to end before the others.

Checks: every item present; each links to its detailed document; the roadmap
in ordered steps, each with its own checks. Then present it and wait for the
owner's approval.

## Plan 4 - MVP (from amendment 4; steps detailed by plan 3)

The fundamental plan: a working application, before a beautiful one and long
before a bigger one. Starts only after plan 3 is approved; plan 3 turns step 0
and each priority into steps with their own checks. The first agent CLI
(Claude Code, unless plan 1 decided otherwise) end to end first (step 0 and
priorities 1 to 5), then the two others, validated the same way; never a
function traded for a visual detail.

0. Foundation: the toolchain installed with the owner's approval (Go is not on
   this machine); the project skeleton (module, folders, build, test commands
   in AGENTS.md); the event and state core with the mock-data mode; the test
   harness (table tests, golden screen snapshots, plan 1's recorded probes as
   fixtures); both screens and the four zones laid out on mock data, so the
   layout exists from the start.
1. Startup configuration: the two questions; provider detection with a clear
   error when the CLI is missing; path entry, expansion and validation (exists,
   is a directory, accessible), current directory prefilled.
2. The chosen provider starts in the chosen directory.
3. The conversation end to end: streaming, multiline input and long pastes,
   clipboard, permission handling.
4. Zones A and C show the real execution information available.
5. Zone D shows the real metrics available, "N/A" otherwise.
6. The visual presentation polished to match the reference images (the layout
   itself exists since step 0).

Out of scope (amendment 4): recent projects, favourites, project browser, git
worktrees, editor integration, model configuration, cross-provider
collaboration, switching provider during a session, session profiles,
orchestration settings.

Done when every acceptance criterion of amendment 4 passes, with each of the
three agent CLIs: launched from a terminal; any of the three selectable;
project path validated; the CLI working in that directory; real prompts with
streamed answers; files inspected and changed with approvals; multiline input,
long pastes and copying reliable; real agent activity shown; telemetry accurate
or marked unavailable; both screens in the sequence and layout of the images.
