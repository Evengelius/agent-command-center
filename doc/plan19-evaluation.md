# Plan 19 evaluation (2026-10-20)

The final check of plan 19, zone B as the centerpiece (brief amendment 15).
It checks the plan's five items in
[`plan.md`, Plan 19](../plan.md#plan-19---zone-b-as-the-centerpiece) against
the yes/no criteria written there before the code: a consistent visual
language for the conversation, after the owner's three images, and runs of
tool calls that fold. What B shows is still the CLI's conversation, entry
for entry; only how it is drawn changes.

The evidence is the tests, the golden frames, six mock runs and two live
runs in a real Windows Terminal, each scene read as text through UI
Automation and captured as a screenshot. Earlier attempts that failed are
recorded below as they were: one on a locked workstation, four on the test
script, one on Claude Code's own cache. The block below is the summary
reported to the owner. The results are not to be rewritten.

```text
Evaluation - Plan 19, zone B as the centerpiece - 19 criteria: 19 PASS
PASS 1.1 "▎YOU  11:30", bar and label #A78BFA, the prompt #F9FAFB, then a rule of ─ from column 2 to the edge (TestConversationLook)
PASS 1.2 "▎CLAUDE  Haiku 5.5", bar and label #60A5FA, the model in Haiku's colour, once per exchange: the exchange's later messages have no label; "▎CODEX  gpt-5.5" (TestConversationLook)
PASS 1.3 claude-haiku-5-5 Haiku 5.5, claude-sonnet-4-5-20250929 Sonnet 4.5; claude-opus-5-5[1m], gpt-5.5 and three other ids as given (TestModelName, 8 cases)
PASS 1.4 "▎✓ RESULT" #4ADE80 over the last message of a completed exchange with a call; none while it runs, after an interrupted or a failed turn, or over a lone answer; after a turn the CLI opens, on its message once it completes, and kept after the next prompt (TestConversationLook)
PASS 1.5 ✓ ✗ ! ○ ◉ and the spinner in their status colours; "✓ Agent  general-purpose · List directory and write found.txt"; a Read with its path; the ground #1E2133 on every span from column 2 to the row's end, none on 16 colours (TestConversationLook)
PASS 1.6 A pending card "! APPROVAL orchestrator · Bash  ls -0  <time>", "  F2 to answer", no ground; answered, one row on the ground, "allowed by you" #4ADE80, "denied by you" #F87171; "approved by auto-review" #4ADE80 (TestConversationLook)
PASS 1.7 A thinking mark between two calls not drawn; after the last call, "  ∴ thinking" (TestConversationLook)
PASS 2.1 4 calls: no fold row; 5: "› 2 earlier tool events · Expand" and 3 rows; 12: "› 9 earlier tool events · 1 failed · 1 running · Expand" and 3 rows; expanded "▾ … · Fold" and 12 (TestToolFold)
PASS 2.2 A message and an approval split the runs, no fold; a pending card between two runs of 6, whole (TestToolFold)
PASS 2.3 ↑ through the 3 calls in view, then the fold row, never a folded call; Enter expands, ← folds back with the cursor on the row, a click does both; at 80x24 the expanded run opens under its row (TestToolFoldKeys)
PASS 2.4 A selection from the prompt to the result copies the entries' text joined as before plan 19, the 12 calls' names included (TestToolFold)
PASS 2.5 Over 20 steps (a session built event by event, its turn ending, a fold expanded and folded, the CLI's turn, a new prompt), at widths 100 and 61, B's row count with the cache equals the count without it (TestToolFold)
PASS 3.1 "notice:0 user:1 assistant:1 tool:1 notice:1 assistant:2 user:3 assistant:3", the second turn opened by the CLI (TestEntryStage)
PASS 4.1 33 golden frames rewritten; a script compared every cell with HEAD: each change inside zone B (under a window, B's box from the dashboard frame of the same size)
PASS 4.2 go build ./... and go vet ./... exit 0, gofmt -l . empty; go test ./... 348 PASS, 1 SKIP (TestDiskListLink), 0 FAIL in 14 packages; the timing step 4/4 PASS (long session delta-to-frame p95 7.0 ms)
PASS 4.3 ui-design.md (Zone B, Approvals, Colours and symbols, the keys), events-and-state.md (Conversation record) and AGENTS.md (status) say what was built; brief.md holds amendment 15
PASS 5.1 Real Windows Terminal, 4 runs, 0 FAIL: the Claude and Codex mocks maximised (156x44) and at 80x24 (paged with Shift+PgDn), labels, rule, tool rows, answered approvals and RESULT in order; screenshots reviewed
PASS 5.2 The 12-call replay, maximised and 80x24, 0 FAIL: "› 9 earlier tool events · 1 failed · Expand" and 3 rows; ↑ four times to the row; Enter expands (12 rows maximised; at 80x24 the row and the next calls in view); ← folds; two clicks expand and fold
PASS 5.3 Live, Claude Code 2.1.303 on Haiku: maximised and 80x24, six Read calls in one run, "› 3 earlier tool events · Expand" and 3 rows, RESULT over the answer, "ok" under its own label with no RESULT; 0 FAIL, configuration PASS, the project unchanged, no process left; USD 0.015 in all
Not verified: none
Verdict: DONE
```

## What the build found

Two choices of the plan were refined while it was built; both are in
`plan.md` and ui-design.md.

- **A failed call shows its error, not its target.** TestSyntheticScenarios
  failed on the first build: a Codex command's target, its PowerShell
  command line, took the row and cut its "exit 1 · fatal: …" line. A failed
  or denied call now reads as before plan 19, `✗ name · error`, and the
  target shows on the other calls.
- **Expanding holds B's view.** Preparing the terminal runs showed that at
  80x24, with B following its end, an expanded run grew under the view and
  pushed its own row, and the cursor, out of it. Expanding now holds B where
  it is, so the calls open under the row; TestToolFoldKeys checks it at
  80x24.

Six existing tests had expectations of the old look and were updated, their
intent kept: TestToolFailureRow (plan 5's `▸` rows now show their status, ✓
or ◉), TestApprovalLineKeepsOutcome (an answered row starts three cells in:
its width sweep starts at 63, the room 60 gave it), TestQuestionCard,
TestEntryCursor, TestFailedToolRow and TestEntryViewer (the cursor's row
now reads `▸ ✓ name`).

## The terminal runs

`p19mock.ps1` uses the plan 17 harness: a Windows Terminal window driven by
UI Automation, a temporary project, store and home, and a PATH of System32
only, so no agent CLI is found. At 80x24, where B shows a few rows, the
script pages B from its top with Shift+PgDn and joins the pages into one
transcript. The 12-call replay, `fold-12.jsonl`, is synthetic: the Claude
mock's events up to its first call, then twelve Read calls of the
orchestrator 300 ms apart, the fourth failed, then an answer
(`mkfold.js`).

| Run | Checks | Result |
| - | - | - |
| Claude mock, 156x44 | the exchange in order: YOU, prompt, rule, CLAUDE Haiku 5.5, the Agent call, the reply, two answered approvals; RESULT over the answer | 0 FAIL |
| Codex mock, 156x44 | YOU, rule, CODEX gpt-5.6-sol, two collabAgent:wait, the answered fileChange; RESULT over "done" | 0 FAIL |
| 12-call replay, 156x44 | the fold row and 3 calls; ↑ ×4 to the row; Enter 12 rows; ← folds; two clicks | 0 FAIL |
| Claude mock, 80x24 | the same as maximised, from the paged transcript | 0 FAIL |
| Codex mock, 80x24 | the same as maximised, from the paged transcript | 0 FAIL |
| 12-call replay, 80x24 | as maximised; expanded, the row and the calls under it in view | 0 FAIL |

Each run: the window closed, no process left, no agent CLI started by the
dashboard, nothing written in the project.

### Earlier attempts

The first, at 01:10, maximised: the workstation locked during the run, its
screenshot the lock screen; the screen could not be read and the quit's keys
stopped ("the target window lost the focus"). The window was closed by
message and no process was left.

```text
p19mock claude max  2026-10-20T01:10:39
[setup] PASS  claude-subagent-approval.jsonl in C:\Users\amolimom\AppData\Local\Temp\p19-claude-max\demo
[size] PASS  maximised True, 44 rows, 156 columns
[replay ended] PASS  'Found 2 files' on screen
[the tail: RESULT over the answer] PASS  ▎✓ RESULT | The subagent found two files | Found 2 files: alpha.txt and beta.md.
[the exchange: labels, rule, tool rows] FAIL  missing after row -1: '▎YOU'
[total] FAIL  1 FAIL
```

Four more failed on the script, each fixed and the run repeated; the screen
was right in each (its text saved with the run):

```text
p19mock claude max  2026-10-20T01:21:55
[the exchange: labels, rule, tool rows] FAIL  missing after row 13: '✓ general-purpose · Write'
```

"allowed by you" is on the approval's own row; the script looked for it on a
later row. Parts on the same row are now marked as such.

```text
p19mock claude small  2026-10-20T01:24:35
[the exchange: labels, rule, tool rows] FAIL  missing after row 9: '▎CLAUDE  Haiku 5.5'
```

At 80x24 B holds six conversation rows: no screen holds the whole exchange.
The script now pages through it.

```text
p19mock claude small  2026-10-20T01:25:39
[the exchange: labels, rule, tool rows] FAIL  missing after row -1: '▎YOU'
[no agent CLI started] FAIL  new claude/codex processes: 2
```

The pages were nested arrays in the script, so the transcript held none of
the screen's text. Two claude or codex processes appeared during that run
and were gone a minute later, when they were listed: the check had counted
every new one, and could not say whose. The dashboard ran with a PATH of
System32 only, where neither CLI is. The check now names each new process
and fails on one only when the dashboard is among its ancestors; the five
runs after it found none at all.

```text
p19mock fold small  2026-10-20T01:28:10
[the exchange: labels, rule, tool rows] FAIL  the tail after the exchange: missing after row 10: '✓ Read  '
```

The replay's tail starts with a call row, the exchange's last row already
matched (f11.go); the joined check looked for a second one.

## The live runs

`p19live.ps1`, a change of plan 18's approved live run, its prompts only:
the dashboard on Claude Code 2.1.303, Haiku through `ANTHROPIC_MODEL` in that
window only, the CLI through the logging wrapper, a temporary project of six
one-word files, the real home so that Claude Code is signed in, and the
configuration check before and after. Prompt 1 asks for six Read calls in
one message, then a one-line answer; prompt 2 is "Reply with only the word
ok." Any card other than a read would be denied and fail the run; none came.

| Size | Calls, from the CLI's log | B after prompt 1 | B after prompt 2 | Cost |
| - | - | - | - | - |
| 156x44 | text, Read ×6, text | CLAUDE Haiku 5.5, the first text, `∴ thinking`, "› 3 earlier tool events · Expand", four.txt to six.txt, RESULT over the answer | YOU, rule, CLAUDE Haiku 5.5, "ok", no RESULT; prompt 1's RESULT kept | USD 0.0049 |
| 80x24 | Read ×6, text | CLAUDE Haiku 5.5, `∴ thinking`, the fold row, three calls, RESULT | the same | USD 0.0048 |

Both: model claude-haiku-5-5, answers "one.txt: apple, two.txt: river,
three.txt: stone, four.txt: cloud, five.txt: maple, six.txt: ember" and
"ok", 0 FAIL, configuration PASS, the project unchanged, no process left.
The first reply maximised was the owner's Px reminder: the owner's global
`~/.claude/CLAUDE.md` asks every Claude Code session for it, and the
sessions the dashboard starts read that file too.

### An earlier run at 80x24

```text
01:33:22 [confcheck] FAIL  exit 1; mixed  C:\Users\…\.claude.json  key claudeCodeFirstTokenDate  not compared (sensitive name) | mixed  C:\Users\…\.claude.json  key oauthAccount  not compared (sensitive name) | mixed  C:\Users\…\.claude\plugins\known_marketplaces.json  changed, not classified (contents not inspected)  sha256=1338fe7db7322a8dc6d9cc6e685b9307ff3caf825cfa9dc7b9b801cf78e64e3f | configuration: FAIL
01:33:22 [total] FAIL  1 FAIL
```

Its display checks all passed (the six calls folded, RESULT, the lone "ok";
USD 0.0050). The configuration check found
`plugins\known_marketplaces.json` changed: Claude Code's marketplace
refresh, its one `lastUpdated` written at 23:32:56 UTC (01:32:56 here),
during the run, as in plan 18; `settings.json` was last written on
2026-10-19 at 23:00, before any run. The run was repeated and passed.

The three live runs cost USD 0.015, API-equivalent.
