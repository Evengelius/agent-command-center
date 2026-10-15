# Current Plan

Four plans from `doc/brief.md` and its three amendments, run one at a time and in
order. Each ends with the evaluation summary of `doc/evaluation.md` and waits
for the owner's approval before the next starts. Until plan 4 is approved: no
code, no dependency installed, no file written outside `doc/` and this plan.

The application is an interactive workspace (amendment 2): a real conversation
with the orchestrator, beside its workflow tree, its live agents and their
telemetry. It drives Claude Code first; Codex CLI and Gemini CLI as far as
their interfaces allow, each optional (amendment 1).

Active: **Plan 1, not started.**

## Plan 1 - Research

Inspect each reference: architecture, implementation approach, dependencies,
licensing, relevant source code, and the data it reads. Do not assume the
reference video or the layout mock-up shows a working application. Never read
credential files.

- One file per reference project in `doc/research/`: `agent-dashboard.md`,
  `claude-dashboard.md`, `ccdash.md`, `jev-pilot.md`, each with: source and
  commit, license, stack and dependencies, architecture, data sources (and
  whether each is documented), which agent CLIs it supports, whether it hosts a
  conversation or only observes, what is reusable and under which license
  terms (the project is proprietary: permissive licenses only).
- One file per agent CLI: `claude-code.md` (including the workflows and
  statusline pages of the brief), `codex-cli.md`, `gemini-cli.md`, each with:
  its official documentation (pages and date); every supported way to drive a
  session from another program (SDK, headless or streaming modes, servers,
  protocols); how it runs subagents or parallel work; where a program can
  observe it (stream events, hooks, session files, status line, telemetry
  export); each marked documented or undocumented.
- `doc/research/integration.md`: the 9 research questions of amendment 2,
  answered for each agent CLI with sources: embedding or controlling a session;
  genuine streaming; subagent lifecycle and tool events; approvals and
  permissions; persistence, cancellation and resumption; reusing an existing
  terminal UI; concurrent subagent output; reliable quota; limits on Windows,
  WSL2, Linux and macOS. Each answer with its limitations and a realistic
  fallback.
- `doc/research/telemetry-sources.md`: for each agent CLI, every way to get
  zone D's values (5-hour and 7-day quota or the CLI's equivalent, reset times,
  model, context, tokens, API-equivalent cost, agent count and concurrency
  limit, duration, errors and retries), documented or not, with its source;
  what is simply unavailable (shown as "N/A").
- `doc/research/terminal-input.md` (amendment 3): what each candidate UI library
  offers for a multiline editor (selection, cursor movement, very long input,
  bracketed paste); clipboard access per terminal and OS (Windows Terminal,
  PowerShell, WSL2, Linux, macOS; OSC 52 and native clipboards); mouse capture
  versus the terminal's own text selection; tmux copy mode; `$EDITOR`; the
  shortcuts each OS and terminal already takes. Each with its source and its
  limits.
- `doc/research/summary.md`: fork versus build from scratch, with the reasons.

Checks: the 11 files exist with their sections; every claim has a source; every
license is an SPDX id read at a recorded commit; every reference, every agent
CLI, all 9 questions and every item of amendment 3 are covered; no credential
file read; nothing outside
`doc/` changed (`git status`).

## Plan 2 - Architecture

Design the application from the research. Zones A to D of amendment 2 (A:
orchestrator and workflow tree; B: interactive conversation, multiline input,
streaming, Markdown and code blocks, tool activity, clarification and approval
requests, history, follow-ups, interruption; C: live subagents, selectable and
expandable, independent from the conversation; D: usage and telemetry), the
interaction flow of amendment 2 (launch, ask, orchestrate, watch, inspect,
approve, report, continue), keyboard navigation and focus, responsive layouts,
and the style of `doc/reference-design.md`. Bubble Tea, Lip Gloss and tmux
unless plan 1 found better; each CLI's own orchestration reused, not rebuilt.

- `doc/architecture.md`: stack with versions; the five separated parts of
  amendment 2 (CLI execution and session management, event collection and
  normalisation, application state, terminal rendering, input and commands);
  one adapter per agent CLI; the integration strategy with Claude Code; how it
  runs on Windows (tmux needs WSL); how it starts.
- `doc/events-and-state.md`: the conceptual events of amendment 2
  (`conversation.message.delta`, `tool.started`, `agent.failed`,
  `approval.requested`, `telemetry.updated`...), each mapped to a real
  documented source per CLI, or marked unavailable; the state model; where
  polling cannot be avoided, and why.
- `doc/ui-design.md`: the revised layout at 3 terminal sizes (text mock-ups);
  the interaction-flow diagram; focus and every key binding; scrolling,
  collapsing, full-screen inspection; animations; colors and symbols mapped to
  the references; every state (loading, streaming, waiting for approval,
  cancelled, no data, error, CLI not installed); how AI messages, observed
  events and inferred labels look different; the input editor of amendment 3
  (multiline editing and selection, paste, clipboard and mouse copy from every
  panel, configurable Enter-to-send and newline keys, `$EDITOR`), with what
  works where across terminals and OSes.

Checks: every item of the brief and of amendment 2 placed in a section; every
choice traced to a plan 1 finding; every conceptual event mapped or marked
unavailable; mock-ups for 3 sizes; no rebuilt workflow engine or terminal
protocol.

## Plan 3 - Reliability

- `doc/reliability.md`: the parts separated as in plan 2 (rendering reads the
  state only); what happens when each agent CLI, each event source or each
  telemetry source is missing; approvals, interruption, cancellation and
  resumption without losing the session; concurrent output; how every CLI's
  configuration stays untouched; JEV optional; the mock-data mode (mock events
  through the same pipeline, never shown as real); the test plan (event
  normalisation and telemetry parsing per CLI, conversation and agent state
  transitions: each state and transition listed; input: pastes preserved byte
  for byte, a multiline paste never submitting, thousands of lines without
  freezing, the same behaviour for every agent CLI).

Checks: each brief item of phase 3 answered; each source from plan 1 has a
fallback; the state machines list every state and transition, and fit all
three CLIs; mock mode can never be mistaken for a live session.

## Plan 4 - Proposal

- `doc/proposal.md`: the 6 items of the brief and the 7 deliverables of
  amendment 2 (revised layout, interaction-flow diagram, event and state
  architecture, integration strategy, reusable components, limitations and
  risks, updated MVP plan), summarised from plans 1-3 with links, explaining
  what amendments 2 and 3 changed in the architecture. The MVP roadmap says in
  which order the agent CLIs and the zones are delivered, includes the input
  editor and clipboard as essential MVP features (amendment 3), and becomes the
  next plans in this file, once approved.

Checks: every item present; each links to its detailed document; the roadmap
in ordered steps, each with its own checks. Then present it and wait for the
owner's approval.
