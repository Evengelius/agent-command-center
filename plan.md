# Current Plan

The active plan with its progress, the owner's standing decisions, and one
entry per closed plan. Each plan comes from `doc/brief.md` and its
amendments, ends with the evaluation summary of `doc/evaluation.md`, and
waits for the owner's approval before the next one starts. This file keeps
the current state only (the owner, 2026-10-18): a closed plan's full text,
criteria and progress log are in git (`git show 8888f59:plan.md` for plans 1
to 13, `git show 84d2ad8:plan.md` for plan 14, `git show b047434:plan.md`
for plan 15, `git show becfb34:plan.md` for plan 16, `git show f00c2d8:plan.md`
for plan 17, `git show 8b3b16b:plan.md` for plan 18, `git show
d84e5d4:plan.md` for plan 20, `git show f939d96:plan.md` for plan 21,
`git show d13ef71:plan.md` for plan 22, `git show e8d3e3b:plan.md` for
plan 23, `git show a3bfe5d:plan.md` for plan 24, `git show cb8f76d:plan.md`
for plan 25, `git show a9a5145:plan.md` for plan 26, `git show 041e7b8:plan.md`
for plan 27, `git show 8fe0077:plan.md` for plan 28, `git show d696f22:plan.md`
for plan 29, `git show 6025513:plan.md` for plan 30, `git show
8e2a721:plan.md` for plan 31, `git show b56d23e:plan.md` for plan 32, `git show
bc5d92b:plan.md` for plan 33, `git show 47b50ce:plan.md` for plan 34, `git show
b43cf46:plan.md` for plan 35, `git show c0cebc4:plan.md` for plan 36, `git show
31c22ef:plan.md` for plan 37, `git show dc9c6d9:plan.md` for plan 38, `git show
55e2bed:plan.md` for plan 39), and its results in its evaluation file.

## The owner's standing decisions

Decisions of closed plans that still apply; `AGENTS.md` and
`doc/evaluation.md` hold the others.

- **Live runs.** Each one is proposed first with its scope, model, expected
  usage and stopping conditions, and runs only once approved (plan 3,
  2026-10-16); Claude runs use Haiku as a per-process override.
- **The configuration check is hash-only**: credential contents are never
  read, and a file that cannot be classified safely is "changed, not
  classified", never reported unchanged (2026-10-16). No parser runs over a
  file that may hold credentials (the mixed-state tokenizer was withdrawn,
  2026-10-16); the one exception is confcheck's key-level comparison of
  `~/.claude.json`, sensitive keys skipped (plan 5).
- **Measured, not assumed.** Performance figures are targets until a test
  measures them; a documented design is never presented as verified runtime
  behaviour (2026-10-16).
- **The integration.** If driving a CLI without a terminal falls short, the
  limit is documented and the smallest adjustment proposed; the architecture
  never changes silently (2026-10-16).
- **Drafts and session data** survive cancellation, crashes and recovery; no
  conversation data is stored beyond the draft, and nothing sensitive goes
  into a log (2026-10-16).
- **Zone A shows values, not notes** (amendment 25, 2026-10-22, plan 26):
  its telemetry pane carries no notes row, legend or counter, now or in
  any later plan; errors, retries, unrecognised lines, freshness and the
  legends live in the details view (Enter on TELEMETRY), where each value
  has its source and age. An undocumented value keeps its "†" mark where
  it is shown.

## Recurring

Tasks tagged **[Recurring]** repeat on an outside event: they are never
closed, never listed as remaining, and each occurrence is reported in the
summary format (the owner, 2026-10-18).

- **[Recurring] After a CLI update.** At session start, `claude --version`
  and `codex --version` are compared with the "Last probed" line of
  `doc/research/probes/README.md`; a newer one is told to the owner with the
  re-check proposed (`doc/evaluation.md`, After a CLI update), which runs
  only once the owner approves it. The dashboard's project bar shows
  "newer than tested" meanwhile. Last occurrence: Claude Code 2.1.303 on
  2026-10-19, approved, every check PASS (P303a to P303d in
  `doc/research/probes/README.md`); it found that plan 14's shape check
  counted an interrupted tool use as unrecognised, fixed the same day at the
  owner's request (P303e).

## Nice to have

Items tagged **[Nice to have]** are future improvements: none is built, and
none is listed as remaining, until the owner asks for one in a new plan
(the owner, 2026-10-18). Amendment 4 names ten "future improvements"
(`doc/brief.md`, Amendment 4, "For this phase, do NOT implement"). Five
remain below. Recent projects, favourites and a project browser were plan 15
(amendment 9); git worktree management was plan 16 (amendment 10). The
automatic VS Code integration was replaced by the owner with a file viewer in
the terminal, plan 17 (amendment 11). The session's model, effort and
Ultracode selection was plan 34 (amendment 34); configuring models beyond the
session stays below.

- **[Nice to have]** Advanced model configuration.
- **[Nice to have]** Cross-provider agent collaboration.
- **[Nice to have]** Provider switching during an active session.
- **[Nice to have]** Complex session profiles.
- **[Nice to have]** Advanced orchestration configuration.
- **[Nice to have]** Google Antigravity as an agent CLI, removed from the
  MVP by amendment 5; the provider interface stays open to it.
- **[Nice to have]** Starting it like an app (the owner, 2026-10-19): a built
  `agent-command-center.exe` started from a Start-menu shortcut, pinnable to
  the taskbar, whose target is `wt.exe -d <projects folder> <path to the
  .exe>`, so that it always opens in Windows Terminal (the only supported
  terminal, amendments 6 and 7) with the project field at that folder; the
  `.exe` is rebuilt after each code change.

## Plan 49 - The rest of phase 0, and phases 3 and 4 of the audit

Brief amendment 50 (2026-10-25): the owner asks for the three phase 0
safeguards still missing, then phase 3 (performance) and phase 4
(maintainability) of the audit's roadmap ([`doc/audit.md`](doc/audit.md),
Part 1, E, and Part 2, C), all of them, though phase 3 was conditional and
phase 4 optional:
- phase 0: the event queue's tests, the write queue's, the overlay matrix;
- phase 3: A, the tool rows' parsing memoised; B, the per-frame scans that
  grow with the session bounded; C, both measured by the timing step;
- phase 4: A, one owner of the overlays; B, the adapters' shared
  lifecycle; C, the git tests in parallel; D, normalised approval
  decisions; E, grapheme-aware widths; F, GIF first-frame decoding.

The work comes in five batches, each checked before the next: the
safeguards first, which pin today's behaviour; then phase 3; then phase
4's A and B, its C and D, its E and F. A change that fixes a defect a
safeguard finds is test-first, as in plan 48. The criteria below were
written before the code. Nothing is committed until the owner has
reviewed the result, and nothing is pushed without the owner's word.

### What was found (2026-10-25, on ecb87bb)

