# Plan 70 evaluation (2026-10-30)

The check of plan 70, the one-time summary suggestion after a resume (brief
amendment 72): once a session is resumed from the home screen and its CLI
has named the session asked for, zone B offers once to put a fixed request
for a summary of the earlier work in the empty draft; nothing is sent
until the owner presses Enter, and Not now dismisses the offer. The owner
asked for it on 2026-10-30, the scope settled over two reviews: "Proceed
with Plan 70 based on the revised design and your latest technical review.
[…] Proceed directly without pausing for another approval checkpoint."

It covers the 12 criteria of [`plan.md`, Plan 70](../plan.md), written
before the code. Nothing is committed or pushed, and no test data is
deleted: the owner reviews the results and the diff first, and runs the
manual acceptance below.

**Revised after the owner's review** (brief amendment 73, the same day):
the suggestion is drawn as the owner's card. [Revision 1](#revision-1-the-owners-card)
holds the card's evaluation, which revises 70.2 and adds 70.13; the
block below stays as it was reported. **Accepted by the owner** the same
day: [10 scenarios of 10 PASS](#the-owners-manual-acceptance-2026-10-30),
on Claude Code and on Codex.

How it was checked:
- **Tests**: the state (`internal/state/summaryoffer_test.go` and three
  cases of the state machines, one per row of events-and-state.md), the
  drawing (`internal/render/summaryoffer_test.go`, at 80x24, 120x40 and
  200x60 and at every width from 1 to 60) and the application through the
  real adapters over both fake CLIs, Claude Code's and Codex's
  (`internal/app/summaryoffer_test.go`, each test run for both).
- **Each change broken on purpose** through `go test -overlay`, the
  repository never written, its code hashed before and after: 34 mutants
  (`scratchpad\plan70\mkmutants70.mjs`, `mutrun70.sh`), run three times,
  the last on the final code.
- **Windows Terminal** (70.9): ACC's profile, maximised and at 120x40,
  plan 69's stand-in `claude.exe` (no CLI ran, no quota).
- **The live runs** (70.10): Claude Code and Codex, each on its own,
  three turns each, in a temporary project, the configuration check
  before and after; nothing deleted afterwards.
- **Build, vet (Windows and Linux), gofmt, the whole suite** after the
  last code edit, `-race` on the changed packages' tests, and **the timing
  step** alone, on mains power.

## The evaluation

```text
Evaluation - Plan 70, the one-time summary suggestion after a resume - 12 criteria: 12 PASS
PASS 70.1 The offer: TestSummaryOfferSet: set for the session asked for, Claude Code's and Codex's, Codex's also when it names the thread before the handshake ends, standing until Ready; TestSummaryOfferNever: none after a message of the owner's sent before the CLI names the session, for a resume Claude Code refuses ("No conversation found with session ID") or one that fails (the handshake's timeout), after a crash resume (Ctrl+R, a new session machine, the earlier offer unanswered) or in a mock run; TestStateMachines' Session row "Ready | the first session.started after a resume from the home screen": none for another ID or a new session, the next turn's session.started leaving it; the row "Ready | owner prepares the summary request, or answers Not now": gone for good, nothing written, no entry, the session Ready and the turn Idle; the Turn row "Submitted | first event of the turn": standing while the message waits, gone once the CLI accepts it; TestSummaryOfferKept: kept by a usage read's telemetry, a turn the CLI opened itself and a send refused during it, a Codex slash command held back and a write that failed, gone once a message is accepted; TestSummaryOfferPrepare and TestSummaryOfferNotNow: projects.json byte for byte unchanged by Prepare summary and by Not now; TestSummaryOfferNotOffered: START SESSION, another session named and a crash resume through the real adapters, none
PASS 70.2 The suggestion drawn: TestSummaryOfferDrawn, both CLIs at 80x24, 120x40 and 200x60: after the resumed note, the question, a blank row, "[ Prepare summary ]    [ Not now ]", a blank row, "Prepare puts a request in the draft." and "Nothing is sent until you press Enter." in one column, blank rows before and between the blocks, every frame row the screen's width; the buttons' cells, first and last, SummaryPrepare and SummaryNotNow, the cells beside them and the question not; TestSummaryOfferRows, every width from 1 to 60: every row within it, every span decoration (never copied), the buttons side by side from 34 columns and one above the other under, every text whole once wrapped, seven rows from 38 columns (the owner's six and a blank row after), not among the session's copied lines; TestSummaryOfferFollows: after a 60-line answer of a turn the CLI opened, B following its end, the last note above the blank row that ends the conversation at each size; TestSummaryOfferFocus: Prepare summary filled with the focus, Not now once selected, neither without; TestSummaryOfferHidden: not drawn while the session is Crashed, Closing, Closed or Handshaking, while a message is Submitted or a turn Running, under zone A's tree at 80x24 or the diff in B's place, nor once the offer is gone, the offer kept by drawing; the golden frames unchanged
PASS 70.3 The keys: TestSummaryOfferKeys, both CLIs: Enter on the empty input with no focus does nothing; ↑ gives the focus, Prepare summary selected; ↑ again with no entry stop above keeps it; after a turn the CLI opened (the suggestion hidden while it ran, the offer kept), ↑ goes on to its call's stop; ↓ past the newest entry gives the focus back; → selects Not now; Esc returns to the input, the suggestion shown; ↑ then ↓ returns too; a typed character leaves the focus and is typed in the draft; F2 leaves it and says "no approval waiting"; a paste's text ends it, and Enter then sends the pasted words; TestSummaryOfferNotNow: ← and → between the buttons; TestSummaryOfferScrolled: B scrolled up, the suggestion out of view, ↑ is the entry cursor's (the call's stop), ↓ past it gives the suggestion the focus and B follows its end; ↑ with text in the draft moves in the draft; the footer "←→ select · Enter choose · ↑ entries · Esc back to the input" while it has the focus (TestSummaryOfferPrepare); the existing key tests unchanged and passing
PASS 70.4 The mouse: TestSummaryOfferMouse, both CLIs: a click on the question changes nothing, the focus not taken; a click on Prepare summary puts the request in the draft, the input focused, nothing written; TestSummaryOfferNotNow: a click on Not now withdraws the offer, the words in the draft kept; the existing click tests unchanged and passing
PASS 70.5 Prepare summary: TestSummaryOfferPrepare, both CLIs: the exact request in the empty draft, the cursor at its end, nothing written, the offer gone, the suggestion and its focus gone, projects.json unchanged; the draft cleared (Ctrl+A, Backspace): not back; Ctrl+Z brings the request back; TestSummaryOfferDraftKept, both CLIs: words typed, then a click on Prepare summary; an image attached and no text, ↑ and Enter; a draft restored at the session's start ("draft restored"), then a click: each time the toast "The draft is not empty: send or clear it, then Prepare summary", the text and the images unchanged, the suggestion kept, nothing written, the focus kept after a refused Enter
PASS 70.6 Not now: TestSummaryOfferNotNow, both CLIs: by keys (↑ → Enter) and by a click: the offer withdrawn, nothing written, the draft and its words unchanged, projects.json unchanged; ↑ afterwards no longer the suggestion's
PASS 70.7 Sent by Enter: TestSummaryOfferPrepare, both CLIs: the prepared request sent byte for byte through the adapters (Claude Code's user message content, Codex's turn/start input text), the same message as an ordinary one sent after it once the text is taken out (their shapes compared), no settings change written (no set_model, apply_flag_settings or set_permission_mode; no thread/settings/update); TestSummaryOfferFirstMessage: a write that fails restores the draft, the turn failed, and keeps the suggestion
PASS 70.8 Nothing else changes: plan 69's tests unchanged and passing (TestLastSession*, TestResume*, the refusals), the crash resume's path as before; the golden frames unchanged (no frame has the suggestion); no adapter, store or telemetry change (the code diff is internal/state, internal/render and internal/app only); go test ./... ok
PASS 70.9 Windows Terminal: ACC's profile, 1.23, maximised (208x59) and at 120x40, plan 69's stand-in claude.exe, no CLI and no quota, the final build (acc-p70.exe, sha256 336c2cd5…, a rebuild of the final tree byte-identical): 14 runs, every poll found. A session started from the home screen and told a word, then quit (the store kept; once more with words left in the draft, the store and the drafts kept). Ctrl+R in a new dashboard: the resumed note and the suggestion under it; ↑ filled Prepare summary and showed the footer, → Not now, ← Prepare summary again, Enter put the request in the draft, the suggestion gone; Enter sent it, "OK (stand-in)" answered under the YOU message. A click on "[ Not now ]": the suggestion gone, nothing sent (no YOU entry, no answer, the badge Idle two seconds later). A click on "[ Prepare summary ]": the request in the draft, nothing sent. The draft restored at the start ("draft restored", "my own draft words" in the input) and a click on Prepare summary: the toast "The draft is not empty: send or clear it, then Prepare summary", the words kept, the suggestion kept, the request not put in. Made smaller with the suggestion shown: Win+Down restoring the maximised window (208x59 to 160x40), and three font zooms of the 120x40 tab (to 90x30, the narrow layout): the suggestion redrawn whole, ↑ giving it the focus and its footer
PASS 70.10 Live, each CLI on its own, the final build, a temporary project, the configuration check PASS before and after each run, three model turns each: Claude Code 2.1.303 on Haiku 5.5 (95 s, session 1fba98f0) and Codex 0.163.0 on gpt-6-sol (87 s, thread 01a18c9d). Each: a session started from the home screen, no suggestion; told an objective and a decision (ORCHID-70 and TEAL-70; GRANITE-70 and AMBER-70); quit; resumed with Ctrl+R: the note and the suggestion; ↑ → Enter (Not now): the suggestion gone, no message, the badge Idle six seconds later; quit; resumed again: the suggestion; Win+Down (59 rows to 40) and maximised again: drawn at both; words typed and a click on Prepare summary: the toast, the words and the suggestion kept, nothing sent; the words cleared, ↑ Enter: the request in the draft, the suggestion gone, nothing sent six seconds later; Enter: a summary naming the objective and the decision, no approval asked, zone A after it 5% of 1M† │ 89.4k │ $0.01 and 6% of 258.4k │ 33k │ N/A (Codex reports none); an ordinary message answered "DONE", the suggestion not back, zone A 136k and 49.7k
PASS 70.11 The checks: go build ./... ok; go vet ./... ok (Windows and Linux); gofmt -l . empty; go test ./... ok (16 packages, 309 s, 12:53) after the last code edit; 34 mutants through -overlay on the final code, the repository never written (its hash 72752151 before and after): 33 caught by a failing test, A6 not, an equivalent mutant (Details), none by the compiler, none void; -race on internal/app, internal/render and internal/state: 0 race reports; the timing step alone, on mains power, at 12:30:33: its 21 tests within their targets
PASS 70.12 The documents: brief amendment 72 (the owner's words); ui-design.md (Zone B, The summary suggestion: its rows, once, keys, mouse, Prepare summary, Not now, the request word for word and its limits; the Transition; Focus and keys' line and B's entries' ↓; TestKeyOverlayRows ok, the overlay's group within 80 columns); events-and-state.md (the Session rows, the Turn row and the offer's paragraph; TestStateMachines ok); architecture.md (The summary suggestion after a resume); AGENTS.md's status and MVP scope (TestAgentsMDWithinCodexLimit ok, 21,830 bytes); plan.md's plan 70; this file
Not verified: the owner's visual review (the colours, the filled button: the shots in Details); a window edge dragged by hand (Win+Down and the font zoom were the automated resizes); an image attached by hand in Windows Terminal (the tests attach one through both adapters); a summary turn that asks for an approval (neither CLI asked in the live runs); a session whose context the CLI has compacted (not produced: it needs a long session); Codex in the stand-in harness (the stand-in is Claude Code's; Codex in Windows Terminal is the live run's); a crash resume and a mock run in Windows Terminal (the tests' alone)
Verdict: DONE
Likely cause: -
Proposed next step: the owner's manual acceptance (below) and review of this report and the diff, then the commits on the owner's approval - waiting for approval
```

The recommendation: **PENDING MANUAL ACCEPTANCE**. Every automated check
and both live runs pass; what is left is the owner's: the look, a window
edge dragged by hand, and the scenarios on the owner's own sessions.

## Revision 1: the owner's card

The owner reviewed this report and its shots (brief amendment 73): a
provisional pass for the function, the four refinements accepted, and
one weakness, the suggestion too close to ordinary conversation text. The
owner's design, a card, replaces its look; nothing else changes. Only the
drawing's code changed: `internal/render/summaryoffer.go` (the card),
`palette.go` (two styles), one argument in `dashboard.go` (B's rows for
the card's height), and the tests (`internal/render/summaryoffer_test.go`
rewritten, one frame check in `internal/app/summaryoffer_test.go`). No
state, application or adapter code changed: the keys, the clicks, the
draft's protection, the request and what the CLIs receive are those
checked above.

```text
Evaluation - Plan 70, revision 1: the owner's card (amendment 73) - 13 criteria: 13 PASS
PASS 70.1 The offer: no state code changed (the revision's code diff is internal/render alone); TestSummaryOfferSet, TestSummaryOfferNever, TestSummaryOfferKept and the machines' three cases pass in the suite; S1 to S6 caught again
PASS 70.2 The suggestion drawn, revised by amendment 73: TestSummaryOfferDrawn, both CLIs, at 80x24, 90x30, 120x40 and 200x60 (B's 11, 17, 27 and 47 rows): the card cell for cell against the owner's design, its twelve rows at the last three and its seven at 80x24, at the left of B's inside under the resumed note, a blank row before and after it, every frame row the screen's width; every cell of the two buttons a click's target, no other cell of the card; TestSummaryOfferRows, every width from 1 to 80: none under 7 columns, else min(w, 55) wide, its edges whole, margins of two cells, the title whole from 34 columns, SUGGESTION alone from 16, none under, the buttons side by side from 41, one rule as wide as the texts, every span decoration, every text whole once wrapped, the design's twelve rows and the blank row from 46; TestSummaryOfferHeights, every room from 1 to 30 rows: the design from 17, without the rule's blank rows from 15, without the edges' too from 13, without the question's too below, edges, title, texts and rule always whole; TestSummaryOfferFollows: after a 60-line answer, B following its end, the card whole, its bottom edge above the last blank row, at the four sizes; TestSummaryOfferFocus, on the true-colour and the 16-colour palettes: without the focus both buttons in the title colour, bold; with it the selected one's brackets bright and bold, its label bright, bold and underlined, the other subdued, not bold; no span of the card with a ground or inverse, none written in its escape codes; the frame the thinking card's style (#7C6BB3; 16 colours 5), the title in the title colour, the question bold, the rule in the frames' grey, the notes grey; TestSummaryOfferHidden: as before; the golden frames unchanged
PASS 70.3 The keys: TestSummaryOfferKeys, TestSummaryOfferNotNow and TestSummaryOfferScrolled pass for both CLIs with the card; A1 to A16 caught as before but A6, the equivalent mutant
PASS 70.4 The mouse: TestSummaryOfferMouse and the clicks of TestSummaryOfferNotNow and TestSummaryOfferDraftKept pass on the card's cells for both CLIs; C5 (the buttons' panel dropped) caught by TestSummaryOfferDrawn and TestSummaryOfferFocus
PASS 70.5 Prepare summary: TestSummaryOfferPrepare (its frame check now the card's title, buttons and two notes) and TestSummaryOfferDraftKept pass for both CLIs
PASS 70.6 Not now: TestSummaryOfferNotNow passes for both CLIs
PASS 70.7 Sent by Enter: TestSummaryOfferPrepare and TestSummaryOfferFirstMessage pass for both CLIs: the request byte for byte, written as an ordinary message, no settings change
PASS 70.8 Nothing else changes: the revision's code diff is internal/render/summaryoffer.go, palette.go (two styles) and dashboard.go (one argument); zones A and C, the header, Source Control and the home screen untouched; the golden frames unchanged; go test ./... ok
PASS 70.9 Windows Terminal: the card's runs (Details, Windows Terminal, the card), the build of the final code (acc-p70b.exe, sha256 6b6683f0…, a rebuild of the final tree byte-identical), plan 69's stand-in, no quota, 15 runs: Ctrl+R, the card under the resumed note; its keys; a click on each button; a restored draft protected; the window made smaller with the card shown, maximised and at 120x40
PASS 70.10 Live: standing from the first evaluation, not run again: the revision changes no code the CLIs see (the request, the send path and the adapters as they were)
PASS 70.11 The checks: go build ./... ok; go vet ./... ok (Windows and Linux); gofmt -l . empty; go test ./... ok (16 packages, 327 s, 16:10) after the last code edit, its first run at 15:56 failing once in TestSynchronizedOutput (Details); 54 mutants through -overlay, the repository never written (its code's hash 139c1bc6 before and after): 53 caught by a failing test, A6 not, the equivalent one, none by the compiler, none void; -race on internal/app, internal/render and internal/state: 0 race reports; the timing step alone, on mains power, at 16:11: its 21 tests within their targets
PASS 70.12 The documents: brief amendment 73 (the owner's design word for word); ui-design.md (Zone B, The summary suggestion: the design, its colours and why, its size and height); architecture.md (the rendering's row); plan.md (D2, 70.2 revised, 70.12, 70.13, step 8); AGENTS.md's status; this revision; TestKeyOverlayRows, TestStateMachines and TestAgentsMDWithinCodexLimit ok (21,838 bytes)
PASS 70.13 The card in Windows Terminal: at 208x59, 120x40, 90x31 and 80x24 (the grids read from the screens), the design whole at the first three and its seven rows with the note above at 80x24; its pixels the palette's (the frame #7C6BB3, the title #C4B5FD, the question #E5E7EB, the rule #3B3F51, the notes #6B7280); the focus moved with ↑ → ←: the selected button #F9FAFB and underlined, the other #A1A9C0, both #C4B5FD without the focus, the terminal's own #0C0C0C behind them in every state; Win+Down (208x59 to 160x40) and the font zoom (to 90x30): the design whole; the corner dragged with the mouse from 120x40 to 90x31 and 81x25 (the seven rows, ↑ giving the focus), then back to 120x41: the design whole, the focus kept, no row left behind
Not verified: the owner's look at the card; the window dragged by the owner's own hand (the harness's drag is the mouse through SendInput); Codex's card in Windows Terminal (the stand-in is Claude Code's; the card does not depend on the CLI, and TestSummaryOfferDrawn draws it for both); a 16-colour terminal on screen (its styles are tested, not seen); the live runs with the card (not run again, 70.10)
Verdict: DONE
Likely cause: -
Proposed next step: the owner's manual acceptance (the checklist below, revised for the card) and review of this report and the diff, then the commits on the owner's approval - waiting for approval
```

The recommendation stays **PENDING MANUAL ACCEPTANCE**.

### What F2 uses, and the colours chosen

F2 is the approval key ("go to the oldest pending approval"). Its
colours (`internal/render/palette.go`; ui-design.md, Colours and symbols):
- **`cAttention`, `#F472B6`** (256: 205; 16 colours: bright magenta), the
  role "Attention", the symbol "!": "needs the owner (approval,
  question)". Its styles `stAttention` and `stAttentionBold` draw a
  pending approval's or question's frame and heading, its keys once
  focused, the footer's "◆ approval waiting · F2 to answer", zone C's ●
  for an agent waiting for the owner, the turn's "!", the entry cursor's
  ▸.
