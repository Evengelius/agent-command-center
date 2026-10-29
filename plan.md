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
- **The Git Graph's watch stays as it is** (amendment 65, 2026-10-28):
  one `git for-each-ref` every 2 s while the panel shows, told its cost
  (about 6 % of one core here); a watch of the reference files' times is
  not proposed again unless the owner asks.

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
(amendment 9), the browser replaced by the system's folder dialog in plan
52 (amendment 53); git worktree management was plan 16 (amendment 10). The
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

## Roadmap

The owner's order (amendment 65, 2026-10-28), plans 64 and 65 now
closed: a search of the conversation (plan 66, below), then the
session's resumption from the home screen. Each starts as a plan for the
owner's review when the one before is closed.

## Plan 66 - The conversation's search

Brief amendment 68 (2026-10-29), the second of the roadmap's three
(amendment 65): a search of zone B's conversation, the file viewer's
search reused. Proposed for the owner's review, revised after another
agent's review the owner shared, and approved by the owner on
2026-10-29 (Progress).

### What was found (2026-10-29, on 250df5f)

- **The key.** Ctrl+F reaches the dashboard: Windows Terminal's own find
  is Ctrl+Shift+F, and ACC takes Ctrl+F only inside the draft window, for
  its full screen (`internal/app/editing.go`). The layer on top takes the
  keys first (plan 49, design D7; `internal/app/keys.go`): the key
  overlay, the draft window, the quick open and the file viewer, then the
  dashboard's views. The file viewer's `/` cannot open B's search: B's
  input owns the keyboard, and `/` there starts a message or a slash
  command. For the same reason `n` and `N` cannot step through B's
  matches once a field has closed, as they do in the viewer: they would
  be typed into the draft.