- **The queues** (audit finding 12).
  - `pipe.Queue` (`internal/provider/pipe/pipe.go`) is tested by nothing
    but `TestReadLine`: not its order across producers, its holds, its
    close, or a reader that does not read.
  - The application's write queue (`internal/app/writer.go`: 64 jobs, one
    worker) has no test of its order, of its refusal when full ("too many
    writes waiting"), or of the close behind it.
- **The overlays** (finding 4).
  - Seven overlays are drawn under conditions written out one by one
    (`render/select.go:166-213`), and they disagree: the commit dropdown's
    omits the windows, the file window's the draft window.
  - The keys (`app/keys.go:37-80`, `:169-228`), the mouse
    (`app/mouse.go:45-245`), `motionFilter` (`:284`) and `View`
    (`app/app.go:373-396`) each encode their own precedence. Each opener
    closes its own list of the others (`models.go`, `modes.go`, the "+"
    band's click, the sidebar's).
  - Only the two selectors close when the frame could not draw them.
- **The per-frame work** (finding 7). The timing step on today's code
  (`scratchpad\plan49\timing-before.txt`) has every test within 16 ms, and:
  - `TestLongSession`: its frames' p95 is 4.26 ms in the session's first
    tenth and 14.94 ms in its last, 3.5 times as much; at its end, p50
    7.17 ms and p95 11.62 ms. Run on its own: 2.2 then 4.5 ms, and an end
    p95 of 5.2 ms.
  - `TestThinkingFrame` (5,522 entries, 721 of them thinking): p95 8.14 ms.
  - A profile of the long session's frames:
    - zone A takes 52 %: it builds the rows of all 240 turns, then keeps
      those in view; `indexItems` alone takes 27 %, a pass over the 6,960
      tools and the 720 agents with a sort;
    - zone B takes 32 %: `lookAll` and the row counts, over the 5,281
      entries, each copied;
    - zone C takes 12 %: `currentTools`, a pass over every tool.
  - Each tool row parses its tool's input JSON two or three times a frame
    (`render/tree.go:287-290`, `:411-436`).
  - `Animating` and `ActivityOn` scan the whole conversation each frame
    (`render/thinking.go:942`, `:1081`); `agentTools` scans every tool
    (`inspect.go:201`).
- **The adapters** (finding 9). `claude/session.go` and
  `codex/session.go` repeat the same constants, spawn, readers, wait and
  drain, write framing, `apart`, `fatal`, `Close` and escalation. They
  differ in the CLI's name and in what a line means.
- **The git tests** (finding 12). `go test ./internal/input` takes 148 s.
  - 21 git tests, all sequential, take 143 s of it; the longest takes 32 s.
  - Each sets git's environment with `t.Setenv` (`gitEnv`), which forbids
    `t.Parallel`.
- **The decisions** (finding 11). The CLIs' words travel through the
  whole program:
  - the keys choose "accept" or "allow" by the provider
    (`app/keys.go:487-497`);
  - the reducer lists all six words (`state/actions.go:441`), maps the
    CLI's words back (`reduce.go:792-794`), and denies an unreadable
    request with "decline" for Codex (`reduce.go:755`);
  - render compares with "acceptForSession" (`approval.go:56`,
    `viewer.go:43`).
- **Widths** (finding 5).
  - `textWidth` and `cut` add up code points (`render/text.go:232-276`).
    Bubble Tea's cell buffer measures grapheme clusters, since Windows
    Terminal reports mode 2027.
  - For ❤️, 👍🏽, 👨‍👩‍👦 and 🇫🇷 the two disagree: 1, 4, 6 and 4 cells
    against 2 each.
  - No golden frame and no mock file holds such a sequence.
- **GIF** (Part 2, C). `inspectGIF` decodes every frame (`gif.DecodeAll`,
  `input/image.go:361`) to keep the first: an animated GIF's frames are
  all decoded and held at once.

### Design

- **D1. The event queue pinned** (phase 0). New tests of `pipe.Queue`, in
  a bubble (`testing/synctest`), where a deadlock fails the test:
  - eight producers pushing at once: every event leaves once, each
    producer's in its order, `Seq` from 1 with no gap;
  - a hold keeps the events after it waiting; released, its events leave
    in their place; dropped, they never leave and leave no gap;
  - a close lets the queued events leave, then closes `Out`, and a push
    after it is dropped; a hold not yet released keeps `Out` open until
    its release, which every write's path makes;
  - a reader that does not read: 100,000 pushes return, then all are read
    in order;
  - in the application, a session's events all reach the reducer, in
    order, before its end.
- **D2. The write queue pinned** (phase 0). New tests in `internal/app`:
  - `c` on an approval card while the CLI holds the answer: the interrupt
    is not written before the answer;
  - the queue full (one write running, 64 waiting): the next is refused at
    once with "too many writes waiting", and the 65 run in order;
  - a write that fails, and one that panics, stop none of those after it;
  - a confirmed quit writes the cancels, the interrupt, then `Close`, in
    that order; with the queue full, `Close` comes at once.
- **D3. The overlay matrix** (phase 0). One table-driven test over ten
  layers: the "+" menu, the model selector, the modes selector, the commit
  dropdown, the staging question, the file viewer, the quick open, the
  draft window, the key overlay and the Worktrees view, over the dashboard
  with its docked panel. For each layer opened as the owner opens it, and
  for each ordered pair (the first open, then the second's own key or
  click), it checks:
  - which layers are open and which is drawn, and that nothing of a closed
    or covered one is drawn;
  - which takes the next key and the next click, and that a closed or
    covered one takes neither;
  - where the focus is after each closing.

  The matrix states the behaviour the layers must have. The cells today's
  code fails are its defects: each is recorded in the progress log, then
  fixed by D7.
- **D4. Tool rows memoised** (3A).
  - A tool's derived fields are computed once for an input: its target,
    and whether that is a path, as `toolTarget`, `toolDetail` and
    `approvalDetail` give them. The key is the input's identity (its first
    byte's address and its length), never its contents, so a new input is
    a new key and an update shows at once.
  - They are held by a new `render.Memo` in the UI, as `WrapCache` holds
    the wrapping: a memo, never a source of what is drawn, so a nil memo
    computes everything with the same frame. What a frame did not use is
    dropped after the next one: the memo holds what two frames draw, at
    most.
- **D5. Scans bounded** (3B). The memo also keeps, for the state it was
  made from:
  - each turn's items (the orchestrator's tools and the top agents, as
    `indexItems` gives them), each agent's tools and its latest call:
    built once, then each new tool added in its place. The reducer lists
    the tools in the order they come (`State.ToolOrder`), and counts the
    changes to an agent's place or name (`State.AgentRev`: an agent
    added, its parent, kind or label set), which rebuild them;
  - the conversation's thinking entries, found as the entries are added:
    `Animating` and `ActivityOn` look at those only.

  Zone A builds only the rows of the window it shows, from a list of the
  tree's rows that says what each row is; `Tree`'s nodes come from the
  same list. Zone B's passes over the entries take them by reference, not
  by copy. Nothing is dropped from the state: the scroll, the inspection,
  the navigation and the folds read the same values.
- **D6. Measured** (3C). The timing step, with `TestToolsToFrame` added to
  it: one open turn of 2,000 tool calls, 30 of them `Write` calls of
  100 KB, while 8 agents run with 50 calls each; each new call is timed
  from Update to its frame.
- **D7. One owner of the overlays** (4A).
  - `render.Layers(s, ui)` gives the layers open, which the frame shows
    and the top one, from one ordered list. From the top: the Worktrees
    view, the key overlay, the draft window, the file window, then at
    most one popup (the "+" menu, the model selector, the modes selector,
    the staging question or the commit dropdown), shown only over the
    dashboard and under no window.
  - `Draw`, the key routing, the mouse routing, `motionFilter` and the
    mouse mode take the layers from it; each layer's handlers are
    unchanged.
  - Opening a popup closes the others (`openPopup`). A popup the frame
    does not show closes, as the selectors already do: a window opened
    over the commit dropdown or the staging question closes them, as a
    click outside does.
  - The popups' anchors are recorded by the draws that make them (the
    editor's rule and its band), not found again by their glyphs.
  - A new panel or popup is one entry in the list, plus its handlers.
- **D8. The adapters' shared lifecycle** (4B).
  - `pipe.Proc` holds what both sessions do the same way: the event queue
    and the lock that orders it, the spawn's first event, the stdout and
    stderr readers with their limit and their recovery, the wait that
    reports the exit last, the line write with its hold, `Apart`,
    `Fatal`, and `Close` with its escalation. Every message keeps its
    words, the CLI's name in them.
  - Each adapter keeps its protocol: its arguments and environment, its
    mapper, what a line calls for, the handshake's hold, its answers and
    its requests.
- **D9. The git tests in parallel** (4C). The package's `TestMain` sets
  git's environment once for every test: an empty global configuration,
  no system one, a test identity. The git tests, each on its own temporary
  repository, then call `t.Parallel`. Those that change the environment
  (`PATH`, the proxy variables) or the package's variables stay
  sequential. No git package of its own is needed.
- **D10. Normalised decisions** (4D).
  - `event.Decision` holds the owner's answers (allow, allow for the
    session, deny, cancel), plus withdrawn, for a request the CLI ended
    itself.
  - The keys, the reducer and render use only these. Each adapter writes
    them in its CLI's words, exactly as today: Claude Code's allow and deny
    (a cancel is a deny), Codex's accept, acceptForSession, decline and
    cancel. Each mapper reads its CLI's words back into them; a word it
    does not know stays as it is, which the reducer keeps, as today, as an
    expiry.
  - The mode that the approvals command carries moves to a field of its
    own.
  - The mock files are written again from the probes (`TestMockFiles
    -update`): only their decisions change.
- **D11. Grapheme widths** (4E).
  - `textWidth`, `cut`, the wrap, the editor's and the content's cells and
    the hit-testing's units take grapheme clusters. A combining mark, a
    variation selector, a zero-width joiner and what it joins, an emoji
    modifier, a flag's second regional indicator, a tag sequence, a keycap
    and Hangul jamo stay with their base.
  - A cluster's width is lipgloss's, cached; a single code point's is
    today's.
  - The segmentation is the dashboard's own, from Go's Unicode tables, so
    no dependency is added. Where it is unsure it keeps more together,
    never less, so a width is never wrong.
  - Text all in ASCII takes a path of its own, faster than today's.
  - The editor's cursor still moves by code point; inside a cluster, it
    is drawn on the cluster.