- **`cPill`, `#3B1531`** (256: 53): the "F2 to answer" pill's ground
  under the pink text in bold (`stPill`), its ends ▐ ▌ (`stPillEnd`); on
  16 colours "[ F2 to answer ]" in bold magenta, no ground.

What it means: an agent blocked until the owner answers. Not an error
(the red `#F87171`, failed or denied), not a warning (the amber of Full
access), not a suggestion. The summary suggestion blocks nothing, so the
pink is not reused: drawn pink, it would read as an approval waiting,
beside the footer's own pink line when one waits. The card takes
existing styles, one new style built from palette colours, and no new
colour:

| Element | Style | Colour | Why |
| - | - | - | - |
| Frame | `stSummaryFrame`, `stThinkBorder` itself | `#7C6BB3` (16: magenta) | zone B's other card's violet, softer than the pink; apart from the zones' frames (grey `#3B3F51`, the focused zone's `#C4B5FD`), so it never reads as a focused panel; the thinking card is never on screen with it (no turn runs while it shows) |
| Title in the top edge | `stTitle` | `#C4B5FD`, bold | the panels' titles |
| Question | `stBold` | `#E5E7EB`, bold | the card's main line |
| Buttons, the card not focused | `stTitle` | `#C4B5FD`, bold | the actions, in the accent |
| The focused button | `stBrightBold`; its label `stSummaryFocus` (new: `stBrightBold` underlined) | `#F9FAFB`, bold, the label underlined | brighter, bolder and underlined: the focus shows without colour |
| The other button, with the focus | `stMuted` | `#A1A9C0`, regular | subdued |
| Rule of hyphens | `stFrame` | `#3B3F51` | the rule between two messages, the input's rule |
| Notes | `stGrey` | `#6B7280` | hints |

