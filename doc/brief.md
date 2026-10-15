# Brief

The owner's request, kept word for word. It is the source of truth for the
project's goal; `plan.md` turns it into steps, and `AGENTS.md` into rules.
Change it only when the owner changes the request, and record the date.

Received 2026-10-15.

---

I want to build a personal, native terminal dashboard for Claude Code multi-agent workflows.

**My goal:** Create a terminal-based command center inspired by the video and screenshot in `references/`. I love its dark, technical aesthetic, colored workflow diagrams, animated progress indicators, and hierarchical visualization of agents.

Before writing any code, research these existing implementations:

- https://github.com/bjornjee/agent-dashboard
- https://github.com/seunggabi/claude-dashboard
- https://github.com/jedarden/ccdash
- https://github.com/Akramovic1/jev-pilot
- https://code.claude.com/docs/en/workflows
- https://code.claude.com/docs/en/statusline

**Phase 1 — Research**

Inspect their architectures, implementation approaches, dependencies, licensing, and relevant source code.

Identify which components we can reuse and which should be built ourselves. Determine whether forking an existing repository is preferable to starting from scratch.

Do not assume that the video represents an existing working application.

**Phase 2 — Architecture**

Design a native terminal application with three primary areas:

1. Left: orchestrator, workflow tree, phase status, and agent hierarchy.
2. Main: live subagent activity, execution logs, progress, agent selection, and diff inspection.
3. Bottom: real 5-hour and 7-day subscription quota, reset timers, token usage, context consumption, model information, and agent statistics.

Prefer Bubble Tea, Lip Gloss, and tmux unless your research identifies a better solution.

Reuse Claude Code's native orchestration wherever possible. Do not rebuild its workflow engine unnecessarily.

Support keyboard navigation, live updates, responsive terminal layouts, and a polished visual style inspired by the reference.

**Phase 3 — Reliability**

- Use officially documented interfaces where possible.
- Separate the data collection layer from the interface.
- Handle unavailable telemetry gracefully.
- Keep existing Claude Code configurations intact.
- Treat JEV routing as optional, not a prerequisite.
- Provide mock-data mode for visual development.
- Implement tests for telemetry parsing and agent state transitions.

**Phase 4 — Proposal**

Before implementing anything, provide:

1. A comparison of the reference projects.
2. A recommended architecture and technology stack.
3. A detailed terminal UI design.
4. A prioritized MVP roadmap.
5. A list of what can be reused versus developed.
6. Technical risks and compatibility constraints.

Do not edit files or install dependencies yet. Present the research and proposed plan, then wait for my approval.

My priority is a beautiful, functional, personalized terminal experience — not creating another complex orchestration framework.

---

The reference video: https://video.twimg.com/amplify_video/2110467312574709760/vid/avc1/1200x1500/UkwR2TP8IvY8ydvI.mp4
(downloaded as `references/reference-video.mp4`).

## Amendment, 2026-10-15: not only Claude Code

The owner: "it must work with Codex or Gemini also, it's multi agent".

The dashboard follows several coding-agent CLIs, not only Claude Code: Claude
Code, OpenAI Codex CLI and Google Gemini CLI. Wherever the brief above says
"Claude Code", read "each supported agent CLI". Claude Code stays the first and
best documented target; each agent CLI is optional, so the dashboard works with
any one of them installed. Hence the project's name, `agent-command-center`.

## Amendment 2, 2026-10-15: interactive conversation and live control center

The owner's update, word for word, with a layout mock-up
(`references/reference-layout-mockup.png`). It changes the project from a
monitoring dashboard into an interactive workspace: the owner talks to the
orchestrator in the application while watching the agents work. Read with
amendment 1: the conversation targets Claude Code first; whether Codex CLI and
Gemini CLI can be driven the same way is a research question, and each stays
optional.

---

# Architecture Update — Interactive Conversation & Live Agent Control Center

I want to extend the terminal dashboard requirements I previously gave you.

The original design is good, but it is missing one crucial feature: **a fully interactive, real-time conversation with the main AI orchestrator.**

This should not be a passive monitoring dashboard. It must function as a complete, interactive Claude Code workspace, where I can communicate with Claude and simultaneously observe the entire multi-agent workflow.

## 1. Updated Terminal Layout

Design the interface around four functional zones:

**A. Left — Orchestrator & Workflow Tree**

- Show the current main orchestrator and its model.
- Display the hierarchical execution plan.
- Show stages, dependencies, active tasks, and completion status.
- Display delegated subagents and their parent-child relationships.
- Update automatically when workflow events occur.
- Allow keyboard navigation through workflow nodes.

**B. Main Upper Area — Interactive AI Conversation**

This is a critical addition.

I want to communicate directly with the orchestrator, just as I would in a normal Claude Code session.

Requirements:

- A persistent interactive conversation panel.
- A multiline user input area.
- Streaming AI responses, displayed incrementally.
- Support for Markdown, syntax highlighting, and code blocks.
- Display meaningful progress messages and tool-use activity.
- Support clarification questions and permission/approval requests.
- Preserve conversation history and session continuity.
- Allow follow-up instructions without restarting the application.
- Handle interruptions and cancellation safely.

The conversation must be real, not simulated or generated from fake agent-status messages.

Investigate how to preserve native Claude Code interaction and functionality using documented interfaces, without unnecessarily recreating Claude's entire terminal protocol.

**C. Main Lower Area — Live Subagent Activity**

