# Plan 6 evaluation (2026-10-17)

The final check of plan 6, plan 5's four known limits: each item of
[`plan.md`, Plan 6](../plan.md#plan-6---plan-5s-known-limits) against the
yes/no criteria written in the progress list before its work started, and
the plan's rules (Windows 11 and Windows Terminal only, no live CLI turn, no
new dependency). The evidence is `plan.md`'s record of each item; the block
below is the summary reported to the owner. The results are not to be
rewritten.

```text
Evaluation - Plan 6, plan 5's known limits - 6 criteria: 6 PASS, 0 FAIL
PASS 1  A failed Codex command's line in B (item 1): 5 of 5; TestCodexCommandExit, TestFailedCommand, TestToolFailureRow unchanged; "✗ commandExecution · exit N · <last non-blank line>", "exit N" alone on blank output; the exit code from the event, never parsed
PASS 2  "↓ N new" beside the marked row (item 2): 5 of 5; TestNewBadgeRow through the real program: the row's start drawn, a drag copies its text without the badge, the marked row never on the badge row
PASS 3  A string over several lines (item 3): 5 of 5; TestMultiLineString, TestBlockComment, TestMarkdownSameText, TestInspectMarkdown; TestStreamToFrame p95 4.00 to 4.21 ms (HEAD 3.04 to 4.75 ms)
PASS 4  Codex's collabAgentToolCall model (item 4): 4 of 4; TestSpawnRequestModel (fixture from the 0.163.0 generated schema), TestRequestedModel, TestAgentModel unchanged; the agent's own model always wins
PASS 5  go build ./... and go vet ./... (also -tags live, -tags timing) exit 0; gofmt -l . prints nothing; go test ./... 226 top-level tests PASS in 14 packages (221 at 6813c0e, +5), 0 FAIL; golden frames unchanged; the changed files have no CR and no BOM
PASS 6  Rules: no live CLI turn, no quota; go.mod and go.sum unchanged since 6813c0e, 4 direct modules; Windows 11 only; none of amendment 4's excluded features
Mutation checks: 2 mutants killed in item 4 (the spawn event's parent, the reducer's guard), 0 survived; none run for items 1 to 3.
Not verified: items 1 to 4 live during a real turn; a spawn request carrying model (no recording has one); the error line itself for a Codex command. Each with its reason below.
Verdict: DONE
```

## Not verified

What no check of this plan settles, and why. None of it is reported as PASS.

- **Items 1 to 4 live during a real turn.** The plan spends no quota, so no
  CLI turn ran. The tests drive the real program's messages and draw the
  frames the terminal receives.
- **A spawn request carrying `model`** (item 4). No Codex recording has one;
  the fixture is built from the field as Codex 0.163.0's generated schema
  documents it ("Model requested for the spawned agent, when applicable").
  `reasoningEffort`, documented beside it, is not used.
- **The error line of a failed Codex command** (item 1). Codex sends one
  output, "aggregated from stdout and stderr", and no notification of an
  agent's command separates the streams. B shows the exit code and the last
  non-blank line, where a failing command most often ends; the viewer
  (Enter) shows the whole output. A limit of the interface, not of the code.

## Failures on record

None of the plan's criteria failed during the work. One defect was found in
review before the full check, and is recorded in `plan.md`, item 4:

| Where | Failure | What followed |
| --- | --- | --- |
| Item 4, review | The spawn event carried no parent: an agent first seen through it would be created without one, and a subagent's spawn would sit under the orchestrator | The mapper sends the caller as `Parent`; the reducer's no-progress guard allows it; `TestSpawnRequestModel` (with a subagent caller) and `TestRequestedModel` cover it, each failing with its fix reverted |

## Criteria amended

None.

## Simulated run (after the evaluation)

The owner asked for a simulated run of the four items before approving:
a replay in a real Windows Terminal, no CLI, no quota. The summary above
stands as it was reported; this run adds to it.

```text
Simulated run - Plan 6 - codex-plan6-limits.jsonl in Windows Terminal: 42 checks, 42 PASS, 0 FAIL
PASS  maximised 21 of 21, default size 21 of 21 (p6sim.ps1, SendInput keys, wheel and drag, UI Automation text, screenshots)
PASS  item 1: "✗ commandExecution · exit 1 · fatal: No configured push destination." in B, not the first line; Enter shows all three output lines
PASS  item 2: "↓ 1 new" beside "27. Checklist step 27 …"; a drag copies the row without the badge; ↑ marks the command above the badge row; the wheel at the bottom follows again
PASS  item 3: 14 of 14 code lines seen; screenshots: Go, Python and TS strings green to their closing delimiter, the markers inside them not comments, the shell backtick ending with its line
PASS  item 4: "/root/list_files gpt-6-mini" in C and its inspection header; "/root/create_found model N/A"
PASS  Ctrl+Q leaves no process; the clipboard restored; TestSyntheticScenarios gains the file; full check 226 top-level tests, 0 FAIL
Mutation checks: 2 killed (B's first line instead of the last; the spawn's model dropped), 0 survived.
Not verified: as above; the run replays a recording with the documented fields put in, not a live turn.
```