No ground anywhere: the inside is zone B's own background, and the
filled button of the first version (`stButton`, ink on `#C4B5FD`) is no
longer used by the suggestion.

### Its size

55 columns, the design's, narrower only where zone B is (B is 72 to 132
columns wide at the four sizes checked, so the card is the design's at
all of them). Its height is the design's where B's conversation rows
hold it with the blank row after it and the resumed note's four rows
above it: 17 rows, B's room from 90x30. At 80x24 B has 11 rows: drawn
whole, the card would push the note out and lose its own top edge and
title as B follows its end. So the card drops its blank rows in a fixed
order (the two around the rule from 16 rows down, the two inside its
edges from 14, the one under the question from 12), never a text, an
edge or the rule: at 80x24, 7 rows, the note and the whole card on
screen together. The design's two blank rows outside the card stay.

### Windows Terminal, the card (70.9 and 70.13)

`scratchpad\term\p70bruns.ps1` on `p70b.ps1` (plan 69's harness with one
step added: the window's bottom-right corner dragged with the mouse,
left button down, eight moves, up), plan 69's stand-in `claude.exe`, no
CLI and no quota, the build of the final code (`acc-p70b.exe`). New
folders for everything (`out-p70b`, `plan70\wt-b`,
`%TEMP%\p40-out-p70b-*`): the first runs' evidence untouched. 15 runs in
301 s, every poll found but one (below); the screens' texts checked in
`plan70\evidence-b\wt-checks.txt`:
- **The card at four sizes**, Ctrl+R in a dashboard on the stored
  session: 208x59 (maximised), 120x40, 90x31 and 80x24 (the grids read
  from the screens). At the first three, the design whole; at 80x24, the
  seven rows with the note above. The pixels of the maximised shot
  (`keyboard-max\k5-zoneb.png`): the frame `#7C6BB3`, the title
  `#C4B5FD`, the question `#E5E7EB`, the rule `#3B3F51`, the notes
  `#6B7280`, on the terminal's own `#0C0C0C`.
