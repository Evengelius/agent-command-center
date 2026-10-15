# Current Plan

Four plans from `doc/brief.md` and its amendment, run one at a time and in
order. Each ends with the evaluation summary of `doc/evaluation.md` and waits
for the owner's approval before the next starts. Until plan 4 is approved: no
code, no dependency installed, no file written outside `doc/` and this plan.

The dashboard covers three agent CLIs, each optional: Claude Code, OpenAI Codex
CLI and Google Gemini CLI.

Active: **Plan 1, not started.**

## Plan 1 - Research

Inspect each reference: architecture, implementation approach, dependencies,
licensing, relevant source code, and the data it reads. Do not assume the
reference video shows a working application. Never read credential files.

- One file per reference project in `doc/research/`: `agent-dashboard.md`,
  `claude-dashboard.md`, `ccdash.md`, `jev-pilot.md`, each with: source and
  commit, license, stack and dependencies, architecture, data sources (and
  whether each is documented), which agent CLIs it supports, what it does well
  for this brief, what is reusable and under which license terms.
- One file per agent CLI: `claude-code.md` (including the workflows and
  statusline pages of the brief), `codex-cli.md`, `gemini-cli.md`, each with:
  its official documentation (pages and date), how it runs subagents or
  parallel work, and where a dashboard can observe it (session files, logs,
  status line or hooks, events, telemetry export), each marked documented or
  undocumented.
- `doc/research/telemetry-sources.md`: for each agent CLI, every way to get
  what the bottom bar needs (5-hour and 7-day quota or the CLI's equivalent,
  reset times, tokens, context, model, agent statistics), documented or not,
  with its source; and what is simply unavailable.
- `doc/research/summary.md`: fork versus build from scratch, with the reasons.

Checks: the 9 files exist with their sections; every claim has a source; every
license is an SPDX id read at a recorded commit; every brief reference and
every agent CLI is covered; no credential file read; nothing outside `doc/`
changed (`git status`).

## Plan 2 - Architecture

Design the application from the research: the three areas of the brief (left:
orchestrator, workflow tree, phase status, agent hierarchy; main: live subagent
activity, logs, progress, agent selection, diff inspection; bottom: quota,
reset timers, tokens, context, model, agent statistics), keyboard navigation,
live updates, responsive layouts, and the style of `doc/reference-design.md`.
Bubble Tea, Lip Gloss and tmux unless plan 1 found better; each CLI's own
orchestration reused, not rebuilt.

- `doc/architecture.md`: stack with versions; one collector per agent CLI
  behind a common model the interface reads, so a CLI can be added or missing
  without touching the interface; data flow from sources to screen; how it runs
  on Windows (tmux needs WSL); how it starts.
- `doc/ui-design.md`: each area at 3 terminal sizes (text mock-ups), with
  several agent CLIs on screen at once; colors and symbols mapped to the
  reference; every key binding; every state (loading, live, no data, error,
  CLI not installed).

Checks: every brief item of phase 2 placed in a section; every choice traced to
a plan 1 finding; mock-ups for 3 sizes; one collector per agent CLI; no rebuilt
workflow engine.

## Plan 3 - Reliability

- `doc/reliability.md`: collection layer separate from the interface (the
  interface only reads the common model); what happens when each agent CLI or
  each telemetry source is missing; how every CLI's configuration stays
  untouched; JEV optional; the mock-data mode; the test plan (telemetry parsing
  per CLI, agent state transitions: each state and transition listed).

Checks: each brief item of phase 3 answered; each telemetry source from plan 1
has a fallback; the state machine lists every state and transition, and fits
all three CLIs.

## Plan 4 - Proposal

- `doc/proposal.md`: the 6 items the brief asks for (comparison, architecture
  and stack, terminal UI design, prioritised MVP roadmap, reuse versus build,
  risks and compatibility constraints), summarised from plans 1-3 with links.
  The MVP roadmap says in which order the agent CLIs are supported, and becomes
  the next plans in this file, once approved.

Checks: the 6 items present; each links to its detailed document; the roadmap
in ordered steps, each with its own checks. Then present it and wait for the
owner's approval.
