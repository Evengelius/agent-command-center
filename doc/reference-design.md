# Reference design

What the owner likes in `references/`, described so that a design decision can
point at it. Two references, with different roles:

| File | Gives | Section |
| --- | --- | --- |
| `reference-screenshot.png` (1200x1500), `reference-video.mp4` (the same screen, animated, 3.4 MB) | the **style**: colors, frames, diagrams, animation | [What the screen shows](#what-the-screen-shows) |
| `reference-layout-mockup.png` (768x646), from amendment 2 of the brief | the **layout**: four zones, with the conversation | [Layout mock-up](#layout-mock-up) |
| `01-startup-configuration.png` (960x721), `02-agent-command-center.png` (960x325), from amendment 4 | the **screen sequence**: startup, then dashboard | [Screen sequence](#screen-sequence) |

**It is a visual reference, not a product.** The reference looks like a
promotional mock-up ("CLAUDE CODE ULTRACODE TIP"). Its model names, flags
(`--effort ultracode`), "JEV fork layer" figures and numbers may not exist or
work as shown. Phase 1 must not assume any of it is real; take the look, not
the claims.

## What the screen shows

A dark terminal window (macOS frame, "~/app - zsh - 90x46"):

- **Title bar**: a banner with three highlighted labels ("CLAUDE CODE ULTRACODE
  TIP", "OPUS 5.5 PLANS", "16 AGENTS RUN"), underlined by a double rule.
- **Legend row**: one colored square per role: purple "opus 5.5 · plans",
  blue "haiku 5.5 · scans", orange "sonnet 5.5 · ships", green "jev · forks".
- **Left column**, purple frame, "OPUS 5.5 orchestrator": a vertical timeline
  with diamond markers ("writes script", "idle mid-run", "signs off"), the
  workflow script shown as code (`phase('scan')`, `pipeline(files, haiku ·
  audit)`, `parallel(fixes, sonnet · patch)`, `agent(review)`), then counters
  ("agents 10/16", "opus context 0.8k") and short notes ("script holds the
  plan", "never reads raw agent output").
- **Top right**, purple frame: the main session box ("writes the workflow",
  an effort meter of filled blocks).
- **Middle right**, green frame: a routing panel ("which file", "which model",
  "retry or stop") with horizontal bar meters and probabilities.
- **Workflow runtime**: a connector line that forks into three phase boxes,
  with animated dots travelling along the connectors:
  - "scan", blue frame, Haiku, a 100% bar, "audits each file", "✓ done";
  - "edit", purple frame, Sonnet, a 46% bar, "own isolated copy", "1/4 agents";
  - "verify", orange frame, Sonnet, an empty 0% bar, "cross-checks all",
    "· queued".
  The three branches join into a final box, "back to Opus · sign off,
  verified findings only".
- **Session log**, framed: timestamped lines, each with a colored model name
  ("21:12:53 haiku 5.5 src/routes/pay.ts · 1 finding").
- **Prompt and status line**: the command line, then a one-line status bar
  ("workflow [edit]  agents [10/16]  opus [idle]  jev [3,366 forks]").

## What the owner wants from it

From the brief: the **dark, technical aesthetic**, **colored workflow
diagrams**, **animated progress indicators** and the **hierarchical
visualization of agents**. Concretely, the elements to carry over:

| Element | In the reference | Why it matters |
| --- | --- | --- |
| One color per model or role | legend row, frame colors, log names | read the screen at a glance; with several agent CLIs (Claude Code, Codex), each CLI and model needs its own color |
| Box-drawing frames with titles | every panel | the technical look |
| Phase boxes on a connector tree | scan / edit / verify | the workflow as a diagram |
| Progress bars with percentages | each phase box | live progress |
| Animated dots on connectors | between boxes | "it is running" without text |
| Status words with symbols | "✓ done", "◓ 1/4 agents", "· queued" | state at a glance |
| Timestamped, colored log | session log | what happened, in order |
| One-line status bar | bottom line | the essentials always visible |

The layout itself differs from the brief's: the brief asks for a left column
(orchestrator, tree, phases), a main area (live activity, logs, diffs) and a
bottom bar (quota, tokens, context, models, statistics). The reference is a
style guide for those areas, not their layout; the layout comes from the
mock-up below.

## Layout mock-up

`reference-layout-mockup.png`, sent with amendment 2 of the brief. Its own
caption says what it is: "Conceptual terminal layout · Mock statuses and usage
figures". Every figure in it (42%, 68%, "Agents 4/16", "Tokens 128K") is made
up: take the arrangement, not the numbers.

```text
+--------------------------------------------------------------- * LIVE --+
| AGENT CONTROL CENTER                                                    |
+------------------------+------------------------------------------------+
| A. ORCHESTRATOR        | B. CLAUDE - CONVERSATION            Streaming  |
|    Opus - Workflow     |    YOU: Refactor the auth system and add tests |
|  v Analyze request     |    CLAUDE: I'll inspect the existing ...       |
|  v Create plan         |    o Running authentication checks...          |
|  o Execute changes     |    > Ask Claude or give instructions...        |
|    |- scan-01 v        +------------------------------------------------+
|    |- scan-02 v        | C. LIVE AGENTS                       4 ACTIVE  |
|    |- edit-01 o        |    v Haiku - Scanning complete                 |
|    `- edit-02 o        |    o Sonnet - Editing auth.ts                  |
|  . Verification        |    o Sonnet - Writing tests                    |
|  . Final review        |    . Sonnet - Verification queued              |
|                        |    [Select agent to inspect logs]              |
+------------------------+------------------------------------------------+
| D. LIVE QUOTA & TELEMETRY                                               |
|    5-hour quota [=====------] 42%      7-day quota [========---] 68%    |
|    Agents 4/16              Context 24%               Tokens 128K       |
+-------------------------------------------------------------------------+
```

(`v` done, `o` in progress, `.` waiting, in place of the mock-up's ✓, ◉ and ○.)

What it fixes, matching zones A to D of amendment 2:

- **A, left, full height**: the orchestrator and its model, then the plan as a
  tree of stages, with each stage's subagents indented under it
  (`scan-01`, `edit-01`...) and a status symbol per line.
- **B, upper right**: the conversation, with a "Streaming" badge, the user's
  and the orchestrator's turns labelled, an in-progress line for what the
  orchestrator is doing, and the input line at the bottom of the panel.
- **C, lower right**: the live agents, one line each (model, task, status), a
  count of active agents in the title, and a hint to select an agent.
- **D, bottom, full width**: the quota bars with percentages, then agents,
  context and tokens on one line.
- A title bar with a "LIVE" badge.

Its palette fits the first reference: dark background, thin rounded frames,
pale violet panel titles, green for done, amber for in progress, grey for
waiting, and a violet and an amber bar for the two quotas.

## Screen sequence

From amendment 4: the application opens on a startup screen (image 1), then
moves to the dashboard (image 2), as one coherent application. Both images are
design mock-ups; image 1 says so itself ("Interactive design mockup - does not
launch a CLI or access your project").

### Image 1: startup configuration

```text
+---------------------------------------------------------------------+
| [>_] AGENT COMMAND CENTER                               New session |
+---------------------------------------------------------------------+
| Configure your workspace                                            |
| Select an AI agent and the project you want to work on.             |
|                                                                     |
| 1. AI Agent *                                                       |
|    (o) Claude Code                                                  |
|    ( ) OpenAI Codex                                                 |
|---------------------------------------------------------------------|
| 2. Project Directory *                                              |
|    ( ~/projects/agent-command-center                          )     |
|    [folder] Browse directories, recent projects, or enter a path    |
|---------------------------------------------------------------------|
| Optional session settings                                       v   |
| (            Preview configuration ->                         )     |
+---------------------------------------------------------------------+
```

A single centred column: a title bar, a heading in violet with one line of
help, two numbered questions each marked required with a red asterisk, radio
buttons for the agent CLI, a rounded text field for the path, thin separators,
and one wide light button. The image lists a third agent, Google Gemini CLI;
amendment 5 removes it, so the list has two: Claude Code and OpenAI Codex.

### Image 2: dashboard, empty state

```text
+---------------------------------------------------------------------+
| [>_] AGENT CONTROL CENTER                                    * LIVE |
+---------------------------------------------------------------------+
| [folder] agent-command-center   [branch] main          Claude Code  |
| ~/projects/agent-command-center                                     |
+---------------------------------+-----------------------------------+
| ORCHESTRATOR                    | CONVERSATION                      |
| o Waiting for instructions      | Ready to work on this project.    |
|                                 | > Ask your agent anything...      |
+---------------------------------+-----------------------------------+
```

A project bar under the title (project name, path, and the agent CLI on the
right), then zones A and B side by side, as they look before the first
prompt. Zones C and D are not in the image: they come from the layout mock-up.

### Where the images and amendment 4's text differ

The text is the specification; the images give the look.

| In the image | The text says | So |
| --- | --- | --- |
| "Browse directories, recent projects, or enter a path" | manual entry only in the MVP; amendment 9 adds recent projects, favourites and a project browser | a text field with the current directory prefilled and a short help line; under it the recent and favourite projects, and Ctrl+O opens the browser (ui-design.md, Startup screen) |
| "Optional session settings" | exactly two mandatory questions; no advanced configuration | not in the MVP |
| "Preview configuration" | proceed once both answers are validated | the button starts the session, after validation |
| "AGENT CONTROL CENTER" (image 2) | "AGENT COMMAND CENTER - LIVE" | the project's name, "AGENT COMMAND CENTER" |
| git branch "main" in the project bar | provider, project name and path | the branch is shown when the project is a git repository, for display only (no git management) |