- **The focus, by keys**: ↑, Prepare summary bright and underlined (the
  buttons' row: `#F9FAFB`, Not now `#A1A9C0`), the suggestion's footer;
  →, the same on Not now and Prepare summary `#A1A9C0`; ←, back; without
  the focus, both `#C4B5FD`; behind them `#0C0C0C` in every state, no
  rectangle. Enter: the request in the draft, the card gone; Enter again:
  sent, "OK (stand-in)" answered. At 80x24 the request's first words were
  not on screen to find (the editor shows its end, the cursor there): the
  draft's end and the card's absence are in `k12`, the message sent and
  answered in `k15`.
- **The mouse**: a click on "[ Not now ]": the card gone, nothing sent; a
  click on "[ Prepare summary ]": the request in the draft; maximised and
  at 120x40.
- **The draft restored** at the start, then a click on Prepare summary:
  the toast "The draft is not empty: send or clear it, then Prepare
  summary", the words kept, the card kept, no request put in (`r5`, both
  sizes).
- **Made smaller with the card shown**: Win+Down (208x59 to 160x40) and
  three font zooms (120x40 to 90x30): the design whole, ↑ giving it the
  focus. **The corner dragged** (`drag-90x30`): the window 1139x770 px to
  868x607 (90x31), then 787x505 (81x25: the card's seven rows, the note
  above, ↑ giving the focus), then back to 1138x777 (120x41): the design
  whole, the focus kept, no row left behind.

### The mutants, the card

`scratchpad\plan70\mkmutants70b.mjs` and `mutrun70b.sh`, into
`plan70\mut-b` (the first runs' `mut` kept): the first set's state and
application mutants unchanged (S1 to S6, A1 to A16; that code did not
change), and 32 for the card (C1 to C32): its width, margin, gap, title
and its fallbacks, the rule, the wrapping, the blank rows' order and
thresholds, the room passed, the styles of each part and focus state,
the underline, the frame's colour, the notes' words. 54 mutants, the
code's hash 139c1bc6 the same before and after: 53 caught by a failing
test, A6 not, the equivalent mutant of the first runs; none by the
compiler, none void (`plan70\mutrun70b.out`).

### The checks, the card

In `scratchpad\plan70\evidence-b`:
- `go build ./...`, `go vet ./...` (Windows, and `GOOS=linux`), `gofmt -l
  .` empty, `git diff --check` clean.
- `go test ./...` at 15:56 (`suite.txt`): 15 packages ok, internal/app
  failed once, in TestSynchronizedOutput/Windows_Terminal: "3 frames
  wrapped, want the redraw and one per key". That test (plan 5, unchanged
  since) replays a mock, where no card is ever offered, and types three
  keys 50 ms apart, expecting a frame each; under the suite's sixteen
  packages at once, two keys fell in one frame. Alone, ten runs out of
  ten passed, and under `-race` it passed too. The suite run again at
  16:10 (`suite-2.txt`): 16 packages ok in 327 s, the code unchanged.