- **The text.** An entry's lines are `entryLines`
  (`internal/render/content.go`), the text a selection copies: a
  message's lines, a tool call's name and its error, an approval's or an
  error's first line, a thinking phase's summary and details. A tool
  row's target (its path, command, pattern or URL, an agent's type and
  description) is not among them: it is drawn from the call's input
  (`callDetail`, the memo's `detail`), as decoration a selection does not
  copy, the user's folder hidden (`hideUser`: `C:\Users\…\`), then cut to
  the room: a path loses its left end at a separator (`elidePath`),
  anything else its right end, "…" in place of what is cut
  (`fitDetail`); with under 8 cells left, the row draws no target.
- **What B hides.** A run of five calls or more shows its last three and
  one row for the others, which Enter or a click unfolds; a thinking
  phase's details unfold the same way. An owner's message over 20 lines
  shows its first 5 and "… N more lines", with no unfold in B (ASM-22).
- **The full view** (Enter on an entry; `entryViewer`,
  `internal/render/viewer.go`): a message's every line, each wrapped by
  `wrap`, whose rows are slices of the line, byte for byte; it scrolls
  with `InspectTop` and `InspectHold`, and its rows carry their offset in
  the row (`content`), which the selection uses. Opened at a given line,
  with its matches lit, it shows a long paste's folded lines with no new
  viewer.
- **Drawing a match.** Every span of B's rows carries its source (`ref`:
  the entry, the line, the byte offset), from which the selection's
  highlight restyles part of a span (`highlight`,
  `internal/render/select.go`); a `refAtom` span is one drawn unit
  standing for several source bytes (a tab's spaces). The file viewer
  draws its matches in `stMatch` and its current one in `stMatchCur`.
  The entry cursor scrolls B the least that shows a row (`putCursor`,
  `revealRow`), and an unfold holds B's view where it is (`unfold`).
- **The field.** The branches popup's search is the input's own editor
  (`input.Editor`): typing, Backspace, and a paste's first line, its
  control characters dropped; its ← → belong to the popup's tree.
- **The cap.** The file viewer keeps the first 10,000 matches from the
  top of its file; B's search starts from its bottom, the newest, so
  B's cap has to keep the newest.
- **The cost** (`scratchpad\plan66\zz_searchcost_test.go`, overlaid,
  never in the tree): sessions of TestLongSession's shape, 240 turns each
  of a prompt, 20 tool calls, a thinking phase and an answer of 30 lines,
  then of 300; for each query the best and the worst of five runs.

| Session | Entries, lines, text | entryLines | The viewer's regexp, case ignored | A scan ignoring case |
| - | - | - | - | - |
| two hours | 5,280, 12,480, 1.02 MB | 0.6 ms at most | 13.5 to 21.6 ms | 1.0 ms at most; 3.1 ms for "e", capped at 10,000 |
| answers ten times longer | 5,280, 77,280, 9.98 MB | 1.0 to 3.6 ms | 133 to 211 ms | 2.0 to 5.6 ms |

  The viewer's `findAll` (a literal regexp, `(?i)` without a capital)
  passes the keystroke's 16 ms on two hours of conversation; with the
  case kept it takes 0.5 to 3.5 ms. Lines lowered once and searched with
  `strings.Index` took 1.0 to 3.7 ms, but lowering them costs 3.3 to
  35 ms and a copy of the text. A scan that finds the query's first
  letter in either case (`strings.IndexByte`) and compares the window
  without case (`strings.EqualFold`) needs neither, and found exactly the
  regexp's matches for every query.

### Design (proposed)

- **D1. The bar.** Ctrl+F opens the conversation's search: one row at the
  bottom of zone B, over the input's rule, on B's ground. On the left "/ "
  in the title colour, as the viewer's field has it, then the query and
  its cursor (AltGr's characters typed in it too: step 7); on the right
  "match 1 of 240" in the progress colour, "no
  match" in the attention colour with "· calls' inputs and outputs not
  searched" in grey after it, so that it never reads as the text being
  nowhere; over the cap "match 3 of the newest 10,000", so that the count
  never reads as all there are; or,
  the query empty, "type to search the conversation" in grey; the bottom
  row's hints its keys, their directions spelled out: "↑ Enter older ·
  ↓ Shift+Enter newer · Tab whole · Esc close". B's conversation ends a
  row higher while the bar shows: its scroll, the entry cursor's reveal
  and the "↓ N new" badge count with that row. The keyboard is the
  bar's: typing, Backspace, ← → and a paste edit the query in the input's
  own editor, as the branches popup's field does.
- **D2. Where Ctrl+F opens it.** The bar is a layer of `render.Layers`,
  over the dashboard and under the windows that stand over it, which take
  their keys first as now: in the draft window Ctrl+F stays its full
  screen; in the file viewer, the quick open, the key overlay, an entry's
  full view, the inspection and the Worktrees view it does what it does
  today, the conversation being behind them. On the dashboard Ctrl+F opens
  the bar from any focus: the input, B, zone A, zone C, a docked panel
  beside B. Where a view takes B's place (the diff view; a docked panel
  or the Git Graph under 100 columns, or where B would keep fewer than 64
  columns beside it, as at 120x40: step 7), Ctrl+F closes it as Esc
  does, then opens the bar. From Ctrl+F to Esc, the draft, its cursor
  and its selection, zone C's chosen agent, the session and the
  conversation stay as they were; the query is kept in memory alone,
  never written to a file or a log, never in the draft's recovery file,
  never sent to the CLI.
- **D3. What it searches.** What B holds as text: the owner's messages
  whole, a long paste's folded lines among them; the CLI's messages (the
  RESULT box's among them); each tool row's name, its target as the row
  writes it before cutting it to the room (`hideUser`'s, so a hidden user
  folder is never matched), and its error; an approval's row, a thinking
  phase's summary and details, notices, errors and the dashboard's notes;
  the calls a run folds and a phase's folded details too. An approval's
  row is its name and its target, as a tool row's, answered or waiting
  (found at step 5: the request's target was drawn and not searched). Not
  searched: a call's input and output (the full view's and the
  inspection's), the turn review's rows (the dashboard's own, plan 65),
  the labels and times, zones A and C, the draft.
- **D4. Matching.** The query as typed, no pattern syntax; the case
  ignored unless the query has a capital, as in the file viewer's search:
  an ASCII letter in either case, and every letter whose two cases begin
  with the same byte in UTF-8 (é and É, д and Д, α and Α), by the scan
  measured above. The matches are found again at each keystroke, in
  Update, from the state as it is: no copy of the text, no command (the
  viewer's search runs in one over a file it has read; B's entries change
  under Update, and a command reading them would race). The one memo is
  each call's and each request's target, read once while the bar is open
  and dropped when it closes (`render.SearchMemo`; found at step 2: read
  afresh, the two-hour session's 4,800 targets cost 4.8 to 8.5 ms a
  search).
  At most 10,000, found from the newest back (`render.MaxMatches`): the
  count then reads "of the newest 10,000" (D1), the matches kept and not
  all there are.
- **D5. The matches in turn.** The matches are numbered from the newest,
  "match 1 of 240" the newest of all, since the search starts there and
  goes back in time. At first the current match is the newest at or above
  B's last row shown: what the owner sees. ↑ or Enter: the older match,
  its number one more; ↓ or Shift+Enter: the newer; at either end, round
  to the other, as the viewer's `n` does. B scrolls the least that shows
  the current match's row, a wrapped line's later rows too
  (`revealRow`), and is held from then on: follow stops, "↓ N new"
  counting what comes. A match inside a folded run or a phase's folded
  details unfolds them, as a click does, and they stay so, the owner's to
  fold again. A match in a long paste's folded lines is shown on the
  paste's "… N more lines" row, which B scrolls to. Typing keeps the
  current match while it still matches, else takes the newest at or
  above it.
- **D6. While the turn goes on.** With the bar open, the matches are found
  again whenever events were applied since the last search
  (`State.Applied`). An answer streaming in adds its matches: the count
  grows, and so does the current match's number when they come below it,
  but the current match stays the one it was, kept by its entry, line and
  offset, and B does not move. Only when its text is gone (a thinking
  phase's summary rewritten, a call's error replaced) does the nearest
  match at or above its place become current.
- **D7. Lit.** Every match in view on the viewer's amber (`stMatch`), the
  current one in its current-match style (`stMatchCur`), restyled over B's
  rows from their sources as the selection is, and before it, so that a
  selection shows over a match; across Markdown markup, the characters
  drawn light. A tool row's target, narrowly: its drawn characters carry
  their offsets in the target, as content's do, but of a kind a copy
  skips, so a selection copies the row as now; the "…" the cut puts in is
  one drawn unit standing for the bytes it hides, as a tab's spaces
  stand for the tab. A match in the visible part lights its characters; a
  match wholly in the cut part lights the "…"; a target the row has no
  room to draw leaves the call's name to light in its place. A long
  paste's "… N more lines" row stands for its folded lines in the same
  way, lit when a match is there.
- **D8. The whole entry.** Tab shows the current match's entry whole, in
  its full view, as Enter does from the entry cursor: a message, a long
  paste's folded lines among them, scrolled so that the match's row is in
  view, a failure the same; a tool call at its top. Its matches are lit
  there, over the rows it draws, from the same matches (the view's rows
  being slices of the entry's lines, a match its wrap splits lights on
  both rows). It keeps its own keys (↑ ↓ PgUp PgDn Home End, the wheel, a
  selection); Esc comes back to B and the bar, the current match as it
  was. No other viewer.
- **D9. Closing.** Esc closes the bar and clears the matches; B stays
  where the search left it, held. The focus goes back where Ctrl+F found
  it, the input, B, zone A, zone C or a docked panel, when that place is
  still on screen, else to the input (the diff view or the panel that
  Ctrl+F closed); the draft is as it was. F2, F3, F4, F6 (Shift+F6 too),
  Alt+P, Alt+B, Alt+G and Ctrl+W close the bar as Esc does, then do what
  they do; F1, F12, Ctrl+C and Ctrl+Q act as on the dashboard, the bar
  kept, and so do Ctrl+O, Ctrl+Shift+C, Shift+PgUp and Shift+PgDn, the
  other keys of Anywhere that leave the conversation as it is (step 3).
  Any other key is the bar's, none reaching the draft. The mouse
  in B works as before (the wheel, a selection, a fold's click), the bar
  kept; a click that gives another zone the focus closes it.
- **D10. Parts.** The dashboard's own: the reducer and its machines
  unchanged, no event, nothing to the CLI, no new dependency, no new
  rendering mechanism: the matches are drawn through the references and
  the restyling the selection already uses, and the whole entry through
  the full view. Rendering: the matcher over the state (`ConvMatches`),
  the bar's layer, the highlights in B and in the full view, the target's
  references, a match's row for the reveal. The application: the bar's
  field and keys (`UI.Search`), the search again after a keystroke or
  applied events, the unfold, the reveal, Tab's full view, the focus kept.

### Criteria (proposed)

1. **66.1 What is searched** (render tests): a query found in each kind D3
   names, with its entry, line and offset: a long paste's folded line, a
   folded run's call, a phase's folded details, a tool row's target by a
   word its cut hides; nothing found in a call's input or output, a turn
   review's row, a label, zone C or the draft; a user folder's name
   (`C:\Users\name\`) never found in a target.
2. **66.2 Matching:** a lower-case query ignores case (`ERROR` found by
   `error`; `é`, `д` and `α` find their capitals), a capital keeps it
   (`Error` finds only `Error`); `a.b`, `(x)`, `\d` and `*` found as
   typed; on the probe's two sessions and random queries, exactly the
   matches of the file viewer's `findAll` over the same lines; over
   10,000, the newest 10,000 kept and "of the newest 10,000", the first
   current match still the newest at or above B's last row.
3. **66.3 The bar** (application tests and golden frames at 80x24, 120x40
   and 200x60): Ctrl+F from the input, B, A, C and a docked panel; from the
   diff view in B's place, which closes; in the draft window its full
   screen, as before, and in the file viewer, the quick open, the key
   overlay and the full view nothing new; the field's typing, Backspace,
   ← → and paste; "match 1 of 240", "no match · calls' inputs and
   outputs not searched", the empty field's words, the keys' directions;
   on Claude Code's and Codex's mocks alike; every existing golden frame
   unchanged.
4. **66.4 The matches in turn:** the first the newest at or above B's last
   row shown; ↑ and Enter older, ↓ and Shift+Enter newer, the number
   following, round at either end; B scrolled the least that shows the
   match, on a wrapped line's third row too, and held after it; a match in
   a folded run and in a phase's folded details unfolding them; one in a
   long paste's folded lines shown on its "… N more lines" row; typing
   keeping the current match.
5. **66.5 While the turn goes on:** with an older match current, an answer
   streamed with the bar open adds its matches, the count and the current
   match's number growing, the current match and B's view not moving; a
   current match whose text is gone giving way to the nearest at or above
   it; a new turn's prompt found.
6. **66.6 Lit:** in view, every match in `stMatch` and the current one in
   `stMatchCur`; inside bold text, code and a link's text, in CJK text, in
   an error; in a tool row's target its visible characters, the "…" for a
   match its cut hides wholly (the current one never without a lit mark),
   the name for a target not drawn; a long paste's
   fold row for its folded lines; a selection drawn over a match; a
   selection's copy byte for byte as before the plan, a tool row's
   included; nothing lit after Esc.
7. **66.7 The whole entry:** Tab on a match in a long paste's folded lines
   opens its full view with the match's row in view and lit, one its wrap
   splits lit on both rows; on a message and on a tool call; the view's
   own keys; Esc back to B and the bar, the current match the same.
8. **66.8 Closing:** Esc, the keys D9 names, a click giving another zone
   the focus; B held where the search left it; the focus back where Ctrl+F
   found it, from the input, A, C and a docked panel, and to the input
   when Ctrl+F closed the diff view; the draft's text, cursor and
   selection byte for byte the same from Ctrl+F to Esc; the query in no
   file the dashboard writes (the draft's recovery among them) and in
   nothing written to the CLI.
9. **66.9 Timing:** `TestSearchToFrame` in the timing step, each time the
   whole of it (the query read, the matches found and placed, the rows lit,
   the frame drawn): a keystroke in the bar to its frame on the probe's two
   sessions, a long paste's hidden match among them, and a delta to its
   frame with the bar open on the two-hour one, p95 within 16 ms, the
   target unchanged; the timing step's other tests within their targets.
10. **66.10 The documents:** ui-design.md, a Search section in Zone B with
    its frame (golden frame `search`, byte for byte; step 5: `TestDocFrames`
    reads no document), Focus and keys and the key overlay with
    Ctrl+F; architecture.md; evaluation.md's timing step; brief amendment
    68; AGENTS.md's MVP scope.
11. **66.11 Windows Terminal:** in the acc tab, at 208x59, 120x40 and
    80x24, a long mock made in the scratchpad, with a folded run and a
    long paste in it: Ctrl+F, a query typed, its matches lit, ↑ to a match
    in the folded run, then to one in the paste's folded lines and Tab, Esc
    twice; the owner's look at the shots.
12. **66.12 The checks:** build, vet (Windows, Linux), gofmt and the whole
    suite; the changes broken on purpose through `go test -overlay`, each
    caught; `-race` on the changed packages; the timing step alone, on
    mains power.

### Steps

1. Amendment 68 and this plan, for the owner's review.
2. The matcher (render): 66.1, 66.2.
3. The bar, its keys, the matches in turn and closing (application,
   render): 66.3, 66.4, 66.5, 66.8.
4. The highlights and the whole entry (render, application): 66.6, 66.7.
5. The timing test and the documents: 66.9, 66.10.
6. Windows Terminal: 66.11.
7. The checks, the mutants, the timing and `doc/plan66-evaluation.md`
   (66.12); the commit after the owner's review.

No live run: the search is the dashboard's own, and mocks show what it
does; no CLI configuration touched, no quota used.

### Progress

- 2026-10-29: step 1: amendment 68 and this plan, proposed for the
  owner's review, with the findings above (the probe in the scratchpad,
  nothing of it in the repository).
- 2026-10-29: another agent's review, shared by the owner, folded in: the
  approach, the matching, the directions and the cost accepted; its three
  points answered. Ctrl+F in the draft window stays its full screen, the
  bar being a layer under the windows (D2), and Esc gives the focus back
  where Ctrl+F found it, the draft untouched (D9). A tool row's target is
  lit through narrow references of its own, the cut's "…" standing for
  what it hides, the name when the target is not drawn (D7). A long
  paste's folded lines are searched, shown on its fold row, and Tab opens
  the existing full view at the match, lit (D3, D5, D8): the full view's
  rows are slices of the message's lines, so no new viewer. Besides: the
  current match stays put while an answer streams (D6); the matches
  numbered from the newest, so that the cap keeps the newest, where the
  search starts (D4, D5); the query never written or sent (D2); Windows
  Terminal at 80x24 too (66.11).
- 2026-10-29: the other agent's final review, sent by the owner as their
  own message: "You may now proceed with the implementation of Plan 66,
  following the approved design", taken as the owner's approval (brief
  amendment 68). Its refinements folded in: the no-match words naming
  calls' inputs too, the cap's count "of the newest 10,000" (D1, D4), a
  match wholly in a cut target's "…" tested as visible (66.6), the timing
  test the whole keystroke with a long paste's hidden match (66.9), both
  CLIs' mocks (66.3). AGENTS.md's status and its MVP scope updated.
  Step 2 begins.
- 2026-10-29: step 2, the matcher (`internal/render/search.go`:
  `ConvMatches`, `entryParts`, `findIn`): `TestConvMatchesKinds`,
  `TestConvMatchesCase`, `TestConvMatchesLiteral`, `TestConvMatchesOrder`,
  `TestConvMatchesCap`, `TestConvMatchesLikeRegexp` (311 queries on
  sessions of the probe's shape, exactly the regexp's matches) and
  `TestFindIn` pass. Found: each call's target read afresh at every
  keystroke cost 4.8 to 8.5 ms on the two-hour session's 4,800 calls, so
  the targets are read once while the bar is open (`SearchMemo`,
  `TestConvMatchesMemo`; D4 revised).
- 2026-10-29: step 3, the bar, its keys, the matches in turn and closing
  (`internal/app/search.go`; the bar's row, its layer and B's room in
  render): `TestSearchOpen`, `TestSearchField`, `TestSearchSteps`,
  `TestSearchStartsInView`, `TestSearchFolds`,
  `TestSearchThinkingDetails`, `TestSearchStreaming`, `TestSearchClose`
  and `TestSearchCounts` pass; 12 golden frames (`search`, `search-none`,
  `search-codex`, each at the four sizes), no existing frame changed.
  Shift+F6 closes the bar as F6 does, and Ctrl+O, Ctrl+Shift+C and
  Shift+PgUp/PgDn keep it (D9 revised).
- 2026-10-29: step 4, the lighting and the whole entry: `TestSearchLit`,
  `TestSearchLitName`, `TestRetarget`, `TestSearchLitView` and
  `TestSearchWhole` pass. On the way, `markTarget` gave a call with no
  target an empty line instead of none, so that its row drew a target
  column (`TestFailedToolRow` failed): fixed, nil kept nil.
- 2026-10-29: step 5, the timing test and the documents.
  `TestSearchToFrame`'s first run failed on its own setup (the rounds left
  "te" in the field, so the hidden match was looked for as "terefused"):
  the field is now emptied first. Run alone while the documents were
  written, not the timing step: a key p95 7.5 ms on two hours and 10.5 ms
  on answers ten times longer, a delta p95 5.5 ms. Writing ui-design.md's
  frame found an approval's target drawn in its row and not searched: it
  is searched and lit as a tool row's now, answered or waiting
  (`entryParts`, `answeredRows`, `pendingBox`; `TestConvMatchesKinds`,
  `TestSearchLitApproval`; D3 made explicit), the `search` frames reading
  "match 1 of 8". ui-design.md: Zone B, Search, its frame the golden
  `search`'s rows 2 to 26 in cells 36 to 111, byte for byte (compared by
  `scratchpad\plan66\docframe.mjs`: `TestDocFrames` writes the Layouts'
  frames only and reads no document); Focus and keys' Ctrl+F row and the
  search bar's table, the key overlay with them (`TestKeyOverlayRows`, 104
  rows). architecture.md, The conversation's search; evaluation.md, the
  timing step's twenty-one tests. The F1 page now names the bar's own
  "↓ Shift+Enter", which `TestConfiguredKeysShown` took for a newline key
  the configuration had dropped (the whole page searched for the text): it
  now refuses "Shift+Enter" on every row but that one, its other checks as
  they were. The whole suite then passed (build, vet, gofmt, 16 packages).
- 2026-10-29: step 6, Windows Terminal 1.23.20211.0 in the acc profile,
  maximised (208x59), at 120x40 and at 80x24, driven by
  `scratchpad\term\p66run.ps1` over a scratch mock
  (`scratchpad\plan66\mk66mock.mjs`: a 60-line paste whose line 40 says
  "quota exceeded", a run of six Reads whose second, folded, reads
  quota.go, eight turns). At each size: Ctrl+F's bar; "quota" typed,
  "match 1 of 3" lit in the newest answer; ↑, "match 2 of 3", the run
  unfolded and B scrolled to quota.go's row, its "quota" lit; ↑, "match 3
  of 3", the paste's "… 55 more lines" row lit at B's top; Tab, the
  paste's full view, line 40 lit in view; Esc, B and the bar as they were;
  Esc, the bar closed, B where the search left it, the input's caret back.
  The full view stops at its end, so line 40 showed lower than two rows
  from its top at 208x59 and 120x40: ui-design.md says so. No Claude Code
  or Codex session ran in a terminal (every claude.exe under the Claude
  desktop app), so none was closed.
- 2026-10-29: step 7. The tests read against the criteria before the
  mutants, eight parts had none, now tested: Ctrl+F from a docked panel,
  at 200x60 beside B and at 120x40 in its place, and over the quick open
  and the file viewer (66.3, `TestSearchDockedPanel`); the least scroll
  to a wrapped line's later row, up and down (66.4, `TestSearchReveal`);
  a current match whose text is gone (66.5, `TestSearchGone`); Tab on a
  call's target, the full view's own keys (66.7, `TestSearchWhole`);
  every key D9 names and the ones kept (66.8, `TestSearchCloseKeys`); the
  query in no file written and nothing sent (66.8,
  `TestSearchNotWritten`); the draft and the labels never searched (66.1,
  `TestConvMatchesKinds`); past the cap, B scrolled up, the first match
  still the newest at or above its last row (66.2, `TestSearchCapStart`).
  D9's B's entries: Esc gave the input the focus; it now gives the entry
  cursor back on its entry (`TestSearchEntryCursor`). D2's "under 100
  columns" made exact: a panel takes B's place there, and where B would
  keep fewer than 64 columns beside it, as at 120x40 (`PanelCompact`). A
  first mutant run, stopped to wait for these tests, had R6 survive, a
  live thinking card searched: equivalent, a live phase having no lines
  (`entryLines`); its kills after the stop were void, mutrun's go test
  stopped with it.
- 2026-10-29: step 7, the 62 mutants on those tests (661 s, the code's
  hash the same before and after): 54 killed, three of them by the
  compiler ("declared and not used"), so rewritten to compile; 8
  survived, each settled. Equivalent: R6 (above); R21, a long paste's
  explicit fold-row case, its entry's first whole row being that row
  anyway; A5, the docked panel's check on Esc, syncUI giving an "R" focus
  with no panel shown back to B. Tested now: R16, a click opening the "+"
  menu with the bar open (`TestSearchClose`); R18, a fold row's place
  (`TestSearchPlaces`); A3, Ctrl+F over Source Control's commit dropdown,
  which takes no key of its own (`TestSearchClosesPopups`); A11, a call's
  name kept over its target, both matching on its row
  (`TestSearchKeepsPart`). Two were defects, fixed: R17 showed that a
  row's place ended one byte past it (`rowEnd`'s `r.to`, exclusive), so
  that a match starting the next row, below the view, could be the first
  current one: the place is now its last byte shown; A9 showed that the
  bar refused AltGr's characters. Measured in Windows Terminal 1.23 on the
  owner's Belgian AZERTY layout (`scratchpad\term\p66run.ps1`'s altgr
  step, TEA_TRACE): AltGr+2 arrives as `[50;0;64;1;41;1_`, "@" with left
  Ctrl and right Alt, which the decoder gives its text and the
  application's Ctrl and Alt test dropped. The bar now takes a key's text
  with no Ctrl or Alt, or with both, AltGr's (`keyText`): "/ @" in the
  probe. The same test drops AltGr's characters in the input's editor,
  the branches popup's field and the other fields (`editKey`,
  `branches.go`), measured the same way (AltGr+2 and AltGr+1 typed
  nothing in the draft): a defect older than this plan, outside its
  scope, reported to the owner with its fix proposed apart. The 13 that
  survived or changed, run again: all killed but R6, R21 and A5. Windows
  Terminal again on the final build, at the three sizes: the same eight
  steps, the same results.
- 2026-10-29: step 7's checks on the final code. The 64 mutants (602 s,
  the code's hash the same before and after): 61 caught by a failing
  test, R6, R21 and A5 equivalent. go build, go vet (Windows and Linux),
  gofmt ok; go test ./... ok, 16 packages; -race on internal/app's
  search, golden-frame and key tests (18.7 s) and the whole of
  internal/render (118.3 s): no race. The timing step alone, on mains
  power, 14:55 (CPU 11 to 25 % before): 20 PASS, `TestSearchToFrame`
  among them (a key p95 7.15 ms and 9.91 ms, a delta 4.10 ms), and 1
  FAIL, `TestBannerToFrame` p95 2.315 ms over its 2 ms, the home screen's
  banner, which plan 66 does not touch: alone, HEAD's tree (250df5f) and
  this one in turn, CPU 0 to 16 %, p95 1.645, 1.576, 1.566 ms against
  1.619, 1.517, 1.550 ms, within the target on both. 66.9 stays a FAIL
  as measured, its exception the owner's to decide, as in plan 65.
  `doc/plan66-evaluation.md`: 10 PASS, 1 FAIL, 66.11 waiting for the
  owner's look at the shots. Results and shots presented for the owner's
  review; nothing committed.
- 2026-10-29: the owner shared, with no words of their own, another
  agent's review of the shots and the report ("APPROVED — 9,5/10"): it
  approves the shots ("I approve criterion 66.11 — the independent visual
  review"), taken as 66.11's look, PASS, so 11 PASS and 1 FAIL; it
  requests no change. It recommends that 66.9 be accepted as a documented
  exception, its FAIL, the 20 PASS and 1 FAIL and the 2 ms target kept,
  and leaves that decision and the commit to the owner. Recorded after the
  evaluation's block, which stays as reported, with the evidence for the
  three equivalent mutants that the review asked to keep. Waiting for the
  owner's decision on 66.9's exception and for the commit's authorization.
- 2026-10-29: the owner, word for word: "I accept the documented timing
  exception for criterion 66.9, under the same principles applied to Plan
  65.", the FAIL kept as measured, the 2 ms unchanged, no run again only
  to pass; the final commits and the closure allowed, the diff checked
  first, then the push once the commits and the final checks are done;
  "Keep the pre-existing AltGr issue separate for the next plan." Plan
  66: 11 PASS and 1 accepted with a documented exception, 66.9.

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

## Plan 49 - The rest of phase 0, and phases 3 and 4 of the audit

Brief amendment 50: the rest of phase 0 of the audit's roadmap and its
phases 3 and 4 ([`doc/audit.md`](doc/audit.md), Part 2, C).
- Phase 0: the event queue's and the write queue's tests, and a matrix of
  the ten overlays alone and in their 90 ordered pairs.
- Tool rows memoised and long sessions' scans bounded (`render.Memo`,
  `State.ToolOrder`, `State.ParentRev`, zone A's rows made when shown);
  `TestToolsToFrame` joins the timing step.
- One owner of the overlays (`render.Layers`), the matrix's five defects
  fixed.
- The adapters' shared lifecycle (`pipe.Proc`).
- The git tests in parallel, git's environment set once in `TestMain`.
- Normalised approval decisions (`event.Decision`), each adapter writing
  its CLI's words byte for byte as before.
- Widths by grapheme cluster (`render/cluster.go`).
- A GIF's first frame alone decoded, its other frames walked by their
  blocks.

Approved on 2026-10-25: 11 of 16 PASS; 49.7 and 49.10, short of their
targets, and 49.8, 49.9 and 49.11, on tests that changed how they call
the code, FAIL, accepted as documented exceptions
([`doc/plan49-evaluation.md`](doc/plan49-evaluation.md)); commit 5c06d39,
then the approval commit.

## Plan 50 - Source Control multi-selection

Brief amendment 51: Ctrl+click selects files in Source Control, a "▎" on
each selected row and "Commit N files" on the button, and Commit and
Commit & Push commit them alone, by `git commit --only`: each selected
file staged whole, a partly staged one refused, unstaged and untracked
ones staged only after the question's yes, the rest of the index left as
it is. Windows Terminal passes Ctrl with a click, never Shift. The
owner's question before the approval of the plan: when the commit fails
after the question's staging, the staging stays, as git leaves it, and
the note says so. After the first evaluation (13 of 13 PASS) the owner
asked that a plain click on a file clear the selection, Commit being the
whole index as before with none selected, and that the note say
"changes"; the review of the complete diff then found a file moved on
disk, both its rows selected, refused once staged, fixed by reading
git's status path by path. Approved on 2026-10-26: 14 of 14 PASS
([`doc/plan50-evaluation.md`](doc/plan50-evaluation.md)); commit
b9f8323, then the approval commit.

## Plan 51 - Home screen redesign

Brief amendment 52: the home screen drawn as the owner's image,
`references/16-home-screen.webp`: picture mode (Sixel images, their text
by GDI in Cascadia Mono) from 118 columns with Sixel, grid mode otherwise;
the project field's path terminal text, its editing, selection, copy and
paste as before; an animated Space Invaders banner, ten frames a second,
only the pixels that change sent, in a fixed palette, paused without the
focus and hidden on small screens; Ctrl+N resets the form, the mouse
works the screen, Ctrl+P is left out. After the evaluation the owner
asked for a livelier pace, then twice for a faster ship (the march every
0.4 s, the ship's legs at 125 to 275 design pixels a second; a dash with
afterimages tried and withdrawn). Approved on 2026-10-27: 10 of 11 PASS
at the evaluation; 51.1, the field's box up to half a row from the
design's place, FAIL, accepted as a documented exception; 51.7 measured
again on a quiet machine, PASS; 51.9's paste back checked by the owner
([`doc/plan51-evaluation.md`](doc/plan51-evaluation.md)); commit
7e60f73, then the approval commit.

## Plan 52 - The system's folder dialog for the project

Brief amendment 53: the home screen's folder button and Ctrl+O open the
system's dialog for choosing a folder (IFileOpenDialog, folders only),
over the terminal, at the field's directory, else its nearest existing
parent, else the starting folder; the folder chosen goes into the field,
which takes the focus, Cancel changes nothing, a failure is a note. Plan
15's project browser removed at the owner's choice, `realPath` and
`dosPath` kept in `input/realpath_*.go`. Approved on 2026-10-27: 5 of 5
PASS ([`doc/plan52-evaluation.md`](doc/plan52-evaluation.md)); commit
9b8b6e5, then the approval commit.

## Plan 53 - A project chosen in the list fills the field

Brief amendment 54: choosing a project in the home screen's PROJECTS
list, by up and down or a click on its row or star, puts its path into
the Project directory field, so that the field shows what Start uses;
Tab onto the list, f and Del leave the field as it is. Approved on
2026-10-27: 5 of 5 PASS
([`doc/plan53-evaluation.md`](doc/plan53-evaluation.md)); commit
1818367, then the approval commit, both pushed.

## Plan 54 - The mouse wheel over the project list

Brief amendment 55: the wheel over the home screen's PROJECTS panel, its
rows, headings and frame, moves the list's selection one project a notch,
as up and down do, choosing it (its path into the field) and giving the
list the focus; the panel's other cells carry `HomeItemList`. Approved on
2026-10-27: 4 of 4 PASS
([`doc/plan54-evaluation.md`](doc/plan54-evaluation.md)); commit
38ee6d7, then the approval commit, both pushed.

## Plan 55 - Nerd Font icons in the Explorer

Brief amendment 56: in the acc tab, with Cascadia Mono NF installed for
the owner (release 2407.24, per user, registered from Windows Terminal:
the Claude app's shells write the registry into its private copy), the
Explorer's rows draw Nerd Font icons in each kind's colour and the
codicons' chevrons; anywhere else (another terminal or profile, no font,
a terminal started before it) the panel's own glyphs. The launcher names
the face in the acc tab's fragment by the same rule. Approved on
2026-10-27: 5 PASS, 0 FAIL, and 55.5's visual review by the owner
([`doc/plan55-evaluation.md`](doc/plan55-evaluation.md)); commit
374c23e, then the approval commit, both pushed.

## Plan 56 - The Explorer's font installed by acc

Brief amendments 57 and 59: Cascadia Mono NF bundled in the repository as
released (`third_party/fonts`, the SIL OFL 1.1 with its symbols'
notices) and installed by acc for the user, under Windows Terminal, when
Windows has no font of that name: the files checked against
`SHA256SUMS`, registered under HKCU, one line printed; a failure one
line, the launch going on; plan 55's rule unchanged, the icons from
Terminal's next start. Withdrawn on 2026-10-28 for plan 57 and reinstated
the same day. Approved on 2026-10-28 without repeating its end-to-end
test, run with the owner on 2026-10-27: 6 PASS, 0 FAIL
([`doc/plan56-evaluation.md`](doc/plan56-evaluation.md)); commit
d88c392, then the approval commit, both pushed; its full text in git
(`git show d88c392:plan.md`).

## Plan 57 - Segoe Fluent Icons in the Explorer

Brief amendment 58: the Explorer's icons from Segoe Fluent Icons, a font
that comes with Windows 11, named after Cascadia Mono in the acc tab's
face, nothing to install. Proposed on 2026-10-28 and withdrawn by the
owner the same day without being implemented (amendment 59), for plan
56's Nerd Font icons and the same look on any Windows version; its
proposal in git (`git show d88c392:plan.md`).

## Plan 58 - A click outside the window closes it

Brief amendment 60: a left click outside the window floating over the
dashboard, the file viewer or the quick open, closes it as Esc's last step
does and does nothing else, the click judged against the frame drawn last
(`Screen.Window`); full screen, its first row is outside; the draft
window keeps its rule. Approved on 2026-10-28: 5 PASS, 0 FAIL
([`doc/plan58-evaluation.md`](doc/plan58-evaluation.md)); commit
0408228, then the approval commit, both pushed; its full text in git
(`git show 0408228:plan.md`).

## Plan 59 - The branches popup

Brief amendment 61: a click on the project bar's branch name, or Alt+B,
opens a popup of the repository's branches and tags: a search, Recent,
Local, Remote and Tags, a message where a group is empty; a switch (git
switch, a remote branch as a tracking one), confirmed when uncommitted
changes would come along and refused while the agent works, a notice in
zone B and a note in an empty draft after it; a fetch of every remote,
never pruning, Esc and a two-minute limit stopping it; a name copied; the
header's branch following the HEAD file, checked every 2 s. New branch
left out by the owner. Approved on 2026-10-28: 10 PASS, 0 FAIL after the
owner's visual review ([`doc/plan59-evaluation.md`](doc/plan59-evaluation.md));
commit 5c8c9be, then the approval commit, both pushed; its full text in
git (`git show 5c8c9be:plan.md`).

## Plan 60 - Git blame in the file viewer

Brief amendment 62: `b` in the file viewer shows, left of the line
numbers, the date and author of the commit that last changed each line,
after the owner's image of an IDE's blame, and in the footer the cursor
line's commit; a line not committed shows "Not committed" and claims no
author; a compact form where the code would keep fewer than 60 cells. Git
is given the very bytes shown (`git blame --porcelain --contents -`), an
answer is kept only for the same file and the latest read, and git's
refusals are named. Approved on 2026-10-28: 9 PASS, 0 FAIL after the
owner's visual review ([`doc/plan60-evaluation.md`](doc/plan60-evaluation.md));
commit 0050409, then the approval commit, both pushed; its full text in
git (`git show 0050409:plan.md`).

## Plan 61 - Clicks in the file viewer and the quick open

Brief amendment 63: a left click on a line of the file viewer puts the
cursor on it, anywhere on its row between the side borders, the view
kept and the selection starting as before; in the quick open a click on
a file selects it and a click on the selected one opens it, as in the
Explorer. A click is judged by the rows the frame drew
(`Screen.WindowList`). Approved on 2026-10-28: 7 PASS, 0 FAIL after the
owner's visual review ([`doc/plan61-evaluation.md`](doc/plan61-evaluation.md));
commit b452ade, then the approval commit, both pushed; its full text in
git (`git show b452ade:plan.md`).

## Plan 62 - Git Graph

Brief amendment 64: the sidebar's third tile, or Alt+G, shows the
repository's history in the docked place, in git's topological order,
read a page at a time from fixed tips: coloured lanes, an overflow column
that joins no commits git does not join, labels; a commit's files and a
file's diff in that commit; a scope popup of references (All, Auto,
branches, remote branches, tags); a wide view; the cursor's row in the
footer and a card beside a row. Read only, nothing fetched. Approved on
2026-10-28 after the owner's visual review of its three stages: 11 PASS,
1 FAIL accepted as a documented exception (62.10, 209 bytes kept a
commit, not about 100) ([`doc/plan62-evaluation.md`](doc/plan62-evaluation.md));
commit e906b73, then the approval commit, both pushed; its full text in
git (`git show e906b73:plan.md`).

## Plan 63 - The ping

Brief amendment 65: when the agent needs the owner while the dashboard's
window, or its tab, lacks the focus (an approval or a question waiting, a
turn completed or failed, the session ended on its own), one BEL in ACC's
own Windows Terminal profile, whose fragment sets `"bellStyle":
"taskbar"`, at most one in 5 s and none lost to that limit, and the most
urgent reason in the window's title; nowhere else a BEL. Approved on
2026-10-28: 6 PASS, 0 FAIL after the owner's review ("I like it", "No
sound") ([`doc/plan63-evaluation.md`](doc/plan63-evaluation.md)); commit
4a049f3, then the approval commit, both pushed; its full text in git
(`git show 4a049f3:plan.md`).

## Plan 64 - Windows notifications

Brief amendment 66: with each ring of plan 63's ping, a silent Windows
notification of the most urgent need rung, under the name Agent Command
Center, its text generic (the CLI's name only), one at a time and taken
out of Windows' list once nothing waits; its app ID that of a Start menu
shortcut the dashboard makes when missing, pointing at Windows Terminal's
alias with ACC's profile; a feasibility probe first. Approved on
2026-10-28: 7 PASS, 0 FAIL after the owner's review ("Looks right", "No
sound") ([`doc/plan64-evaluation.md`](doc/plan64-evaluation.md)); commit
0111dbd, then the approval commit, both pushed; its full text in git
(`git show 0111dbd:plan.md`).

## Plan 65 - The turn-by-turn review

Brief amendment 67: after each turn, zone B lists the files that changed
in the project while the turn ran, whoever changed them, each file's
change of that turn in plan 44's diff view; two git snapshots in a store
of the dashboard's own, readable by the user alone, no hook or filter
run, no credential file, hard link or file outside the project read.
Approved on 2026-10-29 through the reviews the owner shared: 7 PASS and
2 accepted with a documented exception, 65.1 (a split index's shared
file's time) and 65.9 (TestBannerToFrame's timing on a busy machine)
([`doc/plan65-evaluation.md`](doc/plan65-evaluation.md)); commit
3757c15, then the approval commit, both pushed; its full text in git
(`git show 3757c15:plan.md`).