The mock is P7 through the real mapper, with the two `spawnAgent` calls
documented by Codex 0.163.0's schema (one with `model`, one without), a
Markdown answer and a failed `git push` put in (`codexPlan6Limits` in
`internal/provider/mock/files_test.go`). Before the passing runs, the harness
had five errors of its own, fixed and recorded in `plan.md`; one of them was
a fixed pause too short at the default size, after which the viewer opened in
4 of 4 tries. Seen during the run, not a defect: an agent model's source is
drawn nowhere, for a requested model as for a reported one, and
`reasoningEffort` is not shown.

## Items 5 and 6 (added after the simulated run)

After the simulated run, the owner added the two things it had seen: an
agent model's source (item 5) and the reasoning effort (item 6), to be done
before the commit. Their criteria are in `plan.md`, written before the work.
The blocks above stand as they were reported; this one adds to them.

```text
Evaluation - Plan 6, items 5 and 6 - 8 criteria: 8 PASS, 0 FAIL (one amended)
PASS 5.1  Inspection header: "gpt-6-mini (requested)", "claude-haiku-5-5 (reported)", "model N/A (not reported)"; TestAgentSourceAndEffort, golden frames
PASS 5.2  Preview header: "gpt-6-mini (requested) · effort low (requested)" at 200 columns; TestAgentSourceAndEffort; the simulated run at 208 columns
PASS 5.3  (amended) C marks a requested model with a grey "*" and its header "* requested model"; rows unchanged without one; every row fits at 120 and 200
PASS 5.4  Tests through the frame; mutant killed; ui-design.md updated; full check
PASS 6.1  Codex: subAgentActivity.reasoningEffort (P7b "high") reported; the spawn's fills only an empty effort; the agent's own wins in either order; null changes nothing; TestAgentEffort, TestRequestedEffort
PASS 6.2  C: the effort after the model, grey, aligned, only when an agent has one; preview beside the model; inspection always, "effort N/A (not reported)" without one
PASS 6.3  An ended agent takes a model-only or effort-only event and nothing else; TestRequestedEffort
PASS 6.4  Mocks: TestMockFiles passes, none changed (no mock is built from P7b); 4 golden frames changed and reviewed; ui-design.md and reliability.md updated; full check 229 top-level tests PASS in 14 packages (226 + 3), 0 FAIL; go.mod and go.sum unchanged
Mutation checks: 4 killed (the source always "(reported)", the subAgentActivity effort dropped, the guard ignoring the effort, a requested effort overriding the agent's own), 0 survived.
Simulated run extended: 27 of 27 PASS maximised, 26 of 26 at the default size (the preview check maximised only); the first run's 20 checks pass again.
Not verified: a spawn request carrying model or reasoningEffort in a recording (none has one; fixtures from the 0.163.0 generated schema); a live turn.
Verdict: DONE
```

Criterion 5.3 was amended during the work, before its result: it asked
for the word "requested" after a requested model in zone C. With a fixed
effort column beside it, the row cut its activity to "a…" at 200 columns
beside the preview. The mark became a grey "*" with a footnote in C's
header, following zone D's "†".

Before the passing simulated run, the harness had two errors of its own:
the maximised window here is 157 columns, under the preview's 160; at 170
the preview box drops its right-hand header, and the first ↓ only selected
C's first row. The harness now shrinks the font to 208 columns, selects
each row, and restores the font. Seen, not changed: between 160 and about
200 columns the preview box drops its whole right-hand header, the model
included, as every box does when its title does not fit; the inspection
shows the model and effort at every width.

## How the check ran

The main session wrote each item's criteria in `plan.md` before the work; one
implementer agent made the changes; the main session reviewed the diff, fixed
what review found, ran the mutation checks and the full check. The timing
test ran three times on the change and three times on 6813c0e in a temporary
worktree, since removed. The CR and BOM check counted bytes with Node, as
Git Bash reports CR in this folder where there is none. Items 5 and 6 were
made, checked and run by the main session, without an agent.