- `-race` (WinLibs GCC 16.2.0, `CGO_ENABLED=1`): internal/render and
  internal/state ok in 146 s, internal/app ok in 344 s, 0 race reports
  (`race*.log`).
- The timing step alone, on mains power (battery 95%), CPU load before 15
  to 46%, the owner's usual applications open, no other go process, at
  16:11: its 21 tests PASS, among them a key to its frame in a
  10,000-line draft p95 1.04 ms (1.02 ms in the first run), the banner
  p95 1.62 ms against its 2 ms target (1.97 ms) (`timing-step.log`).
- The card's own cost (`card-bench.txt`, a test put in through
  `-overlay`, not in the tree): a whole frame with the card and without
  it, three runs each: at 80x24, 95 to 113 µs and 241 allocations
  against 82 to 92 µs and 197; at 200x60, 304 to 325 µs and 776 against
  278 to 283 µs and 723. Drawn only while the offer stands.

## The owner's manual acceptance (2026-10-30)

The owner ran the acceptance in Windows Terminal, on Claude Code and on
Codex, and reviewed it through the screenshots of the owner's own review,
none of them saved in this repository or the scratchpad: "Final verdict:
PASS — 10 out of 10 manual scenarios accepted." No further change to the
look was asked for.

| Test | Scenario | Verdict |
| - | - | - |
| M01 | The card's look and its keyboard focus | PASS |
| M02 | Not now dismisses it, no request sent | PASS |
| M03 | Prepare summary fills Claude Code's draft, nothing sent | PASS |
| M04 | Text and a restored draft protected | PASS |
| M05 | Windows Terminal resized by hand, the narrow layout among them | PASS |
| M06 | A real Claude Code summary after a resume | PASS |
| M07 | A draft holding only an image, protected | PASS |
| M08 | A Codex session resumed, the card shown | PASS |
| M09 | Prepare summary fills Codex's draft, nothing sent | PASS |
| M10 | A real Codex summary after a resume | PASS |

What the owner observed:
- **M01, the card accepted as drawn**: the violet frame with "SESSION
  RESUMED · SUGGESTION" in its top edge, the inside on the terminal's
  black, the buttons told apart by their text alone with no rectangle,
  the focused one bold and underlined, the rule of hyphens, and the blank
  row between the buttons and the rule, reviewed and to be kept: "Do not
  remove or modify it."
- **M05, resizing**: the card whole in the normal layout and in the
  narrower one where zone A's full presentation collapses; no clipping,
  no corruption, the draft kept.
- **M06, Claude Code**: the resumed session recalled the objective
  ORCHID-70 and the decision TEAL-70, saying their precise meanings were
  not available rather than inventing them; the summary returned, the
  turn completed, the telemetry updated, no tool run seen.
- **M07, an image alone**: a real JPG of about 2.2 MB attached, the draft
  otherwise empty; Prepare summary kept the image and the card, showed the
  warning of a draft that is not empty, and sent no request.
- **M10, Codex**: the resumed thread recalled GRANITE-70 and AMBER-70; a
  concise summary in about three seconds, no tool run seen; the turn
  completed, the telemetry updated.

This settles Revision 1's items not verified but one: the owner's look at
the card (M01), the window resized by the owner's hand (M05), Codex's card
in Windows Terminal (M08, M09), the real CLIs with the card (M06, M10). A
16-colour terminal on screen stays not verified; the blocks above stay as
they were reported.

Not closure blockers, for a later evaluation, as the owner listed them:
- An ordinary message sent without choosing either button: covered by
  the automated tests (TestSummaryOfferFirstMessage, both CLIs), not run
  by hand.
- A summary's quality on a genuinely long session whose context the CLI
  has compacted: a question of the model's context, not of plan 70's
  function. Nothing here claims that a model recovers what compaction
  lost.

### Verification boundaries

| Category | What |
| - | - |
| Manually verified by the owner, on the real CLIs | the card's look and focus; Not now; Prepare summary on both CLIs, nothing sent until Enter; text, a restored draft and an image alone protected; Windows Terminal resized by hand, the narrow layout among them; the summaries on Claude Code (ORCHID-70, TEAL-70) and Codex (GRANITE-70, AMBER-70), the turns completed, the telemetry updated, no tool run seen |
| Verified by the automated tests and the Windows Terminal harness | the offer's lifecycle: none for a new session, another session named, a refused or failed resume, a crash resume or a mock run; withdrawn by the first message the CLI accepts, kept by a failed write; the keys and clicks; every width from 1 to 80 and every room from 1 to 30 rows; both palettes; the corner dragged by the harness |
| Not manually verified | an ordinary first message sent without either button (automated only); a long, compacted session's summary; a 16-colour terminal on screen |

### Approval, commit and push

**Approved and committed (2026-10-30).** The owner's final report, "Final
verdict: PASS — 10 out of 10 manual scenarios accepted.", allows the
documents, the commits and the normal push once the final checks pass,
and asks for a cleanup inventory before any test data is removed.
Commit 3da7b9d (26 files: 20 modified, 6 added), from the files reviewed
in `git status` and `git diff`, each of them plan 70's: its code and
tests in internal/state, internal/render and internal/app, and its
documents. `git diff --cached --check` reported nothing; no file holds a
CR or a BOM, read from the bytes; a scan of the change found no private
path, secret, placeholder or attribution. In that commit, no code file
and no document a test reads changed after the last whole suite began
(17:12:40): build, vet (Windows and Linux) and gofmt clean, 16 packages
ok in 286 s. The author is `Evengelius <molimoabel@gmail.com>`, with no
attribution line, and the working tree was clean after it. Then the
approval commit: plan.md and AGENTS.md close plan 70, and this record;
AGENTS.md is read by a test, so the whole suite runs on the committed
tree. Not pushed yet: the final checks come first, then the push. The
blocks above stay as they were reported.

**Pushed (2026-10-30).** The checks on the committed tree, 4364af5, the
working tree clean before and after: go build ok, go vet ok (Windows and
Linux), gofmt -l . empty; the committed code hashes to the tree every
earlier check ran on (139c1bc6, the mutants' hash, recomputed from the
commits). A binary rebuilt from the commit differs from the one the
Windows Terminal runs used only by what Go writes into it from git (the
commit, and no longer "modified"). The first whole suite on it (17:20,
`-v`) failed TestSynchronizedOutput/Windows_Terminal once more, as the
15:56 run had (Revision 1, The checks); the test passed 20 times of 20
alone, and the whole suite run again, alone, passed: 16 packages ok in
469 s (17:28 to 17:36). The only other go process then was the owner's
own `go run` behind an idle dashboard, started at 17:03 by `acc.cmd`
during the acceptance. Then commits 3da7b9d and 4364af5 fast-forwarded
origin's main from f3b69f9 to 4364af5, nothing forced, origin's main
read back equal to main (`git ls-remote`). The commit record above,
which says "Not pushed yet" as written, stays as it was, the push
following it.

