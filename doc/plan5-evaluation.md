# Plan 5 evaluation (2026-10-17)

The final check of plan 5, the post-MVP improvements: the fifteen items of
[`plan.md`, Plan 5](../plan.md#plan-5---post-mvp-improvements), each against
the yes/no criteria written in the progress list before its work started, and
the plan's rules (Windows 11 and Windows Terminal only, no live CLI turn, no
new dependency, amendment 4's scope). The evidence is `plan.md`'s record of
each item; the block below is the summary reported to the owner. The results
are not to be rewritten.

A failure recorded during the plan stays recorded as it was found, with its
cause and what followed; a criterion takes its latest run. Each one is listed
under [Failures on record](#failures-on-record), and each criterion amended
while building under [Criteria amended](#criteria-amended).

```text
Evaluation - Plan 5, post-MVP improvements - 17 criteria: 17 PASS, 0 FAIL
PASS 1  Step 8's additions live in WT (item 1): 6 of 6; wt.exe, maximised and default size, 22 of 22 automatic checks per run; Markdown styles, key overlay and zoom seen in screenshots; no process left
PASS 2  Codex slash note live (item 2): 4 of 4; Codex 0.163.0, /status typed and Enter pressed by the owner on the physical keyboard; the note shown, nothing sent, no session file; confcheck compare -cli-ran PASS, every file identical
PASS 3  Approval card full view (item 3): 9 of 9; TestApprovalViewer, TestViewerWhole, TestViewerDiff, TestViewerHead
PASS 4  Folded entries expanded in B (item 4): 7 of 7; TestEntryCursor, TestEntryViewerBody, TestEntryViewer, TestKeyOverlayRows
PASS 5  Scrolling in zones A and C (item 5): 8 of 8; TestPaneScroll (tree, list, wheel, zoom and 80x24)
PASS 6  A failed tool's line in B (item 6): 8 of 8; TestToolFailureRow, TestFailedToolRow, TestToolError, TestFailedCommand
PASS 7  Each subagent's model in zone C (item 7): 7 of 7; TestSubagentModel (claude, codex), TestAgentModel, TestAgentModelColumn; P3 "claude-haiku-5-5", P7b "gpt-6-sol", P7 "model N/A"
PASS 8  Allow for the session on Codex (item 8): 7 of 7; TestOfferedDecisions, TestCodexSessionAllow, TestApprovalCards, TestStateMachines, TestViewerHead
PASS 9  Failure cards and session lines copied (item 9): 6 of 6; TestSessionStateCopy, TestSessionCutRow, TestSessionStateMouse
PASS 10 A deny message for a question (item 10): 6 of 6; TestQuestionDeny, TestViewerHead, TestStateMachines
PASS 11 Markdown in the inspection's Messages tab (item 11): 5 of 5; TestInspectMarkdown, TestStyledRow, TestWrapUnchanged (3,000 seeded texts)
PASS 12 Block comments highlighted on every line (item 12): 6 of 6; TestBlockComment, TestMarkdownSameText, TestInspectMarkdown; TestStreamToFrame p95 2.26 to 2.43 ms
PASS 13 256- and 16-colour rendering (item 13): 6 of 6; TestColourProfiles through Bubble Tea's own detection and renderer, TestPalette256, TestPalette16, TestPaletteSwitch, TestTrueColours; 8 of 8 frames byte-equal to the recording before the change
PASS 14 Synchronized output, mode 2026 (item 14): 5 of 5; measured in WT 1.23.20211.0, 92 of 95 writes wrapped after WT's answer; TestSynchronizedOutput with WT's recorded answer and three others
PASS 15 API-key session, no 5-hour window, live context refusal (item 15): simulated only, as the owner asked; TestSyntheticScenarios, TestTelemetryNA, TestDetailsKeys, TestMockFiles PASS at the close
PASS 16 go build ./... and go vet ./... (also -tags live, -tags timing) exit 0; gofmt -l . prints nothing; go test ./... 209 top-level tests PASS in 14 packages (177 at cf58739, +32), 0 FAIL, 0 SKIP; go test -race ./... ok in 14 of 14 packages; the files changed by items 13 and 14 have no CR and no BOM
PASS 17 Rules: no live CLI turn, no quota (item 2 started Codex without a turn); go.mod and go.sum unchanged since cf58739, 4 direct modules; Windows 11 and WT only; no Gemini CLI, Antigravity, VS Code terminal or conhost work; none of amendment 4's excluded features
Mutation checks: 80 mutants killed across items 4 to 14 (4: 1, 5: 2, 6: 1, 7: 5, 8: 10, 9: 12, 10: 13, 11: 12, 12: 8, 13: 12, 14: 4), 0 survived but 1 equivalent mutant (item 14); none run for item 3.
Not verified: the overall look and feel (the owner's call); the items 3 to 10 behaviours live in WT during a real turn; a real 256- or 16-colour terminal; tearing on a fast stream seen by eye; a live acceptForSession offer; the question deny message's round trip; a live failed tool call and failure card; the API-key, no-5h and live refusal cases (simulated); a NUL in a paste; ~\.claude.json's changes. Each with its reason below.
Verdict: DONE
```

## Not verified

What no check of this plan settles, and why. None of it is reported as PASS.

- **The overall look and feel.** The owner's call, as at plan 4.
- **Items 3 to 10 live during a real turn.** The plan spends no quota, so no
  CLI turn ran: the approval viewer (3), the entry cursor and view (4), the
  wheel and keys in A and C (5), a failed call's row (6), a subagent's model
  (7), a failure card copied by hand (9). The tests drive the real program's
  messages and draw the frames the terminal receives; item 1 showed the
  frames those tests draw look as drawn in WT.
- **A live `acceptForSession` offer** (item 8). None was seen; Codex 0.163.0's
  generated schema has no `availableDecisions`, so on that version `s` may
  never be offered. `d` still sends `decline` when a list omits it.
- **The question deny message's round trip** (item 10): how Claude Code
  reacts to it; CC-10 q3 stays open.
- **A real 256- or 16-colour terminal** (item 13). WT reports true colour;
  the other profiles were checked through Bubble Tea's own detection and
  renderer into a buffer, not seen on a screen.
- **Tearing on a fast stream** (item 14). Mode 2026 is measured as wrapped,
  not judged by eye; the first frame is drawn before WT's answer arrives,
  so it is not wrapped.
- **An API-key session, a plan without a 5-hour window, a live
  `get_context_usage` refusal** (item 15). Simulated, at the owner's request;
  each needs a key, another account or a CLI fault.
- **A NUL in a paste.** As at plan 4: the Windows clipboard ends text at the
  first NUL.
- **`~\.claude.json`.** As at plan 4: the hash-only check cannot classify
  Claude Code's own changes; item 2 started Codex only, and every file was
  identical.

Known limits recorded with their items: a failed Codex command's error line
is the first non-blank line of its aggregated output, which can be stdout
(item 6); "↓ N new" can cover the marked row while a view is held (item 4);
a string over several lines is still highlighted on its first line only
(item 12); Codex's `collabAgentToolCall` `model` is not used (item 7).

Outside the plan: Linux and macOS, a later view by the owner's request.

## Failures on record

Each stays as it was recorded in `plan.md`; none is open.

| Where | Failure | What followed |
| --- | --- | --- |
| Item 1, run 1 | Zone-title checks FAIL: a test defect, PowerShell's case-insensitive `$t` overwrote `$T` | Variable renamed; runs 3 and 4 every check PASS |
| Item 1, run 2 | Screenshots without colour: the agent's shell has `NO_COLOR=1`, passed on to the application | The harness removes `NO_COLOR`; every earlier WT run (L5 included) was colourless, their text checks stand |
| Item 2 | One-row check for "Zone D shows the 5h and weekly limits" FAIL: a test defect, the note wraps | The phrase is there on zone B's joined rows and in the screenshot; criterion PASS |
| Item 11 | Found while building: `wrap` looped for ever on a rune wider than the row | Fixed; `TestWrapUnchanged` and a direct case |
| Item 13, probe | In 16 colours the palette collapsed (text, titles, Opus, Haiku, code, keywords all bright blue; statuses and Sonnet bright red) | An explicit 16-colour palette, switched on the colour profile with a full redraw |
| Item 14, mutation | `TERM_PROGRAM=x` survived | An equivalent mutant: Bubble Tea asks under every terminal program but Apple's; `TERM_PROGRAM=Apple_Terminal` killed |

## Criteria amended

Each amended while building, before its check, and recorded in `plan.md`:
item 4, criterion 1 (every failure is expandable); item 5, criterion 5 (the
hint's wording, to fit zone A); item 7, criterion 5 (the inspection header
named); item 11, criterion 1 (an AI message's later rows in text colour);
item 12, criterion 5 (the Limits row replaced by the limit that remains);
item 14, criterion 2 (no `?2026h` or `?2026l`, since the query `?2026$p` is
always written).

## How the check ran

- **Main session.** The deterministic criteria (16, 17) were settled by
  commands run on 2026-10-17 on the working tree, cf58739 plus plan 5's
  uncommitted changes, the race detector included; each item's criteria rest
  on its record in `plan.md`, checked when it finished.
- **Live runs.** Items 1, 2 and 14 started the built executable in Windows
  Terminal; only item 2 started a CLI (Codex, no turn), with the
  configuration check before and after. Before items 2 and 14, no
  terminal-hosted Claude Code or Codex session was open; none was closed.
  Item 1's record does not say.
- **Housekeeping.** Item 13's before-change frames (`*.ansi`), left in
  `internal\app` by a temporary test, were moved to the session scratchpad;
  no test reads them.

## Live verification

Added on 2026-10-17, after the summary above, which stays as it was: the
owner asked to see the Not verified items live before the commit. One Windows
Terminal window per run, the application built from the working tree; the
owner typed the prompts and judged each screen, the main session took the
final screenshot and visible text, then quit with Ctrl+Q. Each live CLI
run had its own empty folder under `%LOCALAPPDATA%\Temp`, a configuration
snapshot before and `confcheck compare -cli-ran` after. Before the first, no
terminal-hosted Claude Code or Codex session was open; none was closed.

```text
Live verification - Plan 5, before the commit - 15 runs: 15 PASS, 0 FAIL (the owner's verdict on each)
PASS L1  Look and feel (t01; Claude Code 2.1.301, Haiku; Markdown, two parallel subagents): "It looks good, it worked fine"
PASS L2  Approval card full view (item 3, t02): the 79-line sample.go full screen, scrolled to its end; Esc back to the card still pending; a from the full view; "allowed by you 18:41:17"
PASS L3  A folded entry expanded in B (item 4, t03): the Bash output, lines 1 to 200, full screen; Esc to the cursor, Esc to the input
PASS L4  Scrolling in zones A and C (item 5, t04): 12 subagents, "0 ACTIVE · 12 DONE"; arrows, pages, Home and End, wheel, fold and zoom in both zones
PASS L5  A failed tool's row (item 6, t05): "✗ Read · File does not exist. …" and "✗ Bash · Exit code 2", each opened full screen
PASS L6  Each subagent's model in C (item 7, t06; Codex 0.163.0): /root/slice and /root/interface, gpt-6-sol each, and in the inspection
PASS L7  Codex approvals (item 8, t07): the cards offered a, d and c, no s; one.txt and two.txt written after a
PASS L8  A failure card copied (item 9, t08): the test's claude.exe ended by the main session; the card selected with the mouse; pasted "The session ended unexpectedly / the CLI exited unexpectedly / exit code 4294967295", no ✗, indentation or key hint
PASS L9  A question denied (item 10, t09): after d, Claude neither asked again nor chose; it waited for the next message; "green" in turn 2 gave the poem
PASS L10 256 colours (item 13, t10; mock, TERM=xterm-256color, WT_SESSION, COLORTERM and TERM_PROGRAM cleared): the main text colours are xterm-256 values (215,215,255; 95,95,135; 215,175,255; grey 68,68,68), none in the true-colour run (t09)
PASS L11 16 colours (item 13, t11; mock, TERM=xterm): Windows Terminal's 16 colours, text, titles, models, checks and badge distinct
PASS L12 Tearing (item 14, t12): a 45 s streamed answer, scrolled during the stream; no tearing seen
PASS S1  Claude API key, context refused (mock): "5h N/A (API key: no subscription window)", "7d N/A (…)", "context 3% of 1M (derived)", "errors 1", no "% resets"
PASS S2  Codex API key (mock): "5h N/A (API key: ChatGPT sign-in needed)", "7d N/A (…)", "credits N/A (…)"; plan and errors under "+5 more" at 120 columns, read in the details by the owner; no "% resets", no "plan plus"
PASS S3  Codex plan without a 5-hour window (mock): "5h N/A (plan has no 5-hour limit)", no 5h gauge; "7d … 1%  resets Thu 10:30"; "errors 0"; "plan plus" under "+2 more" at 120 columns
Configuration: t06, t07 (Codex) PASS, every file identical; t01, t02, t04, t05, t08, t09, t12 (Claude Code) NOT-VERIFIED, ~\.claude.json changed, not classified (t04: known_marketplaces.json too); t03 FAIL, see below
After every run: no acc-live, claude or codex process left outside VS Code; the project folders empty but for sample.go (t02), one.txt and two.txt (t07)
Settled: CC-10 q3, Claude Code's reaction to the question deny message (L9)
Still not verified: a live Codex acceptForSession offer (0.163.0 offered none); a selection from a conversation entry into a failure card (only the card was copied); a NUL in a paste; ~\.claude.json's changes
```

**t03's configuration FAIL**, kept as found:
`~\.claude\plugins\plugin-directory-cache-v2.json` changed at 18:46:05,
`known_marketplaces.json` at 18:46:01 and two files under `plugins\synced` at
18:44:20, while the test's Claude Code ran (about 18:43:30 to 18:46:54). They
are Claude Code's plugin marketplace cache. The application writes no file
there (its only writes are its own temporary drafts); a second Claude Code,
the main session's in VS Code, was running too, and hashes cannot tell which
of the two refreshed it. Plan 1 (criterion 9) and plan 4 (L5) recorded the
same refresh. The other Claude Code runs left both files identical, but for
`known_marketplaces.json` in t04.

**Harness notes.** The first launch of t01 failed: `Start-Process` joined
the arguments unquoted, and Windows Terminal took the title's second word
for the command; fixed by passing one quoted argument line. In t05 the
main session's first Ctrl+Q came while a Bash card was still pending; the
owner allowed it and finished the run before the capture.

## After the commit

Added on 2026-10-17, after commit d42531f, which the sections above
describe and which stay as they were. On the list of what was still open,
the owner decided: the simulated item 15 cases stand ("No need we already
simulated it"); no further work on the Codex `s` offer; Linux and macOS
stay a later view; and the five items below are done.

```text
Follow-up - Plan 5, after d42531f - 6 items: 6 PASS, 0 FAIL
PASS F1 A selection from a conversation entry into a failure card (t13; mock, the claude-subagent-approval recording plus a session end with code 3): the owner's copy held the message text and "The session ended unexpectedly / the CLI exited unexpectedly / exit code 3", no ✗ and no key hint
PASS F2 A NUL in a paste (t14; mock): the clipboard held "before", NUL, "after"; Ctrl+V gave "> before", 1 line, no ^@, the editor responsive. Windows' clipboard ends text at the first NUL (Set-Clipboard "before`0after" reads back 6 characters); a NUL that reaches the editor is kept (TestInputByteForByte)
PASS F3 The Codex thread limit (the owner chose "Call config/read"): config/read once per session after initialized, {"cwd": <project>, "includeLayers": false}; only config.agents.max_concurrent_threads_per_session (fallback max_threads) decoded; every event of the reply carries a fixed placeholder as Raw, so no byte of it is kept or shown (TestConfigReadKeepsOnlyTheLimit, with a marker); live t15c, Codex 0.163.0, no prompt sent: details "agents.limit  not set · config/read"; configuration PASS
PASS F4 Key-level diff of ~\.claude.json in confcheck (the owner chose "Key-level diff"; this test tool only): digests per key, sensitive names never hashed, unknown keys counted as config; a change in no compared key is NOT-VERIFIED (found in review: the first version passed it silently; test tightened)
PASS F5 Test 3 again (t03b; Claude Code 2.1.301 on Haiku; the owner ran the Expand and fold-back steps: "It worked fine"): configuration PASS; ~\.claude.json: cachedExperimentData, cachedGrowthBookFeatures, cachedGrowthBookFeaturesAt changed (state); oauthAccount, claudeCodeFirstTokenDate not compared (sensitive name); no project entry changed; plugin-directory-cache-v2.json and known_marketplaces.json identical
PASS F6 The V2 backend's own limit (the owner's request, after the source reading below): config/read also decodes features.multi_agent_v2.max_concurrent_threads_per_session; when its value minus one differs from agents' limit, or agents' is unset, zone D shows "limit N/A (set per backend; backend not reported)"; probe with -c overrides, no model call: the reply carries it as a table with the number, or as true; configuration PASS (TestConfigReadV2Limit, seven cases; TestTelemetryNA "codex limit per backend")
Full check: build; vet, also -tags live and -tags timing; gofmt empty; go test ./... 221 top-level tests (209 + 12) in 14 packages, all PASS
After every run: no acc-live, claude or codex process left outside VS Code; every project folder empty
Still not verified: a live Codex acceptForSession offer (0.163.0 offered none; not pursued, the owner's decision); Codex's built-in thread limit when unset (6 subagents on the V1 backend, 3 on V2; the backend is never reported: shown "N/A (not set; default not reported)")
```

t03's FAIL above stays on record. The rerun did not reproduce it: the
plugin cache was not refreshed during t03b. Before the runs, no
terminal-hosted Claude Code or Codex session was open. t15 and t15b were
harness attempts that captured the startup screen and the folded zone D;
both left the configuration identical and no process.

**The limit's range**, read in Codex's source at tag `rust-v0.163.0`
(openai/codex, 2026-10-17, the owner's question):
`agents.max_concurrent_threads_per_session` (alias `max_threads`) counts the
spawned agents open at once in a session; its minimum is 1 (0 stops Codex at
startup: "must be at least 1", `codex-rs/core/src/config/mod.rs:4003-4007`;
`#[schemars(range(min = 1))]`, `codex-rs/config/src/config_toml.rs:764-766`)
and it has no maximum (a `usize`). Unset, the V1 backend allows 6
(`DEFAULT_AGENT_MAX_THREADS`, `mod.rs:255`) and V2 allows 3: its default of 4
threads counts the main one (`mod.rs:256`, `mod.rs:1666-1680`). Set, both
backends allow exactly that number (V2 adds the main thread and subtracts it,
`mod.rs:2815-2823`). `features.multi_agent_v2.max_concurrent_threads_per_session`,
when set, overrides it on V2 and counts the main thread; the application reads it
(F6): when the two backends' limits differ, zone D says so and shows N/A,
since the active backend is not reported. `agents.max_depth` (default 1) applies to V1 only.
