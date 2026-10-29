# Plan 66 evaluation (2026-10-29)

The check of plan 66, the conversation's search (brief amendment 68):
Ctrl+F opens a bar at the foot of zone B; what B holds as text is
searched, its folds and a long paste's folded lines included, the case
ignored unless the query has a capital; the newest match first, ↑ or
Enter to the older ones, ↓ or Shift+Enter to the newer; the matches lit on
the file viewer's amber, B scrolled to each; Tab shows the current match's
entry whole. Nothing goes to the CLI, nothing is written, the reducer is
unchanged. Another agent's two reviews, shared by the owner, were folded
into the design; the second, sent as the owner's own message, approved it:
"You may now proceed with the implementation of Plan 66, following the
approved design."

It covers the 12 criteria of [`plan.md`, Plan 66](../plan.md), written
before the code. Nothing is committed or pushed: the owner reviews the
results and the shots first.

How it was checked:
- **Render tests** (`internal/render/search_test.go`, 14 tests): the
  matcher on conversations built entry by entry and on sessions of the
  probe's shape against the file viewer's way of matching, the places a
  search compares with, the lighting on drawn frames, the targets'
  references, the full view's light.
- **Application tests** (`internal/app/search_test.go`, 19 tests) on the
  scripted CLI, a wire rig of the real Claude Code adapter and the
  Explorer's rig: the bar, its keys and field, the steps and the reveal,
  streaming, closing, Tab; the golden frames `search`, `search-none` and
  `search-codex` (Codex's mock) at 79x24, 80x24, 120x40 and 200x60.
- **The timing test** `TestSearchToFrame` (`searchtiming_test.go`, the
  `timing` tag) in the timing step, alone.
- **Each change broken on purpose** through `go test -overlay`, the
  repository never written, its code hashed before and after: 64
  mutants of the render and application parts
  (`scratchpad\plan66\mkmutants66.mjs`, run by plan 51's mutrun).
- **Windows Terminal**: the application built from this tree, in ACC's
  profile, a scratch mock with a folded run and a long paste
  (`scratchpad\plan66\mk66mock.mjs`), maximised (208x59), at 120x40 and at
  80x24 (`scratchpad\term\p66run.ps1`).
- **The race detector** on the search's tests in the two packages the plan
  changed, and **build, vet (Windows and Linux), gofmt, the whole suite**
  after the last code and document edit.

## The evaluation

```text
Evaluation - Plan 66, the conversation's search - 12 criteria: 10 PASS, 1 FAIL
PASS 66.1 What is searched: TestConvMatchesKinds: a query found in each kind D3 names, with its entry, line and offset: an owner's message ("alpha"), a long paste's folded line 12 ("refused"), the CLI's message inside bold ("beta"), a Read's target by its folder, the user's folder hidden ("gamma"), a command's target ("delta"), a failed call's error line as received ("epsilon"), an agent's description ("zeta"), a Write's target ("notes.md"), an approval's row ("eta request"), an answered request's target ("upsilon") and a waiting one's ("phi --"), a notice, an error, a dashboard note, a folded run's first call ("lambda0"), a thinking phase's details line; nothing found in a call's input past its target ("omicron"), its output ("pi output"), an approval's or an error's later lines, the turn review's row ("rho"), zone C's agent ("sigma"), the user folder's name ("name"), what a request would write ("chi body"), the draft in the editor's view ("tau"), the labels ("YOU", "CLAUDE", "RESULT")
PASS 66.2 Matching: TestConvMatchesCase: "error" finds ERROR, Error and error, "Error" and "ERROR" one each, école, дом and αλφα their capitals, a capital query only itself; TestConvMatchesLiteral: a.b, (x), \d and * found as typed, "ee" in "eeeee" twice, never overlapping; TestConvMatchesLikeRegexp: 311 queries (11 fixed, 300 cut at random from the text, a third with a capital first) on 12 turns of the probe's shape, exactly the matches of a regexp built as the file viewer's findAll builds it (QuoteMeta, (?i) without a capital), the probe's two full sessions checked the same way at step 1 (plan.md, What was found); TestConvMatchesCap: 12,000 matches, the newest 10,000 kept, from entry 119's last to entry 20's first, Capped; TestSearchCounts: "match 1 of the newest 10,000"; TestSearchCapStart: 12,000 matches, B three pages up: Capped, "of the newest 10,000", the first current match the newest at or above B's last row shown and not the newest of all; TestFindIn: the scan's edges, a 10,000-letter line read once
PASS 66.3 The bar: TestSearchOpen: Ctrl+F from the input, zone A and zone C, the focus in B, Esc giving each back; from the diff view in B's place, closed; in the draft window its full screen as before; over the key overlay and an entry's full view no bar; TestSearchEntryCursor: from B's entries, their cursor off; TestSearchDockedPanel: from the Explorer beside B at 200x60, the panel kept; at 120x40, in B's place, the panel hidden as B's focus hides it; over the quick open and the file viewer no bar; TestSearchField: typing, Backspace, ← → Home End and a paste's first line edit the query, AltGr's "@" as Windows Terminal sends it (left Ctrl and right Alt) typed, Ctrl or Alt alone typing nothing (">need-le!@"); TestSearchClosesPopups: over Source Control's commit dropdown, which takes no key of its own, the dropdown closed, the bar on top, the keys its; the golden frames search ("match 1 of 8"), search-none ("no match · calls' inputs and outputs not searched") and search-codex ("match 1 of 24", Codex's mock) at 79x24, 80x24, 120x40 and 200x60, "type to search the conversation" and the footer's "↑ Enter older · ↓ Shift+Enter newer · Tab whole · Esc close · F1 keys" in them; no existing golden frame changed (no tracked frame in git's diff)
PASS 66.4 The matches in turn: TestSearchSteps: the first match the newest ("One more needle"), ↑ and Enter older, the number one more each time, ↓ and Shift+Enter newer, round at either end, B held at each; TestSearchStartsInView: B scrolled up, the first match the newest at or above its last row shown; TestSearchPlaces: a row shown stands for its entry up to its last byte, a match from the next byte on below it, a long paste's fold row for its folded lines; TestSearchReveal: a match on a long line's third row or later, above the view, B's top that row, the least scroll up; the newest below the view, on B's last row, the least scroll down; TestSearchFolds: a folded run's call unfolds the run, a long paste's folded line 12 shows its "… N more lines" row; TestSearchThinkingDetails: a phase's details unfold; typing keeps the current match (TestSearchField, TestSearchStreaming's numbers)
PASS 66.5 While the turn goes on: TestSearchStreaming: an older match current, a new prompt and a streamed answer with the bar open, 3 matches more, the current match's number 3 more, the same match, B's top the same, the bar "match N+3 of M+3", a new turn's prompt among them; TestSearchGone: the current match in a streaming answer, the answer ending with other words: one match fewer, the nearest at or above its place current ("One more needle"), B's top the same; TestSearchKeepsPart: a call's name current, its target matching before it on the row, kept by its part after an event
PASS 66.6 Lit: TestSearchLit: at 120x60, 7 matches on the amber, in a message, inside bold text, code and a link's text, in CJK text, in a call's visible target and in a failure's error; the "…" of a target whose cut hides the match lit; the current match, in a long paste's folded lines, lights its "… N more lines" row; a selection drawn over a match; the copy byte for byte as without the search, tool rows included; nothing lit with the bar closed; TestSearchLitApproval: an answered request's visible target lit, a cut one's "…" lit, a waiting request's target row lit as current, the copy unchanged; TestSearchLitName: a target not drawn lights the call's name; TestRetarget: a path's left cut, its "…" standing for the bytes it hides, a description's right cut
PASS 66.7 The whole entry: TestSearchWhole: Tab on an answer opens nothing; on a match in a 300-line paste's folded lines, its full view with "connection refused" in view, not at its top; Home there moves the view, the bar kept; Esc back to B and the bar, the same current match; Tab on a Read's target, the call's full view at its top, Esc back; TestSearchLitView: the full view's match lit, one its wrap splits lit on both rows (row 20 of the view)
PASS 66.8 Closing: TestSearchClose: Esc closes, B held where the search left it; F1 opens the overlay over the bar, kept after; F6 closes the bar and moves the focus; a click on zone C closes it, the focus C's; one on "+" closes it, the menu then on top; TestSearchCloseKeys: F2, F3, F4 (its toast: no modes in this rig), F6, Shift+F6, Alt+P, Alt+B, Alt+G and Ctrl+W each close the bar, F4, F6, Shift+F6 and Ctrl+W seen acting; F12, Shift+PgUp and Ctrl+O act with the bar kept; Ctrl+G and Ctrl+K do nothing, the draft window shut, the query and the draft unchanged; TestSearchOpen, TestSearchEntryCursor and TestSearchDockedPanel: the focus back to the input, A, C, B's entries (their cursor on its entry again) and the docked panel, and to the input when Ctrl+F closed the diff view or the panel gave way; TestSearchField: the draft's text, cursor and selection byte for byte the same from Ctrl+F to Esc, nothing sent; TestSearchNotWritten (the real Claude Code adapter, a drafts folder of its own): the query typed and pasted, in no file of the drafts' folder and in no line written to the CLI, the stored draft as it was
FAIL 66.9 Timing: expected TestSearchToFrame within 16 ms at p95 and the timing step's other tests within their targets; found TestSearchToFrame within (a key in the bar to its frame at 200x60, 407 keys: p95 7.15 ms on two hours of conversation, 9.91 ms on answers ten times longer, a 2,000-line paste's hidden match among them; with the bar open, a delta to its frame, 400 deltas: p95 4.10 ms), 19 of the other 20 within theirs, and TestBannerToFrame, the home screen's banner, p95 2.315 ms over its 2 ms (the timing step alone, on mains power, 14:55:02, 87.3 s)
PASS 66.10 The documents: ui-design.md, Zone B, Search, its frame golden frame search's rows 2 to 26 in cells 36 to 111 byte for byte (scratchpad\plan66\docframe.mjs: TestDocFrames reads no document), Focus and keys' Ctrl+F row and the search bar's table, the key overlay with them (TestKeyOverlayRows, 104 rows, none cut at 80 columns); architecture.md, The conversation's search; evaluation.md's timing step with TestSearchToFrame, twenty-one tests; brief amendment 68; AGENTS.md's status and MVP scope (21,684 bytes, TestAgentsMDWithinCodexLimit)
PASS 66.12 The checks: go build ./... ok; go vet ./... ok (Windows and Linux); gofmt -l . empty; go test ./... ok (16 packages) after the last code edit; 64 mutants through -overlay on the final code, the repository never written (its hash the same before and after): 61 caught by a failing test, R6, R21 and A5 equivalent (Details, The mutants); -race on internal/app's search, golden-frame and key tests and on the whole of internal/render: no race; the timing step alone, on mains power (66.9)
Not verified: 66.11's last part, the owner's look at the shots. Its measured part holds: in ACC's profile, Windows Terminal 1.23.20211.0, maximised (208x59), at 120x40 and at 80x24, the final build and the scratch mock: Ctrl+F, "quota" typed, "match 1 of 3" lit in the newest answer; ↑, "match 2 of 3", the folded run open, B at quota.go's row, its "quota" lit; ↑, "match 3 of 3", the paste's "… 55 more lines" row lit at B's top; Tab, the paste's full view, line 40 lit; Esc, B and the bar; Esc, the bar closed, B where the search left it, the input's caret back (24 shots)
Verdict: NOT DONE
Likely cause: 66.9: the workstation's load, 11 to 25 % CPU before the run (The timing step, below): the banner's code is not plan 66's, and on a quieter machine, 0 to 16 % CPU, its p95 on HEAD's tree and on this one in turn was the same, 1.645, 1.576 and 1.566 ms against 1.619, 1.517 and 1.550 ms, all within the 2 ms
Proposed next step: 66.9: the owner's decision, a documented timing exception as in plan 65 or the timing step again on a quiet machine; 66.11: the owner's look at the shots and this report - waiting for approval
```

**66.11, and the review the owner shared (2026-10-29).** The owner shared,
with no words of their own, another agent's review of the shots and of this
report: its verdict in French ("APPROVED — 9,5/10"), then the review in
English. It approves the shots, "I approve criterion 66.11 — the
independent visual review", taken as 66.11's look: PASS, so 11 PASS and 1
FAIL. It requests no change, visual or technical. It took the narrowest
shot it saw for about 90x30 and accepted 80x24 as reported; the 80x24 shot
sent is `search-60x18\paste.png`. It accepts the three equivalent mutants,
their evidence kept (The mutants), and recommends that 66.9 be accepted as
a documented exception: its FAIL, the timing step's 20 PASS and 1 FAIL and
the 2 ms target kept as they are, and no run made only to reach 21 PASS. It
leaves that decision and the commit's authorization to the owner; this
report waits for both.

**The timing exception, accepted (2026-10-29).** The owner sent, word for
word: "I accept the documented timing exception for criterion 66.9, under
the same principles applied to Plan 65." The measured FAIL stays recorded,
never reclassified as a PASS, its threshold unchanged, no test run again
only to pass; the three equivalent mutants' evidence stays as written. The
same message allowed the final commits and the closure of plan 66, the
diff checked first to hold plan 66's files alone, and the push once the
commits and the final checks are done. Recorded as asked:

| | |
| - | - |
| Measured result | FAIL: TestBannerToFrame p95 2.315 ms, target 2 ms, unchanged |
| The complete timing step, 14:55:02 | 20 PASS, 1 FAIL; not all twenty-one |
| Likely cause | the workstation's load (11 to 25 % CPU before the run), supported by the comparison with HEAD's tree in turn |
| Regression | none of plan 66's shown by the evidence: the banner's code is not plan 66's, its p95 the same on both trees in turn |
| The owner's decision | a documented timing exception, accepted |
| Status | accepted with an exception, not a PASS |

Plan 66: 12 criteria, 11 PASS and 1 accepted with a documented exception,
66.9 (TestBannerToFrame's timing at 14:55). The AltGr characters that the
input's editor and the other fields drop stay apart, for the next plan
(the owner: "Keep the pre-existing AltGr issue separate for the next
plan.").

**Approved and committed (2026-10-29).** Commit a5116aa (39 files: 21
modified, 18 added), from the files reviewed in `git status` and `git
diff`, each of them plan 66's. `git diff --cached --check` reported the
three 79x24 golden frames' trailing blank rows alone ("new blank line at
EOF"): the too-small screen drawn at the terminal's 24 rows, as every
79x24 frame is and TestGoldenFrames requires, plan 65's four the same
(`git diff --check 3757c15~1 3757c15`). No file holds a CR or a BOM, read
from the bytes. A scan of the change found no leftover, private path or
company name; two test comments that cited the scratchpad's probe file
cite plan 66's findings instead, checked by vet, gofmt and the render
package's search tests. The whole suite had passed after the last change
of behaviour, and no test reads the documents edited since. The author
is `Evengelius <molimoabel@gmail.com>`, with no attribution line, and the
working tree was clean after it: PASS. Then the approval commit: plan.md
and AGENTS.md close plan 66, and this record; `go test ./internal/arch`,
which reads AGENTS.md, ok after its edit. Not pushed yet: the final
repository checks come first, then the push the owner allowed. The block
above stays as it was reported.

**Pushed (2026-10-29).** The owner allowed the push once the commits and
the final repository checks were done ("You may push once the commits and
final repository checks are complete."). The checks on the committed tree,
219d71f, the working tree clean before and after: go build ok, go vet ok
(Windows and Linux), gofmt -l . empty, go test ./... ok (16 packages). Then
commits a5116aa and 219d71f fast-forwarded origin's main from 250df5f to
219d71f, nothing forced, origin's main read back equal to main (`git
ls-remote`). The commit record above, which says "Not pushed yet" as
written, stays as it was, the push following it.

## Details

### What the build found and changed

- **The targets' memo** (D4 revised at step 2). Each call's target is
  read from its input as its row reads it (`toolDetail`): read afresh at
  every keystroke, the two-hour session's 4,800 calls cost 4.8 to 8.5 ms a
  search. The bar keeps each call's and each request's target, one string
  each, from its opening to its closing (`SearchMemo`), and drops them.
  No text of the conversation is copied; no command reads the state.
- **The closing keys** (D9 refined at step 3). Shift+F6 closes the bar as
  F6 does; Ctrl+O, Ctrl+Shift+C, Shift+PgUp and Shift+PgDn act with the
  bar kept, as F1, F12 and Ctrl+C do: keys of Anywhere that leave the
  conversation as it is.
- **A call with no target** (step 4). `markTarget` gave a call with no
  target an empty line instead of none, and its row drew a target column
  ("Bash   ·"); `TestFailedToolRow` failed: nil is kept nil.
- **A request's target** (D3 made explicit at step 5). Writing
  ui-design.md's frame showed the Write request's "….txt" drawn and not
  searched, since a request's row searched its name alone. Its target is
  now searched and lit as a call's, answered or waiting (`entryParts`,
  `answeredRows`, `pendingBox`): the frame's count went from 7 to 8.
- **The F1 page's Shift+Enter** (step 5). The bar's own "↓ Shift+Enter"
  row made `TestConfiguredKeysShown` fail: it refused the text
  "Shift+Enter" anywhere on the F1 page once Shift+Enter is no newline
  key. It now refuses it on every row but that one, its other checks as
  they were.
- **The timing test's own setup** (step 5). Its first run failed: the
  rounds of typed and erased queries ended with "te" in the field, so the
  hidden match was looked for as "terefused". The field is emptied first.
- **B's entries** (D9, step 7). Esc gave the input the focus even when
  Ctrl+F came from B's entries, which D9 names apart from the input; it
  now gives their cursor back on its entry, on its review row while that
  row is there.
- **The compact rule's words** (D2, step 7). The docked panel takes zone
  B's place under 100 columns, and also wherever B would keep fewer than
  64 columns beside it, as at 120x40 (`PanelCompact`); D2 and ui-design.md
  said only the first.
- **The tests read against the criteria** (step 7). Eight parts of the
  criteria had no test; each has one now (plan.md, Progress, step 7).
- **A row's place** (D5, step 7, found by mutant R17). B's last row shown
  stood for its entry up to the end of its bytes, exclusive, read as
  inclusive: a match starting the next row, below the view, could be the
  first current one. The place is now the row's last byte shown
  (`rowEnd`), pinned by `TestSearchPlaces`.
- **AltGr** (step 7, found by mutant A9). The bar dropped every character
  typed with AltGr. Measured in Windows Terminal 1.23 on the owner's
  Belgian AZERTY layout, TEA_TRACE recording what arrived: AltGr+2 comes
  as the record `[50;0;64;1;41;1_`, "@" with left Ctrl and right Alt, the
  decoder gives it its text, and the test of no Ctrl and no Alt dropped
  it. The bar now takes a key's text with neither or both (`keyText`); the
  probe typed "/ @". **Outside this plan:** the same test drops AltGr's
  characters in the input's own editor and the other fields (`editKey`,
  `branches.go`): AltGr+2 and AltGr+1 typed nothing in the draft. That
  test is the committed one, which this plan does not touch (`keys.go`'s
  diff is Ctrl+F's and Ctrl+W's alone). `@`, `#`, `\`, `|`, `{`, `}`,
  `[`, `]` and `~` can only be pasted there on this keyboard. Not changed
  here: its fix is proposed apart.
- **The full view's end.** Tab scrolls the full view two rows above the
  match where the view's length allows; at 208x59 and 120x40 the 60-line
  paste's view stopped at its end, line 40 in view lower down.
- **Tab on a request does nothing.** D8 opens what Enter opens from the
  entry cursor, which never stops on a request; a request's own view opens
  from its card or by a double click. ui-design.md says so.

### The mutants

Broken through `go test -overlay` by plan 51's mutrun, the repository
never written (its code hashed before and after, the same: d595a8bb), on
the final code: 64 mutants, 602 s; 61 caught by a failing test (the first
one named below), 3 equivalent. The runs before it, on the tests of step 6
and then of step 7, found the two defects the Details list (A row's place,
AltGr) and what the tests lacked: R16, R18, A3 and A11, each tested since.
Three mutants the compiler had killed there, not a test ("declared and not
used": R36, A11, A19), were rewritten to compile.

| Mutant | Broken | Caught by |
| - | - | - |
| R1 | the case always ignored | TestConvMatchesCase |
| R2 | the capital never looked for | TestConvMatchesCase |
| R3 | the matches overlapping | TestConvMatchesLiteral |
| R4 | one past the cap | TestConvMatchesCap |
| R5 | an entry's oldest match first | TestConvMatchesLiteral |
| R6 | the live thinking card searched | equivalent: a live phase has no lines (`entryLines`, `thoughtShown`), so it is never found |
| R7 | no tool target searched | TestConvMatchesKinds |
| R8 | no request's target searched | TestConvMatchesKinds |
| R9 | the targets never kept | TestConvMatchesMemo |
| R10 | a failed call's error not searched | TestConvMatchesKinds |
| R11 | the cap's words as all there are | TestSearchCounts |
| R12 | no match without the scope | TestGoldenFrames |
| R13 | no bar row | TestGoldenFrames |
| R14 | the footer without the bar's keys | TestGoldenFrames |
| R15 | the draft's caret drawn under the bar | TestGoldenFrames |
| R16 | the bar over a popup | TestSearchClose |
| R17 | a match at the place not at or before it | TestSearchPlaces |
| R18 | the fold row standing for its first line alone | TestSearchPlaces |
| R19 | a folded run not opened | TestSearchFolds |
| R20 | a phase's details not opened | TestSearchThinkingDetails |
| R21 | a folded line not on its fold row | equivalent: without it a folded line goes to its entry's first whole row, the fold row |
| R22 | a match's row the line's first | TestSearchReveal |
| R23 | no bottom place | TestSearchStartsInView |
| R24 | no name for a target not drawn | TestSearchLitName |
| R25 | the cut's "…" never lit | TestSearchLit |
| R26 | the fold row never lit | TestSearchLit |
| R27 | no current match | TestSearchLit |
| R28 | the matches lit over the selection | TestSearchLit |
| R29 | a left cut's "…" a drawn part | TestSearchLit |
| R30 | a right cut's "…" unmarked | TestRetarget |
| R31 | no target an empty line | TestFailedToolRow |
| R32 | the fold row unmarked | TestSearchLit |
| R33 | a call's cut target without references | TestSearchLit |
| R34 | an answered request's target unmarked | TestSearchLitApproval |
| R35 | a waiting request's target unmarked | TestSearchLitApproval |
| R38 | a row's place one past its last byte | TestSearchPlaces |
| R36 | the full view's current match not marked | TestSearchLitView |
| R37 | a match's view row from the row's start | TestSearchLitView |
| A1 | the diff kept under the bar | TestSearchOpen |
| A2 | the entry cursor kept | TestSearchEntryCursor |
| A3 | the popups kept | TestSearchClosesPopups |
| A4 | the focus not back to A or C | TestSearchOpen |
| A5 | the focus back to a panel no longer beside B | equivalent: syncUI gives an "R" focus with no panel shown back to B |
| A6 | ↑ to the newer | TestSearchSteps |
| A7 | the leaving keys not closing the bar | TestSearchClose |
| A8 | the dashboard's keys taken by the bar | TestSearchClose |
| A9 | a Ctrl or Alt chord typed | TestSearchField |
| A26 | AltGr's characters refused | TestSearchField |
| A10 | a paste's every line | TestSearchField |
| A11 | the current match not kept | TestSearchKeepsPart |
| A12 | the first match the newest of all | TestSearchCapStart |
| A13 | typing not showing the match | TestSearchFolds |
| A14 | no round at the ends | TestSearchSteps |
| A15 | the match put at the view's top | TestSearchReveal |
| A16 | B not held | TestSearchSteps |
| A17 | the folds not opened | TestSearchFolds |
| A18 | Tab's view at its top | TestSearchWhole |
| A19 | a click elsewhere keeping the bar | TestSearchClose |
| A20 | the matches not found again after events | TestSearchStreaming |
| A21 | a paste into the draft | TestSearchField |
| A22 | Ctrl+F over a window | TestGoldenFrames |
| A23 | Ctrl+F over a full view | TestSearchOpen |
| A25 | Esc not giving B's entry cursor back | TestSearchEntryCursor |
| A24 | Ctrl+W under the bar | TestSearchCloseKeys |

The three equivalent, and the code that makes each so:
- **R6** (`searchable` without its `thinkLive` test). A live phase's entry
  has no lines: `entryLines` gives a thinking entry none until
  `thoughtShown`, which needs the phase ended (`!t.Live()`,
  `internal/render/thinking.go`), so nothing of a live card is ever found,
  with the test or without it.
- **R21** (`ConvMatchRow` without its fold-row case). A match in a long
  paste's folded lines is on no drawn line, so `ConvMatchRow` falls back to
  the entry's first row that shows it whole (`rowEnd`'s `AllLines`); for a
  long paste that row is its fold row, the one the case returns.
- **A5** (`closeSearch` giving the docked panel its focus without checking
  that the panel is still beside B). After every update `syncUI` gives an
  "R" focus with no panel shown back to B (`internal/app/app.go`: "the
  docked panel hidden: zone B has the focus back"), and a panel in B's
  place is hidden whenever B has the focus, the bar's included; with the
  check or without, Esc ends with the input's focus there.

### The checks

| Check | Result |
| - | - |
| build, vet (Windows, Linux), gofmt | ok, ok, ok, nothing printed |
| `go test ./...`, after the last code edit | 16 packages ok; internal/app 215.8 s, internal/input 235.4 s, internal/render 29.4 s |
| the mutants, mutrun, on the final code | 64: 61 caught by a failing test, R6, R21 and A5 equivalent, 602 s |
| the code's hash, before and after the mutants | the same, d595a8bb |
| `-race` | internal/app, `TestSearch`, `TestGoldenFrames`, `TestConfiguredKeysShown` and `TestFailedToolRow`, 18.7 s; internal/render whole, 118.3 s: no race |
| Windows Terminal, the final build | 208x59, 120x40 and 80x24, the eight steps each as 66.11 says; the AltGr probe at 120x40 |
| the timing step, alone, on mains power | below |

### The timing step

The command of evaluation.md, the twenty-one tests, alone, at 14:55:02:
on mains power, battery 100 %; CPU load over the 2 s before, 25, 21, 25,
11 and 18 %; the windows open, Chrome, Claude, Edge, Teams, Outlook,
Notepad, Task Manager, Move Mouse, Settings; Defender idle (0 s of CPU).

| Test | Measured | Target | Result |
| - | - | - | - |
| TestKeystrokeToFrame | 10,000-line draft p95 1.05 ms; in the draft window 1.12 ms | 16 ms | PASS |
| TestDraftKeystrokes | 50,000 lines, 8 saves, p95 2.53 ms | 16 ms | PASS |
| TestKeystrokeCost | reports only | | PASS |
| TestLongSession | a delta at the end, p95 3.08 ms | 16 ms | PASS |
| TestThinkingFrame | p95 2.39 ms | 16 ms | PASS |
| TestPasteToFrame | the worst 1.66 ms | 100 ms | PASS |
| TestStreamToFrame | p95 4.66 ms | 16 ms | PASS |
| TestSelectorToFrame | p95 1.03 to 1.52 ms | 16 ms | PASS |
| TestMotionToFrame | p95 1.01 to 2.00 ms; idle motion all dropped | 16 ms | PASS |
| TestDiffToFrame | p95 1.00 to 1.08 ms | 16 ms | PASS |
| TestMessageToFrame | p95 0.73 and 1.64 ms | 16 ms | PASS |
| TestOutputToFrame | p95 1.02 ms | 16 ms | PASS |
| TestToolsToFrame | p95 2.69 ms | 16 ms | PASS |
| TestBannerToFrame | p95 2.315 ms, max 12.5 ms, 11,355 bytes a frame | 2 ms | FAIL |
| TestHomeKeysToFrame | p95 0.20 ms | 16 ms | PASS |
| TestHomePictureTime | the worst 226 ms | 260 ms | PASS |
| TestBranchesFrameCost | p95 1.96 ms | 16 ms | PASS |
| TestBlameFrameCost | p95 1.51 ms | 16 ms | PASS |
| TestGraphFrameCost | p95 1.05 ms | 16 ms | PASS |
| TestTurnReviewFrameCost | p95 2.00 ms | 16 ms | PASS |
| TestSearchToFrame | keys, two hours: p50 4.00, p95 7.15, max 32.1 ms; answers ten times longer: p50 7.51, p95 9.91, max 15.4 ms; deltas, the bar open: p50 2.38, p95 4.10, max 7.26 ms | 16 ms | PASS |

TestSearchToFrame's one key of 32.1 ms in the two-hour session is a single
frame, its p95 7.15 ms; a run of it alone earlier the same day, while the
documents were written, gave p95 7.52 ms, 10.46 ms and 5.52 ms.

**The banner's miss, investigated** (evaluation.md: a miss is a FAIL, never
rerun until it passes). The banner is the home screen's (plan 51) and plan
66 changes nothing on its path. `TestBannerToFrame` alone on HEAD's tree
(250df5f, read out with `git archive` into the scratchpad) and on this one,
in turn, three rounds, the machine quieter (0 to 16 % CPU before): HEAD
p95 1.645, 1.576 and 1.566 ms; this tree 1.619, 1.517 and 1.550 ms; p50 678
to 688 µs on both. The same on both trees and within the 2 ms there, so
no regression of plan 66; the miss in the timing step goes with its load,
as plan 65's did (2.053 ms on a load of 10 to 41 %), accepted then as a
documented exception. The FAIL stays as measured; the decision is the
owner's.

### Windows Terminal

The shots are in `scratchpad\term\out-p66run\`, a folder each size
(`search-max`, `search-90x30`, `search-60x18`), eight steps each: `start`,
`bar`, `typed`, `run`, `paste`, `whole`, `back`, `closed`, each a PNG and
its text, with the run's `log.txt`; taken again on the final build, the
same results at each step. `altgr-90x30` is the AltGr probe: AltGr+2 and
AltGr+1 in the draft (`editor`, `editor2`, nothing typed), then AltGr+2 in
the bar (`query`, "/ @"), its `trace.log` the records received. The ones to look at first, at 208x59:
`typed` (the newest match lit), `run` (the folded run open, quota.go's
"quota" lit), `paste` (the fold row lit, the run's match on the plain
amber), `whole` (line 40 lit in the full view). No Claude Code or Codex
session ran in a terminal, so none was closed.