**No test data removed.** The owner's final report asks for an inventory
first and a separate instruction before anything is deleted; the
inventory went to the owner with the report. Everything stays: the live
runs' projects and their entries in ACC's `projects.json`, Claude Code's
session `1fba98f0-…` and Codex's thread `01a18c9d-…` (the owner's
acceptance resumed both), the harness's temporary folders and the
scratchpad's evidence.

**The test data removed (2026-10-30).** After the push, on the owner's
"Run recommended cleanup", each target checked first against the
inventory, the owner's ACC quit, no ACC, Codex or test CLI running.
Everything went to the Recycle Bin, none of it deleted for good, so each
item can be restored until the bin is emptied (`scratchpad\plan70\cleanup`:
the script, its results and the store's backup):
- The harness's disposable data: `%TEMP%\p70demo`, the 14
  `%TEMP%\p40-out-p70-*` and 15 `%TEMP%\p40-out-p70b-*` folders, five
  rebuildable binaries in the scratchpad, the mutants' 88 source copies
  and their overlays (their logs kept), the harness's seeds and the
  scratch tests (`plan70\wt`, `wt-b`, `card`).
- The registry's two test entries, `acc-p70-live-claude-123906` and
  `acc-p70-live-codex-124102`, alone: the nine others unchanged and in
  order, the file round-tripping byte for byte first, a copy of it kept
  (`projects.json.before`).
- The two test projects' folders, both empty.
- Claude Code's test session `1fba98f0-…`: its project folder (the
  transcript and an empty `memory`) and its empty `session-env` folder; no
  other file named it but this work's own session transcript.

Kept, each for a reason:
- Codex's thread `01a18c9d-…`: Codex's shared databases (its logs, state
  and thread history) record it. Its writer lock went when the owner's
  ACC closed Codex, before the cleanup began.
- The evidence this file names (`scratchpad\plan70\evidence`,
  `evidence-b`, the mutants' logs and summaries, `scratchpad\term\out-p70*`)
  and the scripts; the harness files earlier plans share (`Term.cs`,
  `Extra.cs`, `Burst.cs`, `p69.ps1`, plan 69's stand-in).
- The registry's five older test projects (`acc-demo`, four
  `acc-p39-live-*`): not plan 70's.
- Px, its configuration and its process: outside every cleanup.

**Outside plan 70, for the owner to decide.** TestSynchronizedOutput
(plan 5, unchanged since) failed in two of today's seven whole-suite runs,
each time under the suite's own parallel packages, never alone (30 runs
of 30): it types three keys 50 ms apart and expects a frame for each, and
under that load two keys can share a frame. It replays a mock, where no
summary offer is ever made. Not changed here.

## Details

### What the requirements asked, and where

- **The suggestion** (the owner's section 1): drawn after the resumed
  note, which is unchanged, from `summaryRows`
  (`internal/render/summaryoffer.go`), while `SummaryShown`: the offer
  (`SessionState.SummaryOffered`, `internal/state/state.go`) standing, the
  session `Ready` and the turn `Idle`. Set once per verified resume in
  `started()`; withdrawn by Prepare summary, Not now and the first
  message the CLI accepts (`toRunning`); never stored. Not for a new
  session, a failed or refused resume, another session named, a crash
  resume or a mock run. Zone B hidden: the offer waits.
- **Prepare summary** (section 2): `prepareSummary`
  (`internal/app/summaryoffer.go`) puts the request in the empty draft
  through the editor's own insertion (so Ctrl+Z takes it out), the cursor
  at its end; Enter sends it through the existing path, its draft
  recovery, failure restoration and approvals unchanged. Over text, an
  image or a restored draft: a toast, nothing else.
- **Not now** (section 3): `summaryNotNow`, the state's
  `SummaryDismissed`; nothing written, no token used.
- **Keys and mouse** (section 4): `summaryKey`, `focusSummary`,
  `pastNewest`, the ↑ of `dashboardKey`, `summaryClick`; Enter on an
  unfocused empty input does nothing.
- **The request** (section 5): `summaryRequest`, the owner's words; its
  limits written in ui-design.md ("Its limits": the request asks,
  nothing enforces it; the CLI's permissions and approvals decide; a
  compacted context may lack details).
- **The exclusions** (section 6): no "+" menu item, no Alt+S, no permanent
  button, no automatic summary, no history restored, no transcript read
  or stored, no stored dismissal, no model or effort change, no
  telemetry, no compaction detection; zone A, zone C, Source Control and
  the home screen untouched.

### The design as built, and what changed on the way

The design D1 to D8 stands; four refinements, each written into
`plan.md`'s design and the documents:
- **The offer ends when the CLI accepts the first message** (D1), not
  when it is sent: the owner's "first successfully submitted ordinary
  message"; a write that fails keeps it. While a turn is under way, a
  message being sent among them, the suggestion is not drawn (D2).
- **↑ keeps the focus where no entry stop is above** (D3): right after a
  resume the note is no stop, and ↑ would have dropped the focus for
  nothing.
- **Text a paste puts in the draft ends the focus** (D3), as a typed
  character does: no key comes with a paste, and Enter would otherwise
  have gone to the suggestion.
- **A blank row after the suggestion** (D2), above the input's rule, as
  after the last message (amendment 17): found by
  TestSummaryOfferFollows, the last note against the rule in a
  conversation taller than zone B.

The footer names the suggestion's keys while it has the focus, and F1's
overlay has their group, as every other focus's.

### The model calls

Six, all in the live runs: per CLI, the objective and decision told, the
summary, and the ordinary message after it (three on Haiku 5.5, three on
gpt-6-sol). No model call anywhere else: the tests and the Windows
Terminal runs use fakes and the stand-in; Not now and Prepare summary
make none (L70.5 and L70.8: no message, the badge Idle).

### The summaries

- **Claude Code** (Haiku 5.5): "**Goal:** The stated objective is
  ORCHID-70. The decision reached is TEAL-70. I don't have any further
  detail on either…", then decisions, what was completed ("Only the
  acknowledgment"), what is open and a next step. It also named "two
  `/usage` snapshots": the dashboard's own usage reads (plan 24, "/usage"
  written as a user message, answered with no model call) are in Claude
  Code's context, so a summary can mention them (outside plan 70, below).
