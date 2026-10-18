# Plan 14 evaluation (2026-10-18)

The final check of plan 14, nothing left open: its four items of
[`plan.md`, Plan 14](../plan.md#plan-14---nothing-left-open) against the
yes/no criteria written in the progress list before each item's work
started, and the plan's rules (the owner's Claude quota, Haiku as a
per-process override, the dashboard only in Windows Terminal maximised, each
run in a temporary project outside the repository, the configuration check
before and after, no CLI configuration file or system setting changed, no new
dependency). The evidence is `plan.md`'s record of each item, the probe
recording and the live-run captures. The block below is the summary reported
to the owner. The results are not to be rewritten.

```text
Evaluation - Plan 14, nothing left open - 18 criteria: 18 PASS
PASS 1.1 P14a (Haiku, Claude Code 2.1.302): a can_use_tool with tool_name "AskUserQuestion" and the documented input {questions:[{question, header, options:[{label, description}], multiSelect}]}, plus a new requires_user_interaction:true; recorded as p14a-claude-question-answer.jsonl (68 lines)
PASS 1.2 The allow with updatedInput {questions, answers} ("Rouge café ✓"; "Monday, Tuesday") gave a tool result that is not an error and carries both answers: "Your questions have been answered: \"Which colour?\"=\"Rouge café ✓\", \"Which days?\"=\"Monday, Tuesday\". …"
PASS 1.3 The model's next text: Colour: "Rouge café ✓"; Days: "Monday, Tuesday", the labels byte for byte
PASS 1.4 P14a configuration check before and after (-cli-ran): PASS, configuration files identical, ~/.claude.json changed only in cache-state keys; the folder deleted, no CLI process left
PASS 1.5 The card answers: 1-4 choose (toggle on a multiSelect question), ← → select the question, a sends once every question has a choice (otherwise a toast), d and c unchanged; the response is allow with updatedInput {questions, answers}; TestQuestionAnswerKeys, TestAnswerQuestion, TestQuestionCard, TestQuestionAnswerRules
PASS 1.6 Live run (attempt 2, Haiku, Windows Terminal maximised): keys 2, 1, 3, a; the card showed ● Blue, [x] Monday, [x] Friday, then "✓ orchestrator · AskUserQuestion answered by you 12:22:17", "Which colour? → Blue", "Which days? → Monday, Friday"; the wrapper's log holds the allow with answers {"Which colour?":"Blue","Which days?":"Monday, Friday"}; the model replied "Colour: Blue; Days: Monday, Friday"; configuration check PASS; the folder deleted, no process left (Ctrl+Q 3.1 s)
PASS 2.1 doc/research/claude-code.md section 10: every question settled with evidence or closed with the reason the dashboard does not depend on it; no question left open
PASS 2.2 doc/research/codex-cli.md section 10: q1, q2, q4, q7, q8 settled; q3, q5, q6, q9, q10 closed with the reason; no question left open
PASS 2.3 architecture.md, reliability.md (every "Not verified" row), events-and-state.md and ui-design.md say how each question ended
PASS 2.4 CC-10 q13: TestEchoUnicode sends "café ✓ — 日本\r\nline 2\n" through the fake CLI's pipes and reads it back byte for byte; P14a carried "Rouge café ✓" both ways live
PASS 2.5 CC-10 q10: get_context_usage answered 914 ms after it was sent, with no assistant message, result or rate_limit_event after it (P14a)
PASS 3.1 The owner's closures of 2026-10-18 recorded: Claude Code's Auto fallback (claude-code.md section 11, not provoked); the Codex subagent model or effort and its default limit (codex-cli.md section 10); the checks by eye and the excluded list (ui-design.md, reliability.md)
PASS 4.1 go build ./... and go vet ./... (also -tags live, -tags timing) exit 0; gofmt -l . prints nothing
PASS 4.2 go test ./... 267 top-level tests PASS in 14 packages, 0 FAIL (baseline 8888f59 261, +6: the five question tests and TestCompactBoundary)
PASS 4.3 The timing step run alone (an input and a render path changed): TestKeystrokeToFrame p95 1.07 ms, TestDraftKeystrokes p95 2.01 ms, TestKeystrokeCost, TestLongSession delta-to-frame p95 3.05 ms, all PASS (target p95 <= 16 ms)
PASS 4.4 The 40 changed and new files (this one included) carry no CR and no BOM
PASS 4.5 A search of the living documents finds no open item without its outcome: the remaining hits are section headings, pointers to reliability.md's outcomes, or historical files (proposal.md, the plan evaluations, plan 1's reference studies)
PASS 4.6 The summary written here and in plan.md's progress list
Not verified: nothing within plan 14's scope. The owner closed, without a check: Claude Code's Auto fallback to a prompt (it follows Claude Code's own mechanism), the Codex subagent model or effort at launch and its default limit, the checks by eye and the excluded list.
Verdict: DONE
```

## The question card, from probe to live run

- **The answer format** (P14a). Claude Code asks with a `can_use_tool` for
  `AskUserQuestion` and takes the answer as an allow whose `updatedInput`
  keeps `questions` unchanged and adds `answers`, keyed by the question
  text; a multiSelect answer is its labels joined by ", ". The details are
  in `doc/research/claude-code.md`, section 10, item 3.
- **The card.** Each question lists its options with their keys; 1 to 4
  choose, ← and → move between questions, `a` sends. Once answered, the
  card reads "answered by you HH:MM:SS" and one "question → answer" line
  per question (`doc/ui-design.md`, Questions). A deny still works as plan
  5 found: the model asks in text instead.
