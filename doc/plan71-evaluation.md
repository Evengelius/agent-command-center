# Plan 71 evaluation (2026-10-30)

The check of plan 71, consistent Markdown rendering across ACC (brief
amendment 74): Markdown is drawn as formatted content wherever ACC presents
it as such, its markers hidden, and every raw context keeps its text
exactly. The owner reported the defect after plan 70's closure, a Claude
Code answer in zone B showing `**Goal:**`, `**Decisions:**`,
`**Completed:**`, `**Open:**` and `**Recommended next step:**` as written
and Markdown files likewise, and asked for the underlying problem to be
fixed consistently: "Do not fix only the specific screenshot or the
specific labels shown above."

It covers the 11 criteria of [`plan.md`, Plan 71](../plan.md), written
before the code. **Nothing is committed or pushed**: as the owner asked,
the implementation and this evidence go to an independent review first.

How it was checked:
- **Tests**: the renderer (`internal/render/markdown_test.go`: 52 cases of
  its constructs, 4,000 random lines' invariants, the copy, the wrap, the
  styles, the memo); zone B through both CLIs' events and recordings,
  streaming and completed, at 80x24, 120x40 and 200x60; the inspection
  (`inspectmd_test.go`); the file viewer's preview
  (`mdpreview_test.go`) and its `m` key in the application
  (`internal/app/markdown_test.go`).
- **Each change broken on purpose** through `go test -overlay`, the
  repository never written, its code hashed before and after: 50 mutants
  (`scratchpad\plan71\mkmutants71.mjs`, `mutrun71.sh`), one at a time.
- **The golden frames**: 116 rewritten, each compared with HEAD by a
  script and the rest by eye.
- **Windows Terminal** (71.9): ACC's profile, maximised, at 120x40 and at
  80x24, a stand-in `claude.exe` streaming a Markdown answer (no CLI ran,
  no quota).
- **Build, vet (Windows and Linux), gofmt, the whole suite** after the last
  edit, `-race` on the changed packages and **the timing step** alone, on
  mains power. No Go build or test ran beside another (amendment 74).

## The evaluation

```text
Evaluation - Plan 71, consistent Markdown rendering across ACC - 11 criteria: 10 PASS, 1 FAIL
PASS 71.1 The renderer: TestMarkdownShown, 52 cases, each line's shown text, width and styles: strong, emphasis, both, strikethrough, nested and mixed; code spans literal (`**not bold** \*` kept), in strong; links "label (url)", the label blue underlined, the URL grey, images, a strong link, a destination's angle brackets hidden; escapes (`\*not em\*` is "*not em*"); snake_case and an underscore opening inside a word kept; unclosed bold, code, link and strikethrough as written; lone and space-flanked markers kept ("2 * 3 * 4"; "*a * b*" italic whole); headings without their #s, "#hashtag" kept; quotes with their bars, nested, holding a list and a heading; bullets • ◦ ▪ by depth, numbers in the bullet colour, a task box as written; rules of -, * and _ across the row; fences hidden, the language amber, a closing fence an empty row, code literal in a fence and in an unclosed one; a table's header bold, its rule ├─┼─┤, cells padded, escaped pipes and pipes in code; no table without its rule; wide characters, a tab, an empty line; control characters drawn as their marks (^[, ^G); a file's byte order mark hidden; every marker's bytes hidden (no cell, stNone). TestMarkdownLines: the block kinds, fences carried over, nested lists at any indent, tables only with their rule, the byte order mark. TestMarkdownCells, 4,000 random lines of Markdown's pieces, controls and broken UTF-8: every cell its line's bytes, in order and apart, no control character drawn, every byte styled, the wrap keeping every cell but its edges' spaces, a copy widened over hidden bytes only. TestMarkdownStyles: each style made from its block's base, with its attributes, painted again with the 16-colour palette and back. TestBlockComment, TestMultiLineString: the code highlighting as before, the fences' markers hidden
PASS 71.2 Zone B: TestMarkdownZoneB, the owner's answer through the reducer's events after a resume, Claude Code and Codex, streaming and completed, at 80x24, 120x40 and 200x60: no ** nor backtick drawn; "Goal:", "Decisions:", "Completed:", "Open:" and "Recommended next step:" bold (the last in view at every size, all from 40 rows); the list's bullets; every row within the frame; a label's first cell hitting its source byte after the markers; the answer copying as its source, a label as "**Goal:**", a word of it as "Goal". TestMarkdownRecordings, through each CLI's mapper: Claude Code's streamed "**59**" (P22o) bold; Codex's "Created [hello.txt](<…>) containing `hello`." (P12x) as "Created hello.txt (<HOME>/…/hello.txt) containing hello.", hello.txt a link, hello code. TestMarkdownWrap and TestMarkdownZoneBHang: the wrap by the cells shown, a list item's and a quote's later rows under their text. TestMarkdownWiden: the copy's ends. TestSearchLit unchanged and passing (the bold, code and link "needle" lit); the entry cursor's, folds' and follow's tests unchanged and passing
PASS 71.3 Thinking and zone C: TestCodexSummaryDetails and TestCodexSummaryLines: Codex's summary as Markdown in a phase's details, the dashboard's lines as they were; TestCodexSummaryCopies: copied as Codex sent it; TestMarkdownExcerpts: the full view, an agent's row ("completed · Done: ok and a link (u)"), zone C's timeline (an AI message without markers, a notice as written, a 26 KB message's two rows ending with "…"), the excerpt read only as far as its rows need; TestInspectMarkdown: the Messages tab in zone B's styles, markers hidden, the start's description and the end's summary rendered, the notice and every other row as before, scrolled top to bottom at three sizes, the copy what the rows show; TestMarkdownRecordings: a Codex subagent's "**0 files**" (P7c) bold in the inspection, plain in the timeline
PASS 71.4 The file viewer: TestMarkdownPreview at 80x24, 120x40 and 200x60: one row per source line, numbered as before, every row within the window, the constructs rendered, no marker drawn, "preview · " and "m source"; m's source as before, "source · " and "m preview"; a text file unchanged; the cursor line's ground; blame's date and author and an edit mark's "▌" beside the preview; a match lit where the preview draws it, its column the cells shown before it (10, the source's 12); the shift by the cells shown; a long line cut with "…"; the copy the source, widened; a byte order mark's first heading rendered. TestMarkdownFileKey in the application: readme.md opened as its preview; m to its source and back, the shift reset; n bringing a match 30 bold words in into view by each view's column; g 1 in the preview; m in main.go doing nothing and typing nothing
PASS 71.5 Raw stays raw: TestMarkdownRaw: the owner's message, the draft, a notice, a dashboard note, a Bash target and its error, a Write's approval preview and a diff's lines of a Markdown file, all as written; TestMarkdownPreview: a Markdown file's source view; the code blocks' content literal (71.1); the golden frames of Source Control's, the turn review's and the Git Graph's diffs changed only in zone C's timeline line ("- alpha.txt" now "• alpha.txt"), their diffs as before
PASS 71.6 Streaming: TestMarkdownStreaming, the owner's answer seven bytes a delta on each CLI: at each step its rows those of the same text completed, every row within zone B at 80x24, "**Goal:" as written until its closer came; TestMarkdownCache, a long answer streamed 97 bytes a delta: every frame with the memo the frame without it, at three sizes, scrolled, in the preview and the inspection, a line no frame draws dropped after the next; TestStreamToFrame in the timing step: p50 2.00 ms, p95 4.29 ms (target p95 ≤ 16 ms)
PASS 71.7 Golden frames: 116 rewritten (TestGoldenFrames and TestThinkingFrames), each against HEAD by scratchpad\plan71\review\frames.mjs: the same rows, each as wide, the same words once the old frame's markers are taken out: 86 so; the other 30 read by eye: rows coming into view or leaving it as zone B, following its end, grew shorter, a popup or a card cutting B's text at another word, two blank rows under a short conversation, the timeline's bullets; nothing else changed
PASS 71.8 Nothing else changes: the code diff is internal/render and internal/app/file.go's m and match column; zone A's, zone C's layout, plan 70's card, Source Control's with its multi-selection, the diffs', the keys' and the mouse's tests unchanged and passing; three application tests that asserted the old literal markers ("**59**", "**Computing f(10)**") now assert the rendered text and the markers' absence; go test ./... ok
PASS 71.9 Windows Terminal: ACC's profile, the final code's build (acc-p71.exe, sha256 d1056dab…), plan 71's stand-in claude.exe streaming a Markdown answer in 61 deltas of 12 bytes, 50 ms apart (no CLI, no quota): 7 runs, every poll found, maximised (208x59), at 120x40 and at 80x24. The answer while it streamed, a table's row drawn as it arrived, the cursor after it; finished, its heading, bold labels, nested bullets, italic and struck words, code spans, the link and its grey URL, the quote's bar, the boxed table, "go" amber over literal coloured code, the rule; Shift+PgUp at 80x24, the URL wrapped onto its own row, the quote's bar on its wrapped row; at 120x40 a nested item's wrapped row under its text. README.md from the Explorer's quick open, its preview at the three sizes ("preview · ", every construct, a long line cut with "…", "m source"); m to its source (Chroma's colours, the markers, "source · ", "m preview") and back; "/needle" lit 30 bold words in, the view shifted to it, "match 1 of 1"; bom.md, written with a byte order mark, its first heading rendered
FAIL 71.10 The checks: expected build, vet, gofmt, the suite, the mutants, -race and the timing step's 21 tests within their targets; found go build ./... ok; go vet ./... ok (Windows and Linux); gofmt -l . empty; go test ./... ok (16 packages, 1,137 tests, 364 s) after the last edit; 50 mutants through -overlay, one at a time, all caught by a failing test (45, then the 5 that did not compile rewritten to and caught), none survived, the code's hash e8431f30 before and after, its non-test code (5e5d97d3) the final one; -race on internal/render (195 s) and internal/app (377 s): exit 0, no race report; the timing step alone, on mains power, at 20:46:09 (88.6 s): 20 of 21 within their targets, TestSearchToFrame's answers ten times longer p95 16.57 ms over its 16 ms (p50 10.02 ms; its two hours case p95 5.02 ms and its streamed deltas 4.07 ms)
PASS 71.11 The documents: brief amendment 74 (the owner's words); ui-design.md (Markdown: where, the renderer, its look, malformed and escaped, what it leaves out, safety, streaming, copy and search, the inspection; Thinking; Search; the file viewer's preview and its m; Focus and keys; Rendering; Colours; Selection and copy; TestKeyOverlayRows, 105 rows); architecture.md (Markdown rendering; ASM-17); AGENTS.md's status and MVP scope (TestAgentsMDWithinCodexLimit ok, 21,851 bytes); plan.md's plan 71; this file
Not verified: the owner's look at the shots (scratchpad\term\out-p71, Details); a live answer from Claude Code and from Codex in Windows Terminal (no live run is in the plan's criteria: the tests replay both CLIs' recordings and events, and the stand-in streams as Claude Code does); a window edge dragged by hand
Verdict: NOT DONE
Likely cause: 71.10: this machine's run-to-run variance on the search's heaviest case, whose p95 sits near its target, not plan 71's frame: the same test on HEAD's tree and on this one in turn, one at a time, read p95 10.6, 12.2 and 12.0 ms (HEAD) against 12.1, 12.8 and 12.4 ms (plan 71); with each key's Update and View timed apart, three runs each, the frame (View, where plan 71 works) read p50 1.0 to 2.0 ms on both trees, and the search's matcher (Update, code plan 71 does not touch) p50 7.5 to 9.8 ms, its p95 from 10.1 to 21.8 ms between runs of one binary (Details, The timing step); plans 66 and 69 read this case at p95 9.9 and 9.6 ms
Proposed next step: 71.10: the owner's decision, a documented timing exception as in plans 65 and 66, or the timing step again on a quiet machine; the owner's independent review of this report and the diff, then the commits on the owner's approval - waiting for approval
```

The recommendation: **ready for the owner's independent review**, with
71.10's timing miss to decide. Every functional check passes, both CLIs'
paths included; what is left is the owner's: the timing decision, the
look of the shots and a live run on the owner's own sessions (the manual
steps below).

## 71.10, timed again on a quiet machine (2026-10-30)