- **Codex** (gpt-6-sol): "Goal: GRANITE-70. Decision: AMBER-70 was
  chosen. Completed: I acknowledged that decision with "OK." No files were
  read, tools run, or changes made.", then what is open and a next step.

### The mutants

Three runs of the same set through `-overlay`, three at a time, a mutant
caught at its first failing test (`scratchpad\plan70\mut-run1`,
`mut-run2`, `mut\logs`):
- **First run** (33 mutants, the code at 1c3bb59c): 28 caught; 5 not:
  A9 (the focus without B following its end), A10 (↑ taking the
  suggestion out of view), A11 (↑ with text in the draft taken), R7 (the
  suggestion's rows not counted in the conversation's) and A6. Tests
  added for the first four: TestSummaryOfferScrolled (A9, A10, A11) and
  TestSummaryOfferFollows (R7), which also found the missing blank row;
  R12 added for it.
- **Second run** (34, the code at be508b56): 33 caught, A6 not. One kill
  of A1 also listed TestEntryViewer, which no change of plan 70 reaches
  (no offer there): its thinking phase's exit, a step on the clock, not
  done under three test processes at once; 20 runs alone and 15 under
  load passed. A1 stands caught by TestSummaryOfferNotNow in all runs.
- **A6, equivalent**: ↓ while the suggestion has the focus, taken out of
  its explicit case, reaches the editor, which moves nothing: the focus
  exists only with no text in the draft. Kept explicit for the reader.
- **Final run** (34, the final code at 72752151, one assertion added to
  TestSummaryOfferPrepare after the second run): 33 caught, A6 not, none
  by the compiler, none void; the code's hash the same before and after.

### The checks

- `go build ./...`, `go vet ./...` (Windows, and `GOOS=linux`), `gofmt -l
  .` empty.
- `go test ./...`: 16 packages ok in 309 s, at 12:53:39, after the last
  edit (`scratchpad\plan70\evidence\suite-final.txt`); at 12:30 (250 s)
  before that assertion.
- `-race` (WinLibs GCC 16.2.0, `CGO_ENABLED=1`): internal/app 365.5 s,
  internal/render 156.5 s, internal/state 14.6 s, all ok, 0 race reports
  (`scratchpad\plan70\evidence\race-*.log`).

### The timing step

On mains power (battery 98%), CPU load before 12 to 30%, the owner's
usual applications open (browsers, Teams, Outlook, Notepad, Task
Manager), no other go process; 12:30:33, the command of
doc/evaluation.md, 90 s: the 21 tests PASS. Among them: a key to its
frame in a 10,000-line draft p95 1.02 ms, in a 50,000-line one 3.01 ms;
the conversation's search over two hours p95 7.18 ms (answers ten times
longer 12.58 ms); the banner p95 1.973 ms against its 2 ms target; a
2,000-line answer's delta p95 5.09 ms (`evidence\timing-step.log`).
The suggestion itself is not timed: drawn only while the offer stands,
right after a resume, its rows are three short texts wrapped and seven
rows a frame.

### Windows Terminal (70.9)

`scratchpad\term\p70runs.ps1` on plan 69's `p69.ps1`, the setup
`plan70\seed70.ps1` seeding each run's temporary app data from the first
runs' (`plan70\wt`); the shots and texts in `scratchpad\term\out-p70\<run>-<size>`
(the zone B crops `*-zoneb.png`). Each run's home, app data and PATH are
temporary: nothing of the owner's is read or written. The colours as
drawn: the question and the buttons in the title colour, the focused
button filled with ink on it, the notes grey (`keyboard-max\k5-zoneb.png`,
`k7-zoneb.png`; `smaller-90x30\s7.png`). That was the first version's
look; the card's is in Revision 1.

### The live runs (70.10)

`scratchpad\term\p70live.ps1`, the evidence in
`scratchpad\term\out-p70-live-claude` and `out-p70-live-codex`
(results.txt, a shot and its text at each step, the configuration
compare). The owner's recent projects were counted first (8, then 9):
the cap of 10 dropped none of them (L70.11). In the Codex run's
`07c-protected.png` the footer's toast is under the taskbar, which popped
up over the window; the script read the toast in the screen's text
before the shot.

### Kept, as the owner asked (no test data deleted)

- The live runs' projects, `%LOCALAPPDATA%\Temp\acc-p70-live-claude-123906`
  and `acc-p70-live-codex-124102`; their entries in ACC's
  `projects.json`, each with its session (the store now holds 10 recent
  projects, its cap, and the favourite: the next new project drops the
  oldest recent one); Claude Code's session `1fba98f0-…` and Codex's
  thread `01a18c9d-…` in each CLI's own history.
- The Windows Terminal runs' temporary folders, `%TEMP%\p70demo` and 14
  `%TEMP%\p40-out-p70-*`; for the card, 15 `%TEMP%\p40-out-p70b-*`.
- The scratchpad's evidence, scripts and mutants (`scratchpad\plan70`,
  `scratchpad\term\out-p70*`); for the card, `plan70\evidence-b`,
  `plan70\mut-b`, `plan70\wt-b`, `plan70\card` and `term\out-p70b`.

### Outside plan 70, for the owner to decide

- **The usage reads in Claude Code's context.** The summary named "two
  `/usage` snapshots": since plan 24 the dashboard reads Claude Code's
  usage by writing "/usage" as a user message, which Claude Code answers
  with no model call but keeps in the session's context. Harmless to the
  summary's substance; a summary may mention them.
- **The recent projects at their cap** after the live runs (above).

## Manual acceptance checklist (for the owner)

Plan 70 checked by hand in Windows Terminal, each CLI on its own: a
result on one is not one on the other. **Quota: three turns per CLI**
(2.2, 6.2 and 6.3: three on Haiku, three on Codex's model); a resume, the
suggestion, Not now and Prepare summary make no model call. **Nothing is
deleted or reconfigured**, and no recent project is added (ACC's store is
at its cap of 10, revised for Revision 1): the live runs' projects are
used again, `%TEMP%\acc-p70-live-claude-123906` and
`%TEMP%\acc-p70-live-codex-124102`, already among ACC's recent projects;
each test session stays in its CLI's own history; `ANTHROPIC_MODEL` lives
in one tab only. `acc.cmd` runs the working tree (`go run`), the code the
Windows Terminal runs built. Each step: what to do, what to see, PASS
when it matches; record it only once seen.

**1. Prerequisites** (no quota)
- 1.1 In any PowerShell tab:
  `Test-Path "$env:TEMP\acc-p70-live-claude-123906", "$env:TEMP\acc-p70-live-codex-124102"`.
  PASS: True, True.
- 1.2 Maximise Windows Terminal; in a PowerShell tab:
  `Set-Location "$env:TEMP\acc-p70-live-claude-123906"`, then
  `$env:ANTHROPIC_MODEL = 'haiku'`, then
  `& C:\javadev\projets\agent-command-center\acc.cmd` (it opens ACC's
  9 pt tab). PASS: the home screen, Claude Code selected, the project
  field ending in `acc-p70-live-claude-123906`, and "→ Claude Code's last
  session here: … · Ctrl+R continues it" (the live run's session
  1fba98f0, which knows ORCHID-70 and TEAL-70). For the card alone, with
  no quota, go to 3.2 now.

