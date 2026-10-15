# Reference design

What the owner likes in `references/`, described so that a design decision can
point at it. The files are `reference-screenshot.png` (1200x1500) and
`reference-video.mp4` (the same screen, animated, 3.4 MB).

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
| One color per model or role | legend row, frame colors, log names | read the screen at a glance; with several agent CLIs (Claude Code, Codex, Gemini), each CLI and model needs its own color |
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
style guide for those three areas, not their layout.