The owner chose to run the timing step again once the machine was quiet
("Re-time when quiet"); the block above stays as it was reported. A
waiter (`scratchpad\plan71\wait71.ps1`) ran the step's command word for
word, alone, once the laptop was on mains power and Windows' processor
counter averaged at most 15 % over ten seconds, no second of it over
30 %, with no Go process running. A first waiter stopped when the
session's connection dropped, before it ran anything; the second, started
detached, waited from 22:18:53. The step ran at 22:22:15, on mains power
(battery 81 %), the counter at 2.9 to 22.8 % (mean 13.3 %), WMI's load as
the first run recorded it at 23 to 26 %, fewer applications open than
for the first (no Chrome, Edge, Teams or Task Manager): 89.2 s
(`evidence\timing2-step.log`, UTF-16 as PowerShell wrote it, and its
UTF-8 copy; `timing2-conditions.txt`; `timing2-wait.txt`).

20 of its 21 tests were within their targets again, a different one over:
- **TestSearchToFrame within**: answers ten times longer p50 8.51, p95
  14.22, max 19.16 ms against its 16 ms; two hours p95 6.12 ms; deltas,
  the bar open, p95 4.61 ms.
- **TestBannerToFrame over**: the home screen's banner, p95 2.25 ms over
  its 2 ms (p50 0.77, max 4.40 ms; 11,355 bytes a frame). Plan 71 does
  not touch the banner or the home screen. Run on HEAD's tree and on this
  one in turn, three times each, alone, its p95 read 1.88, 1.85 and
  1.80 ms (HEAD) and 1.86, 1.58 and 1.78 ms (plan 71), all within 2 ms,
  the same 11,355 bytes a frame; in the first run it read 1.72 ms, and
  plan 66 recorded the same miss (2.315 ms).

| Test | Measured, the second run | Target | Result |
| - | - | - | - |
| TestKeystrokeToFrame | 10,000-line draft: p95 1.10 ms; the draft window: p95 1.61 ms | 16 ms | PASS |
| TestDraftKeystrokes | 50,000 lines, 8 saves: p95 2.62 ms | 16 ms | PASS |
| TestKeystrokeCost | 251 µs and 598 allocations a key (10,000 lines); 626 µs (50,000) | none | PASS |
| TestLongSession | delta to frame at the end: p95 3.00 ms; heap 9.3 MiB | 16 ms | PASS |
| TestThinkingFrame | p95 4.18 ms | 16 ms | PASS |
| TestPasteToFrame | 50,000 lines: max 1.50 ms | 100 ms | PASS |
| TestStreamToFrame | 2,000-line answer: p50 2.51, p95 5.76, max 9.54 ms | 16 ms | PASS |
| TestSelectorToFrame | p95 1.04 to 2.01 ms | 16 ms | PASS |
| TestMotionToFrame | p95 1.25 to 2.01 ms; idle reports all dropped | 16 ms | PASS |
| TestDiffToFrame | p95 1.00 to 1.20 ms | 16 ms | PASS |
| TestMessageToFrame | p95 1.12 and 1.96 ms | 16 ms | PASS |
| TestOutputToFrame | 64 MiB of output: p95 1.04 ms | 16 ms | PASS |
| TestToolsToFrame | p95 2.91 ms | 16 ms | PASS |
| TestBannerToFrame | p95 2.25 ms | 2 ms | FAIL |
| TestHomeKeysToFrame | p95 0.19 ms | 16 ms | PASS |
| TestHomePictureTime | max 221 ms | 260 ms | PASS |
| TestBranchesFrameCost | p95 2.98 ms | 16 ms | PASS |
| TestBlameFrameCost | p95 1.60 ms | 16 ms | PASS |
| TestGraphFrameCost | p95 1.01 ms | 16 ms | PASS |
| TestTurnReviewFrameCost | p95 2.01 ms | 16 ms | PASS |
| TestSearchToFrame | keys, two hours: p95 6.12 ms; answers ten times longer: p50 8.51, p95 14.22, max 19.16 ms; deltas, the bar open: p95 4.61 ms | 16 ms | PASS |