**2. A conversation to summarize** (1 turn; the project's last session
becomes this new one, the earlier one staying in Claude Code's history)
- 2.1 Enter (START SESSION). PASS: "Ready to work on this project.", and
  no suggestion in zone B.
- 2.2 Type `This is a short planning conversation; do not read files or run anything. Our objective is SAFFRON-70. We decided COBALT-70. Reply with the single word OK, and nothing else.`
  and Enter. PASS: "OK" answered. **1 turn.**
- 2.3 Ctrl+Q, then Ctrl+Q again if it asks.

**3. The suggestion** (no quota)
- 3.1 `& C:\javadev\projets\agent-command-center\acc.cmd` again. PASS:
  "→ Claude Code's last session here: … · Ctrl+R continues it".
- 3.2 Ctrl+R. PASS: zone B's "Resumed Claude Code session <8
  characters> · …" unchanged, and under it, once, the card of amendment
  73 (revised after the owner's review):

  ```text
  ╭─ SESSION RESUMED · SUGGESTION ──────────────────────╮
  │                                                     │
  │  Want a summary of your earlier work?               │
  │                                                     │
  │  [ Prepare summary ]     [ Not now ]                │
  │                                                     │
  │  -------------------------------------------------  │
  │                                                     │
  │  The request will be added to your draft.           │
  │  Press Enter to send it.                            │
  │                                                     │
  ╰─────────────────────────────────────────────────────╯
  ```

- 3.3 Judge the look: the frame a soft violet, its title in the top
  edge, pale violet and bold; the question bold; the rule of hyphens
  subdued grey; the notes grey; the buttons pale violet and bold; no
  filled background anywhere; nothing cut. PASS: as you want it.
- 3.4 ↑. PASS: "Prepare summary" bright white, bold and underlined
  between its brackets, no light rectangle behind it, "[ Not now ]"
  subdued grey; the footer "←→ select · Enter choose · ↑ entries · Esc
  back to the input · F1 keys". → moves the bright, underlined label to
  "Not now", ← back; Esc: both buttons pale violet again.
- 3.5 F1. PASS: the overlay lists "THE SUMMARY SUGGESTION (after a
  resume, ↑ from an empty input)". Esc closes it.

**4. Not now** (no quota)
- 4.1 ↑, →, Enter. PASS: the suggestion gone, the input empty, no new
  message in zone B, the badge "Idle", zone A's values unchanged.
- 4.2 Ctrl+Q (twice if asked).

**5. Draft protection and resizing** (no quota)
- 5.1 `& …\acc.cmd`, Ctrl+R. PASS: the suggestion again.
- 5.2 Type `my own words`, then click "[ Prepare summary ]". PASS: the
  footer's toast "The draft is not empty: send or clear it, then Prepare
  summary"; your words unchanged; the suggestion still there; nothing
  sent.
- 5.3 Ctrl+A, Backspace; "+", Upload from computer, choose any image;
  click "[ Prepare summary ]". PASS: the same toast, the image's chip
  kept, nothing sent. Remove the image (✕ on its chip).
- 5.4 With the suggestion shown, drag the window's corner to make it
  smaller, down to about 80x24 (just above "Terminal too small"), then
  larger again. PASS: the card whole within zone B at every size, its
  frame and title never cut; under about 30 rows its blank rows go
  (first the two around the rule, then the two inside its edges, then
  the one under the question), the resumed note still shown above it at
  80x24; no leftover rows once the window grows back, the card as in
  3.2.
- 5.5 (Optional, a restored draft) Type `restored words`, Ctrl+Q twice,
  `& …\acc.cmd`, Ctrl+R. PASS: the toast "draft restored", the words in
  the input, the suggestion shown; a click on "[ Prepare summary ]" gives
  the toast of 5.2. Then Ctrl+A, Backspace.

**6. Prepare and send** (2 turns)
- 6.1 With the input empty, click "[ Prepare summary ]" (or ↑, Enter).
  PASS: the request in the input, word for word, the cursor at its end,
  the suggestion gone; wait ten seconds: nothing sent, the badge "Idle".
  Ctrl+A, Backspace: the suggestion does not come back; Ctrl+Z: the
  request back.
- 6.2 Enter. PASS: a summary naming SAFFRON-70 and COBALT-70 (or saying
  what it lacks); zone A's values updated after the turn. If an approval
  card comes, answer it as you wish and note it (the request asks for no
  tool and enforces nothing). **1 turn.**
- 6.3 Type `Thank you. Reply with the single word DONE, and nothing else.`
  and Enter. PASS: "DONE", the suggestion not back. **1 turn.**
- 6.4 Ctrl+Q (twice if asked).

**7. Codex** (3 turns): 1.2 to 6.4 again in a new tab, without
`ANTHROPIC_MODEL`, in `$env:TEMP\acc-p70-live-codex-124102` (its last
thread 01a18c9d knows GRANITE-70 and AMBER-70), OpenAI Codex chosen on
the home screen (↓) before Enter or Ctrl+R, the words `OPAL-70` and
`CEDAR-70`. Codex's note reads "Resumed OpenAI Codex thread …" and "its
earlier turns are not shown here".

**8. Not offered** (no quota, either CLI)
- 8.1 A new session: 2.1 checks it.
- 8.2 A mock run: `& C:\javadev\projets\agent-command-center\acc.cmd --mock C:\javadev\projets\agent-command-center\testdata\mock\claude-subagent-approval.jsonl`.
  PASS: no suggestion.
- 8.3 (Optional) A crash resume: during a resumed session with the
  suggestion shown, end the CLI's process (Task Manager, the `claude.exe`
  or `codex.exe` under ACC's tab); "The session ended unexpectedly";
  Ctrl+R. PASS: the session back, no suggestion.
