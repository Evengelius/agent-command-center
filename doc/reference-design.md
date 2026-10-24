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
| `08-input-editor.png` (1685x139), from amendment 29 | **the input editor's row**: a "+" box, the pill "Fable 5.1 High · Ultracode", "⚡ Auto" at the right end, design placeholders | [The input editor's row](#the-input-editors-row) |
| `09-plus-menu.png` (492x196), from amendment 30 | **the "+" menu**: the "+" clicked, on an active background, a menu above it with a folder icon and "Upload from computer" highlighted under the mouse | [The "+" menu](#the-plus-menu) |

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
edge and its state word, the footer's "F4 Manual/Auto" ("F4 modes" since
plan 39) and the key page.

## The input editor's row

`08-input-editor.png` (1685x139), sent with amendment 29 of the brief
(plan 29): zone B's input editor, magnified, as the owner wants its lower
row. Top to bottom: the prompt row, "> " in lavender, the cursor block,
"Ask Claude Code anything..." in grey; a thin grey separator across the
box; then the row: at the left a small rounded box with a white "+", a
bordered square on the dark background; after a gap a dark grey rounded
pill reading "Fable 5.1 High" in white, a lavender dot, "Ultracode" in
violet bold; at the right end a lightning bolt and "Auto" in white, a
cell or two from the edge. The owner: the "+" will later add a file, the
pill will later be the model selection, "Auto" the effort selection;
design only for now, no functionality.

What plan 29 takes from it: the helper line's key hints go; the row holds
the "+" and the pill on zone B's tool rows' ground (the image's grey,
`#1D2129`, against the ground's `#1E2133`), "↯ Auto" at the right end (the
mode pill since plan 39, [The modes selector](#the-modes-selector)), a
separator above the row; the pill names the session's real model, "High
· Ultracode" standing as the owner's placeholders until an effort and a
mode have a source. What it keeps: the editor's dynamic statuses (a turn
running, a paste, a notice, the quit question, a keys problem) and the
line count of a multi-line draft, in the row's free middle, since the row
is where they were; the keys stay in the footer and on the F1 page.

## The "+" menu

`09-plus-menu.png` (492x196), sent with amendment 30 of the brief (plan
30): the editor's "+" after a click, magnified. The "+" box has an
indigo background (`#242D52`) and its "+" in lavender-white; above it a
rounded box with a grey border (`#333447`) and a dark fill, a small tail
pointing down at the "+", holding a folder outline icon and "Upload from
computer" in lavender-white (`#D9D5FC`) on an indigo highlight
(`#272745`): the item under the mouse. The owner: the message must be
"Upload from computer" with that folder icon; the active background on
the clicked "+"; the hover colour on the item under the mouse; the
design first, the upload itself later.

What plan 30 takes from it: a click on the band opens a three-row box
above it, its bottom edge on the editor's rule, in the frame's colour,
with " 🗀 Upload from computer " in the thinking edge's lavender-white;
the band and the hovered item on a new active ground, the image's
indigo; the terminal asked for every mouse motion while the menu is
open, so the hover follows the pointer. What differs: the tail, which a
one-cell grid cannot draw; the box's fill, the background itself.

## The image preview

`10-chat-image-preview.png` (926x308), sent with amendment 32 of the
brief (plan 32): the chat's message box after the owner pasted an image.
Above the text, a 150x150 thumbnail: the image cut from its centre to a
square, its corners rounded by about 7 pixels, a 1-pixel light-grey
border (`#CCCCCB` on the white box); the text starts 31 pixels under it;
the send and voice icons at the right of the last row. The owner: paste
an image in the terminal chat as here; a preview before sending as here.

What plan 32 takes from it: each attached image as a square thumbnail,
the thumbnails side by side one cell apart above the chips (the owner, on
the screenshots: the space between them "way shorter"), each cut from
its centre, its corners rounded, a 1-pixel border, drawn with Sixel in
Windows Terminal, 14 columns by 7 rows (140x140 image pixels, the
terminal giving a cell 10x20 of them) from 50 rows, 10 by 5 below. What
differs: the border in the frame's colour, the image's light grey
suiting a white box and not the dark one; plan 31's chips under the
thumbnails, between them and the text; no blank row under them, rows
being scarce.

## The model selector

`11-model-selector.webp` (1354x1162), sent with amendment 34 of the brief
(plan 34): the popup a click on the editor row's model pill opens, which
the owner calls the final approved visual specification. Colours read
from its pixels (the median of each element's strokes).

- A box on a near-black ground (`#131519`, the page around it `#0F1014`),
  a lavender border (`#CA92F2`) about 3 px wide with rounded corners, and
  a small tail at the bottom-left pointing down, in the border's colour.
- The top row: "CLAUDE MODELS" in bold lavender (`#CC97F2`) at the left,
  "Current session" in grey (`#9299A9`) at the right.
- Five models, each two lines: the name in white, the description under
  it in grey (`#848FA2`): "Default (recommended)" / "Opus 5.5 · Best for
  everyday, complex tasks"; "Fable 5.1" / "For your toughest challenges";
  "Opus 5.5" / "For complex work and everyday tasks"; "Sonnet 5.5" /
  "Most efficient for simpler tasks"; "Haiku 5.5" / "Fastest for quick
  answers". The names line up, the descriptions under them. From one
  model's name to the next is 2.75 times the line height; from the header
  to the first name, 1.8.
- The second model, Fable 5.1, lies on a band (`#342B47`) with rounded
  corners across the box's inner width, a pink "▶" (`#E286D0`) before its
  name, its description lighter on the band (`#A3A3BF`) and a white "✓"
  at the band's right end, level with the name.
- A thin grey line (`#5E616D`), "▶ More models" in white with a white "›"
  at the right end, a second line.
- "Effort:" in white, then a dark rounded box (`#232631`) reading "XHigh"
  in white with a white chevron at its right.
- "Ultracode" in white, "Enables dynamic workflows" in grey (`#878D9F`),
  and at the right end a lavender pill toggle (`#CA84FB`) with a white
  knob at its right: on.
- The bottom border closely beneath the Ultracode row: no footer, no
  further row.

What plan 34 takes from it, in cells, the box's top-left corner at column
0 and row 0, the box W cells wide:

| Element | Cells | Colour |
| - | - | - |
| Border | rounded box-drawing corners and lines; "▾" in the bottom edge at column 1, above the pill's first cell | `#CA92F2` |
| Header, row 1 | "CLAUDE MODELS" or "CODEX MODELS", bold, from column 2; "Current session" ending at column W-3 | `#CA92F2`; `#9299A9` |
| Models, from row 3 | a name row and a description row, both from column 5, one blank row between models (the image's 2.75 lines rounded to 3) | names in the bright white of the palette; descriptions `#848FA2` |
| Cursor | the band from column 2 to column W-3 over the model's two rows, "▶" at column 3 of the name row | band `#342B47`, "▶" `#E286D0`, the description on the band in the muted grey of the palette |
| Model in force | "✓" at column W-5 of its name row | the bright white |
| Separators | a line from column 2 to column W-3, right under the last model, and right under "More models" | `#5E616D` |
| More models | "▶" (collapsed) or "▼" (expanded) at column 3, "More models" at column 5, "›" at column W-5 | the bright white |
| Effort, two rows under the second line | since plan 37 (The effort slider, below): "Effort (<level>)" at column 3, "Controls reasoning depth" at the settings rows' description column (20 for Claude Code's), the slider's pill in columns W-19 to W-4, with Sixel an image over W-19 to W-3 whose last cell carries the band, or W-19 to W-4 in text | white; `#848FA2`; the slider's own |
| Ultracode, two rows under the effort | "Ultracode" at column 3, "Enables dynamic workflows" at the settings rows' description column (plan 37), the toggle in columns W-7 to W-4, with Sixel an image over W-7 to W-3 whose last cell carries the band (plan 35), or W-8 to W-4 in text | white; `#878D9F`; track `#CA84FB`, knob white |
| Bottom edge | the row right under the Ultracode row | `#CA92F2` |

W is 63 (the image's width over its cells), the longest name or
description plus 12, or what the settings rows take (66 for Claude Code's
levels, plan 37), whichever is largest; 25 rows in all for Claude Code's
five models with More models, 23 for Codex's, which has no Ultracode row.

What differs: one size of type, the terminal's, where the image's names
and header are larger than its descriptions; the 2.75 lines between
models rounded to three rows; the tail drawn as "▾" in the bottom edge,
where plan 30's menu, the same one-cell grid, left its tail out; the
box's ground the dashboard's own background, as plan 30's; the band's
and the box's corners square, a cell being the smallest unit. The toggle,
since plan 35 (brief amendment 36, the owner's image of it, a track of
76x40 pixels and a knob of 32), is a pill of 40x20 pixels over four
cells, in an image of five whose last cell carries the band: its ends
half circles, its knob 16 pixels across with a margin of 2, 2:1 where
the owner's track is 1.9:1, and a little wider on screen
where the font's cells are wider than 1:2, Windows Terminal drawing 10x20
pixels in a cell (2.1:1 at 120x40, 48x23 pixels). Without Sixel it is five cells of
Powerline's half circles, its knob as tall as the track, with no margin.
The effort control is plan 37's slider, not the image's box (The effort
slider, below).

### The effort slider

Brief amendment 38: the owner's five images of one slider at five levels,
lowest to highest, each about 300x50 pixels on `#151719`, "the
authoritative visual references" for the selector's effort control:

- One pill, 289x36 pixels, its ends fully rounded, its track `#272B2D`.
- A white disc 34 pixels across as the thumb, its centres at the five
  levels evenly spaced from one end's centre to the other's.
- A 4x4-pixel dot at each level, `#697276` on the track and `#95A9D4` on
  the fill, the thumb's own hidden under it.
- At the four lower levels, a fill of `#6784C1` from the left end to the
  thumb: the same colour at each, its length growing.
- At the highest, the whole pill lit by a gradient from `#4C3F99` at the
  left through `#B288DF` at about 45% to `#815AB3` near the thumb, with
  eleven pale lavender specks of 2 to 4 pixels.

What plan 37 takes from them: the pill drawn at one terminal row, 160x20
pixels over 16 cells (the images' 8:1), as a Sixel image of 17 cells
with the band's last cell; the thumb 18.9 pixels across, the dots 2.3,
the levels from x 10 to x 150; the colours above, the gradient sampled
along the fifth image's middle row; the specks where that image has
them; the highest level the model's last, Ultra for a Codex model that
lists it. Beside each image scaled to one row in Windows Terminal 1.23,
the slider matches at each level (`scratchpad\plan37\probe-wt.png`).

What differs: the images show one slider; the dashboard has no level
names on it, the label "Effort (<level>)" naming the level in force; a
level asked for and not yet in force is ringed, a held effort greyed,
which the images do not show; the thumb glides between levels (180 ms);
without Sixel the slider is text, its gradient a ground a cell, its thumb
two half circles as tall as the pill.

## The modes selector

Brief amendment 40 (plan 39): three images, "authoritative visual design
references", the popups lossy WebP of 1254x1254 pixels, the pill 258x92:

- `35.webp`, Claude Code's popup: a charcoal ground `#111518`, a thin
  border near `#B064F4` with rounded corners and a "V" let into the bottom
  edge near its right end; "MODES" in `#B86AF6`, bold, and "Current
  session" in `#A0AECC` on the first row; four options, each an outline
  icon at the left in `#AEBCDC` (a raised hand, code brackets, a scroll,
  a lightning), its title in `#F0F0F2` and its description, two lines
  (three for Auto), in `#95A3BF`, broken after "approval", "selected",
  "and", "that" and "pause" (32 characters at most a line). Auto selected:
  a rounded rectangle of `#2E2842` from above its title to below its last
  line, nearly the box's width, its title `#F2EFF6`, its description
  `#B09FD5`, its lightning `#E874F2`, a white check (`#F6F2F8`) at the
  right.
- `36.webp`, Codex's popup: the same box with three options, a hand, a
  shield with a prompt, a hexagon with "!"; Approve for me selected, its
  shield `#EEE0FA`; Full access's hexagon `#F4C46E` and title `#F5C673`,
  its description muted; the descriptions broken after "files",
  "detected" and "internet" (35 characters at most).
- `37.png`, the composer's pill: a fill of `#21223A`, a thin `#7E4AE1`
  outline with rounded corners (their radius 0.36 of its height), the
  lightning `#E982F6` at the left, "Auto" in `#DBD9E6`, no chevron.

What plan 39 takes from them: the box in cells, W = 19 + the longest
description line, the heading row, a half row before each option and
after the last, the titles and descriptions at column 11, the colours
above, the selected option's band, the check, Full access's amber; the
icons as the dashboard's own drawings on a 24-unit grid, the band's
rounded ends and the check as Sixel images in two strips beside the
text, so the band can move under the pointer without redrawing them; the
pill as a Sixel image a row high, its title drawn by GDI at 13 pixels.
Read back from Windows Terminal's screenshots (plan 39's evaluation,
39.14), every colour of the box and the pill is within 3 levels of the
image's, but the anti-aliased label's brightest pixel (10).

What differs: the terminal's monospace font for the images' proportional
one, and its cells for their free layout, so the gaps between options are
half rows; the pill's title is larger against its height than the
image's, which would be unreadable at one row; the "V" is the frame's
"▾" above the pill's middle; the box's ground also fills the outer half
of its frame's cells, a halo of half a cell barely darker than the
terminal's own ground (`#111518` against `#0C0C0C`); the images show no
hover, no greyed option and no confirmation, which follow the same
colours (the hovered option on the band without the check; the
confirmation's heading and Turn on Full access in amber).