So 71.10, by the letter, missed in both runs, each time by one test over
its target, a different one each time, each in code plan 71 does not
change (the search's matcher, the banner), each within its target on
HEAD's tree and on this one when run alone in turn. The decision stays
the owner's: a documented timing exception, as in plans 65 and 66, or
another run.

## The owner's decision on 71.10 (2026-10-30)

The owner accepted 71.10 as a documented performance exception, in its
own words: "Decision: Accept criterion 71.10 as a documented performance
exception. The available evidence is sufficient. No additional timing run
is required." Its rationale: the two runs, each 20 of 21 ("Run 1: 20/21
passed. `TestSearchToFrame` exceeded its 16 ms target by 0.57 ms." and
"Run 2: 20/21 passed. `TestSearchToFrame` passed at p95 14.22 ms, while
`TestBannerToFrame` exceeded its 2 ms target by 0.25 ms."), and that "the
subsequent baseline comparisons provide reasonable evidence that these
marginal failures are caused by execution variability rather than a
significant Plan 71 regression": the search test passed under quieter
conditions; the banner was not modified by plan 71; three alternating
baseline comparisons showed no banner regression; plan 66 documented
similar variability.

**71.10: ACCEPTED EXCEPTION**, not PASS. Both runs stand as reported, their
failures with them: the evaluation block above (TestSearchToFrame's
answers ten times longer, p95 16.57 ms over 16 ms) and the second run
(TestBannerToFrame, p95 2.25 ms over 2 ms). The baseline comparisons that
justify it are in this file: the search on HEAD's tree and this one in
turn (p95 10.6 to 12.2 ms against 12.1 to 12.8 ms) and with Update and
View timed apart ([The timing step](#the-timing-step)); the search within
its target on the quiet run, and the banner on both trees in turn, p95
1.58 to 1.88 ms, within 2 ms ([71.10, timed again on a quiet
machine](#7110-timed-again-on-a-quiet-machine-2026-10-30)). As the owner
asked, no further timing run is made to obtain 21 of 21, and no
speculative performance change.

Plan 71's evaluation stands at 10 PASS and 71.10 an accepted exception.
The plan is **not closed**: the owner's manual acceptance in Windows
Terminal comes next, one test at a time, each reviewed before the next.

## Manual acceptance

The owner checked plan 71 by hand in Windows Terminal, one test at a time,
as for plan 70; each test's result was reviewed before the next was given.
`acc.cmd` runs the working tree (`go run`), so the code checked is plan
71's as it stood, nothing committed. No recent project is added: the
tests use temporary projects already among ACC's recent ones. The campaign
found the multiline defect (The multiline defect, below), in AGENTS.md's
preview and in a Codex answer in zone B; after its fix the owner repeated
the affected tests and recorded, on 2026-10-31, the results below, in the
owner's words. The campaign is complete: the owner asked that it not be
repeated.

| Test | Scenario | Given | Result, the owner's (2026-10-31) |
| - | - | - | - |
| M71-01 | A real Claude Code answer rendered in zone B, the owner's own message raw | 2026-10-30 | "Real Claude Haiku answer in Zone B: multiline rendering manually verified." |
| AGENTS.md's preview | The repository's AGENTS.md in the file viewer's preview, its spans over lines | 2026-10-31 | "`AGENTS.md` preview: multiline rendering manually verified." |
| Codex | A Codex answer in zone B, its spans over lines | 2026-10-31 | "Codex: automated post-fix coverage verified; no additional live manual retest performed." |

**M71-01, a real Claude Code answer rendered** (1 turn of quota). The
project `%TEMP%\acc-p39-live-claude-134810`, empty, among ACC's recent
projects since plan 39. The prompt asks Claude Code to reply with a fixed
Markdown text, every construct of 71.1 in it; the owner's message, which
holds the same text, is drawn raw, and the answer rendered. What to look
for, the evidence (two screenshots) and the PASS conditions are in the
instructions given to the owner, kept with its result once reviewed.

## The multiline defect (2026-10-31)

The owner confirmed a defect in the manual acceptance: inline Markdown
spanning two physical lines was shown with its markers. In AGENTS.md's
preview, `**Claude Code first, then OpenAI Codex` / `CLI**` and
`**Status: plan 71 active since 2026-10-30, consistent Markdown rendering`
/ `across ACC (brief amendment 74).**`; in a Codex answer in zone B,
`**This bold span begins on one physical line` / `and ends on the next
physical line.**` and `*This italic span also begins on one line` / `and
ends on another.*`. The owner asked for the root cause to be fixed in the
shared renderer, with no provider-specific workaround, everything validated
kept, diffs and the source view raw, and no new dependency.

**Root cause.** The renderer read a line's inline constructs from that
line alone: `mdBuild.show` parsed `inline(at, len(src), …)`, so an opener
whose closer is on a later line found none before its line's end, and both
were shown as written. Plan 71 had listed it as a limitation ("a construct
over two lines (each line is rendered on its own)", Delivery report, 4),
but it is valid Markdown: CommonMark reads a paragraph's inline text whole,
its line endings in it, and a hard-wrapped paragraph, as AGENTS.md's, has
spans over its lines. Every surface draws through the renderer, so every
surface showed it.

**The fix**, in the shared renderer only (`markdown.go`, `content.go`, and
`cards.go` and `inspect.go` reading through the frame's cache): no provider
code, no new dependency; the diffs, the source view, the owner's messages
and the draft never call it.
- `mdLines` groups a block's lines into paragraphs as CommonMark does
  (`spans`, `mdSegment`): a blank line, a heading, a table, a rule, a fence
  or code ends one; text goes on it, as a list item's or a quote's later
  lines do; a list item begins one; a quote's text goes on a quote's
  paragraph as deep.
- `pairPara` reads a paragraph's lines in order, each by its own parse's
  top level (`lineDelims`, a dry run of the same `inline`): a run of
  backticks shown as written with one as long on a later line opens a code
  span to the first of them, whatever that line makes of it; a run that
  may close closes the nearest opener of its marker and length on an
  earlier line, before it may open one, the openers between dropped; one
  shown as written that may open is an opener; after a marker that closes
  a span, or opens a code span, its line is read again from there, as it
  is drawn. The opening and closing rules are the line's own (`mayOpen`,
  `mayClose`, as `emphasis` and `closeRun`).
- Each line keeps its own source, offsets and wrap. It records what it
  starts inside (`mdLine.carry`) and its markers that open or close a span
  over lines (`mdLine.cross`); `para` hides those markers and parses each
  part between them as the line's own, from inside what is open. A copy
  widens over a span's markers when it takes in its text from its first
  line to its last (`crossSpans` in `mdWiden`). The lines' runs are kept a
  frame after their last use, as the wraps are (`WrapCache.lineDelims`).
- Found while checking a first version of the pairing, before this
  delivery, and fixed: a closer before punctuation on a line that opens
  another span after it (`CLI**, the only **two** supported`: the line's
  own parse took the closer for an opener, the markers shown again); a
  code span closing on a line with code spans of its own (`` `go test ``
  / `` ./...` and `go vet` ``); a closer right after another span's closer
  (`c~~**`).

**Tests**, `internal/render/mdspans_test.go`, seven:
- `TestMarkdownAcrossLines`, 47 paragraphs, each line's shown text and its
  bytes' styles: the AGENTS.md paragraphs and the Codex answer word for
  word; bold, italic, both, strikethrough, over two and three lines,
  nested, beside a span of the line's own, from inside the span its
  closer's line opens; a list item's, a quote's and a quote's item's
  lines; code spans over lines, their text literal, of two backticks, in a
  span; fenced code literal; no span over a blank line, a line of spaces,
  a heading, a list item, a quote, a quote's heading or item, a fence, a
  rule, a table, two items, quotes of two depths; escaped, unclosed, of
  lengths that differ, a closer at a line's or a quote's text's start, an
  opener at a line's end, spaced stars, an unmatched backtick, an
  underscore in a word.
- `TestMarkdownAcrossLinesSurfaces`: zone B for each CLI, the preview, the
  inspection's Messages tab, Codex's summary in a phase's details, zone C's
  timeline.
- `TestMarkdownAcrossLinesStreaming`: Codex's answer 1, 3, 7 and 11 bytes
  a delta on each CLI: at each step its rows those of the same text
  completed, the opener as written until its closer comes on the next
  line.
- `TestMarkdownAcrossLinesSource`: every span drawn is its source's bytes
  at its offset; a click on "CLI" hits line 2, byte 0; the copy of the
  span whole is `**Claude Code first, then OpenAI Codex\nCLI**`, of a word
  the word, short of its end or from its middle without the markers, of
  the lines whole their source; the search finds "OpenAI Codex" at its
  source's offset and lights it, finds `**` twice and lights nothing; the
  preview copies as zone B.
- `TestMarkdownAcrossLinesRaw`: the owner's message, a Markdown file's
  source view and a diff's lines keep AGENTS.md's markers as written.
- `TestMarkdownAcrossLinesCache`: the kinds through the cache in two
  frames those read anew, the runs dropped two frames after their last
  use; streamed 9 bytes a delta, every frame with the cache the frame
  without it.
- `TestMarkdownAcrossLinesRandom`: 3,000 random paragraphs of Markdown's
  pieces: every span over lines within one paragraph, its markers hidden;
  each line's parse beginning where its paragraph's text does.

**Mutants** (`scratchpad\plan71\mkmutants71b.mjs`, `mutrun71b.sh`, through
`-overlay`, one at a time, the code hashed before and after: unchanged):
37 on the fix, each run first against `TestMarkdown…` and then the whole
render package: 33 caught; 3 survived and 1 did not compile. The
survivors: P2 (a backtick opening a code span with no closer, its effect
seen only by a span over lines after it), P13 and P14 (an underscore
inside a word opening or closing over lines): their cases added, S3
rewritten to compile, the four run again: all caught. None survives.

**The checks**, one at a time, no Go run beside another, after the last
edit of the code (its hash `a170c8b7` over the changed non-test Go files):
gofmt empty; `go build ./...` and `go vet ./...` (Windows and Linux) ok;
`go test ./...` ok (16 packages, 367 s), and again after the last test
added (321 s); `-race` on internal/render (188 s) and internal/app (443
s): exit 0, no race report; the golden frames unchanged (none rewritten:
`TestGoldenFrames` and `TestThinkingFrames` pass as they are, no frame
holding a span over lines).

**The timing step**, run once for the fix, at 01:34:38 on 2026-10-31, as
the guide has it: on mains power (battery 100 %), no Go build or test
beside it, the CPU load recorded, not a gate (the processor counter 4.6
to 25.9 %, WMI 24 to 32 %; a wait for a quiet CPU, the owner's condition
for 71.10's second run, read means of 15.7 to 56.3 %, most 16 to 33 %,
over 12 minutes and was stopped);
open: Claude, Edge, Outlook, Paint, Notepad, Move Mouse, the system's
settings. 20 of 21 within their targets; TestStreamToFrame p50 3.00 ms,
p95 6.21 ms, max 14.71 ms (target p95 ≤ 16 ms; 2.00 and 4.29 ms in 71.10's
first run); TestSearchToFrame's answers ten times longer p95 13.74 ms;
**TestBannerToFrame p95 2.73 ms, over its 2 ms**, a FAIL. The banner's
frame is the home screen's, whose code (`home*.go`) calls none of the
Markdown renderer's; the same test read 2.25 ms in 71.10's second run and
within 2 ms on HEAD's tree and plan 71's in turn. Not run again (the
owner's decision on 71.10: no timing run to obtain 21 of 21).

**What the fix costs a frame**, measured apart (a scratch test through
`-overlay`, `scratchpad\plan71\cost`, `testing.Benchmark`, in turn): the
kinds of TestStreamToFrame's 2,000-line answer, its lines' runs cached as
in a frame, 670 to 970 µs a frame with the pairing against 430 to 550 µs
without it (its first version of `spans` taken out, S1), 26 allocations
and 390 KB against 2 and 164 KB; most of it the runs' cache filled anew
each frame. Sized from the frame before, as the wraps' cache is, it read
570 to 860 µs, 10 allocations and 279 KB: not applied, as the owner asked
for no speculative performance change; the owner's to decide.

```text
Evaluation - Plan 71, the multiline defect - 10 criteria: 9 PASS, 1 FAIL
PASS F1 Spans over a paragraph's lines: TestMarkdownAcrossLines: AGENTS.md's two paragraphs and the Codex answer word for word, "Claude Code first, then OpenAI Codex" / "CLI", "Status: …" / "across ACC (brief amendment 74).", "This bold span …" / "… physical line." bold, "This italic span …" / "… another." italic, no * drawn; strikethrough, both, over two and three lines, nested, beside a span of the line's own, a closer before punctuation with a span after it, a closer before a letter, a closer after another's, a span from inside the span its closer's line opens; a list item's, a quote's and a quote's item's lines; code spans over lines, of two backticks, in a span, a code span's closer before its line's own
PASS F2 Paragraph boundaries: no span over a blank line, a line of spaces, a heading, a list item, a quote, a quote's heading, a quote's item, a fence, a rule, a table, two list items, quotes of two depths; TestMarkdownAcrossLinesRandom, 3,000 random paragraphs: every span over lines within one paragraph's lines, its markers hidden, each line's parse beginning where its paragraph's text does
PASS F3 Escaped, unmatched and code as before: \*\* escaped, unclosed, lengths that differ, a closer at a line's or a quote's text's start, an opener at a line's end, spaced stars, an unmatched backtick (a span after it still pairs), an underscore in a word opening or closing nothing, code spans' and fenced code's text literal; TestMarkdownShown's 52 cases and every earlier Markdown test unchanged and passing
PASS F4 One renderer everywhere: TestMarkdownAcrossLinesSurfaces: zone B for Claude Code and Codex, the preview, the inspection's Messages tab, Codex's summary in a phase's details and zone C's timeline without markers, the spans bold
PASS F5 Streaming: TestMarkdownAcrossLinesStreaming, 1, 3, 7 and 11 bytes a delta on each CLI: at each step the rows those of the same text completed, the opener as written until its closer came on the next line, then hidden; TestMarkdownAcrossLinesCache: streamed 9 bytes a delta, every frame with the cache the frame without it; the kinds through the cache in two frames those read anew, the runs dropped two frames after their last use
PASS F6 The source: TestMarkdownAcrossLinesSource: every span drawn its source's bytes at its offset; a click on "CLI" at line 2, byte 0; the copy of the span whole "**Claude Code first, then OpenAI Codex\nCLI**", a word the word, short of its end and from its middle without the markers, the lines whole their source, the preview's the same; the search finding "OpenAI Codex" at its source offset and lighting it, "**" found twice and lighting nothing
PASS F7 Raw stays raw, nothing else changes: TestMarkdownAcrossLinesRaw: the owner's message, a Markdown file's source view and a diff's lines with AGENTS.md's markers as written; TestMarkdownRaw unchanged and passing; the golden frames unchanged, none rewritten; no provider code, no dependency added
PASS F8 The checks: gofmt -l . empty; go build ./... ok; go vet ./... ok (Windows and Linux); go test ./... ok (16 packages, 367 s, and 321 s after the last test added); 37 mutants through -overlay, one at a time: 33 caught, 3 survivors' cases added and 1 rewritten to compile, the four then caught, none survives, the code's hash unchanged by each run; -race on internal/render (188 s) and internal/app (443 s): exit 0, no race report
FAIL F9 The timing step: expected its 21 tests within their targets; found 20 of 21, TestStreamToFrame p95 6.21 ms (16 ms), TestBannerToFrame p95 2.73 ms over its 2 ms (p50 1.04 ms), at 01:34:38 on mains power, the CPU 4.6 to 25.9 % (WMI 24 to 32 %)
PASS F10 The documents: ui-design.md (Markdown: the renderer, Over a paragraph's lines, Malformed and escaped, Not rendered, Streaming; Selection and copy), architecture.md (Markdown rendering), plan.md's progress, this section
Not verified: the fix in Windows Terminal (the tests draw the frames the terminal shows; the owner repeats the affected manual tests)
Verdict: NOT DONE
Likely cause: F9: the home screen's banner, which no Markdown code draws (home*.go), at this machine's load; the same test read 2.25 ms in 71.10's second run and within 2 ms on both trees in turn
Proposed next step: F9: the owner's decision, as for 71.10; the owner's review of the fix and the manual tests again - waiting for approval
```

**The owner's decision on F9 (2026-10-31).** Asked whether 71.10's
accepted exception covers the banner's miss, the owner answered "Yes, it
covers it". **F9: ACCEPTED EXCEPTION**, under 71.10's, not PASS. The run
stands as reported, its failure with it (TestBannerToFrame p95 2.73 ms
over 2 ms); the reasons are 71.10's: the banner is not modified by plan 71
and is not on the fix's path (`home*.go` call none of the Markdown
renderer), and it read within 2 ms on HEAD's tree and plan 71's in turn.
No further timing run is made and no speculative performance change; the
sized cache stays the owner's to decide. The fix's evaluation stands at 9
PASS and F9 an accepted exception. Plan 71 is not closed: the owner's
review of the fix and the affected manual tests come next.

**Limitations** (ui-design.md, Markdown, Not rendered): a link or an image
over two lines is shown as written; a closer of another length than its
opener does not pair (`**a` closed by `b*`; CommonMark splits such runs);
a run opens before a non-space and closes after one, as within a line,
not by CommonMark's punctuation rules; a code span whose opening backtick
its line's own emphasis or link encloses stays the line's (CommonMark
gives the code span precedence).

To repeat by hand: AGENTS.md's preview (its two paragraphs above, `m` to
the source and back) and a Codex answer in zone B with the owner's two
spans; nothing committed.

## The Markdown cache (2026-10-31)

The owner decided to include the Markdown cache's optimisation in plan 71
before its closure: ACC is meant for daily use, responsive and frugal over
long sessions, continuous streaming and large conversations. The request:
investigate the cache (`WrapCache`, its `cur`/`old`, `shown`/`shownOld`
and `delims`/`delimsOld` generations, the paragraph runs and their
pairing), choose the best measured trade-off between frame time,
allocation, hit rate, retention and complexity, keep every result exact,
bound memory, measure before and after, and finish the documents. No
timing campaign (71.10 and F9 stay ACCEPTED EXCEPTIONS) and no manual test
again; focused benchmarks authorised.

**The baseline, measured first.** `internal/render/mdbench_test.go`, 18
benchmarks drawing whole frames through one `WrapCache` and one `Memo`, as
the program does: zone B unchanged (a 2,000-line answer of
TestStreamToFrame's lines, 2,000 lines of hard-wrapped Markdown with spans
over lines, 500 exchanges at the end and scrolled, a 100 KB line),
streaming (line by line from nothing to 2,000 lines, a byte a frame at
2,000 lines, 7 bytes a frame, into one 1,000-line paragraph, into a
1,000-line Go block, a 100 KB line a kilobyte a frame), resized (200 and
160 columns in turn), scrolled a row a frame, the inspection of 100
messages, a 5,000-line Markdown file's preview scrolled and shown as its
source and back. Go's clock steps by 0.5 ms on this machine, too coarse
for one frame: a frame's tail is its cost at the largest sizes (the
"at-2000" and "at-1000" streams, the resize) and what it allocates.

**Where the time and the bytes went** (CPU and allocation profiles, before
the change):
- An entry's Markdown kinds (`mdLines`: each line's block, a code block's
  strings and comments, a table's widths, the spans over a paragraph's
  lines) were cached nowhere: every frame read them again for every answer
  it drew, 37 % of an unchanged frame with a 2,000-line answer in view, the
  code block's scan 22 % of it.
- A streaming answer is drawn twice a frame, counted then drawn: its kinds
  were read twice.
- The per-line caches start every frame anew: every line of every answer
  drawn was looked up and put again in a map made afresh (the wraps' 600
  KB a frame); the paragraph runs' map grew from nothing each frame (24
  allocations, 220 KB at 2,000 lines).
- An entry's rows were added to an empty slice, grown a quarter at a time:
  2 MB for a 2,000-line answer, twice for a streaming one.
- A line's wrap built its whole drawing (cells, style runs, a code line's
  colours) and dropped it; its cells were copied from a temporary per
  segment into a slice grown from nothing: a 100 KB line streaming
  allocated 28 MB a frame, a resize 44 MB.
- A Markdown file's kinds were kept until another file was previewed,
  after the viewer had closed.
- A record kept for the tests (where `para` begins) allocated on every
  Markdown line drawn.

**The design.** A cache of each Markdown text a frame draws, by its place,
the same from frame to frame (`mdKey`: zone B's entry, or the inspected
agent's message), holding its lines, their kinds and their wraps at one
width and base (`mdDoc`; `WrapCache.markdown`, `WrapCache.wraps`):
- drawn again with the same text, its lines, kinds and wraps are taken as
  they are: an unchanged answer costs one comparison of its text, the same
  string, and a frame drawing it twice reads it once;
- grown, all its lines but the last those of the text before (compared
  line by line), as a streamed answer's are, it is read again only from
  where the text before stops being sure (`readDoc`, `grownFrom`,
  `mdRead`): its last line, or before it the paragraph or the run of rows
  starting with "|" that the line before the last is in, which the last
  line, still growing, may go on; there the state is a fence's (its
  marker, language and open string or comment, `mdState`), saved when the
  last line is in a fence; the kinds before are taken, and so are their
  wraps at the same width and base;
- changed in any other way (replaced, cut, a line inside changed), read
  whole;
- a line's wrap is found from its cells alone, no style and no colour
  (`mdWrap`, `mdBuild.measure`), built in a scratch the cache keeps for the
  next line when it is at most 4,096 cells (`maxScratch`); a line's
  drawing builds its cells into a slice sized once, a cell a byte at most;
- an answer's rows are added at once, as many as its wraps
  (`mdCardText`);
- every generation of the cache is dropped two frames after its last use,
  as before, its next maps made as large as the last frame's (`sized`);
  the preview's kinds are kept while the viewer shows the file, as its
  preview or its source, and dropped two frames after (`Memo.keepFile`);
  the tests' record of where `para` begins is kept only when they ask
  (`mdBuild.trace`).

Why this one, against the alternatives measured or weighed: the runs'
map sized alone (the measure of 2026-10-31 before this request) took 15 %
and left every frame reading every line; a cache keyed by the text would
hash the whole text each frame and give a streamed answer no text to grow
from; pairing a paragraph's spans across frames, its open runs carried,
would save the last paragraph's pairing at a cost of complexity, where the
paragraph is read again whole (paragraph-at-1000 below); keeping an
entry's rows across frames would keep what changes every frame (its card,
the cursor's row). The cache found the bytes: a frame now does what
changed.

**Before and after**, the 18 benchmarks, five runs each in turn on the
code before (`scratchpad\plan71\cachebase`, through `-overlay`) and after
(`scratchpad\plan71\ab71.sh`, `abtable.mjs`; the load not controlled, the
ranges as found), a frame's median time, its bytes and allocations:

| A frame | Before, ms (range) | After, ms (range) | Before MB, allocations | After MB, allocations |
| - | - | - | - | - |
| Unchanged, a 2,000-line answer | 2.48 (2.27–3.37) | 0.60 (0.53–0.72) | 3.52, 992 | 0.82, 941 |
| Unchanged, 2,000 lines of hard-wrapped Markdown | 2.58 (2.33–4.68) | 0.72 (0.54–0.90) | 2.40, 8,790 | 0.66, 989 |
| Unchanged, 500 exchanges | 1.77 (1.58–2.02) | 1.69 (1.51–2.05) | 1.35, 1,777 | 1.33, 1,761 |
| Unchanged, 500 exchanges scrolled | 1.76 (1.67–3.04) | 1.66 (1.60–1.81) | 1.36, 1,794 | 1.34, 1,778 |
| Unchanged, a 100 KB line | 1.51 (1.48–1.70) | 1.42 (1.34–1.52) | 1.09, 1,059 | 0.95, 1,051 |
| Streaming, an answer line by line to 2,000 | 2.58 (2.15–2.79) | 1.14 (1.06–1.19) | 1.55, 1,044 | 0.85, 992 |
| Streaming, a byte at 2,000 lines | 9.20 (9.07–9.47) | 2.41 (2.11–3.92) | 6.20, 1,051 | 1.89, 977 |
| Streaming, Markdown line by line to 2,000 | 2.52 (2.24–2.64) | 1.23 (1.13–1.71) | 1.25, 5,108 | 0.72, 1,082 |
| Streaming, a byte at 2,000 lines of Markdown | 6.88 (6.72–7.18) | 2.04 (1.99–2.14) | 4.03, 16,603 | 1.34, 1,020 |
| Streaming, Markdown 7 bytes a frame | 1.39 (1.17–1.45) | 1.05 (0.95–1.32) | 0.63, 2,234 | 0.43, 1,073 |
| Streaming into a 1,000-line paragraph | 3.77 (3.62–3.82) | 2.40 (2.09–2.60) | 1.97, 9,133 | 1.10, 5,054 |
| Streaming into a 1,000-line Go block | 4.89 (4.77–5.28) | 1.55 (1.39–1.90) | 1.92, 1,084 | 0.98, 1,055 |
| Streaming a 100 KB line, 1 KB a frame | 24.34 (23.04–25.63) | 8.04 (7.61–11.67) | 28.00, 17,958 | 6.37, 1,105 |
| Resized, a 2,000-line answer | 53.47 (49.53–58.59) | 17.17 (16.70–19.28) | 44.22, 26,623 | 2.29, 4,223 |
| Scrolled a row a frame, 500 exchanges | 1.89 (1.71–2.05) | 1.78 (1.63–2.04) | 1.44, 1,830 | 1.39, 1,799 |
| The inspection, 100 messages | 9.99 (9.16–10.01) | 5.88 (5.53–8.12) | 3.98, 19,340 | 2.72, 11,522 |
| A 5,000-line preview, scrolled | 1.13 (1.11–1.22) | 1.20 (1.07–1.31) | 0.41, 1,264 | 0.40, 1,253 |
| A 5,000-line preview, its source and back | 1.38 (1.25–1.69) | 1.24 (1.13–1.30) | 0.53, 1,447 | 0.43, 1,305 |

An unchanged answer 3.6 to 4.2 times faster, with a quarter of the bytes;
a streaming one 1.3 to 3.8 times (the least where a frame reads its last
paragraph again: 7 bytes a frame, the 1,000-line paragraph), with a third
to two thirds of the bytes; a resize and a 100 KB line streaming 3 times
faster, with a twentieth and a quarter of the bytes. The conversation of 500 exchanges,
the scroll and the preview are within their ranges: their frames' cost is
elsewhere, zone A's workflow tree (53 % of the 500 exchanges' bytes) and
the window's drawing (the preview's time), which this request leaves as
they are. No workload is slower beyond its range.

