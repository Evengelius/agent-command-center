# Reference design

What the owner likes in `references/`, described so that a design decision can
point at it. Two references, with different roles:

| File | Gives | Section |
| --- | --- | --- |
| `reference-screenshot.png` (1200x1500), `reference-video.mp4` (the same screen, animated, 3.4 MB) | the **style**: colors, frames, diagrams, animation | [What the screen shows](#what-the-screen-shows) |
| `reference-layout-mockup.png` (768x646), from amendment 2 of the brief | the **layout**: four zones, with the conversation | [Layout mock-up](#layout-mock-up) |
| `01-startup-configuration.png` (960x721), `02-agent-command-center.png` (960x325), from amendment 4 | the **screen sequence**: startup, then dashboard | [Screen sequence](#screen-sequence) |
| `03-zone-a-telemetry.webp` (1312x1199), from amendment 22 | **zone A with the telemetry**: the orchestrator's header, the stages, session usage, the quota windows and the notes, zone D gone | [Zone A with the telemetry](#zone-a-with-the-telemetry) |
| `04-zone-c-timeline.png` (553x315), from amendment 26 | **zone C with the execution timeline**: the heading inside the frame, the agents' panel and the timeline panel with the agent's status | [Zone C with the execution timeline](#zone-c-with-the-execution-timeline) |
| `05-zone-c-timeline-close.png` (1128x373), `06-zone-c-timeline-close.webp` (1403x469), from amendment 27 | **zone C's close-ups**: the lines in the title colour, the card's two columns and their bar, the panel as wide as its card, the model grey | [Zone C with the execution timeline](#zone-c-with-the-execution-timeline) |
| `07-top-right.webp` (1685x504), from amendment 28 | **the top right**: "● LIVE" over the provider's name and version alone, over zone B's edge with the session's state | [The top right](#the-top-right) |

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

Since plan 20 the owner's two approved templates (brief, amendment 16,
which describes them) are the target design for the dashboard's geometry
and zone B: A and B side by side in the upper row, ending on the same row,
C and D the whole width under them, and zone B's cards. They replace this
mock-up's right-column split; its zones and palette stay
([ui-design.md, Layouts](ui-design.md#layouts-at-three-sizes)).

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

## Zone A with the telemetry

`03-zone-a-telemetry.webp`, sent with amendment 22 of the brief (plan 24):
"the final approved design" of the dashboard's upper part, drawn over a
session of the owner's (the demo of plan 22's amendment 20). What it
shows, left to right and top to bottom:

- **Zone A**, a rounded lavender frame about 59 columns wide, beside zone
  B. Its first row inside the frame: "▾ A · ORCHESTRATOR" in lavender. The
  second row: a lavender flower "✿", then "Claude Opus 5.5" in white, and
  at the row's right end "✓ COMPLETED" in green. A thin grey rule under it.
- **"› STAGES · inferred from activity"**: "STAGES" in white, the rest
  grey. Three turns, each a block: a status symbol and "TURN n" in white
  with its duration at the right ("16s", "10s", "1m 23s"), then two
  indented grey lines: "Agent launched" and "Bash · Write · Results" for
  turn 1; "Started by CLI · report completed" for turn 2; "Agent
  general-purpose" and "Write · Read · Results" for turn 3, whose whole
  block sits on a slightly lighter ground, the selected turn.
- A rule, then **"✦ TELEMETRY"** in lavender.
- **"▾ SESSION USAGE"**: three columns separated by thin vertical rules,
  "CONTEXT", "TOKENS" and "EST. COST" in grey capitals; under them "5% of
  1M" with a short lavender bar, "314.8k" with "session total" under it,
  "$0.59" with "API-equivalent" under it.
- **"▾ QUOTA WINDOWS"**: three windows, each three rows: "Session (5hr)"
  with "0%" at the right, a lavender bar on a dark track across the
  column, "Resets in 4h 12m" in grey; "Weekly (7 day)" and "17%", its
  bar a sixth filled, "Resets in 5d 3h"; "Fable limit" and "0%", an
  empty track, "Resets in 5d 3h". A blank row between the windows.
- A rule, then a note in grey: "ⓘ Values are from the provider when
  available." and "Estimates are marked accordingly." on a second row.
- **Zone B** to the right as plan 20 left it: "B · CONVERSATION" in its
  top edge, YOU and CLAUDE messages with their bars, a thinking summary
  row, a tool group, a RESULT box, the input "> Ask Claude Code
  anything…" with "Enter send · Ctrl+J newline".
- **Zone C** under both, full width: "C · LIVE AGENTS", an agent's two
  rows on the left and its preview on the right.
- No zone D, and no footer in the image (it ends under zone C).

What plan 24 takes from it: zone A's order of sections, the heading inside
the frame, the status on the model row, the stages' compact blocks, the
three usage columns, the quota windows' rows and lavender bars, the notes.

## Zone C with the execution timeline

`04-zone-c-timeline.png`, sent with amendment 26 of the brief (plan 27):
zone C alone, drawn over a session of the owner's (2026-10-22, 17:11: a
general-purpose subagent on Fable 5.1 listing a folder and writing
found.txt), a rounded dark card about 79 columns wide and 17 rows tall.
What it shows, top to bottom:

- **The header row**, inside the frame: "C · LIVE AGENTS" in white, bold,
  at the left; "0 ACTIVE · 1 DONE" in grey at the right end. A thin rule
  under it, across the card.
- **Two panels** under the rule, separated by a thin vertical line from
  the panel headings to the bottom: the left one about two fifths of the
  width.
- **"AGENTS"**, the left panel's heading, in light blue, bold. Under it
  the agent's two rows: a green "✓", "general-purpose" in white bold,
  "fable-5-1" in grey, then "…" (the rest of the row cut); the second row
  in grey, "00:41   30.1k tok", a thin vertical bar, then "tool:" and
  "Write" in white bold.
- **"EXECUTION TIMELINE"**, the right panel's heading, in the same light
  blue, bold, and at its right end "✓ COMPLETED" in green: the agent's
  status, dynamic ("Obviously the status you see on the right,
  "completed" is dynamic").
- **Four entries**, separated by thin grey rules, none after the last.
  Each: the time "17:11:31" in grey, the role, "notice" in blue or
  "assistant" in green, a grey vertical bar, then the text wrapped onto
  two rows at most, the bar drawn on both rows, the second row ending in
  "…" when the text goes on: "Working directory: C:\Users\…\AppData\Local\
  Temp\acc-demo …", "I'll start with the directory listing as specified.",
  "The listing shows three entries and no `found.txt`, so I'll write it
  now.", "Done. Results as plain data: …".
- No preview heading with the agent's label and model, no origin label on
  the rows, no key hint row.

What plan 27 takes from it: the heading inside the frame, the two panels
and their headings, the vertical bar before the agent's tool, the
timeline's entries with their role column, bar, two-row wrapping and
rules, and the agent's status at the timeline heading's right end.
What it keeps from the earlier decisions: the real events decide every
value (the model, the statuses, the turns' durations and tools, the
percentages, the reset times), the symbols of [Colours and
symbols](ui-design.md#colours-and-symbols), the palette, and zones B and C
as plans 20 to 22 left them. The mock-up's figures ("5% of 1M", "$0.59",
"17%") are illustrative, as its "Agent launched" and "Results" lines.

**The close-ups** `05-zone-c-timeline-close.png` and
`06-zone-c-timeline-close.webp`, sent with amendment 27 after the owner
found plan 27's first pass not respecting the design ("the separation "|"
is not respected either", "on the left, there is too much space"), show
the same card larger and settle what the small image left open:

- "C · LIVE AGENTS", "AGENTS" and "EXECUTION TIMELINE" are all in the same
  lavender, bold: the title colour. The rule under the header and the
  vertical line between the panels are in that lavender too, thin, not the
  dim grey of the card's border.
- The agent's card has two columns on a slightly lighter band: the left
  one "✓ general-purpose" over "00:41   30.1k tok", the right one
  "fable-5-1" over "tool: Write", a grey vertical bar between them on both
  rows, the label white bold, the model grey, "Write" white bold. Nothing
  else: no state word, no task text.
- The left panel is as wide as the card, about two fifths of the
  close-up's width, which is itself about 108 columns wide: on a 208-column
  screen the same share left most of the panel empty ("too much space"),
  so the panel takes its width from the card.
- Each timeline entry's grey bar and the thin rules between entries, as
  the small image showed.

What plan 27's second pass takes from them: design D7 to D9 in plan.md
(the lines in the title colour, the card's two columns with their bar on
both rows and nothing else, the selected agent's rows on zone B's tool
rows' ground, the panel as wide as its cards, the model grey).

## The top right

`07-top-right.webp` (1685x504), sent with amendment 28 of the brief (plan
28): the screen's top right corner of a live Claude Code session,
magnified. Top to bottom: "● LIVE", a green dot and the word in green, at
the right end of the title row; "Claude Code 2.1.303" in white under it,
at the right end of the project bar, nothing after it; then zone B's top
edge in lavender with "Idle" in grey let into it before the corner. The
owner: "Idle" is just the state of the session, this can stay. And it is
dynamic.

What plan 28 takes from it: the project bar's right end is the provider's
name and version alone while approvals are Manual with no change asked;
the mode shows there only while it is Auto or changing, since amendment
8 made Auto visible on purpose. What it keeps: the title row, zone B's
edge and its state word, the footer's "F4 Manual/Auto" and the key page.