Under the conversation, show live execution information.

- Active, waiting, completed, and failed agents.
- Model assigned to each agent.
- Current task and relevant file paths.
- Real tool executions and observable progress.
- Agent-specific logs and outputs.
- Verification and test results.
- Retry information.
- Ability to select an agent and expand its detailed activity.
- Ability to return to the orchestrator conversation.

Agent activity must update independently from the conversation.

I should be able to watch multiple agents running while Claude is talking to me.

**D. Bottom — Live Usage & Telemetry**

Create a persistent, compact telemetry panel containing:

- 5-hour subscription quota usage.
- 7-day subscription quota usage.
- Quota reset times, when available.
- Current model and context-window usage.
- Token consumption and estimated API-equivalent costs, clearly distinguished from subscription usage.
- Active agent count and concurrency limit.
- Execution duration.
- Errors and retry counters.

Use real telemetry from supported sources. If a value is unavailable, display "N/A" rather than inventing it.

## 2. Core Interaction Model

The expected experience is:

1. I launch the custom terminal application.
2. I type a request in the conversation input.
3. Claude begins responding and orchestrating the work.
4. The workflow tree updates automatically.
5. Worker agents appear and execute their tasks.
6. Their activities update live without blocking the conversation.
7. I can inspect agents, review outputs, and approve actions.
8. Claude reports the results in the same conversation.
9. I continue the conversation without leaving the application.

The interface should feel like an IDE-style AI command center running entirely in a terminal.

## 3. Event-Driven Architecture

Important: The AI should NOT regenerate or redraw the interface through LLM calls.

Use a persistent TUI that renders application state.

Investigate a state-management/event architecture with events such as:

- conversation.message.delta
- conversation.message.completed
- tool.started
- tool.completed
- workflow.stage.changed
- agent.started
- agent.progress
- agent.completed
- agent.failed
- approval.requested
- telemetry.updated

These are conceptual event names, not assumptions about actual Claude Code API events. Map them to real documented event sources where possible.

Separate:

- Claude execution and session management.
- Event collection and normalization.
- Application state.
- Terminal rendering.
- User input and command handling.

Avoid polling unnecessarily when event-based integration is available.

## 4. User Experience

Prioritize a beautiful, responsive terminal experience inspired by the video reference.

Include:

- Smooth, restrained animations.
- Clear color-coded agent statuses.
- Keyboard shortcuts and panel focus management.
- Scrollable conversation and log history.
- Expandable and collapsible panels.
- Responsive terminal sizing.
- Full-screen agent inspection.
- Real-time updates without screen flickering.
- A clear distinction between AI messages, observed execution events, and inferred status labels.

The input panel must remain easy to access, including while background agents are working.

## 5. Research Questions

Before implementation, investigate:

1. Can Claude Code's interactive session be embedded or controlled through supported interfaces?
2. Which approaches support genuine streaming responses?
3. How can we access subagent lifecycle events and tool activity?
4. How can approval prompts and permission handling be preserved?
5. How can session persistence, cancellation, and resumption work?
6. Can we reuse an existing terminal UI implementation instead of rebuilding it?
7. How should we handle concurrent subagent output?
8. How can quota information be collected reliably?
9. What are the limitations on Windows, WSL2, Linux, and macOS?

Do not assume that all requested runtime data is exposed by Claude Code. Identify integration limitations and propose realistic fallbacks.

## 6. Deliverables

Update the previous architectural proposal to incorporate these requirements.

Provide:

- A revised terminal UI layout.
- An interaction-flow diagram.
- A proposed event and state architecture.
- Recommended integration strategy with Claude Code.
- Components reusable from the previously identified GitHub repositories.
- Technical limitations and risks.
- An updated MVP implementation plan.

**Do not begin coding yet.**

Incorporate this into the existing proposal, explain any architectural changes, and wait for my approval before implementation.

**The ultimate goal is a fully interactive Claude Code experience with a beautiful, continuously updating multi-agent mission-control interface — not merely a log viewer.**

## Amendment 3, 2026-10-15: full clipboard and multiline input

The owner's additional requirement, word for word. It applies to the
conversation input of zone B (amendment 2) and to every panel text can be
selected from, for every agent CLI.

---

## Additional Requirement â€” Full Clipboard & Multiline Input Support

The interactive conversation must support a proper, full-featured terminal text editor, not a basic single-line input field.

Requirements:

- Support multiline typing and pasting of very long prompts, code snippets, logs, and documents.
- Support standard system clipboard copy/paste.
- Allow mouse-based text selection and copying from conversations, agent logs, code blocks, and other panels.
- Support keyboard text selection, cursor movement, editing, and deletion.
- Use bracketed paste handling so multiline pastes never accidentally submit messages.
- Provide a scrollable input area that handles thousands of lines without freezing the TUI.
- Implement configurable Enter-to-send and newline shortcuts.
- Optionally allow editing a prompt in an external editor via $EDITOR.
- Preserve pasted content, whitespace, indentation, and line breaks.
- Ensure clipboard operations work correctly across Windows Terminal, PowerShell, WSL2, Linux, and macOS wherever supported.

Investigate terminal limitations, including mouse capture, tmux copy mode, clipboard integration, and OS-specific shortcuts.

**Important:** This functionality must be provider-independent and work with Claude Code, Codex, Gemini CLI, and future supported agents.

Treat clipboard support and reliable multiline editing as essential MVP features, not optional enhancements.