**Memory and invalidation.** Every part is a memo: a nil cache draws the
same frame. A text is taken again only when equal; from where it grew only
when all its lines but the last are those of the text before, compared,
the point it is read again from being before anything a growing last line
can change; its wraps only at the same width and base. Kinds and wraps are
never changed once made; a new text has new slices, the wraps of the
lines it carried over shared. What is kept is what the last two frames
used: the texts, the wraps, the lines drawn, the runs; a text scrolled out
of view, the inspection closed, a file no longer shown are dropped at the
second frame after. The scratch keeps at most 4,096 cells (196 KB); a
longer line's are dropped with it. The entries' counts are one per entry
of the conversation, which only grows, as before.

**Tests**, `internal/render/mdcache_test.go`, seven:
- `TestMarkdownCacheStream`: 400 random texts of every block kind and of
  spans over lines, fences, strings and tables, streamed 1 to 40 bytes a
  frame, split as zone B's answers and as the inspection's messages, at
  widths that change: every frame's lines, kinds, wraps and rows' count
  those a nil cache gives (1,577 texts read whole, 9,475 from where they
  grew); a frame reading a text twice reads it once; a text replaced, cut
  or changed inside read whole.
- `TestMarkdownCacheCases`, 16 sequences: a span closing on the next line,
  a blank last line growing into the paragraph's, a table's rule arriving,
  a row widening its table, code streaming in a fence with a raw string
  open, a fence closing, an item and a quote going on, a byte order mark,
  the last line changed, a line inside changed, the line before the last
  changed, cut, cut after a paragraph, replaced, back to the text before;
  each read from where it grew or whole as expected, then at another width
  and base.
- `TestMarkdownCacheWrap`: a wrap from the cells alone, in the scratch,
  breaking every random line of every kind where its drawing's cells do,
  at widths 1 to 80; a wrap allocates its rows alone.