- **D12. GIF's first frame** (4F).
  - `gif.Decode` decodes the first frame only. The rest of the file is
    walked by its blocks, without decoding them, to tell one frame from
    several and to check the structure, as `DecodeAll` did.
  - What is sent stays byte for byte: a single frame's file as it is, an
    animated GIF's first frame as PNG, said so.
  - A later frame's damaged pixel data, never used, is no longer a
    refusal.

### Criteria

1. **49.1 The event queue.** The D1 tests pass on today's code and again
   at the end. A mutant of `Forward` that skips the hold, and one that
   numbers the dropped events, each fail them.
2. **49.2 The write queue.** The D2 tests pass on today's code and at the
   end. A mutant that runs each write in its own goroutine fails the order
   test.
3. **49.3 The overlay matrix.**
   - It covers the ten layers alone, and each ordered pair the owner can
     reach.
   - On today's code it fails only in the cells the progress log records as
     defects.
   - At the end it passes whole.
4. **49.4 Tool rows memoised.**
   - A frame of 30 `Write` calls with 100 KB inputs, drawn again with
     nothing changed, parses no input (the memo counts its misses).
   - A call's new input or status shows in the next frame.
   - Two frames after a call left the screen, the memo no longer holds it.
5. **49.5 The memos change no frame.**
   - Every mock file and probe replay, and the long session, drawn after
     each event with the memo and without it: the same frames.
   - The 220 golden frames unchanged.
6. **49.6 Scans bounded.**
   - Once built, the memo's index takes in a tool added to the long
     session's state by visiting that tool alone (it counts its visits).
   - `Animating` and `ActivityOn` visit no entry but the thinking ones.
   - Zone A draws only its window's rows.
7. **49.7 Performance.** The timing step, run alone as
   `doc/evaluation.md` says, before and after, on this machine:
   - `TestLongSession`: the last tenth's p95 at most 1.5 times the first
     tenth's (today 3.5 times); the end's p95 at most half of today's, in
     the same conditions;
   - `TestThinkingFrame`: p95 at most half of today's 8.14 ms;
   - `TestToolsToFrame`: p95 at most 16 ms, measured on today's code first;
   - every other test within its target, none slower than today's beyond
     the step's noise (the old tree timed in turn when one looks slower).
8. **49.8 One owner of the overlays.**
   - `Draw`, the keys, the mouse, `motionFilter` and the mouse mode read
     the layers from `render.Layers` only; no other place lists the
     overlays' precedence.
   - The overlay matrix passes, its defects fixed. The key and mouse suites
     and the golden frames are unchanged.
9. **49.9 The shared lifecycle.**
   - Both adapters' sessions run on `pipe.Proc`, and the code it replaces
     is gone from both.
   - Every adapter test passes unchanged, and the probe replays give the
     same events. Each message the lifecycle writes is word for word
     today's.
   - `pipe.Proc`'s own tests: the exit is the last event; a reader still
     holding the pipes is given up after 2 s, with its message; `Close`
     escalates at its steps.
10. **49.10 The git tests.**
    - `go test ./internal/input -count=1` within 60 s (today 148 s), every
      test passing.
    - It passes three times in a row (`-count=3`), and under `-race`.
    - No git test touches a repository outside its temporary folders.
11. **49.11 Normalised decisions.**
    - No CLI word for a decision is left in `internal/state`,
      `internal/app` or `internal/render`.
    - For each decision and each CLI, the line written is byte for byte
      today's: a table test over both adapters' `Answer`, run first on
      today's code.
    - The approval tests pass unchanged and the golden frames are
      unchanged; the mock files differ in their decisions only.
12. **49.12 Grapheme widths.**
    - For combining marks, VS15 and VS16, a skin tone, ZWJ sequences,
      flags, a tag sequence, a keycap, Hangul jamo, an Indic conjunct and
      CJK: `textWidth` equals `lipgloss.Width`, and `cut` never splits a
      cluster and keeps the most that fits. `FuzzTextWidth` runs 60 s
      without either disagreeing.
    - A conversation, a tool row, a file and a commit message holding
      them: every row of the frame is exactly the screen's width by
      `lipgloss.Width`. On today's code, the rows with a ❤️, a 👍🏽, a
      👨‍👩‍👦 or a 🇫🇷 are not.
    - The golden frames unchanged; the keystroke and stream timing tests
      within their targets.
