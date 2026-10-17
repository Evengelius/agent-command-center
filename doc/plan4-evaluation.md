# Plan 4 evaluation (2026-10-17)

The final check of plan 4, the MVP: amendment 4's ten acceptance criteria
(`doc/brief.md`, "any of the three providers" read as the two of amendment 5),
each for Claude Code and for Codex, and the hard constraints of `AGENTS.md`
that apply to the whole build. The evidence is `plan.md`'s record of each step
and live run; the block below is the summary reported to the owner. The
results are not to be rewritten.

A failure recorded during the plan stays recorded as it was found, with its
cause and what followed; a criterion takes its latest run. Each one is listed
under [Failures on record](#failures-on-record).

```text
Evaluation - Plan 4, the MVP (amendment 4's acceptance criteria) - 14 criteria: 14 PASS, 0 FAIL
PASS 1  Launched from a terminal: L5 started the built executable with wt.exe in a new, maximised Windows Terminal window, for each CLI (acc-l5c-112245, acc-l5x-125316); the startup screen showed the CLI and the folder, "Ready" 2 to 3 s after Enter
PASS 2  Either CLI selectable: TestDetect (neither, either, both), TestStartupGate (the other CLI stays selectable, --provider only prefills), TestParseArgs ("--provider gemini" refused); detected live: Claude Code 2.1.301, Codex 0.145.0 and 0.163.0
PASS 3  Project path validated: TestResolvePath (44 cases), TestCheckProject, TestCheckProjectNotReadable, TestStartupMessageFrames, TestStartupGate
PASS 4  The CLI works in that directory: Claude Code L1 (cwd checked) and L2d ("Ready, cwd" the folder, allow.txt written there); Codex L3 and L4d (cwd in thread/start); L5 found.txt written in the chosen folder; each folder held only its expected file
PASS 5  Real prompts, streamed answers: L2d and L4d, three turns each; turn 2 interrupted mid-stream 8 ms (Claude Code) and 9 ms (Codex) after Ctrl+C; turn 3 answered from the resumed session after a crash and Ctrl+R; L5 one subagent turn per CLI
PASS 6  Files inspected and changed with approvals: L2d Glob then Write allowed with F2, a; L4d fileChange accepted, then read back; L3 decline and cancel; L5 3 approvals (Claude Code) and 1 (Codex) answered in the dashboard, "allowed by you" in B; TestApprovalCards
PASS 7  Multiline input, long pastes, copying: TestInputByteForByte and TestPasteNeverSubmits with the Claude Code, Codex and mock adapters; real Windows Terminal 43 of 43 checks maximised and 42 of 42 at the default size (Shift+Enter, paste, copy, mouse selection, emoji); L5 Shift-drag with mouse capture at both sizes; keystroke timing on mains power p95 1.0 ms, draft p95 2.5 ms (target 16 ms)
PASS 8  Real agent activity: L2d and L4d activityShown nil (stages and tools from the live events); L5 live subagents in zone C, Claude Code "0 ACTIVE · 1 DONE", Codex 0.163.0 "0 ACTIVE · 2 DONE" with no phantom "/root" row; TestAgentsEndCleanly over every recording
PASS 9  Telemetry accurate or marked unavailable: step 7's parsing tests on the probes' exact lines; every zone D value from a named source or N/A with its reason (L2d, L4d); L5 against each CLI's own screen, read by the owner: Codex equal, Claude Code within 1 point and 1 minute, both differences explained
PASS 10 Both screens follow the reference layout and sequence: golden frames of the startup screen and the dashboard for each CLI at 80x24, 120x40 and 200x60; step 8's comparison with reference-design.md; the owner's approval of the look at steps 1, 7 and 8
PASS 11 Configuration untouched: the latest live run per CLI is PASS (Codex acc-l5x-125316) or NOT-VERIFIED only for ~\.claude.json, which Claude Code writes itself (acc-l5c-112245); acc-l5c-105614's FAIL kept below
PASS 12 Dependencies: 4 direct modules, as approved (bubbletea v2.1.0, lipgloss v2.0.6, x/clipboard v0.11.0, x/sys v0.49.0); ultraviolet replaced by a patched copy of its MIT source at the owner's request (third_party/ultraviolet, PATCHES.md, THIRD_PARTY.md), no new module
PASS 13 Scope: none of the features amendment 4 leaves out; no Gemini CLI or Antigravity adapter or option, only a test that refuses the flag (amendment 5); no VS Code terminal or conhost adaptation (amendments 6 and 7)
PASS 14 go build ./... and go vet ./... (also -tags live, -tags timing) exit 0; gofmt -l . prints nothing; go test ./... 177 top-level tests PASS in 14 packages, 0 FAIL, 0 SKIP, at 7e0d746; go test -race ./... ok in every package (step 4's MinGW)
Not verified: the overall look and feel (the owner's call, below); the 256- and 16-colour terminals; synchronized output; live Windows Terminal rendering of step 8's Markdown styles, zone zoom and help overlay; physical key presses; a NUL in a paste; an API-key session; a plan without a 5-hour window; a live get_context_usage refusal; the Codex slash-command note live; ~\.claude.json's changes; who wrote the files changed during acc-l5c-105614. Each with its reason below.
Verdict: DONE
```

## Not verified

What no check of this plan settles, and why. None of it is reported as PASS.

- **The overall look and feel.** The owner's call. The owner approved the
  look on mock demos in Windows Terminal at steps 1, 7 and 8 ("I like it a
  lot" at step 8) and watched the L5 live runs, but never judged the
  application as a whole on a live session; that is the owner's to decide at
  this approval.
- **The 256- and 16-colour terminals.** Windows Terminal reports true colour,
  so every run and golden frame used it. Lip Gloss's downsampling of the
  palette to 256 and to 16 colours was neither tested nor seen; whether the
  zones, statuses and Markdown styles stay distinct there is unknown.
- **Synchronized output** (mode 2026). Not measured; tearing on a fast stream
  was not looked for.
- **Live Windows Terminal rendering of step 8's additions.** The Markdown
  styling, the zone zoom and the help overlay were checked on golden frames
  and seen by the owner on mock data only. The L5 harness reads the screen's
  text through UI Automation, which carries no colour or style.
- **Physical key presses.** Every Windows Terminal check used SendInput, which
  Windows Terminal receives as it does a keyboard, but no physical key.
- **A NUL in a paste.** The Windows clipboard ends text at the first NUL, so
  one cannot be pasted. The editor's handling is tested offline only.
- **An API-key session, a plan without a 5-hour window, a live
  `get_context_usage` refusal.** The first needs a key and API billing, the
  second another account, and the third cannot be caused on a real CLI. Each
  is simulated on mock data (step 7).
- **The Codex slash-command note, live.** The recorded wire settles it
  (`TestCodexSlashCommandNote`); a live check would spend Codex quota.
- **`~\.claude.json`.** Every Claude Code run changes it, and the hash-only
  configuration check the owner approved cannot classify the change; its
  contents are never read.
- **acc-l5c-105614's file changes.** The application writes no file, but that
  it caused none of those changes is inferred from their timestamps (the
  Codex update and the owner's own sessions), not verified.

Outside the MVP, not checked: Linux and macOS (one Windows 11 machine);
each subagent's model in zone C, a later improvement at the owner's request
(zone C shows "model N/A" for Codex children today).

## Failures on record

Each stays as it was recorded in `plan.md`; none is open.

| Where | Failure | What followed |
| --- | --- | --- |
| Step 2, L1 (c), first run | Claude Code refused the test prompt's standalone `sleep 30`, so no interrupt was sent | The test's fault; the prompt fixed, the check run again alone: PASS |
| Step 3, criterion 9 | Shift+Enter sent the message in Windows Terminal; in conhost also three input cells (emoji twice, right-click paste) | Windows Terminal fixed at step 4 (win32-input-mode, patched ultraviolet): 43 of 43 and 42 of 42; conhost out of scope by amendment 7 |
| Step 4, L2 runs 1 to 3 | Run 1 a test defect; run 2 an application defect (a streaming answer marked finished); run 3 two test defects (resume argument order, quit timing) | Each fixed; run 4 passed every check, and L2d again on the final build |
| Step 5, check 8 | Keystroke timing missed 16 ms under machine load | The timing procedure revised; PASS under it, and at step 8 on mains power (p95 1.0 ms) |
| Step 5, L3 run 1; L4 runs 1 and 2 | Test defects: thread and cwd read too early; the default model's fallback not followed; a second request stopped the run | Each fixed; the last L3 and L4 runs PASS, and L4d on the final build |
| Step 6, criterion 1 | P3b gave 2 stages, not 1, against the original wording | The owner amended the criterion on 2026-10-17 (Claude Code opens a second turn for the subagent's report); PASS under it |
| Step 7, criterion 6 | Long-session frame time p95 18 to 26 ms | Bounded log, zone C and B drawn only where shown; PASS (the first result kept) |
| L5, acc-l5c-105614 | Configuration FAIL: `~\.codex\config.toml` changed, and `~\.claude.json` and `known_marketplaces.json` changed, not classified | From the timestamps: the Codex 0.163.0 update and the owner's own Codex and Claude Code sessions during the run; the rerun acc-l5c-112245 NOT-VERIFIED for `~\.claude.json` only |
| L5, acc-l5x-112401 (Codex 0.163.0) | L2.1, Q1, Q2: a phantom "/root" agent that never ended, so zone C never reached "0 ACTIVE" and Ctrl+Q asked to confirm; L2.3: the frame changed by input from outside the harness | Probe P7b; mapper and reducer fixed, `TestAgentsEndCleanly` added; acc-l5x-114152 and acc-l5x-125316 every check PASS |

## How the check ran

- **Main session only, no verifier agent.** The deterministic criteria
  (2, 3, 12 to 14) were settled by commands run on 2026-10-17 at 7e0d746,
  with a clean working tree, the race detector included; the live criteria
  rest on the runs recorded in `plan.md`, each checked when it ran.
- **The owner's reading.** Criterion 9's comparison with each CLI's own usage
  screen is the owner's reading, reported as given.
- **No new live run.** Criteria 1, 8 and 11 take the latest L5 runs, which
  passed after the fixes; a further run would use quota without settling
  anything those runs leave open.

## Live demo before approval (2026-10-17)

At the owner's request, before approving: the built executable at 7e0d746,
started with `wt.exe` in a new, maximised Windows Terminal window, once per
CLI, each in a fresh empty folder under `%LOCALAPPDATA%\Temp` with the
configuration check before and after; the owner drove both sessions. No
Claude Code or Codex session was open in a terminal; none was closed.

- **Claude Code** (`acc-demo-c-133714`, Haiku for that process only): a
  greeting, then four Explore subagents in parallel, all four finished
  (13:40:38 to 13:41:15) and a combined answer. Configuration NOT-VERIFIED for
  `~\.claude.json` only; the folder stayed empty; no CLI process left.
- **Codex 0.163.0** (`acc-demo-x-134301`, `gpt-6-sol`): a greeting, then four
  children (`/root/structure`, `/root/config`, `/root/issues`, then
  `/root/tests`), each reporting to `/root`, all four completed (13:44:13 to
  13:44:35), a combined answer and a last turn. Configuration PASS; the folder
  stayed empty; no CLI process left.
- Evidence: each CLI's own session files for the two folders, read for the
  prompts, the subagents and their ends; the owner's reading of zone C: "They
  were all done".

## The owner's decision (2026-10-17)

After the live demo above, the owner approved plan 4: "Yes do so". The block
above stays as reported; the items under [Not verified](#not-verified) stay
not verified.