- **Live attempt 1 not counted.** The run script waited for a card header
  "! APPROVAL ", but a question card's header reads "? QUESTION ", so the
  script never focused it and timed out after 300 s. The card was answered
  later (Red, Monday at 12:19:53) by the owner by hand, as the owner
  confirmed. The round trip worked the same way: the wrapper's log
  holds the allow with those answers, and the model replied "Colour: Red;
  Days: Monday". The configuration check after it passed; the folder was
  deleted and no process was left. Attempt 2, with the script fixed, is the
  evidence for criterion 1.6.

## What the owner closed

- **Claude Code's Auto falling back to a prompt** after repeated classifier
  blocks: not provoked. The dashboard follows Claude Code's own mechanism; a
  fallback arrives as a `can_use_tool`, and every `can_use_tool` is already
  a card.
- **A Codex subagent launch naming its model or effort, and Codex's default
  subagent limit:** closed.
- **The checks by eye** (look and feel, 256 and 16 colours, flicker,
  physical keys, a NUL in a paste) **and the excluded list:** closed.

## Items 5 to 7 (2026-10-18)

The owner added three items after the block above ("I agree with all"):
the documents hold the latest state (5), a CLI update is never silent (6),
and the full check for both (7), against the criteria of
[`plan.md`, Plan 14](../plan.md#plan-14---nothing-left-open). The block
below is their summary as reported to the owner; the block above is
unchanged. The results are not to be rewritten.

```text
Evaluation - Plan 14, items 5 to 7 - 15 criteria: 15 PASS
PASS 5.1 plan.md holds the active plan with its progress, "The owner's standing decisions" and one entry per closed plan under its exact "## Plan N - ..." heading (approval, counts, evaluation file, commits); 10 plan.md# links in doc/, AGENTS.md, CLAUDE.md and plan.md, 0 unresolved; 332 lines before this record (under 400)
PASS 5.2 The standing decisions of the closed plans' logs kept: in plan.md (live runs, hash-only configuration check, measured not assumed, the integration, drafts), in doc/evaluation.md (The race detector: WinLibs GCC 16.2.0 UCRT r2, its SHA-256, no PATH change), and where they already lived (ASM-17's renderer in architecture.md, plan 11's network item in plan11-evaluation.md)
PASS 5.3 doc/proposal.md deleted; no living file links to it (the only mentions are plan.md's history entries naming the deletion and commit d5b910e); architecture.md cites d5b910e where it cited the proposal
PASS 5.4 Item 5 changed neither doc/brief.md, the evaluation files, plan 1's reference studies nor doc/research/summary.md; doc/research/gemini-cli.md's only change (the closed open questions) was made under item 3, before items 5 and 6 were approved
PASS 5.5 AGENTS.md's status line and layout say what plan.md, the design documents and the evaluation files now hold; working rule "Documents hold the latest state; git holds the history."
PASS 6.1 claude.TestedVersion "2.1.302" and codex.TestedVersion "0.163.0", reported by Detect as Detection.Tested; TestTestedVersion in each adapter reads the README's "Last probed" bullet and fails if it differs (it failed once on the wrapped line before the test read the whole bullet)
PASS 6.2 The project bar reads "Claude Code 2.1.303 newer than tested · ..." with the mark in the attention colour when the CLI is newer; nothing when equal, older, unparsable, empty, without a detection entry, or in a mock session (TestNewerThanTested, 12 cases; TestProjectBarNewerThanTested, 5); TestGoldenFrames passes without -update, no golden frame changed
PASS 6.3 A Claude Code tool result with no permission_decision for its id in tool_result_meta (absent, without the decision, another id's, another shape) is one Unrecognised event, Type "user/tool_result_meta", whose reason names permission_decision; the classifier's verdicts still map (TestToolResultWithoutPermissionDecision, 6 cases); the 9 Claude recordings, 17 tool results, replay with none (TestRecordingsHavePermissionDecision)
PASS 6.4 doc/evaluation.md, "After a CLI update": the session-start check and the re-check (TestLiveGate, the probe drivers, new recordings in the replay tests with no unrecognised event, the "Last probed" line and TestedVersion updated together), run only once the owner approves it
PASS 6.5 AGENTS.md working rule "Check the CLI versions at session start"; the check of 2026-10-18: Claude Code 2.1.302 and Codex CLI 0.163.0, both equal to the last probed, no update
PASS 7.1 go build ./... and go vet ./... (also -tags live, -tags timing) exit 0; gofmt -l . prints nothing
PASS 7.2 go test ./... 273 top-level tests PASS in 14 packages, 0 FAIL (baseline 267, +6: TestNewerThanTested, TestProjectBarNewerThanTested, TestTestedVersion twice, TestToolResultWithoutPermissionDecision, TestRecordingsHavePermissionDecision)
PASS 7.3 The timing step run alone (the project bar is a render path): TestKeystrokeToFrame p95 1.05 ms, TestDraftKeystrokes p95 2.03 ms, TestKeystrokeCost, TestLongSession delta-to-frame p95 3.29 ms, all PASS (target p95 <= 16 ms)
PASS 7.4 The 49 changed and new files (this one included) carry no CR and no BOM
PASS 7.5 This second block written, the first one unchanged; the results in plan.md's progress list
Not verified: the "newer than tested" mark against a real newer CLI (none exists today; the next update shows it, and the re-check runs only on the owner's approval).
Verdict: DONE
```

- **The recorded key.** `tool_result_meta` entries carry `"id"`, not
  `"tool_use_id"`: the recordings show it. Plan 13's
  `TestVerdictsOnlyForTheClassifier` used `"tool_use_id"`, so its verdicts
  had an empty tool id; it now uses the recorded key, with the same
  expected counts.