- `TestMarkdownCacheFrames`: every frame through the cache the frame
  without it: zone B streaming 13 bytes a frame after a finished answer,
  scrolled and at widths that change, the turn ended (the result's box),
  the inspection of an agent whose messages grow, a preview scrolled and
  shown as its source and back.
- `TestMarkdownCacheRetention`: a long answer out of view dropped, its
  wraps too; a long line's cells not kept; the file's kinds kept while
  previewed (not read again) and while its source is shown, dropped once
  the viewer closed; the inspection's texts dropped once it closed.
- `TestMarkdownCacheBytes`, a guard on what a frame allocates, which the
  load does not change: unchanged 0.82 MB (at most 1.3), streaming 1.87 MB
  (3), resized 2.31 MB (4), a long line streaming 7.32 MB (10).
- `TestMarkdownAcrossLinesCache` (`mdspans_test.go`) through the new cache.

**Mutants** (`scratchpad\plan71\mkmutants71c.mjs`, `mutrun71c.sh`,
through `-overlay`, one at a time, the code hashed before and after:
unchanged): 27 on the cache. The first run, 25: 23 caught, 1 survived (S2,
the wrap built with its styles and colours: the same rows, only the bytes
differ) and 1 did not compile (R5); a check of a wrap's allocations added
(which found the tests' record allocating on every line, now kept only
for them), R5 rewritten to compile, and two mutants of that record added:
the four caught. None survives.

**The checks**, one at a time, no Go run beside another: gofmt empty;
`go build ./...` and `go vet ./...` (Windows and Linux) ok after the last
code change (its hash `e41de931` over the changed non-test Go files);
`go test ./...` once after it, ok (16 packages, 396 s); `-race` on
internal/render, a first run failing on `TestMarkdownCacheBytes` alone (no
race report: under -race a frame allocates more, 1.34 MB where it
allocates 0.82 MB), the guard then judged outside -race only and logged
under it (`raceBuild`, as internal/app's timings are), the run again: exit
0 in 169 s, no race report; internal/app exit 0 in 371 s, no race report. The golden frames
unchanged. No timing step, as the owner asked.

```text
Evaluation - Plan 71, the Markdown cache - 8 criteria: 8 PASS, 0 FAIL
PASS C1 Exact: TestMarkdownCacheFrames, every frame through the cache the frame without it, zone B streaming 13 bytes a frame, scrolled and at widths that change, the turn ended, the inspection of an agent whose messages grow, a preview scrolled and shown as its source and back; TestMarkdownCacheStream, 400 random texts streamed 1 to 40 bytes a frame, both splits: every frame's lines, kinds, wraps and rows' count those of a nil cache (1,577 read whole, 9,475 from where they grew); TestMarkdownCacheWrap: a wrap from the cells alone breaking every random line where its drawing does; TestMarkdownCache, TestMarkdownAcrossLinesCache and every earlier test unchanged and passing
PASS C2 Invalidation: TestMarkdownCacheCases, 16 sequences, read from where they grew or whole as expected (a span closing on the next line, a blank line growing into the paragraph, a table's rule arriving, a row widening its table, a fence's raw string open, a fence closing, an item and a quote going on, a byte order mark, the last line changed, a line inside changed, the line before the last changed, cut, cut after a paragraph, replaced, back to the text before), then at another width and base, each equal to the nil cache's
PASS C3 Memory: TestMarkdownCacheRetention: a long answer out of view dropped at the second frame, its wraps too (at most 500 wraps and 200 lines drawn kept); a 100 KB line's cells not kept (the scratch at most 4,096 cells); the file's kinds kept while previewed, not read again, and while its source is shown, dropped two frames after the viewer closed; the inspection's texts dropped two frames after it closed
PASS C4 Faster, no workload slower: the 18 benchmarks, five runs each in turn on the code before and after: unchanged 2,000-line answer 2.48 to 0.60 ms, 3.52 to 0.82 MB; a byte streamed at 2,000 lines 9.20 to 2.41 ms, 6.20 to 1.89 MB; resized 53.47 to 17.17 ms, 44.22 to 2.29 MB; a 100 KB line streamed 24.34 to 8.04 ms, 28.00 to 6.37 MB; the inspection 9.99 to 5.88 ms; the conversation of 500 exchanges, the scroll and the preview within their ranges
PASS C5 Guarded: TestMarkdownCacheBytes, a frame's bytes, which the load does not change: unchanged 0.82 MB (at most 1.3), streaming 1.87 MB (3), resized 2.29 MB (4), a long line streaming 7.32 MB (10), judged outside -race; TestMarkdownCacheWrap: a wrap allocates its rows alone
PASS C6 Mutants: 27 through -overlay, one at a time, the code's hash unchanged by each run: the first 25, 23 caught, S2 surviving (a wrap built with its styles: the same rows) and R5 not compiling; the check of a wrap's allocations added, R5 rewritten, two mutants of the tests' record added: the four caught; none survives
PASS C7 The checks: gofmt -l . empty; go build ./... ok; go vet ./... ok (Windows and Linux); go test ./... ok once after the last code change (16 packages, 396 s); -race on internal/render (169 s) and internal/app (371 s), exit 0, no race report
PASS C8 The documents: ui-design.md (Markdown, Streaming: the cache and its numbers); architecture.md (Markdown rendering: the cache's parts); this section; the delivery report's coverage, manual acceptance and diff; plan.md's progress
Not verified: the cache in Windows Terminal (no manual test again, as the owner asked); a frame's percentiles (Go's clock steps by 0.5 ms here: the tail is measured at the largest sizes and by the bytes)
Verdict: DONE
```

**Trade-offs.** One more cache in `WrapCache`, of a few entries (the texts
two frames drew), each holding its lines' kinds and wraps (about 150 bytes
a line): an answer in view is kept, not read again. The cache reads a
grown text again from its last paragraph: a paragraph that streams is
paired again whole each frame, bounded by the paragraph. A frame drawing
another text at a place reads it whole, as before.

**What it leaves** (measured above; outside the Markdown cache, or
inherent):
- An entry's rows in view are built whole each frame, a streaming one's
  twice (counted, then drawn): about 0.5 MB for 2,000 lines, now at once;
  zone B's row building.
- The inspection builds every row of every message each frame and keeps
  every line drawn for two frames (the shown lines' cache): its frame and
  its memory follow the inspected agent's messages (5.9 ms and 2.7 MB for
  100 messages of 20 lines). Drawing only the rows shown would change the
  Messages tab's structure.
- A resize wraps every line drawn again at its new width: 17 ms for a
  2,000-line answer in view.
- A line that streams is drawn again whole each frame: 8 ms for a 100 KB
  line.
- A conversation of 500 exchanges spends its frame in zone A's tree, not
  in Markdown.

## Approval and closure (2026-10-31)

**Approved.** After an independent review of plan 71, its consistent
Markdown rendering, the multiline span correction and the Markdown cache,
the owner wrote: "I APPROVE the implementation. No blocking issues were
identified.", and authorised the documents' closure, one commit and its
push to the current branch once the final Git checks find no unexpected
change. **Plan 71: CLOSED.**

The results stand as they were reported, the failures with them:
- the plan's 11 criteria: 71.1 to 71.9 and 71.11 PASS; 71.10 an ACCEPTED
  EXCEPTION, not PASS, its two timing runs as reported, TestSearchToFrame
  at p95 16.57 ms over its 16 ms, then TestBannerToFrame at p95 2.25 ms
  over its 2 ms (The evaluation; 71.10, timed again on a quiet machine;
  The owner's decision on 71.10);
- the multiline defect's 10: F1 to F8 and F10 PASS; F9 an ACCEPTED
  EXCEPTION, not PASS, TestBannerToFrame at p95 2.73 ms over its 2 ms,
  under 71.10's (The multiline defect);
- the Markdown cache's 8: C1 to C8 PASS, its benchmarks before and after,
  its tests and its mutants as reported (The Markdown cache).

The manual acceptance, as the owner recorded it (Manual acceptance):
AGENTS.md's preview and a real Claude Haiku answer in zone B verified by
hand after the fix. Codex was not tested live again after the fix: its
coverage since is the automated tests', both CLIs' events and recordings
through zone B and the spans over lines streamed on each CLI; its live
evidence is from before the fix, the campaign's Codex answer in zone B,
in which the owner found the defect.

For the closure, as the owner asked, no test, benchmark, race run or
manual test was repeated: the checks are Git's and the documents'.
plan.md's section of plan 71, its design, criteria, steps and progress,
is kept word for word under The plan as it ran, as the plan's one commit
holds it, plan.md keeping the current state only. The commit, "Plan 71:
Consistent Markdown rendering and cache optimization", holds the whole
plan, its code, tests, golden frames and documents; it is pushed after
the final Git checks. Its hash is git's: a commit cannot hold its own.

## Delivery report

### 1. Root cause

The renderer of plan 4, step 8 (`internal/render/markdown.go`;
ui-design.md, Markdown, before plan 71) "styles and never rewrites": every
character of AI text was drawn as received, markers included, only greyed,
so that a row's text, its wrap and a selection's copy were the plain
text's. `**Goal:**` was drawn as written by design, not by accident: the
renderer had no notion of a byte hidden or replaced. Markdown files were
drawn by the file viewer as their source, coloured by Chroma's Markdown
lexer, markers and all.

### 2. Affected surfaces

Rendered now, each drawn through the same display model:
- zone B: the CLI's answers, Claude Code's and Codex's, streaming and
  finished, after a resume too, in the RESULT box too (`cardText`,
  `crow.content`);
- Codex's reasoning summary in a thinking phase's details and its full
  view (`detailRows`, `thoughtView`), the dashboard's own lines there as
  they are;
- zone C: the inspection's Messages tab (`inspectMessages`), its start
  line's description and end line's summary; an agent's summary on its
  row (`agentRow`); the execution timeline's line of an AI message
  (`timelineEntry`);
- the file viewer: a `.md`, `.markdown`, `.mdown`, `.mkd` or `.mkdn` file's
  preview (`previewSpans`), its source a key away.

Raw, as written: the owner's messages, the draft and the editor; the
dashboard's notes, notices and errors; tool rows and outputs; approval
cards and their previews, Codex's auto-review rows; Source Control and
every diff (Source Control's, the turn review's, the Git Graph's), blame;
a Markdown file's source view; a code block's content. A plan approval
shows its request's JSON as received.

### 3. Implementation

- **One display model** (`markdown.go`): each source line's block kind is
  classified as before (its fences, a code block's strings and comments)
  and now a table's rows; `mdShow` says how each byte is shown: as it is,
  in a style; hidden (a marker, an escape's backslash, a fence's marker, a
  byte order mark); or replaced by a glyph that keeps the bytes it stands
  for (`cell.sub`: a bullet, a quote's bar, a table's borders and
  padding, a rule, the space between a link's label and its URL). Inline
  constructs nest (`inline`, `emphasis`, `closeRun`, `link`, `codeEnd`),
  their styles made from the block's base and painted again with the
  palette (`mdStyle`, `mdStyles`, `paint`).
- **The wrap counts what is shown** (`wrapCells`, `WrapCache.mdRanges`,
  the cache's key with the line's kind); a list item's and a quote's later
  rows hang under their text. Rows keep their byte ranges and spans their
  source offsets, so the entry cursor, the search's light, the selection
  and the hit test work on the source unchanged.
- **The lines a frame draws are kept as shown** for the next frame, as the
  wraps are (`WrapCache.show`, swept the same way), and a row's cells are
  found by offset (`rowCells`, `runAt`). Without it a 20 KB line took
  82 ms a frame; with it 0.3 to 0.5 ms (7.3 ms at HEAD).
- **Copy stays the source** (`mdWiden`, `widenLine`, `mdPair`): at each
  end of a selection, the hidden markers of a construct it takes in whole
  come with it; a word of a construct copies as the word.
- **Excerpts** (`mdText`): an agent's row, the inspection's start and end,
  the timeline, which reads a message only as far as its two rows need.
- **The file viewer's preview**: `MarkdownFile`, `previewing`,
  `previewSpans` (matches, the cursor's ground, the shift, atoms),
  `FileColumnAt` for a match's column, the lines' kinds kept by the file's
  lines (`Memo.fileMd`), the title's "preview · " or "source · ", the
  footer's "m source" or "m preview"; `m` in `internal/app/file.go`.
- No new dependency, no state, adapter, editor, Source Control or diff
  change.

### 4. Markdown coverage and limitations

Covered: headings (`#` to `######`, closing `#`s), strong, emphasis, both,
strikethrough (`~~`), inline code (any number of backticks), bulleted
(`-`, `*`, `+`) and numbered (`1.`, `1)`) lists at any depth, quotes at
any depth holding a list or a heading, links, images (their alt text),
angle-bracketed destinations, rules (`---`, `***`, `___`, spaced), GitHub
tables (header, rule, escaped pipes, pipes in code, ragged rows), fenced
code blocks (backticks or tildes, any length, a language, an unclosed
one), escapes before ASCII punctuation, a file's byte order mark. Since
the multiline fix (2026-10-31, The multiline defect), strong, emphasis,
both, strikethrough and code spans over the lines of a paragraph: a closer
closes the nearest opener of its marker and length on an earlier line of
it, before it may open one; a code span runs to the first run of
backticks as long on a later line; a paragraph ends at a blank line, a
heading, a table, a rule, a fence or code, and a list item or a quote
begins another; streamed, an opener shows as written until its closer
comes, on its line or a later one.

Not rendered, shown as written: a link or an image over two lines (a
line's links are its own); a closer of another length than its opener
(`**a` closed by `b*`; CommonMark splits such runs); a code span whose
opening backtick its line's own emphasis or link encloses (CommonMark
gives the code span precedence); a setext heading's underline (`===`;
`---` is a rule); an indented code block (four spaces: text); HTML;
reference links;
autolinks (`<https://…>`); a link's title (shown in its brackets); a task
list's box (`[ ]`); a table's column alignment (cells start left); a row of
a table that does not start with `|`. A table wider than its rows wraps
them, and while a table streams its columns widen as longer cells arrive.
In a Codex reasoning summary, the first line follows the dashboard's
"Codex's summary, its own text: " on its row, so a block marker there (a
heading's `#`, a list's `-`) is shown as written. While a line streams, a
marker shows as received until its closer arrives, then hides (D6). A run
opens before a non-space and closes after one, within a line and over
lines alike, not by CommonMark's rules on punctuation beside it.

### 5. Testing

- **New tests**: `internal/render/markdown_test.go` (TestMarkdownLines,
  TestMarkdownShown with 52 cases, TestMarkdownCells, TestMarkdownWiden,
  TestMarkdownWrap, TestMarkdownZoneB, TestMarkdownStreaming,
  TestMarkdownRecordings, TestMarkdownRaw, TestMarkdownExcerpts,
  TestMarkdownCache, TestMarkdownStyles, TestMarkdownZoneBHang, with
  TestBlockComment and TestMultiLineString kept),
  `internal/render/inspectmd_test.go` (TestInspectMarkdown rewritten,
  TestWrapUnchanged kept), `internal/render/mdpreview_test.go`
  (TestMarkdownPreview), `internal/app/markdown_test.go`
  (TestMarkdownFileKey).
- **Retired**: TestMarkdownSameText and TestStyledRow, whose contract (the
  markers drawn, the plain text's cells) is what amendment 74 reversed.
- **Changed expectations**: three application tests that asserted the
  literal markers now assert the rendered text and the markers' absence
  (`hook_test.go`, `summary_test.go` twice); two render tests' Codex
  summary rows (`summary_test.go`); `TestKeyOverlayRows`, 105 rows with
  `m`; the calls of `codeRow` and `inspectMessages` given their new
  argument in four tests.
- **Results**: 71.1 to 71.10 above; the whole suite after the last edit,
  1,137 tests in 16 packages, exit 0.

### 6. Regression assessment

- **Raw views and diffs**: the code that draws them is untouched;
  TestMarkdownRaw and the source view check them as written; the golden
  frames of every diff view changed only in zone C's timeline line.
- **Zone B's layout**: rows are as wide as before, the wrap only
  shortened by the hidden markers; the frames' review found nothing else.
- **Streaming**: a streaming answer's rows equal the completed answer's
  at every step (TestMarkdownStreaming); the memo draws what no memo draws
  (TestMarkdownCache); TestStreamToFrame's p95 4.29 ms.
- **Selection, copy, search, the entry cursor, the mouse**: their tests
  unchanged and passing; the copy is the source, as before for every line
  without Markdown.
- **Performance**: zone B and the inspection faster than at HEAD on long
  lines (a 100 KB line 0.4 to 1.1 ms a frame, 43.6 ms at HEAD; the
  inspection of 100 KB 3 ms, 30.9 ms at HEAD); the timing step's 20 other
  tests within their targets; TestSearchToFrame's heaviest case over its
  target in the step's run, the miss placed in the search's unchanged
  matcher and this machine's variance (71.10).
- **Plan 70's card, zone A, zone C's layout, Source Control and its
  multi-selection, both CLIs' adapters**: unchanged code, their tests
  passing.

### 7. Final diff summary

Against 3a5e368, nothing committed; with the multiline correction and the
Markdown cache of 2026-10-31 (first reported: code 14 files +1,254 −217,
tests 10 files +992 −240 and 2 new files of 269 lines, documents 5 files
+426 −53):
- **Code** (14 files, +2,007 −241): `internal/render/markdown.go` (the
  display model; the spans over a paragraph's lines, `spans`,
  `mdSegment`, `pairPara`, `lineDelims`, `mayOpen`, `mayClose`, `para`,
  `crossSpans`; the reading resumed from a line, `mdRead` and `mdState`;
  the wrap from the cells alone, `mdWrap`; +1,286 −124), `content.go` (the
  wrap over shown cells, the rows from them, the caches: `WrapCache.show`,
  `lineDelims`, `markdown`, `wraps`, `mdDoc`, `readDoc`, `grownFrom`,
  `sized`, the scratch; +348 −60), `file.go` (the preview, `MarkdownFile`,
  `FileColumnAt`, the file's kinds kept while shown; +152 −4), `cards.go`
  (`mdCardText`, the rows added at once), `inspect.go` (the messages
  through the cache), `memo.go` (`Memo.fileMd`, `keepFile`, the kinds
  dropped), `thinking.go` (Codex's summary), `select.go` (the copy's
  widening), `palette.go` (`mdStyles`, `cursorStyles`), `editor.go`
  (`cell.sub`), `zonec.go`, `dashboard.go`, `keys.go` (`m`);
  `internal/app/file.go` (`m`, the match's column).
- **Tests** (10 files, +992 −240, and 7 new files of 1,390 lines:
  `internal/app/markdown_test.go`, `internal/render/mdpreview_test.go`,
  `mdspans_test.go` (the spans over lines), `mdcache_test.go` (the
  cache), `mdbench_test.go` (its 18 benchmarks), `race_on_test.go` and
  `race_off_test.go`).
- **Golden frames**: 116 files, +539 −539, none since the first report.
- **Documents**: `doc/brief.md` (amendment 74), `doc/ui-design.md`,
  `doc/architecture.md`, `AGENTS.md`, `plan.md` (5 files, +557 −53), and
  this file.

### 8. Manual verification in Windows Terminal

For the owner, on the build of this tree (`go build -o acc.exe
./cmd/agent-command-center`, or `.\acc` from the repository), in ACC's
Windows Terminal profile, maximised first, then at 120x40 and at 80x24
(`wt -p "Agent Command Center" --size 90,30` and `--size 60,18`). Each
step says what to see.

1. **The owner's case.** Resume a session from the home screen (Ctrl+R),
   Prepare summary, Enter. The answer streams; `**Goal:**`,
   `**Decisions:**`, `**Completed:**`, `**Open:**` and `**Recommended next
   step:**` read **Goal:**, **Decisions:** and so on, bold, no asterisk
   left once each closes. A list shows `•` bullets, its wrapped rows under
   their text; inline code shows without backticks, in blue.
2. **Constructs.** Ask for an answer with a table, a fenced code block, a
   quote and a link: the table boxed (`│`, `├─┼─┤`), its header bold; the
   code block's fences gone, its language amber, its code coloured and
   literal; the quote behind a grey `│`, italic; the link "label (url)".
3. **Streaming.** While an answer streams, a marker may show until its
   closer arrives, then disappears; nothing beyond zone B's edge, no row
   jumping when a code fence closes.
4. **Both CLIs.** The same with `.\acc --provider codex`: Codex's file
   links "name (path)", without the angle brackets.
5. **Copy.** Drag over a bold label, release, paste in Notepad:
   `**Goal:**`. Double-click one word of it: the word alone. Select the
   whole answer: its Markdown source.
6. **Search.** Ctrl+F, a word inside a bold label: lit where it is drawn.
7. **The file viewer.** F3, then `/` in the Explorer, a `.md` file (for
   instance `README.md` or `doc/brief.md`), Enter: "preview · " in the
   title, headings without `#`, bullets, tables; `m`: the source, markers
   and Chroma's colours, "source · "; `m` again: the preview. In the
   preview: `/` and a word, n and N; ← →; `g` and a line number; `b` for
   blame; a click on a line; a drag copies the source.
8. **Raw stays raw.** Type `**x**` in the input: it stays `**x**`, and so
   does your sent message in zone B. Open a changed `.md` file's diff in
   Source Control: its lines as written, `+`/`−` and all.
9. **Resizing.** Win+Down from maximised, or drag a corner: the answer
   rewraps by what is shown; nothing overflows.

## Details

### The design as built, and what changed on the way

The design D1 to D8 stands; these refinements are written into
`plan.md`'s design and the documents:
- **A copy's ends take in a construct's markers only when they take in
  the construct whole** (D3): widening over every hidden byte at the ends
  made a double-clicked word of `**Goal:**` copy as `**Goal`; the pairs of
  markers the renderer records (`mdPair`) decide instead.
- **The lines a frame draws are kept as shown** (D2): a 20 KB line drawn
  row by row from scratch took 82 ms a frame (each row showed the whole
  line anew); with `WrapCache.show` and cells found by offset, 0.3 to
  0.5 ms. The same memo serves the inspection (30.9 ms at HEAD for 100 KB
  of messages, 3 ms now) and the preview.
- **A link's `]` is a space glyph**, where the wrap may break, so a long
  URL goes to its own row instead of carrying its label with it (seen on
  P12x's Codex answer at 80 columns).
- **A destination in angle brackets** shows without them (P12x: Codex
  links the files it writes so).
- **A file's byte order mark** is hidden and its first line read after
  it: found while preparing the Windows Terminal check, a README written
  by Windows' tools would have shown its first heading raw.
- **Zone C's timeline excerpt** reads a message only as far as its two
  rows need (`mdText`'s room), where reading all of it each frame would
  have cost a message's length.
- **The inspection keeps its paragraphs** (tabs as four spaces, C0
  controls dropped) as before plan 71; its rows copy what they show.

### The golden frames

`scratchpad\plan71\review\frames.mjs` compared each of the 116 frames
with HEAD's: the same number of rows, each as wide, and the same words
once the old frame's markers are taken out (backticks, `**`, `__`, `~~`,
a link's brackets; a bullet's `-` and a table's `|` counted as `•` and
`│`). 86 frames were explained so. The other 30, read by eye: zone B
follows its end, so an answer two rows shorter brings two rows into view
at the top (a speaker label, a tool row, the start of an earlier answer);
a popup or a card over zone B cuts its text at another word (the branches
popup, the Git Graph's card and scope popup, the draft window); a
conversation shorter than zone B ends with blank rows; zone C's timeline
draws a list's `-` as `•`. One frame's rows 31 to 34 counted 201 cells
in the script, the image chip `🗀` counted wide: the same rows as HEAD's
33 and 34, moved up by two.

### The mutants

50, through `-overlay` (`scratchpad\plan71\mkmutants71.mjs`,
`mutrun71.sh`), one at a time, each caught at its first failing test. A
first run, stopped after four, found M2 (an underscore opening inside a
word) and M3 (a closer after a space) surviving: the tests had no case of
either; both added, with the byte order mark's. The full run on the code
after those additions: 45 caught, 5 that did not compile (a variable left
unused); those 5 rewritten to compile, and caught (`mutrun71-run2.out`).
The code's hash e8431f30 before and after both runs; after them only
tests were added (blame and the marks beside the preview, go to line, the
draft), the non-test code's hash 5e5d97d3 the same before and after.

| Mutant | What it breaks | Caught by |
| - | - | - |
| M1 | strong drawn as emphasis | TestMarkdownShown, TestInspectMarkdown, TestMarkdownPreview, TestMarkdownRecordings |
| M2 | an underscore opening inside a word | TestMarkdownShown |
| M3 | a closer after a space | TestMarkdownShown |
| M4 | an escape's backslash shown | TestMarkdownShown, TestMarkdownCells |
| M5 | a code span's content parsed | TestMarkdownShown, TestInspectMarkdown, TestMarkdownRecordings |
| M6 | no space between a link's label and URL | TestMarkdownShown, TestMarkdownExcerpts, TestMarkdownRecordings |
| M7 | angle brackets kept | TestMarkdownRecordings |
| M8 | a heading's #s shown | TestMarkdownShown, TestInspectMarkdown, TestMarkdownPreview |
| M9 | one bullet at every depth | TestMarkdownShown, TestMarkdownPreview, TestMarkdownWrap |
| M10, M11 | no hang for a list item, a quote | TestMarkdownWrap, TestMarkdownZoneBHang |
| M12 | cells not padded to their column | TestMarkdownShown, TestInspectMarkdown, TestMarkdownPreview |
| M13 | a table without its rule | TestMarkdownLines |
| M14 | no list at four spaces in | TestMarkdownLines, TestMarkdownShown |
| M15 | a fence's marker shown | TestMarkdownShown, TestInspectMarkdown, TestMarkdownPreview |
| M16 | strong styled italic | TestMarkdownStyles |
| M17, M18 | a construct's markers copied when not taken in whole; a heading's #s with a part of it | TestMarkdownWiden, TestMarkdownCells, TestMarkdownZoneB, TestMarkdownPreview |
| M19 | the excerpt reading the whole message | TestMarkdownExcerpts |
| M20 | zone B's copy not widened | TestMarkdownZoneB |
| M21 | a single ~ striking | TestMarkdownShown |
| B1, B2 | the byte order mark shown; the first line not classified after it | TestMarkdownShown, TestMarkdownLines, TestMarkdownPreview |
| C1 | the wrap by the source's cells | TestMarkdownWrap |
| C2 | the memo's key without the width | TestMarkdownCache |
| C3 | the memo never swept | TestMarkdownCache |
| C4 | a row's first cell lost | TestConversationLook, TestCodexSummaryDetails, and others |
| C5 | a span across hidden bytes | TestMarkdownRecordings |
| C6 | no hang in zone B | TestMarkdownZoneBHang |
| C7 | zone B's answers raw | TestMarkdownZoneB, TestMarkdownRecordings, TestMarkdownZoneBHang |
| T1, T2 | Codex's summary raw in the details, the full view | TestCodexSummaryDetails, TestCodexSummaryLines, TestMarkdownExcerpts |
| I1, I2 | the Messages tab raw; the start line raw | TestInspectMarkdown, TestMarkdownRecordings |
| D1, Z1 | an agent's row raw; a notice rendered in the timeline | TestMarkdownExcerpts |
| F1 to F8 | `.md` not previewed; the source previewed; no cursor ground; no match lit; a match's column; no m hint; no mode in the title; the preview's copy not widened | TestMarkdownPreview |
| S1 | zone B's copy not widened (the selection's side) | TestMarkdownZoneB |
| A1 to A3 | m keeping the shift; m in any file; a match's source column in the preview | TestMarkdownFileKey |
| P1, P2 | the Markdown styles, the cursor line's, not painted again | TestMarkdownStyles |

### The checks

- `go build ./...` ok; `go vet ./...` ok, and with `GOOS=linux`;
  `gofmt -l .` printed nothing.
- `go test ./... -count=1` after the last edit: exit 0, 16 packages, 364 s
  (`scratchpad\plan71\evidence\suite-final.txt`); 1,137 tests listed.
- `-race`, WinLibs GCC 16.2.0, `CGO_ENABLED=1`, one package at a time:
  `internal/render` exit 0 in 195 s, `internal/app` exit 0 in 377 s, no
  race report (`evidence\race-render.log`, `race-app.log`).
- The installed CLIs, Claude Code 2.1.303 and Codex 0.163.0, are the last
  probed versions.

### The timing step

Alone, on mains power (battery 98 %), the CPU at 19 to 25 % before it,
the owner's usual applications open (Chrome, Edge, Teams, Outlook,
Notepad, Paint, Task Manager, Settings, Move Mouse, the Claude app), no
other Go process; at 20:46:09, 88.6 s (`evidence\timing-step.log`,
`timing-conditions.txt`):

| Test | Measured | Target | Result |
| - | - | - | - |
| TestKeystrokeToFrame | 10,000-line draft: p95 1.07 ms; the draft window: p95 1.13 ms | 16 ms | PASS |
| TestDraftKeystrokes | 50,000 lines, 9 saves: p95 3.00 ms | 16 ms | PASS |
| TestKeystrokeCost | 249 µs and 598 allocations a key (10,000 lines); 655 µs (50,000) | none | PASS |
| TestLongSession | delta to frame at the end: p95 2.76 ms; heap 9.3 MiB | 16 ms | PASS |
| TestThinkingFrame | p95 2.51 ms | 16 ms | PASS |
| TestPasteToFrame | 50,000 lines: max 1.13 ms | 100 ms | PASS |
| TestStreamToFrame | 2,000-line answer: p50 2.00, p95 4.29, max 12.48 ms | 16 ms | PASS |
| TestSelectorToFrame | p95 1.05 to 1.51 ms | 16 ms | PASS |
| TestMotionToFrame | p95 1.05 to 1.76 ms; idle reports all dropped | 16 ms | PASS |
| TestDiffToFrame | p95 1.02 to 1.53 ms | 16 ms | PASS |
| TestMessageToFrame | p95 1.24 and 1.60 ms | 16 ms | PASS |
| TestOutputToFrame | 64 MiB of output: p95 1.02 ms | 16 ms | PASS |
| TestToolsToFrame | p95 2.52 ms | 16 ms | PASS |
| TestBannerToFrame | p95 1.72 ms | 2 ms | PASS |
| TestHomeKeysToFrame | p95 0.17 ms | 16 ms | PASS |
| TestHomePictureTime | max 203 ms | 260 ms | PASS |
| TestBranchesFrameCost | p95 2.12 ms | 16 ms | PASS |
| TestBlameFrameCost | p95 1.33 ms | 16 ms | PASS |
| TestGraphFrameCost | p95 1.00 ms | 16 ms | PASS |
| TestTurnReviewFrameCost | p95 1.51 ms | 16 ms | PASS |
| TestSearchToFrame | keys, two hours: p95 5.02 ms; answers ten times longer: p50 10.02, p95 16.57, max 21.13 ms; deltas, the bar open: p95 4.07 ms | 16 ms | FAIL |

The miss was investigated as the guide asks, without rerunning the step:
- **The same test on the last commit**: HEAD's tree (`git archive` of
  3a5e368 in the scratchpad) and this one, their test binaries built once
  and run in turn, alone: answers ten times longer p95 10.56, 12.20 and
  12.00 ms (HEAD) against 12.06, 12.81 and 12.41 ms (plan 71), p50 8.07 to
  9.12 against 9.06 to 9.69 ms.
- **CPU profiles** of both, restricted to the timed frames
  (`-focus=frameTime`): the search's matcher (`ConvMatches`, unchanged
  code) about 43 % of the samples in both, the frame (`Draw`) 17 %;
  the second profile ran about 45 % slower throughout, the unchanged
  matcher included.
- **Update and View apart** (a scratch test through `-overlay`, three runs
  each in turn): the frame, where plan 71 works, p50 1.0 to 2.0 ms on both
  trees; the search's Update p50 7.5 to 9.8 ms, its p95 from 10.1 to
  21.8 ms between runs of one binary.

Plans 66 and 69 read this case at p95 9.91 and 9.56 ms; today's HEAD reads
10.6 to 12.2 ms.

### Windows Terminal (71.9)

`scratchpad\term\p71runs.ps1` on `p70b.ps1`, the final code's build
`acc-p71.exe` (sha256 d1056dab68e580ec…), plan 71's stand-in
`scratchpad\plan71\standin` (`claude.exe`, Claude Code's stream-json, the
answer streamed in 61 `text_delta` events of 12 bytes, 50 ms apart),
`%TEMP%\p71demo` as the project with `README.md` and `bom.md`; screens,
their text and crops in `scratchpad\term\out-p71\<run>`, the runs' log in
`scratchpad\plan71\wt-runs.txt`:

| Run | Size | What it shows |
| - | - | - |
| answer | 208x59, 120x40, 80x24 | a5 while streaming; a7 finished; a9 after Shift+PgUp; a11 after ↑ (no entry stop: an answer is shown whole) |
| file | 208x59, 120x40, 80x24 | b10 README.md's preview; b13 after m, its source; b19 after m and "/needle", the match lit |
| bom | 120x40 | c10 bom.md's preview, its first heading rendered |

Before the runs no Claude Code or Codex session was open in a terminal
(the owner's rule for terminal tests); the Claude app and this session's
own process were left as they were.

### Kept

The scratch evidence stays in the scratchpad, nothing deleted:
`scratchpad\plan71` (the mutants' copies and logs, the review script, the
stand-in, the profiles, the evidence logs, HEAD's tree under `base`) and
`scratchpad\term\out-p71`; `%TEMP%\p71demo` and the harness's
`%TEMP%\p40-out-p71-*` folders. No recent project was added to the
owner's own registry: the harness ran with temporary application data.

## The plan as it ran

plan.md's section of plan 71 as it stood when the owner approved it on
2026-10-31, word for word, its headings one level down: plan.md keeps the
current state only (AGENTS.md, Working rules), and this file the plan's
whole record, as no earlier commit holds it.

### Plan 71 - Consistent Markdown rendering across ACC

Brief amendment 74 (2026-10-30): Markdown shown as formatted content
wherever ACC presents it as such, its markers hidden, raw contexts kept
literal; to be carried out without an implementation approval
checkpoint, then the owner's independent review, nothing committed
before it.

#### What was found (2026-10-30, on 3a5e368)

- **The root cause.** Zone B's renderer (`internal/render/markdown.go`,
  plan 4, step 8; ui-design.md, Markdown) "styles and never rewrites":
  every character is drawn as received, markers included, only greyed,
  so that a row's text, its wrap and a selection's copy are the plain
  text's. `**Goal:**` is drawn as written by design.
- **How a row is built.** An entry's content is its source lines
  (`entryLines`); each wraps into byte ranges of its line
  (`wrapRanges`, cached by `WrapCache`); a row draws its range's bytes,
  styled by runs over the whole line (`mdRuns`), each span keeping its
  source offset (`refText`; a tab or a control character an atom,
  `refAtom`, its bytes kept). Selection, copy (`Screen.Text`: the source
  between two places), the search's light (`lightSearch`) and the entry
  cursor all work on those offsets.
- **The surfaces that show Markdown.** Zone B: the CLI's answers, both
  CLIs', streaming and finished, the RESULT box included (`cardText`,
  `assistant` only); Codex's reasoning summary, in a thinking phase's
  details and its full view, drawn as plain text. Zone C: an agent's
  messages in the inspection's Messages tab (`inspectMessages`, the same
  styles-only rule); the first line of an agent's summary on its row and
  of its description and summary in the inspection, plain. The file
  viewer (plan 17): a `.md` file is its source, coloured by Chroma's
  Markdown lexer, markers and all.
- **The raw ones.** The owner's messages and the editor; tool rows and
  outputs; approval cards and their previews; Source Control and its
  diffs, the turn review's diffs, the Git Graph, blame; the dashboard's
  notes and errors; zone A's stage labels (the owner's prompts). A plan
  approval shows its request's JSON as received, not Markdown.
- **No new dependency is needed**: the renderer is our own (ASM-17);
  Glamour was measured at p95 590 ms a frame on a streamed 2,000-line
  answer in plan 4, and the dependencies are fixed.

#### Design

- **D1 One display model.** `markdown.go` keeps classifying each line's
  block (its fences, code and their strings and comments) and now also
  says how each byte is shown: as it is, in a style; hidden (a marker, an
  escape's backslash); or replaced by a glyph (a list's bullet, a quote's
  bar, a table's borders and padding, a rule). Inline constructs nest:
  strong, emphasis, both, strikethrough, code spans (their content
  literal), links and images, escapes; a marker with no closing one is
  shown as received. Every surface draws through it; none parses
  Markdown on its own.
- **D2 The wrap counts what is shown.** A line's cells are its shown
  bytes and glyphs, hidden bytes none; the wrap fills rows with them, a
  list item's and a quote's later rows hanging under the text. A row
  keeps its byte range and each span its source offset, a glyph an atom
  with its bytes, as a tab: the entry cursor, the search, the selection
  and the wrap cache keep working on the source. The cache's key gains
  the line's Markdown kind; the lines a frame draws are kept as shown for
  the next one, as the wraps are, and a row's cells are found by offset,
  so a long line's rows cost their own cells (step 6's measure: one line
  of 20 KB took 82 ms a frame without it).
- **D3 Copy stays the source.** A selection copies the source of what it
  covers, as ui-design.md's Selection and copy has it (the content, not
  the drawing): the markers between its ends included; at each end, the
  hidden markers of a construct whose content it takes in whole (an
  emphasis's runs, a code span's backticks, an escape's backslash, a
  link's bracket; a heading's `#`s and a fence's marker when it takes in
  all their line shows), so a word of one copies as the word. The search
  finds the source; it lights what of a match is shown.
- **D4 The look**, from the palette: strong bold, emphasis italic, both
  bold italic, strikethrough struck, each on its block's colour (styles
  made from the base, painted again with the palette), code blue,
  headings pale violet bold without their `#`s, a bullet `•` (`◦`
  nested, `▪` deeper) and numbers in the bullet colour, a quote's bar
  `│` grey before italic text, a link's label blue underlined then its
  URL grey in brackets (a destination's angle brackets hidden), a fence's
  language amber on its opening row (both fences' backticks hidden), code
  literal with its colours, a rule across the row grey (`#6B7280`, the
  markers' grey before), a table's cells padded to their column, its
  borders `│` and its header's rule `├─┼─┤` grey, its header bold.
- **D5 Where.** Rendered: zone B's answers of both CLIs, streaming,
  finished, in the RESULT box; Codex's reasoning summary in a phase's
  details and its full view; the inspection's Messages tab; an agent's
  summary on its row and its description and summary in the inspection,
  inline. A Markdown file in the file viewer: its preview, one row per
  source line as now, `m` showing its source and back; line numbers, the
  cursor, go to line, blame, the edit marks, the search and the
  selection unchanged. Raw: everything listed above, a Markdown file's
  source view, code blocks' contents.
- **D6 Streaming.** A line renders as it arrives; a marker shows as
  received until its closing one comes, then hides; an unclosed fence is
  code to the cursor, as now. Finished lines keep their cached rows.
- **D7 Safety.** Shown bytes go through the cells as now (control
  characters and escape sequences drawn visibly, never sent); glyphs are
  fixed strings; nothing reaches the terminal from the text but its
  characters.
- **D8 Nothing else.** No new dependency; the state, the adapters, the
  editor, Source Control and the diffs unchanged.

#### Criteria

- **71.1 The renderer**: each construct of D4, nested, mixed with plain
  text, escaped, unclosed and malformed, at word and line edges, in its
  shown text and styles; every shown cell's source bytes; no control
  character or escape sequence drawn as such.
- **71.2 Zone B**: the owner's `**Goal:**` answer drawn without markers,
  bold, for each CLI's fake, streaming and finished, at 80x24, 120x40
  and 200x60; long lines wrapped by what is shown, every row within B;
  copy, search, the entry cursor, folds and follow as before.
- **71.3 Thinking and zone C**: Codex's summary rendered in a phase's
  details and full view, the dashboard's own lines unchanged; the
  Messages tab, an agent's summary line and the inspection's lines
  rendered.
- **71.4 The file viewer**: a `.md` file opens as its preview, `m`
  toggles; numbers, cursor, go to line, blame, search, shift, marks and
  selection work in both; any other file unchanged.
- **71.5 Raw stays raw**: the owner's messages, the editor, tool outputs,
  approval previews, Source Control's and the turn review's diffs of a
  `.md` file, code blocks: their markers drawn as before.
- **71.6 Streaming**: rows within B at each step of a line whose markers
  close later; the timing step's streamed answer within its budget.
- **71.7 Golden frames**: each changed frame reviewed as a diff, nothing
  changed but Markdown's markers and glyphs.
- **71.8 Nothing else changes**: zone A, zone C's layout, plan 70's card,
  Source Control and its multi-selection, the diffs, the keys and the
  mouse, both CLIs: their tests unchanged and passing.
- **71.9 Windows Terminal**, the stand-in (no quota): an answer with
  Markdown in zone B, maximised, at 120x40 and narrow; a `.md` file
  previewed, `m` toggling to its source.
- **71.10 The checks**: build, vet (Windows and Linux), gofmt, the whole
  suite after the last edit, mutants through `-overlay`, `-race` on the
  changed packages, the timing step alone.
- **71.11 The documents**: brief amendment 74; ui-design.md (Markdown;
  the file viewer's preview and its key; Focus and keys; Selection and
  copy); architecture.md; AGENTS.md's status and scope; this plan;
  `doc/plan71-evaluation.md`.

#### Steps

1. This plan, amendment 74, AGENTS.md's status.
2. The display model and its tests (D1, D2, D4, D7).
3. Zone B, thinking and zone C (D5), with their tests.
4. The file viewer's preview and its key (D5), with their tests.
5. The documents.
6. The checks: the suite, the golden frames reviewed, the mutants,
   `-race`, the timing step.
7. Windows Terminal with the stand-in (71.9).
8. `doc/plan71-evaluation.md`; the owner's review, then the commits.

#### Progress

- 2026-10-30: step 1: this plan, amendment 74 and AGENTS.md's status,
  from the owner's request (amendment 74).
- 2026-10-30: step 2: the display model (`markdown.go`: `mdShow`,
  `mdBuild`, the inline constructs, `mdStyle`, the copy's `mdPair` and
  `mdWiden`; `cell.sub`; `wrapCells`, `rowCells`, `shownRow`,
  `shownPlain`); `markdown_test.go` rewritten (the constructs, 4,000 random
  lines' invariants, the copy, the wrap), `TestMarkdownSameText`,
  `TestStyledRow` and their styles-only contract retired.
- 2026-10-30: step 3: zone B (`cardText`, `crow.content`), Codex's summary
  in the details and the full view, the inspection, an agent's row, zone
  C's timeline (its excerpt read as far as two rows need); a line's shown
  form kept by `WrapCache.show` after a 20 KB line measured 82 ms a frame
  (0.3 to 0.5 ms with it, 7.3 ms at HEAD; 100 KB 0.4 to 1.1 ms, 43.6 ms at
  HEAD; the inspection of 100 KB 3 ms, 30.9 ms at HEAD); a link's `]` a
  space glyph, where the wrap may break; tests for both CLIs' events,
  their recordings (P22o, P12x, P7c), streaming, the raw surfaces and the
  memo. Three app tests asserted the old literal markers and now assert
  the rendered text and the markers' absence.
- 2026-10-30: step 4: the file viewer's preview (`previewSpans`,
  `filePreview`, `Memo.fileMd`, `FileColumnAt`), `m` and its hints; its
  render and application tests.
- 2026-10-30: step 5: ui-design.md (Markdown, Thinking, Search, the file
  viewer and its key, Focus and keys, Rendering, Colours, Selection and
  copy), architecture.md (Markdown rendering, ASM-17), this plan's designs
  D2 to D4; `TestKeyOverlayRows` counts 105 rows.
- 2026-10-30: step 6, the golden frames: 116 rewritten, each against HEAD
  by a script (`scratchpad\plan71\review\frames.mjs`: the same rows, each
  as wide, the same words once the old frame's markers are taken out):
  86 explained so, the other 30 read by eye, rows coming into view or
  leaving it as zone B, following its end, grew shorter, a popup or a card
  cutting B's text at another word, two blank rows under a short
  conversation, the timeline's bullets; nothing else changed.
- 2026-10-30: step 6, the mutants (`scratchpad\plan71\mkmutants71.mjs`,
  `mutrun71.sh`, through `-overlay`, one at a time): a first run stopped
  after M2 and M3 survived, the tests lacking an underscore opening inside
  a word with a closer after it and a closer after a space; their cases
  added, with a byte order mark's (a `.md` file written by Windows' tools
  showed its first heading raw: fixed, `mdLine.bom`). The full run on that
  code, 50 mutants: 45 caught by a failing test, 5 that did not compile
  rewritten to compile and then caught; none survived; the code's hash
  e8431f30 before and after.
- 2026-10-30: step 6, the other checks: `-race` on internal/render and
  internal/app, one at a time, no race report; tests added for blame and
  the marks beside the preview, go to line in it and the draft kept raw
  (the non-test code's hash 5e5d97d3 unchanged); build, vet (Windows and
  Linux), gofmt, `go test ./...` ok after the last edit (16 packages,
  1,137 tests). The timing step alone, on mains power: 20 of 21 within
  their targets, TestStreamToFrame p95 4.29 ms; TestSearchToFrame's answers
  ten times longer p95 16.57 ms over its 16 ms, a FAIL. Investigated
  without rerunning the step: on HEAD's tree and this one in turn, p95
  10.6 to 12.2 ms against 12.1 to 12.8 ms; with Update and View apart,
  the frame p50 1.0 to 2.0 ms on both, the search's unchanged matcher p50
  7.5 to 9.8 ms with a p95 from 10.1 to 21.8 ms between runs of one
  binary.
- 2026-10-30: step 7, Windows Terminal: the stand-in streaming a Markdown
  answer, 7 runs at 208x59, 120x40 and 80x24, every poll found: the answer
  while streaming and finished, scrolled; README.md's preview, m to its
  source and back, a search lit past the window's edge; a file with a
  byte order mark.
- 2026-10-30: step 8, `doc/plan71-evaluation.md`: 10 PASS, 71.10 FAIL
  (the timing miss), the 8-item delivery report and the manual steps;
  waiting for the owner's independent review and decision on 71.10;
  nothing committed.
- 2026-10-30: 71.10 timed again on a quiet machine, the owner's choice
  ("Re-time when quiet"): at 22:22:15, on mains power, the processor
  counter's mean 13.3 %: TestSearchToFrame within (p95 14.22 ms), and
  TestBannerToFrame, the home screen's banner, p95 2.25 ms over its 2 ms,
  20 of 21 again; the banner within 2 ms on HEAD's tree and this one in
  turn (p95 1.58 to 1.88 ms). Recorded beside the first run in the
  evaluation, the first block unchanged; the decision on 71.10 the
  owner's.
- 2026-10-30: the owner accepted 71.10 as a documented performance
  exception ("The available evidence is sufficient. No additional timing
  run is required."): 71.10 an ACCEPTED EXCEPTION, not PASS, both runs and
  their failures kept as reported, the decision and its baseline
  comparisons in the evaluation; no further timing run, no speculative
  performance change. The evaluation: 10 PASS and 71.10 an accepted
  exception. Plan 71 stays open for the owner's manual acceptance in
  Windows Terminal, one test at a time, each reviewed before the next:
  M71-01, a real Claude Code answer rendered, given; nothing committed.
- 2026-10-31: the owner confirmed a defect: a span over two lines of a
  paragraph shown with its markers (AGENTS.md's preview, a Codex answer in
  zone B). Its cause, the renderer reading a line's inline constructs from
  that line alone; fixed in the shared renderer: a paragraph's lines
  grouped as CommonMark does, its runs paired over them (`spans`,
  `pairPara`), each line drawn from inside what the lines above it opened
  (`mdLine.carry` and `cross`, `para`), the copy widened over such a span
  (`crossSpans`). Three cases a first version of the pairing missed found
  and fixed before the delivery (a closer before punctuation on a line
  opening another span, a code span closing on a line with its own, a
  closer right after another's). `mdspans_test.go`: 47 paragraphs, the
  AGENTS.md and Codex reproductions word for word, every surface,
  streaming, the source, the copy, the search, the cache, 3,000 random
  paragraphs. ui-design.md (Markdown; Selection and copy) and
  architecture.md updated. The checks: 37 mutants through `-overlay`, the
  3 survivors' cases added and the one that did not compile rewritten, all
  then caught; gofmt, build, vet (Windows and Linux), `go test ./...` (16
  packages) after the last edit, `-race` on internal/render and
  internal/app with no race report, the golden frames unchanged. The
  timing step once, on mains power, the CPU 4.6 to 25.9 %: 20 of 21,
  TestStreamToFrame p95 6.21 ms, TestBannerToFrame p95 2.73 ms over its
  2 ms (the home screen's banner, no Markdown code on its path); the fix's
  cost a frame measured apart, 2,000 lines 670 to 970 µs against 430 to
  550 µs without it, a sized cache 570 to 860 µs, not applied. The
  evidence in `doc/plan71-evaluation.md`, The multiline defect: 9 PASS, F9
  (the timing step) a FAIL, the owner's to decide. Nothing committed; the
  owner repeats the affected manual tests.
- 2026-10-31: the owner's decision on F9, covered by 71.10's exception
  ("Yes, it covers it"): F9 an ACCEPTED EXCEPTION, not PASS, the run and
  its failure kept as reported; the fix's evaluation 9 PASS and F9 an
  accepted exception. No further timing run, no speculative performance
  change (the sized cache the owner's to decide). Plan 71 stays open for
  the owner's review of the fix and the affected manual tests; nothing
  committed.
- 2026-10-31: the owner's results of the manual tests repeated after the
  fix ("`AGENTS.md` preview: multiline rendering manually verified.";
  "Real Claude Haiku answer in Zone B: multiline rendering manually
  verified."; "Codex: automated post-fix coverage verified; no additional
  live manual retest performed."), the campaign complete; and the owner's
  decision to include the Markdown cache's optimisation in plan 71 before
  its closure. The baseline first, `mdbench_test.go`'s 18 benchmarks, and
  the profiles: an answer's kinds read again every frame (37 % of an
  unchanged frame), twice for a streaming one, the per-line caches filled
  anew each frame, the rows grown a quarter at a time, a wrap building and
  dropping a line's whole drawing. The cache of each Markdown text drawn,
  by place, its lines, kinds and wraps taken unchanged, read again from
  where a grown text stops being sure (its last paragraph or table, a
  fence's state kept), whole on any other change (`WrapCache.markdown`,
  `mdRead`, `wraps`); a wrap from the cells alone in a capped scratch
  (`mdWrap`); the rows added at once; the preview's kinds dropped once the
  viewer no longer shows the file. Before and after, five runs each in
  turn: an unchanged 2,000-line answer 2.48 to 0.60 ms and 3.52 to 0.82 MB
  a frame, streamed at its end 9.20 to 2.41 ms, resized 53.47 to 17.17 ms
  and 44 to 2.3 MB, no workload slower. `mdcache_test.go`, seven tests
  (exact frames, 400 random streamed texts, 16 cases of invalidation,
  retention, a guard on a frame's bytes); 27 mutants, one survivor's check
  added and one rewritten, none surviving; gofmt, build, vet (Windows and
  Linux), `go test ./...` once after the last code change, `-race` on
  internal/render and internal/app, no race report (the bytes guard judged outside -race only, a frame allocating more under it); the evaluation's coverage,
  manual acceptance and diff updated. The evidence in
  `doc/plan71-evaluation.md`, The Markdown cache: 8 PASS. Nothing
  committed.
