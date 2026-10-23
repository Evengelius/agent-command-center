# Plan 34 evaluation (2026-10-23)

The check of plan 34, the model selector, its effort and Ultracode (brief
amendment 34: the owner's request and its image,
`references/11-model-selector.webp`, "the final approved visual
specification"). It covers the 21 criteria written in
[`plan.md`, Plan 34](../plan.md) before the code, on the code with the
independent review's fixes (below). Nothing is committed: the owner
reviews the implementation and the screenshots first.

How it was checked:
- Deterministic tests, none starting a CLI: `internal/provider/claude`
  (TestModelsOfP34c, TestSettingsOfP34c, TestSettingsLineKept,
  TestEffortLock, TestChangeAnswered; TestModelSettingsWire, the real
  adapter and process plumbing over a fake Claude Code answering as P34c
  and P34c3 did), `internal/provider/codex` (TestModelsOfP34x,
  TestTurnStartRefused, TestModelListGuarded; TestModelChangeWire, the
  real adapter over a fake app-server serving P34x's pages),
  `internal/state` (TestChooseClaudeModel, TestRefusedAndKept,
  TestEffortChoices, TestUltracodeChoices, TestCodexNextTurn,
  TestModelsNewProcess, TestEffortLabel, TestTurnRefused,
  TestChoiceDuringTurn, TestChangeDroppedAtEnd), `internal/render`
  (TestSelectorGeometry, TestSelectorStates, TestEffortList,
  TestSelectorMarks, TestSelectorTail, TestModelPill, TestEditorRowGap,
  TestKeyOverlayRows), `internal/app` (TestSelectorKeys,
  TestSelectorMouse, TestSelectorScroll, TestSelectorKeepsDraft,
  TestSelectorNoRoom, TestSelectorSendKey, TestSelectorPillHidden,
  TestSelectorWithoutModels), and the golden frames models-claude,
  models-claude-more, models-claude-effort, models-codex and
  models-codex-more at 79x24, 80x24, 120x40 and 200x60, drawn from the
  mock files made from P34c and P34x. Each test added for a review
  finding was run once against the code without its fix, and failed.
- Windows Terminal 1.23.20211.0 in mock mode: the executable built from
  the final tree (`scratchpad\term\acc-p34.exe`, SHA-256 `6f700c8d…`),
  driven by `scratchpad\term\p34demo.ps1` with a temporary home and app
  data and no CLI on the PATH, maximised (188x53 after two Ctrl+minus, so
  the whole popup shows) and at 120x40.
- Live, with no message sent, so no model call:
  `scratchpad\term\p34live.ps1`, the same executable on Claude Code
  2.1.303 and Codex 0.163.0 in a temporary project outside the
  repository, maximised, the configuration check before and after, the
  31 variables this desktop session puts in its children's environment
  dropped first (with them, Claude Code's descriptions differ from the
  owner's terminal's). No terminal-hosted Claude Code or Codex session
  was open, so none was closed.
- Build, vet, gofmt, the whole suite, the race detector over every
  package, the timing step alone; an independent review of the diff (the
  `verifier` agent), twice.

```text
Evaluation - Plan 34, the model selector, its effort and Ultracode - 21 criteria: 20 PASS, 1 FAIL
PASS 34.1 TestModelsOfP34c: P34c's initialize answer lists 13 models once, from initialize, in the CLI's order (default, opus, fable, sonnet, haiku, then claude-haiku-4-5-20251001, claude-sonnet-5, claude-opus-5, claude-fable-5, claude-opus-4-8, claude-opus-4-7, claude-opus-4-6, claude-sonnet-4-6), with its names and descriptions ("Default (recommended)" / "Opus 5.5 · Best for everyday, complex tasks", "Fable 5.1" / "For your toughest challenges", "Haiku 5.5" / "Fastest for quick answers"); the golden frames models-claude: "CLAUDE MODELS", those five with P34c's descriptions, then "▶ More models"; models-claude-more and TestSelectorScroll: the eight others in that order, from Haiku 4.5 to Sonnet 4.6; no "GPT" in Claude Code's selector
PASS 34.2 TestModelChangeWire: the fake app-server serving P34x's pages of one model receives 7 model/list requests, {} then {"cursor":"1"} to {"cursor":"6"}, and 7 models are listed; TestModelsOfP34x: gpt-6.1-sol (default), gpt-6-astra, gpt-6-sol, gpt-6-luna, gpt-5.6-sol, gpt-5.6-terra, gpt-5.6-luna, the same 7 from the list asked with hidden models; the golden frames models-codex: "CODEX MODELS", GPT-6.1-Sol (default), GPT-6-Astra, GPT-6-Sol, GPT-6-Luna, GPT-5.6-Sol, More models; models-codex-more: GPT-5.6-Terra and GPT-5.6-Luna; TestSelectorStates: no Ultracode row and no "CLAUDE" in Codex's, its bottom edge under Effort
PASS 34.3 TestSettingsOfP34c: the first read is {claude-fable-5-1, fable, max, Ultracode on and available, get_settings, undocumented}; TestModelPill " Fable 5.1 Max · Ultracode "; TestModelsOfP34x: the thread starts on {gpt-6-sol, high, thread/start}; the golden frames: "▶ Fable 5.1 … ✓" and "▶ GPT-6-Sol … ✓" on the band, the pills "Fable 5.1 Max · Ultracode" and "GPT-6-Sol High"
PASS 34.4 TestSelectorScroll: Opus 4.6 in force opens the selector with More models expanded, "▶ Opus 4.6" on screen with its "✓"
PASS 34.5 TestSelectorScroll on P34c's 13 models at 120x40: collapsed 67x25, expanded by Space the same 67x25, "▼ More models" on the list's first row (row 15 of 41); PgUp to row 0, PgDn to 17, to 24 (the end, Sonnet 4.6 shown), PgDn there stays 24; the wheel up over the list 21; Home 0; ↓ thirteen times to Sonnet 4.6, the list at row 23
PASS 34.6 TestModelsOfP34c: Fable 5.1 low, medium, high, xhigh, max; Opus 4.6 low, medium, high, max; Haiku 4.5 none; TestModelsOfP34x: GPT-6-Luna low to max, GPT-6-Sol and GPT-6.1-Sol with ultra; TestEffortList on screen: GPT-6-Sol Low, Medium (default), High, XHigh, Max, Ultra; GPT-6-Luna Low, Medium (default), High, XHigh, Max; Fable 5.1 Low to Max, no "(default)"; Opus 4.6 Low, Medium, High, Max; TestSelectorStates: Haiku 4.5's box "N/A" at column 14, no chevron; TestEffortChoices: XHigh not asked on Opus 4.6, nothing on Haiku 4.5
PASS 34.7 TestModelSettingsWire, the control requests written: initialize, get_settings, apply_flag_settings {"effortLevel":"low","ultracode":true}, get_settings, apply_flag_settings {"ultracode":false}, get_settings, set_model {"model":"claude-opus-4-6"}, get_settings, set_model {"model":"claude-nope"}, get_settings; TestChooseClaudeModel: a model asked as {Choice: claude-opus-4-6} alone
PASS 34.8 TestEffortLock: CLAUDE_CODE_EFFORT_LEVEL in the read's env, or the CLI's environment, holds the effort; TestSelectorStates: "Max" at column 14, no chevron; TestEffortChoices: no command, zone B "the effort is set by CLAUDE_CODE_EFFORT_LEVEL (Max): the dashboard does not change it", the details view "Max, held by CLAUDE_CODE_EFFORT_LEVEL"; live L34.5 on the owner's settings
PASS 34.9 TestModelChangeWire: no line written by the change; the next message's turn/start carries "model":"gpt-6.1-sol" and "effort":"high"; its answer is followed by thread/read {threadId}, whose values reach the state and zone B's speaker on the same thread; TestCodexNextTurn: GPT-6-Luna chosen with Ultra in force takes its default, "→ GPT-6-Luna Medium (next turn)"; GPT-6.1-Sol "→ GPT-6.1-Sol High (next turn)"; TestSelectorStates: the pill "GPT-6-Sol High" with "→ GPT-6.1-Sol High (next turn)" beside it
PASS 34.10 TestModelSettingsWire: Ultracode off sent alone, the read keeping Low; TestUltracodeChoices: no effort and no model in the request; live L34.7: Ultracode off, then on, the pill "Sonnet 5.5 Max" then "Sonnet 5.5 Max · Ultracode"
PASS 34.11 TestSelectorStates: with ultracodeAvailable false the row is grey, its toggle "▐●   ▌" (off); TestUltracodeChoices: no command, zone B "Claude Code reports Ultracode unavailable with Opus 4.6", or "Ultracode's state is not known"; TestModelSettingsWire: the read after Opus 4.6 reports it unavailable; TestModelPill: " · Ultracode" only while applied.ultracode is true
PASS 34.12 TestSettingsOfP34c: P34c's refusal "set_model refused by Claude Code: Model 'claude-not-a-model-34' not found"; TestModelSettingsWire: "Model 'claude-nope' not found" in zone B, the model in force and zone B's speaker unchanged; TestRefusedAndKept: an accepted level the read does not show leaves Max and says "Claude Code kept the effort Max (asked: Low)"
PASS 34.13 TestChooseClaudeModel: the model in force, zone A's and zone B's model unchanged until the read; TestSelectorMouse: the pill "Fable 5.1 High · Ultracode" after the click on Sonnet 5.5, "Sonnet 5.5 High · Ultracode" only after the answer; TestCodexNextTurn and TestModelChangeWire: Codex's values only from thread/read; TestEffortList and TestSelectorMarks: the box and the "✓" stay on what is in force, a Codex level asked for a grey "…"
PASS 34.14 TestSelectorKeys: Alt+P opens on the model in force and closes; ↑↓ move; Enter chooses Sonnet 5.5 with a draft written, nothing sent, the draft kept; Esc closes the effort list, then the selector; TestSelectorScroll: Space, PgUp, PgDn, Home; TestSelectorSendKey: Ctrl+Enter configured as the send key chooses, sends nothing; TestSelectorNoRoom: a paste closes it and goes to the draft
PASS 34.15 TestSelectorMouse: a click on the pill opens; the hover moves the cursor to Sonnet 5.5; a click chooses it; More models expands; the effort box opens its list, a second click closes it, XHigh is chosen; Ultracode turned off; a click on the draft closes the selector, the focus on B; TestSelectorScroll: the wheel
PASS 34.16 TestSelectorKeepsDraft: two lines of text, two images from the dialog, the cursor at the line's start, the focus on zone A; the selector opened by Alt+P and by the pill, a model chosen, the effort list opened and closed, More models expanded, closed by Esc (the focus still A) and by a click on zone B's title: the text, the cursor, the two images in order unchanged, nothing sent
PASS 34.17 TestModelSettingsWire: one spawn through four changes and a refusal; TestModelChangeWire: the same thread id before and after the change
FAIL 34.18 Build, vet and gofmt clean; the suite, 15 packages ok, 623 top-level tests (588 on the last commit); the race detector (WinLibs GCC, CGO_ENABLED=1) over every package on the final code, 0 data races. The golden frames: 67 changed and 20 new (the selector's), each reviewed; but not only by the pill's values, as the criterion says: the editor row in 54 of them (the pill's real values, and at 120x40 a turn's status now beside the shorter pill), the thinking verb in 15 (the line of models or settings added to 10 mock files shifts the sequence numbers that pick the verb), the Explorer's and the quick open's listing in 6 (the two new mock files). No regression in them; an exception for the owner to accept
PASS 34.19 The timing step alone, 8 of 8 within target: TestSelectorToFrame Alt+P p95 1.00 ms (120x40) and 1.10 ms (208x59), ↑↓ in the expanded list 1.05 ms and 1.31 ms; keystroke-to-frame 1.10 ms (10,000-line draft) and 1.11 ms (draft window), 2.37 ms in a 50,000-line draft with 8 saves; TestKeystrokeCost; the long session's deltas 6.05 ms (the last commit's tree and this one timed in turn: 4.59 and 5.52 ms, 5.00 and 4.90 ms); paste-to-frame p50 2.11 ms; the stream 4.00 ms; the thinking card 4.70 ms (targets 16 ms, 100 ms for the paste)
PASS 34.20 Windows Terminal, the final build, maximised and at 120x40: Claude Code's popup 67x25 (P34c's longest description) and Codex's 64x23, collapsed, with More models expanded and scrolled by the wheel, the effort list, Codex's change for the next turn: the lavender border with rounded corners; "CLAUDE MODELS" or "CODEX MODELS" and "Current session"; two rows a model, one blank row between; the band with the pink "▶" and the "✓"; the separators around "More models" and its "›"; the effort box; the toggle with "Enables dynamic workflows"; the bottom edge right under the Ultracode row (Codex's under Effort); no footer; the "▾" above the pill's first cell; the departures reference-design.md lists (one type size, 2.75 lines rounded to three rows, the tail as "▾", square band corners, the toggle in half blocks)
PASS 34.21 Live, no message, the final build: Claude Code 2.1.303, 11 of 11: the pill "Fable 5.1 Max · Ultracode"; the five first models, More models, "Enables dynamic workflows", nothing of Codex's; More models expanded, then paged through: all 13 of P34c's names on screen, each on a row of its own; Max held by the owner's variable; Sonnet 5.5 in force after the read; Ultracode off then on, Max kept; Fable 5.1 back; no message sent; configuration PASS before and after. Codex 0.163.0, 9 of 9: the pill "GPT-6-Sol High"; the five first models with "GPT-6.1-Sol (default)", More models, no Ultracode row, nothing of Claude Code's; all 7 of P34x's names; "→ GPT-6.1-Sol High (next turn)"; choosing GPT-6-Sol again cancels it; no message sent; configuration PASS
Not verified: a Codex turn that carries a change, live (it needs a message, so a model call: the wire is checked against the fake app-server replaying P34x's shapes); Ultracode unavailable in a live dashboard (P34c3 recorded it with Opus 4.6, replayed by the tests); what a Claude Code change does to a running turn (not probed: the dashboard waits for the turn's end); get_settings is undocumented, so a CLI update may drop it (the dashboard then shows why, not values)
Verdict: DONE, waiting for the owner's review; 34.18 is an exception for the owner to accept or refuse
```

**The independent review.** The `verifier` agent read the diff against
the criteria and ran its own tests in copies of the tree. Its first pass
found four defects: an open selector the frame could not draw (a tall
draft at 80x24, a turn's status in the pill's place, a full view) still
took keys and could change the model unseen; Alt+P moved the focus to
zone B; a `turn/start` Codex refused left the turn "Submitted", only
Ctrl+Q left (an older gap a carried change makes reachable); a status
text that just fitted touched "↯ Auto". And six doubts: `model/list`
followed a cursor that came back without end, and a refused page left
"has not listed its models yet" for good; Codex's effort box showed the
level asked for; the "▾" left the pill when the box was pushed left; a
configured send key closed the popup and sent the draft; Claude Code
changes during a turn (not probed); a change waiting when the session
ended stayed on screen. All were fixed (plan.md, Progress), each with a
test that fails without the fix. Its second pass confirmed the ten fixes
and found three small points, fixed too: a refused message's entry in
zone B with no word under it; the same notice repeated by a second
click; and, while a turn's status has the editor row alone (plan 29's
rule: at 120x40 with the longer pills), the selector does not open, the
toast now saying it needs the pill on screen. Plan 29's row is kept, so
during such a turn a Codex change waits for the turn's end; a wider
window (the acc launcher's 208x59) shows the pill throughout.

**The owner's choice after the report (2026-10-23).** Told that their
Claude Code settings held the effort at Max, so the dashboard could not
change it, the owner chose to remove the variable: "the goal is to
control it through my tool". The blocks above stay as reported.

```text
Follow-up - the effort freed for the dashboard - 1 check: 1 PASS
PASS F1 At the owner's request, "CLAUDE_CODE_EFFORT_LEVEL": "max" removed from the env block of ~/.claude/settings.json, the file otherwise identical (parsed and compared with the original before it was written; 945 to 906 bytes). Live, no message, the final build (out-p34-live-claude-free), 11 of 11: the session starts at Fable 5.1 XHigh, the level the same file's modelSettings give it; the effort box has its chevron and its list opens, no notice; Low, then Max, each in force once the read confirms it, Ultracode kept on; Sonnet 5.5 at Max; Ultracode off then on, Max kept; Fable 5.1 back at Max; all 13 models on screen; no message sent; configuration PASS before and after
Beyond the dashboard: every Claude Code session started from now on (the desktop app, VS Code, terminals) begins at those per-model levels (Fable 5.1 XHigh, Opus 5.5 High, the others their default) instead of Max; sessions already running keep Max until restarted. To restore it: put "CLAUDE_CODE_EFFORT_LEVEL": "max" back in that env block.
```

The owner then used it in a session of their own, with messages, so
model calls; read from Claude Code's transcript of that session (its
model and effort fields only):

```text
Follow-up - the owner's session with the effort freed - 1 check: 1 PASS
PASS O1 The effort chosen in the dashboard is the one Claude Code sends: Opus 5.5 at Max, then Medium chosen in the selector between two messages; Claude Code's transcript records output_config.effort "max" on every API request of the turns before the change (21:25:18 to 21:27:17 UTC) and "medium" on every request from the next message on (21:30:15 UTC onward), each response's effort the same; the pill read "Opus 5.5 Medium". The model's first answer after the change could not name its level (it reported a number); asked again, it read it from that transcript
```

The owner's Codex session the same evening, read from Codex's own record
of it (the rollout file; its model and effort fields only):

```text
Follow-up - the owner's Codex session - 1 check: 1 PASS
PASS O2 What the pill shows is what Codex runs: "GPT-6-Sol High" in the dashboard; Codex's rollout records, on each of the session's four turns (21:34:44 to 21:35:49 UTC), turn_context model "gpt-6-sol" and effort "high", and thread_settings_applied the same. The model said it could not see them: Codex sends the model and the effort as the request's settings, not as text in the conversation, so the dashboard is the place to read them
```

**Amendment 35 (2026-10-24): the agent told its settings.** Having seen
that neither CLI's model knows the model and effort it runs, the owner
asked for an opt-in line telling it (plan.md, design D11, criteria 22 to
26, written before the code). The blocks above stay as reported.

```text
Evaluation - Plan 34, amendment 35, the agent told its settings - 5 criteria: 5 PASS, 0 FAIL
PASS 34.22 TestLoadAnnounce: {"models": {"announce": true}} on; false, absent and an empty section off; "yes" gives "config.json: models.announce must be true or false, not a string; the agent is not told its model", 1 "… not a number …", another setting "config.json: unknown setting models.tell; …", a models that is not an object "config.json: models must be an object such as {"announce": true}; …"; a broken file off with no second problem (LoadKeys names it); the keys and Codex's proxy read as before beside a wrong models section. TestAnnounceStartup: on, the grey line "the agent is told its model and effort after each change (config.json)"; "yes", the red problem line; off or absent, neither
PASS 34.23 TestAnnounceClaude: the first message "first" as typed; after Sonnet 5.5 confirmed by the read, the written text exactly "[Agent Command Center] From this message on: model Sonnet 5.5 (claude-sonnet-5-5), effort Max, Ultracode on." then a newline and "second"; the next "third" as typed; Opus 4.6 then Sonnet 5.5 again before a message, "fourth" as typed; Ultracode off, "… Ultracode off." before "fifth". TestAnnounceOff: off, "second" as typed after the change. TestAnnounceWire, through the application: the scripted Claude Code session receives "first", the line and "second", then "third". TestAnnounceSlash: "/compact" as typed, the line on the next message. TestAnnounceImagesOnly: a message of images alone is the line alone. TestAnnounceResume: after a crash and a resume, the restarted CLI's Fable 5.1 XHigh told on the first message
PASS 34.24 TestAnnounceCodex: GPT-6.1-Sol waiting for the next message with High, the written text "[Agent Command Center] From this message on: model GPT-6.1-Sol (gpt-6.1-sol), effort High." then a newline and the text, no Ultracode part; the Codex adapter writes the message's text as the turn/start input as before, with model and effort (TestModelChangeWire)
PASS 34.25 TestAnnounceGivenBack: a failed write and a refused turn/start give back "second" as typed, and each next send carries the line again; given back, then a slash command accepted, the line is still due on the next message. TestAnnounceWire: the message waiting for its acceptance, which the stored draft keeps, is "second". TestAnnounceImagesOnly: no line in the draft
PASS 34.26 TestAnnounceLineGrey: zone B draws the line in the grey style and "Can you refactor the parser?" in the message's bright style (with the grey style removed, the test fails); TestAnnounceClaude: zone A's turn name "second"; TestAnnounceResume: a replayed prompt carrying a line names its turn "replayed"
Checks: build, vet and gofmt clean; the suite, 15 packages ok, 634 top-level tests (623 before the amendment), no golden frame changed (off by default); the race detector on internal/state, internal/input, internal/render and internal/app, 0 data races
Not verified: a model reading the line, which needs a message and so a model call (the owner's to approve)
Verdict: DONE, waiting for the owner's review
```

Withdrawn the same day by the owner: "I don't like it actually, remove
this last implementation". The implementation was removed (its setting,
line, tests and design text); the block above stays as it was reported.

**The runs before the review's fixes.** The Windows Terminal screenshots
and the live checks were made first with the build before the fixes
(`out-p34-live-claude-build1`, `out-p34-live-codex-build1`: 10 of 10 and
8 of 8, configuration PASS), then with the final build (`-build2`: the
same), then once more with the final build after a step was added to the
live harness, L34.4b, which pages through the expanded list and finds
every model's name on screen: the results above.

**The commit and the push (2026-10-24), after the owner's approval.** The
owner approved plan 34 on 2026-10-24, the exception 34.18 accepted; commit
47b50ce (167 files: 125 modified, 42 added) and the approval commit were
pushed to `origin`; `origin/main` equals `main`, the working tree is
clean, and the author is `Evengelius <molimoabel@gmail.com>`, with no
attribution line: PASS. The blocks above stay as they were reported.