13. **49.13 GIF's first frame.**
    - The test GIFs send the same bytes as today: the single frame's file,
      and the animated one's first frame as PNG.
    - A 200-frame 640x480 animated GIF is inspected with under 8 MiB
      allocated (today's measured first).
    - A file cut after its first frame, or holding an unknown block, is
      refused, as today.
    - `FuzzInspectData` runs 60 s without a crash.
14. **49.14 No regression.**
    - Build, vet and gofmt clean; `go test ./...` passes after the last
      document edit.
    - The golden frames and the probe replays unchanged; the mock files
      changed by D10 only.
    - `-race` passes on input, app, state, render, the providers and pipe.
15. **49.15 Mutations.** Each change is broken on purpose through `go test
    -overlay`, and each mutant is caught.
16. **49.16 Windows Terminal.** At 208x59 and 120x40, on a temporary
    project and repository:
    - each popup and window opened in turn, and over each other as the
      matrix pairs them: the frames as before;
    - a mock replay holding ❤️, 👍🏽, 👨‍👩‍👦 and 🇫🇷 in a message: zone B's
      border straight on every row;
    - an animated GIF attached: its first frame previewed, and named
      "animated: the first frame is sent".

### Steps

1. The plan and brief amendment 50.
2. Phase 0: D1, D2, D3 (49.1 to 49.3).
3. Phase 3: D4 to D6 (49.4 to 49.7).
4. Phase 4: D7 and D8 (49.8, 49.9); D9 and D10 (49.10, 49.11); D11 and
   D12 (49.12, 49.13).
5. The checks (49.14, 49.15), Windows Terminal (49.16), the documents, the
   evaluation (`doc/plan49-evaluation.md`), the report.

### Progress

- 2026-10-25: step 1 done; step 2 in progress.
- 2026-10-25: step 2 done (phase 0, no production change); step 3 in
  progress.
  - D1: `pipe/queue_test.go`, five tests in bubbles (eight producers, the
    holds, the close, a hold across the close, 100,000 events unread), and
    `TestSessionEventsInOrder` (2,048 deltas through the listen, then the
    end). All pass on today's code.
    - The last one first failed: the test fed the session's channel after
      `readyScript`, whose `pushEvents` sends an `eventsMsg` itself, and
      every `eventsMsg` arms a listen, so two listens shared the channel.
      In the program only the listen makes them. The test now sends the
      handshake through the channel too.
  - D2: `writer_test.go`, five tests (c's cancel then interrupt, the quit's
    order then its close, the full queue, a failure and a panic, the close
    at once when full). The scripted session's `Answer` can be held
    (`hold`) and logs its calls in order. All pass on today's code.
  - Mutants through `-overlay` (`scratchpad\plan49\mutate49.mjs`): 8 of 8
    caught (`Forward` past a hold, dropped events numbered, a push after
    the close kept, the close dropping the queue, a dropped hold
    released, each write in its own goroutine, a full queue waiting, the
    close queued behind a full queue).
  - D3: `layers_test.go`, 100 cells: the ten layers alone and the 90
    ordered pairs, at 200x60 with Source Control docked, a frame drawn
    after every message as the program does. Recorded first
    (`ACC_LAYERS_RECORD`, `scratchpad\plan49\layers-today.txt`), each cell
    reviewed. 95 hold the approved behaviour; 5 are defects, kept as
    expected failures in `layerDefects` until D7:
    - the commit dropdown, and the staging question, stay open under the
      Worktrees view (Ctrl+W) and are back after it;
    - the staging question stays open under the key overlay (F1), and
      takes the next click there;
    - with the model or the modes selector opened over the staging
      question (Alt+P, F4), the next click closes the hidden question and
      not the selector on top.
  - The matrix's 100 cells take about 13 s and run no faster side by side:
    `runeWidth` takes one mutex for every character past U+02FF, which
    D11 makes lock-free.
- 2026-10-25: step 3 (phase 3) in progress.
  - `TestToolsToFrame` (D6) was written first and run on a clean copy of
    today's code (`git archive HEAD`, in the scratchpad): p50 52.97 ms,
    p95 61.12 ms, max 70.47 ms, over the 16 ms target. Each frame parsed
    the 30 inputs of 100 KB twice for the tree's rows.
  - D4 and D5: `render.Memo` (`memo.go`), held in `UI.Memo` and made by the
    application, swept at each frame's end as `WrapCache` is.
    - It reads an input once, keyed by its identity: tree rows, zone B's
      call rows, answered approvals, the inspection's calls.
    - It keeps an index of the turns' items, the agents' calls and their
      children. The reducer adds `ToolOrder` and `ParentRev`; `ParentRev`
      moves only when an agent known before changes its parent, one check
      at the end of the agent handler. Agents and calls added since are put
      in their places; `indexItems`, `agentTools`, `children` and
      `currentTools`' scan of every call are now used only without a memo.
    - It finds the thinking entries as they come; `Animating` and
      `ActivityOn` walk those only.
    - Zone A's rows carry what they draw (`treeRow.kind`); a row's line is
      made by `treeRow.line` when zone A shows it, and `Tree` makes none.
      The moving dot reads each row's first span as it would be drawn
      (`lead`), so it picks the same rows as before without drawing them.
    - Zone B's passes take entries by reference, and the row count is
      looked up without a closure holding a copy of each entry (it was
      made for every entry, every frame).
  - Measured on its own after D4 and D5: `TestToolsToFrame` p50 1.00 ms,
    p95 2.01 ms; `TestLongSession`'s last tenth's p95 1.01 to 1.14 times
    the first tenth's (before about 2), its end's p95 2.0 to 2.4 ms
    (before 5.1 to 5.3); `TestThinkingFrame` p95 5.0 to 5.3 ms. The whole
    step's numbers come at step 5, after D11's widths.
  - Render draws on one goroutine: `ground.on` makes tinted styles on first
    use, writing shared maps. The new tests that drew in parallel subtests
    (the memo's comparison, the matrix) crashed or could, so both run their
    cells one after the other; the comparison draws every event at
    200x60, and every third and the last also unfolded and at 80x24.
  - Tests: `memo_test.go`. `TestMemoChangesNoFrame` replays the 13 mock
    files and 35 recordings (5,545 events) and finds the same frames with
    the memo and without it. `TestMemoInputs`, `TestMemoIndex`,
    `TestMemoThinking` and `TestTreeRowsLazy` cover criteria 49.4 and
    49.6. The golden frames are unchanged.
  - The whole suite passed after D4 to D6.
- 2026-10-25: step 4 (phase 4) in progress; D7 done.
  - `render/layers.go`: `Layer`, `Layers(s, ui)` and its `Stack` (the
    layers shown, the top one, the tooltip). `Draw` takes the full-screen
    views and the overlays from it, and draws the frame with a popup not
    shown closed (`Stack.only`), so nothing of it, hint or image, is drawn.
  - `app/layers.go`: `closePopups`, called by every opener (the "+" band,
    both selectors, the commit dropdown and the question, the draft
    window, the file viewer, the quick open, the key overlay, the
    Worktrees view); `closeUnshown`, the selectors' closing in `View`
    made general (the "+" menu and the dropdown now say when they were
    drawn).
  - The keys go to the top layer first; a key it passes on goes to the
    dashboard, never to a window under it: the key overlay passed Ctrl+R
    and F to the hidden draft window before (`TestPassedKeyGoesToTheDashboard`,
    which fails on today's code). The mouse, `motionFilter`, the mouse
    mode, the sidebar's hover and the paste take the top layer too; a
    paste under the key overlay, which has no field, goes nowhere (it went
    into the hidden draft or window).
  - The editor's rule is recorded where zone B draws it (`Screen.rule`);
    `editorRule` no longer looks for its dashes. The dropdown's room check
    still reads the panel's border, a check of fit rather than an anchor.
  - The matrix passes whole, its five defects fixed; on a clean copy of
    today's code it still fails in exactly those five. The app suite,
    with its golden frames and key and mouse tests, passes unchanged.
- 2026-10-25: D8 done.
  - `pipe/proc.go`: `Proc`, made with `New` and started with `Run` once
    the session holds it (its reader calls the session's `line`). It holds
    the queue, session.spawned, both readers with their limit and
    recovery, the wait and its drain, `Write` with the line's hold,
    `Push`, `Problem`, `Apart`, `Fatal`, `Close` and its escalation; its
    messages are the old ones with the CLI's name in them.
  - Both sessions keep their protocol: Claude Code's `line` with the
    handshake's hold and the read after a change, Codex's `line` with its
    replies; `mapOut`, `holdIn`, the answers and the requests unchanged. The
    adapters lost 464 lines and gained 61.
  - Every adapter test passes. The two that reached into the internals now
    moved, `TestReaderPanic` in each, drive the reader through `Proc`, with
    the same assertions. `TestProcEndsLast`, `TestProcDrain`,
    `TestProcCloseEscalates` and `TestProcWrite` test `Proc` on its own,
    in bubbles.
- 2026-10-25: D9 done.
  - `TestMain` sets git's environment once; `gitEnv` only skips without
    git. 17 git tests and the subtests of the five longest call
    `t.Parallel`. Two subtests that changed the process's environment
    became sequential tests of their own, with the same assertions:
    `TestSourceCommitNoIdentity` (it unset git's identity with
    `os.Unsetenv`, given back only by the old per-test `t.Setenv`) and
    `TestSourcePushHidesPassword` (the proxy variables). Sequential still:
    the tests that change `PATH` or the proxy variables, and those that
    lower a package limit (`TestSourceStates`, `TestGitDiff`, the
    Explorer's listing).
  - Measured back to back on a clean copy of today's code and on this one:
    345 s and 197 s for `go test ./internal/input`. The same tests of
    today took 148 s this morning: this machine's git calls vary that
    much with its load. One `git --version` takes about 0.26 s here; 12
    shells running them side by side gain 1.6 times, a Go program running
    init, add and commit from 12 goroutines 2.3 to 2.9 times. So the
    parallel tests bring less than the cores would suggest.
- 2026-10-25: D10 done.
  - `event.Decision`: Allow, AllowSession, Deny, Cancel, Withdrawn. The
    session interface's `Answer`, the state's commands, approvals and
    actions, the keys and render use them; `Command.Mode` carries the
    approvals command's mode, which `Decision` carried.
  - Claude Code's adapter writes Allow as allow and Deny and Cancel as
    deny, and refuses AllowSession, which no request of its offers; it reads
    allow, deny and its withdrawal back. Codex's writes accept,
    acceptForSession, decline and cancel, and reads them back, with the
    older approved, denied and abort the reducer used to take; an
    unknown word stays as it is and expires the approval, as before. The
    auto-review verdict's accept and reject are read as Allow and Deny.
  - `TestAnswerLines` in each adapter, run first on the clean copy of
    today's code with today's words: its 9 lines (5 for Claude Code, 4 for
    Codex) are written byte for byte the same by the new code.
  - The four Codex mock files, written again, differ in their decisions
    only (accept becomes allow, resolved withdrawn); Claude Code's already
    used the dashboard's words. The tests that passed Codex's words, or
    compared with them, now use the dashboard's; their wire lines are
    unchanged. Plan 47's `TestUnreadableWordInMock` now expects deny for
    both CLIs, Codex's adapter writing it as decline.
- 2026-10-25: D11 done.
  - `render/cluster.go`: `next` (a cluster's end and width), `joins`
    (UAX #29's rules as the dashboard keeps them), and each code point's
    facts (its width, lipgloss's, and its properties), measured once and
    read without a lock under U+20000. A cluster of several code points
    takes lipgloss's width, kept for 4,096 clusters.
  - `textWidth` counts ASCII runs a byte at a time, steps back one byte
    only before a mark (nothing else joins ASCII), and decodes each other
    code point once. `cut`, a new `tailCut`, the wrap, the cells of the
    editor, the content, the file viewer and the diff, the editor's row
    count, the hit-testing's units, the chips' and the selector's column
    cuts, `cellText`, `MsgAt`, `elide`, `cutRight` and the startup field
    take clusters. The editor's cursor inside a cluster is drawn on it;
    the plan 4 reference test now says so, and its random lines hold
    clusters.
  - Found on the way: lipgloss gives the soft hyphen (U+00AD) and the C1
    controls no cell, where today's `runeWidth` counted one; a single code
    point's width is now lipgloss's everywhere. Go 1.27's tables are
    Unicode 17, which has 20 linkers for GB9c, not 6.
  - Checked against uax29 v2.7.0, the segmenter lipgloss measures with, in
    a scratch program (`scratchpad\plan49\clusters`): every code point's
    width and properties, then 5,000,000 random strings from every class
    of code point: no cluster of uax29's split, no width different. 5% of
    those strings keep two of uax29's clusters together (a joiner before a
    code point that is not a pictograph, a linker before a letter that is
    not a consonant); none of `TestClusterWidths`' texts does.
  - Tests: `TestClusterWidths` (15 kinds of cluster, ASCII around them or
    not; `cut` and `tailCut` keep the most whole clusters), `FuzzTextWidth`
    (60 s, 497,513 inputs, no disagreement with lipgloss), and
    `TestClusterFrames` in the app: a message, a tool row, a file and a
    commit message holding the clusters, every row exactly the screen's
    width at 120x40 and 200x60. On a clean copy of today's code, 26 rows
    are not. The golden frames are unchanged.
  - `BenchmarkTextWidth`, back to back with today's code: 180 bytes of
    ASCII 142 to 194 ns (today 255 to 282), a rule of 200 cells 2.05 to
    2.18 µs (today 3.0 to 3.4), text mixing CJK, emoji and accents the
    same (457 to 481 ns, today 455 to 517). The render suite takes 19 s
    (78 to 106 s before).
- 2026-10-25: D12 done; step 4 done, step 5 in progress.
  - `inspectGIF` decodes the first frame with `gif.Decode`, then
    `gifFrames` (`input/imagegif.go`) walks the file's blocks as
    `DecodeAll` read them, without decoding pixels: it counts the frames
    and refuses what `DecodeAll` refused of the structure, in its words.
  - `TestGIFSent`, run first on the clean copy of today's code: five GIFs
    (imagetest's single frame and animation, three frames of other sizes
    and offsets with disposals and a transparent colour, frames with their
    own colour tables and no global one, comment and plain text
    extensions) send the same bytes, by length and SHA-256.
  - `TestGIFRefused`, recorded the same way: cut after the first frame, no
    trailer, an unknown block, an unknown extension, a graphic control's
    size or terminator, a later frame's bounds, code size or data cut:
    refused in today's words. A later frame's damaged pixels, refused
    before ("lzw: invalid code"), are now taken: the first frame is sent.
  - `TestGIFMemory`: a 200-frame 640x480 GIF allocates 2.6 MiB (65.8 MiB
    on today's code).
  - `FuzzInspectData` ran 62 s (811,610 inputs) without a crash. A new
    `FuzzGIFFrames` checks the walk takes every GIF `DecodeAll` reads,
    with as many frames: 61 s, 493,515 inputs, none.
- 2026-10-25: step 5 in progress.
  - The whole suite passed (2 min 28 s); `-race` passed on input, app,
    state, render, the five provider packages and pipe (7 min 36 s).
  - Mutations (`scratchpad\plan49\mutate49.mjs`, 35 mutants, phase 0's
    eight among them): the first run caught 31. Two layer mutants (no
    `closeUnshown`, no `closePopups`) are each caught by the whole app
    suite, not by the matrix alone: each mechanism backs the other up, and
    `TestSelectorNoRoom` and `TestCommitWorkflowB` pin them. Two showed test
    gaps: `TestTreeRowsLazy` drew no call row (every turn folded), so it now
    unfolds them too; and the cells by code point changed only where rows
    break, not their widths, so `TestClusterCells` checks the cells, the
    wrap and the editor's row count. Both caught after. No mutant code
    left in the tree.
  - Windows Terminal 1.23.20211.0 (`scratchpad\term\p49.ps1`, `p49call.ps1`):
    - one kind of cluster a paragraph (`mkmock49b.mjs`), maximised: on
      today's build zone B's right border breaks on the rows of the VS16
      heart, the skin tone, the family, the flag, the keycap, the jamo
      and the rainbow flag; on the new build it is straight on every row
      but the Devanagari conjunct's (क्ष), which the terminal draws two
      cells wide where lipgloss, and Bubble Tea's renderer with it,
      counts one;
    - each layer opened in turn and the former defects' pairs, both
      sizes, both builds (`claude-thinking.jsonl`, a temporary repository
      from `mkrepo49.mjs`): the frames are the same but where the defects
      were fixed (today's build brings the staging question back after the
      key overlay), and the 200-frame GIF attached through "Upload from
      computer" shows its first frame and the toast "anim.gif: animated:
      the first frame is sent" (the chip "anim.gif · 6 KB").
  - The input package alone: 79.1 s with the CPU 23% busy before, and
    three runs in a row pass (275 s). Its floor is the sequential tests,
    24.5 s, then the parallel phase, which the number of git processes
    Windows starts bounds: `TestSourceConflicts` takes 5.8 s alone and
    38 to 64 s in the package. Splitting the two longest tests' cases into
    parallel subtests made the package slower (100 s), so they are as D9
    left them; criterion 49.10's 60 s is not reached. Back to back with
    today's code on a busier machine: 188.6 s today, 114.2 s now.
  - The timing step alone on both trees: every test within its target;
    `TestToolsToFrame` p95 42.19 ms today, 2.40 ms now; `TestThinkingFrame`
    2.55 ms (half of 8.14 ms is 4.07); `TestLongSession`'s last tenth 0.95
    times its first, but its end's p95 3.00 ms against today's 5.42 ms,
    and over eight runs in turn medians of 2.62 and 5.06 ms: 52%, not the
    half criterion 49.7 asked. What remains is zone B's passes over every
    entry each frame (the looks and the row counts, finding 7's
    `look.go`), which D5 made take entries by reference without bounding.
  - The five fuzz targets, 60 s each with the evaluation guide's flags:
    none failed (`FuzzMapperOut` in both adapters too, D10 having changed
    their mappers).
  - Windows Terminal, two more passes: the Explorer, the quick open and the
    viewer on a file of clusters; the staging question under the Worktrees
    view and under each selector: today's build shows those defects in the
    terminal, the new build does not.
- 2026-10-25: step 5 done. `doc/plan49-evaluation.md`: 11 PASS, 5 FAIL
  (49.7 and 49.10 short of their targets; 49.8, 49.9 and 49.11 on the word
  "unchanged" about tests that changed how they call the code, not what
  they expect). Waiting for the owner's review; nothing committed.

## Closed plans

One entry each: what the plan did, its approval, its evaluation file and
its commits. Their criteria and progress logs: `git show 8888f59:plan.md`
(plans 1 to 13), `git show 84d2ad8:plan.md` (plan 14), `git show
b047434:plan.md` (plan 15), `git show becfb34:plan.md` (plan 16), `git show
f00c2d8:plan.md` (plan 17), `git show 8b3b16b:plan.md` (plan 18), `git show
d84e5d4:plan.md` (plan 20), `git show f939d96:plan.md` (plan 21), `git show
d13ef71:plan.md` (plan 22), `git show e8d3e3b:plan.md` (plan 23), `git show
a3bfe5d:plan.md` (plan 24), `git show cb8f76d:plan.md` (plan 25), `git show
a9a5145:plan.md` (plan 26).

## Plan 1 - Research

The agent CLIs and how to drive them, telemetry, terminal input, the
reference projects, and the decision to build rather than fork
(`doc/research/`). Accepted on 2026-10-16, 9 PASS and criteria 8 and 9 kept as
FAIL, documented exceptions
([`doc/research/plan1-evaluation.md`](doc/research/plan1-evaluation.md));
commit 7e4d67c.

## Plan 2 - Architecture and reliability

`doc/architecture.md`, `doc/events-and-state.md`, `doc/ui-design.md` and
`doc/reliability.md`. Approved on 2026-10-16, 15 of 15 PASS
([`doc/plan2-evaluation.md`](doc/plan2-evaluation.md)); commit d5b910e.

## Plan 3 - Proposal

The MVP roadmap and the decisions it asked for (stack, dependencies, live
runs, configuration check). Approved on 2026-10-16, 12 of 12 PASS; commit
d5b910e, whose `doc/proposal.md` holds the roadmap. The design documents
and the code have replaced it since.

## Plan 4 - MVP (from amendment 4; steps detailed by plan 3)

The MVP, steps 0 to 8: startup, Claude Code, the streaming conversation,
approvals and resumption, Codex, agent activity, telemetry, the UI.
Approved on 2026-10-17 after a live demo on both CLIs, 14 of 14 PASS
([`doc/plan4-evaluation.md`](doc/plan4-evaluation.md)); commits ded53d8 to
cf58739.

## Plan 5 - Post-MVP improvements

The list plan 4 left open, Windows only. Approved on 2026-10-17, 17 of 17
PASS, then 15 of 15 live and 6 of 6 for the follow-up
([`doc/plan5-evaluation.md`](doc/plan5-evaluation.md)); commits d42531f,
897f640, 6813c0e.

## Plan 6 - Plan 5's known limits

Four known limits fixed, then items 5 and 6 added by the owner. Approved on
2026-10-17, 6 of 6 and 8 of 8 PASS
([`doc/plan6-evaluation.md`](doc/plan6-evaluation.md)); commits df273fe,
0814d67.

## Plan 7 - The open list after plan 6

The open list cleaned. Approved on 2026-10-17, 7 of 7 PASS
([`doc/plan7-evaluation.md`](doc/plan7-evaluation.md)); commit bffe9a5.

## Plan 8 - Codex's allow for the session

`acceptForSession` on Codex 0.163.0. Approved on 2026-10-17, 6 of 6 PASS
and its follow-up 3 of 3 ([`doc/plan8-evaluation.md`](doc/plan8-evaluation.md));
commits e75d591, 1daeec1.

## Plan 9 - Plan 8's items left not verified

Two items verified, the third recorded as not verifiable. Approved on
2026-10-18, 6 of 6 PASS
([`doc/plan9-evaluation.md`](doc/plan9-evaluation.md)); commits
c9576d9, 8dea129.

## Plan 10 - A spawned Codex agent's label and the timing step

Both items verified. Approved on 2026-10-18, 7 of 7 PASS
([`doc/plan10-evaluation.md`](doc/plan10-evaluation.md)); commits b71c114,
621609e.

## Plan 11 - A live approval offering the allow for the session

The live approval closed as not verifiable, by the owner's choice.
Approved on 2026-10-18, 3 of 3 PASS
([`doc/plan11-evaluation.md`](doc/plan11-evaluation.md)); commits 81f3bf8,
cb206b6.

## Plan 12 - Manual or Auto approvals

Brief amendment 8: a per-session switch between Manual and each CLI's own
Auto. Approved on 2026-10-18, 18 of 19 PASS and 1 FAIL accepted
([`doc/plan12-evaluation.md`](doc/plan12-evaluation.md)); commits cb31557,
83fa023.

## Plan 13 - A reviewer's refusal live

Claude Code's classifier denial seen live and a render defect it exposed
fixed; Codex's reviewer not reached, closed by the owner. Approved on
2026-10-18, 14 PASS and 3 not observed
([`doc/plan13-evaluation.md`](doc/plan13-evaluation.md));
commits d68bb55, 8888f59.

## Plan 14 - Nothing left open

Every open question of the research files settled or closed by the owner,
Claude Code's question card built and seen live, the documents reduced to
the latest state and a CLI update made visible. Approved on 2026-10-18, 18
PASS and 15 PASS ([`doc/plan14-evaluation.md`](doc/plan14-evaluation.md));
commits 0206ac1, ce06b97.

## Plan 15 - Recent projects, favourites and a project browser

Brief amendment 9: the startup screen's project list (`projects.json`,
favourites first) and its Ctrl+O browser. Approved on 2026-10-18, 21 PASS
([`doc/plan15-evaluation.md`](doc/plan15-evaluation.md)); commits bd9ffa3,
b047434.

## Plan 16 - Git worktree management

Brief amendment 10: the Worktrees view (Ctrl+W, at startup and in session)
with create, list, diff, merge and remove, git through `os/exec`. Approved
on 2026-10-18, 26 PASS
([`doc/plan16-evaluation.md`](doc/plan16-evaluation.md)); commits ce95ba2,
fad6d20, 642c5b9, becfb34.

## Plan 17 - A file viewer in the terminal

Brief amendments 11, 12 and 13: a file viewer in the terminal with Chroma's
highlighting, an Explorer of the project (F3) and the draft in a window
(Ctrl+G), with no external editor. Approved on 2026-10-19, 91 PASS
([`doc/plan17-evaluation.md`](doc/plan17-evaluation.md)); commits 1748808,
10c3e74, 904f9a6, 749cef4.

## Plan 18 - An adaptive layout

Brief amendment 14: zone C sized by the agents in play, a strip of 4 rows (3
under 100 columns) when there are none, and cleared at each new prompt.
Approved on 2026-10-19, 22 PASS
([`doc/plan18-evaluation.md`](doc/plan18-evaluation.md)); commit 8b3b16b.

## Plan 19 - Zone B as the centerpiece

Brief amendment 15: one speaker label per exchange, RESULT over an
exchange's outcome, compact tool rows and folds of long runs of calls, each
entry's turn (`Entry.Stage`). Done on 2026-10-20, 19 PASS
([`doc/plan19-evaluation.md`](doc/plan19-evaluation.md)). The owner
reviewed it in a demo the same day and asked for plan 20 (amendment 16),
which keeps its logic and replaces its look; not committed on its own, it
was committed with plan 20 once the owner approved that, on 2026-10-20;
commit d84e5d4.

## Plan 20 - Strict conformance to the approved templates

Brief amendments 16 and 17: the owner's two templates as one design
system, A and B in one upper row and C full width under them, zone B's
messages as blocks with a rule between them, the tool rows, the folds, the
pending approval's pink box and RESULT. Approved on 2026-10-20 after the
owner's live review, 32 of 32 PASS
([`doc/plan20-evaluation.md`](doc/plan20-evaluation.md)); commit d84e5d4,
with plan 19.

## Plan 21 - What the review of d84e5d4 found

A read-only review of commit d84e5d4: zone D's session time past 2,000
events, two timed tests outside the timing step, Claude Code's
undocumented `api_message_id` labelled with a fallback, zone C's floor at
160x24 to 160x26, and documents promising what the code did not do (the
send and newline keys built in `config.json`, the others corrected).
Approved on 2026-10-20 with the recommended choices, 38 PASS and 2 FAIL,
then the long session's frame time fixed at the owner's request, 6 PASS
([`doc/plan21-evaluation.md`](doc/plan21-evaluation.md)); commits bdc4250,
f939d96.

## Plan 22 - A live thinking indicator

Brief amendments 18 to 20: zone B's live thinking card for Claude Code and
Codex, with its animations, a summary row for each thinking phase that
opens on its details, and an activity line while a turn works; corrections
C1 to C12 from a read-only investigation
([`doc/plan22-investigation.md`](doc/plan22-investigation.md)); Codex
started through the local proxy that `config.json` names (amendment 19);
Codex's reasoning summaries asked for and Claude Code's hook events known
(amendment 20), checked live with one Codex turn (L20). Approved on
2026-10-21: 31 of 31 PASS, then the corrections 17 of 18 with 5.14 fixed,
amendment 20 13 of 13, and L20 8 of 8 once two criteria were corrected
([`doc/plan22-evaluation.md`](doc/plan22-evaluation.md)); commits
e6895c6, 6306bfb and d13ef71, then the approval commit.

## Plan 23 - The acc launcher in the project

Brief amendment 21: `acc`, typed in Windows Terminal, opens the dashboard
in a tab of its own at the owner's size, a 9 pt font. `acc.cmd` at the root
runs `scripts\acc.ps1`, which writes a Terminal fragment for the profile
and makes Terminal reload. The day's machine-local changes were undone.
Approved on 2026-10-22, 6 of 6 PASS before the commit, then committed and
pushed ([`doc/plan23-evaluation.md`](doc/plan23-evaluation.md)); commit
e8d3e3b, then the approval commit.

## Plan 24 - Zone A with the telemetry, zone D removed

Brief amendment 22: zone A redrawn after the owner's image, holding the
orchestrator's dynamic status, the stages as blocks and zone D's telemetry
(session usage, quota windows with lavender bars and countdowns, the notes,
the details view); zone D removed; Fable and Mythos as model families.
Amendment 23, after the owner's live session: the Claude adapter reads the
`/usage` report after the handshake and after each turn, so the
model-scoped "Fable limit" shows live, the exchange kept out of the
conversation. Approved on 2026-10-22: 35 of 35 PASS with the conformance
table 22 rows PASS, amendment 23 6 of 6 and its live check 8 of 8
([`doc/plan24-evaluation.md`](doc/plan24-evaluation.md)); commits 6209267
and a3bfe5d, then the approval commit.

## Plan 25 - The unrecognised lines named

Brief amendment 24: the owner's scenario 1 run on Fable 5.1 ended with
"unrecognised 2" and no way to see which lines. The details view now lists
the last 20 unrecognised lines with their time, type, reason and text; a
replay tool names them from a logging wrapper's record; the owner's two,
named by a logged rerun, were `system/task_summary`, known with no event
since. Approved on 2026-10-22: 7 of 7 PASS, two live runs
([`doc/plan25-evaluation.md`](doc/plan25-evaluation.md)); commit cb8f76d,
then the approval commit.

## Plan 26 - Zone A without notes

Brief amendment 25: the owner asked to get rid of zone A's notes
("unrecognised 1", "quota updated", "† undocumented field", "cost at API
rates") completely and forever. The telemetry pane now ends with the last
quota window's rows; the counts, ages and legends live in the details view,
where an undocumented value keeps its "†" and the heading explains it; a
standing decision. Approved on 2026-10-22: 6 of 6 PASS, 51 golden frames
reviewed ([`doc/plan26-evaluation.md`](doc/plan26-evaluation.md)); commit
a9a5145, then the approval commit.

## Plan 27 - Zone C with the execution timeline

Brief amendments 26 and 27: the owner's design of zone C and its
close-ups, to respect strictly. Zone C from 100 columns: the header
inside the frame, a rule and the panels' line in the title colour, the
agents' panel as wide as its two-column cards (the label over the clock
and the tokens, a grey bar, the model over the current tool; the selected
agent's rows on a band) and the selected agent's execution timeline with
its status at the heading's right end; the narrow layout keeps plan 18's
row. Approved on 2026-10-22: 15 of 16 PASS, 27.6 corrected with a note
at the owner's decision, 46 golden frames, Windows Terminal screenshots
at 208x59 and 120x40 against the images
([`doc/plan27-evaluation.md`](doc/plan27-evaluation.md)); commit 041e7b8,
then the approval commit.

## Plan 28 - The top right after the owner's image

Brief amendment 28: the owner's close-up of the top right, "● LIVE" over
"Claude Code 2.1.303" over zone B's edge with "Idle". The project bar's
right end is the provider's name and version alone while approvals are
Manual with no change asked; the mode shows there only as Auto or while
changing. Approved on 2026-10-22: 6 of 6 PASS, 27 golden frames, a
Windows Terminal screenshot against the image
([`doc/plan28-evaluation.md`](doc/plan28-evaluation.md)); commit 8fe0077,
then the approval commit.

## Plan 29 - The input editor's row after the owner's image

Brief amendment 29: the owner's image of the editor's row, design only.
Under the draft a separator, then " + " and the pill " <model> High ·
Ultracode " on the tool rows' ground, the editor's messages and the line
count in the free middle, "↯ Auto" at the right end; the "+", "High ·
Ultracode" and "Auto" are placeholders with no function. Zone B is one
row shorter. Approved on 2026-10-22: 7 of 7 PASS, 59 golden frames,
Windows Terminal screenshots at 208x59 and 120x40 against the image
([`doc/plan29-evaluation.md`](doc/plan29-evaluation.md)); commit d696f22,
then the approval commit.

## Plan 30 - The "+" menu after the owner's image

Brief amendment 30: the owner's image of the "+" clicked, design first. A
click on the editor row's band opens a three-row box above the editor's
rule, "🗀  Upload from computer", the band and the item under the
pointer on the active ground (`#272745`), the terminal asked for every
mouse motion only while it is open; Esc, any key or a click elsewhere
closes it; the item does nothing yet, the upload being a later plan's.
Approved on 2026-10-22: 6 of 6 PASS, four golden frames added, Windows
Terminal screenshots with a real click and pointer moves at 208x59 and
120x40 against the image
([`doc/plan30-evaluation.md`](doc/plan30-evaluation.md)); commit 6025513,
then the approval commit.

## Plan 31 - Image upload

Brief amendment 31: the "+" menu's "Upload from computer" works. The
system's Open dialog, several images at once, each checked by its bytes
(PNG, JPEG, GIF, BMP, TIFF, ICO and WebP accepted, the orientation
applied, conversions to PNG, an image over a provider's limits scaled
down and said so; HEIC, AVIF, JPEG XL, JPEG 2000, PSD, TGA, DDS, HDR,
EXR, camera raws and SVG refused by name); chips under the editor's
rule, removable; sent only when the owner submits, as base64 image
blocks to Claude Code and `image` inputs holding data URLs to Codex, no
event keeping the bytes (Codex writes them back). Approved on
2026-10-23: 9 of 9 PASS, eight golden frames added, the real dialog in
Windows Terminal, live turns on Claude Code (Haiku 5.5) and Codex
(gpt-6-sol) naming the colours and the text, and a third on Codex
checking the echo fix
([`doc/plan31-evaluation.md`](doc/plan31-evaluation.md)); commit 8e2a721,
then the approval commit.

## Plan 32 - Image paste and preview

Brief amendment 32: paste an image as in the owner's chat, and a preview
before sending. Alt+V attaches the clipboard's images (copied files, the
"PNG" format's bytes, else the bitmap as PNG); Ctrl+V pastes text as
before and, when Windows Terminal keeps its press for want of text,
attaches the image on the key's lone release (undocumented, probe P32a);
a paste of the copied files' paths attaches them when all are images;
a pasted image is held in memory only. With Sixel graphics, the images
show as thumbnails between the rule and the chips, rounded squares cut
from their centre, 14x7 cells from 50 rows, 10x5 from 30, side by side
one cell apart (the owner's remark on the screenshots), drawn after the
frame and again after any frame that changes their rows, a vacated
place written over with the frame's cells. Approved on 2026-10-23: 10 of
10 PASS, four golden frames added, Windows Terminal at 208x59 and
120x40 with the owner's clipboard saved and restored, the remark in a
follow-up ([`doc/plan32-evaluation.md`](doc/plan32-evaluation.md));
commit b56d23e, then the approval commit.

## Plan 33 - Unsent drafts and images through crashes and restarts

Brief amendment 33: whether unsent image attachments survive a crash, a
forced termination, a restart and a resumed session; an investigation
first, then its nine corrections. Each dashboard's draft has a folder of
its own, its lock held while the dashboard owns it, taken back by the
project's next start (its own CLI's first, then the newest); images are
saved at once, the text every 500 ms; pasted images are kept as
`<sha256>.png` until their message is accepted; a message waiting for
its acceptance is stored with what was typed since; Back forgets the
attachments; a damaged list is set aside; a changed file is attached
again as it is now; refused renames and failed saves are tried again;
unused files are swept at each start. An independent review found five
defects, fixed; on the owner's questions, a Codex session resumed with
images, a slow and an offline share were simulated, fixed where they
lost data, and a logoff was not simulated. Approved on 2026-10-23: 14 of
14 PASS and the follow-up
([`doc/plan33-evaluation.md`](doc/plan33-evaluation.md)); commit
bc5d92b, then the approval commit.

## Plan 34 - The model selector, its effort and Ultracode

Brief amendment 34: a click on the editor row's model pill, or Alt+P,
opens the active CLI's own models with the effort and, for Claude Code,
Ultracode, after the owner's image. Claude Code's list comes from
`initialize`, its changes go by `set_model` and `apply_flag_settings`
and are read back by the undocumented `get_settings`; Codex's list from
every page of `model/list`, its changes ride the next `turn/start`,
read back by `thread/read`. The screen shows a value only once the CLI
reports it in force; an effort held by `CLAUDE_CODE_EFFORT_LEVEL` is
shown and never changed; Claude Code's changes wait for a turn's end. An
independent review found four defects and six doubts, fixed, and three
small points on its second pass. At the owner's request
`CLAUDE_CODE_EFFORT_LEVEL` was removed from their Claude Code settings,
and the dashboard then set the effort live; amendment 35, an opt-in line
telling the agent its settings, was built and withdrawn. Approved on
2026-10-24: 20 of 21 PASS, 34.18 (golden frames changed beyond the
pill's values) accepted
([`doc/plan34-evaluation.md`](doc/plan34-evaluation.md)); commit
47b50ce, then the approval commit.

## Plan 35 - The Ultracode toggle after the owner's image

Brief amendment 36: the owner's image of the Ultracode toggle, a lavender
pill with fully rounded ends and a round white knob, against the
dashboard's `▐   ●▌`. Where the terminal draws Sixel, the toggle is an
image over blank cells of its row, drawn as the preview's thumbnails are:
a pill of 40x20 pixels in an image of five cells whose last carries the
band, since the renderer writes every cell between a row's first and last
changes and a band cell after the image made it blink when the band came
or went (bursts of captures in Windows Terminal); without Sixel, a text
pill of Powerline's half circles. Approved on 2026-10-24: 8 of 8 PASS
([`doc/plan35-evaluation.md`](doc/plan35-evaluation.md)); commit
b43cf46, then the approval commit.

## Plan 36 - The selector's cursor and a pointer that has not moved

Brief amendment 37: Alt+P could reopen the model selector with its
cursor on the row under a resting pointer, so that Enter acted on that
row. The probe (Bubble Tea's `TEA_TRACE`) showed Windows Terminal
reporting a pointer put down by software, without its moving, once the
selector asked for every motion, its reports wandering a cell. The
selector now moves its cursor only when the pointer comes onto another
row than the last report's, the first report after Alt+P only recording
where it rests. Approved on 2026-10-24: 7 of 7 PASS
([`doc/plan36-evaluation.md`](doc/plan36-evaluation.md)); commit
c0cebc4, then the approval commit.

## Plan 37 - The effort slider after the owner's images

Brief amendment 38: the owner's five images of one pill-shaped effort
slider at five levels replaced the selector's effort dropdown. Effort
reads "Effort (<level>)", "Controls reasoning depth" and the slider: the
model's own levels as dots, a fill of `#6784C1` growing to the thumb,
the whole pill lit by a purple gradient with specks at the highest
level, a level asked for ringed, a held effort greyed; with Sixel an
image of 170x20 pixels over 17 blank cells, its thumb gliding in 180 ms;
without, 16 text cells. Effort and Ultracode share label, description
and control columns; ← → and a click on the slider ask for a level.
Approved on 2026-10-24: 9 of 9 PASS
([`doc/plan37-evaluation.md`](doc/plan37-evaluation.md)); commit
31c22ef, then the approval commit.

## Plan 38 - Instant Codex model and effort changes, as in VS Code

Brief amendment 39: a Codex model or effort change took effect only with
the next message, where VS Code's Codex extension changes it at once. The
dashboard now asks for Codex's experimental API at initialize, as that
extension does, and writes a change between turns at once with
`thread/settings/update`, the `thread/settings/updated` that follows
putting it in force (P38x: 1 to 3 ms); a change asked during a turn is
written at its end; a Codex that refuses the update takes every change
with the next message, as before. Approved on 2026-10-24: 12 of 12 PASS
([`doc/plan38-evaluation.md`](doc/plan38-evaluation.md)); commit
dc9c6d9, then the approval commit.

## Plan 39 - The permission modes selector after the owner's images

Brief amendment 40: the editor row's right-hand "Auto" became the mode
pill of the owner's third image, and opens a "MODES" popup after the
first two: the active CLI's own modes (Claude Code's Manual, Edit
automatically, Plan and Auto; Codex's Ask for approval, Approve for me and
Full access), each applied through the CLI (`set_permission_mode`; the
experimental `thread/settings/update` with the presets of VS Code's Codex
extension) and shown in force only once the CLI reports it; Codex's Full
access only after a confirmation. The icons, the check and the band's
rounded ends are Sixel strips beside the text cells, the pill a Sixel
image whose title Windows' GDI draws. Approved on 2026-10-24: 15 of 15
PASS ([`doc/plan39-evaluation.md`](doc/plan39-evaluation.md)); commit
55e2bed, then the approval commit.

## Plan 40 - No empty rows in zone C's execution timeline

Brief amendment 41: on Codex, zone C's execution timeline and the
inspection's Messages tab showed a subagent's rows "assistant │" with no
text, one for each of its reasoning steps: since plan 22 a subagent's
reasoning summaries leave assistant entries holding only a thinking mark,
which zone B never drew. Neither draws an entry with no text now (a
thinking mark, a message that ended empty, a notice with none), a rule
only between two drawn entries. Approved on 2026-10-24: 6 of 6 PASS
([`doc/plan40-evaluation.md`](doc/plan40-evaluation.md)); commit
4158751, then the approval commit.

## Plan 41 - No authorization on low-risk review rows

Brief amendment 42: a Codex auto-review row rated low risk read
"authorization unknown": Codex's reviewer answers a low-risk action with
a bare allow, and Codex then reports the authorization as unknown with a
stock rationale. After a preview, the owner chose to drop the
authorization from review rows rated low risk, in zone B, zone C's
timeline and the inspection; at medium, high and critical it stays.
Approved on 2026-10-24: 7 of 7 PASS
([`doc/plan41-evaluation.md`](doc/plan41-evaluation.md)); commit
3904f13, then the approval commit.

## Plan 42 - The mouse in zone B: a click to fold, a double click to open

Brief amendment 43: beside the keyboard's path, which the owner approved
and which stays as it is, a click anywhere on a foldable item's row in
zone B (a run of calls, a thinking phase's summary row) unfolds it or
folds it again, and a double click on a row of an entry Enter shows whole
opens its full view as Enter does, an approval's rows the approval whole
as from its card. The mouse puts the entry cursor nowhere and shows none
of the keyboard's hints. On those rows a double click no longer selects a
word; a drag still does. Approved on 2026-10-24: 7 of 7 PASS
([`doc/plan42-evaluation.md`](doc/plan42-evaluation.md)); commit
11f794d, then the approval commit.

## Plan 43 - A right activity sidebar with the Explorer docked

Brief amendment 44: a narrow activity sidebar right of zone B, from 100
columns, with the Explorer's and Skills' icons after the owner's two
images. The Explorer, a floating window since plan 17, is a panel docked
between zone B and the sidebar where zone B keeps 64 columns, and in zone
B's place where it does not and under 100 columns; F3 and the Explorer's
icon show and hide that one panel, the focus with it, F3 keeping its file
meaning. The icons' four looks are Sixel images, their hover from
every motion reported, filtered before Update; Skills is a placeholder.
The review led to a third patch of the vendored ultraviolet: a row
holding a wide character is written only as far as it changed, so the
icons beside it stay. Approved on 2026-10-25: 15 of 15 PASS
([`doc/plan43-evaluation.md`](doc/plan43-evaluation.md)); commit
48bde52, then the approval commit.

## Plan 44 - Source Control and a Git diff viewer

Brief amendment 45: the sidebar's second icon, Skills' placeholder until
then, is Source Control. Its panel shares the Explorer's docked place and
lists the project's Git changes from one git status read, the Explorer's
too: MERGE CHANGES, CHANGES and STAGED CHANGES, each file with its real
status, a file staged and changed again in both. A file opens its diff in
zone B's place, the panel beside it from 118 columns: git's blobs read raw
by cat-file and the working file, compared by a Myers diff of our own,
unified or side by side, with line numbers, green and red rows and
navigation between changes; read only. The engine was built by an
implementer subagent. Approved on 2026-10-25: 14 of 14 PASS
([`doc/plan44-evaluation.md`](doc/plan44-evaluation.md)); commit
d9d7d83, then the approval commit.

## Plan 45 - Commit, Push and Commit & Push in Source Control

Brief amendment 46: under Source Control's repository row, a message
field, a purple "✓ Commit" button with a separator and a chevron, and a
dropdown with one item, Commit & Push, after the owner's image. Commit
takes the index as staged (asking before staging anything), refuses a
staged set other than the one shown and counts once HEAD is a new commit
on the old one; Push goes to the configured upstream only, never forced;
the main action is read from git's branch line. After the first report
the owner asked for two additions: the staged files' contents compared
too, and a long operation's elapsed time with Esc cancelling a push (a
Ctrl+Break to git's own process group, behind a check of its lock files;
probes P45a to P45e). The engine's first version was built by an
implementer subagent. Approved on 2026-10-25: 19 of 19 PASS
([`doc/plan45-evaluation.md`](doc/plan45-evaluation.md)); commit
f71e022, then the approval commit.

## Plan 46 - Three fixes from the audit

Brief amendment 47: the first three fixes of the read-only audit of
2026-10-25 ([`doc/audit.md`](doc/audit.md)), each test-first.
- The worktree merge reads the main worktree first, refuses a merge in
  progress or uncommitted changes, and aborts only its own conflict
  (MERGE_HEAD the branch's tip).
- A refused Codex `config/read` or `model/list` no longer ends the
  handshake (`event.Problem.Optional`).
- An opened untracked folder in Source Control is read again after each
  status read and each opening.

Approved on 2026-10-25: 10 of 10 PASS
([`doc/plan46-evaluation.md`](doc/plan46-evaluation.md)); commit 9375233,
then the approval commit.

## Plan 47 - The rest of the audit's phase 1

Brief amendment 48: the rest of phase 1 of the audit's roadmap
([`doc/audit.md`](doc/audit.md), Part 2, C), each fix test-first.
- Ctrl+Q while a commit, push or staging runs asks to confirm, naming
  it.
- An answer to a request the CLI withdrew first expires the approval and
  the session goes on (`provider.ErrNoPending`).
- The reducer's Mode case returns, so mode events leave no false note,
  and a mock replay of Codex declines in Codex's word.
- The Explorer opened over the staging question drops it.
- `realPath` strips the `\\?\` prefix; two stale comments are corrected.
- `TestStateMachines`, failing since plan 46's commit, is repaired.

Approved on 2026-10-25: 10 of 10 PASS
([`doc/plan47-evaluation.md`](doc/plan47-evaluation.md)); commit d44bc0f,
then the approval commit.

## Plan 48 - The audit's phase 2: robustness at the boundaries

Brief amendment 49: phase 2 of the audit's roadmap
([`doc/audit.md`](doc/audit.md), Part 2, C), each fix test-first.
- The TIFF and ICO readers are bounded: the tags read from the file when
  used, an icon's PNG size checked before decoding.
- A running command's streamed output keeps its last 64 KiB, the rest
  counted and said.
- The Claude Code reader never writes to stdin.
- Git calls are bounded by `WaitDelay` once git has ended.
- A panic in the dashboard's own goroutines ends its session in order
  (`Fatal`, `Panicked`); fuzz targets for the image readers and both
  mappers.
- Bidi controls and zero-width spaces are drawn as `<U+XXXX>` on the
  approval card.

Approved on 2026-10-25: 13 of 14 PASS; 48.5 FAIL, its prediction about
the old code, accepted as a documented exception
([`doc/plan48-evaluation.md`](doc/plan48-evaluation.md)); commit 067aeb4,
then the approval commit.
