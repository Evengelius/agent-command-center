# Plan 10 evaluation (2026-10-18)

The final check of plan 10, a spawned Codex agent's label and the timing
step: its two items of
[`plan.md`, Plan 10](../plan.md#plan-10---a-spawned-codex-agents-label-and-the-timing-step)
against the yes/no criteria written in the progress list before their work
started, and the plan's rules (no live run, no quota: the recording P9a is
the evidence; no new dependency). The evidence is `plan.md`'s record of
each item; the block below is the summary reported to the owner. The
results are not to be rewritten.

```text
Evaluation - Plan 10, a spawned Codex agent's label and the timing step - 7 criteria: 7 PASS, 0 FAIL
PASS 1.1 Mapper: P9a's completed spawnAgent gives the child 01a14bd7-1754-76f2-b210-de4cf274f58b one AgentProgress, label "task: Count the files in the c…", description "Count the files in the current directory and reply with the number only.", model gpt-5.6-luna, effort low, requested, parent the orchestrator (TestSpawnRequestRecorded); a prompt with no model or effort gives the label alone, a blank prompt nothing (TestSpawnLabel, 9 prompts; TestAgentEffort)
PASS 1.2 State: a spawn label fills an empty label, "unannounced agent" or a previous spawn label; the agent's own label replaces it and is never replaced by it, in either order; the description fills only an empty one; a completed agent takes the label, description and model with an empty history (TestRequestedLabel, 7 steps)
PASS 1.3 Dashboard, P9a replayed (new mock codex-spawn-request.jsonl): C row "✓ task: Count the files in the… gpt-5.6-luna* low completed" at 120x40, "✓ task: Count the … gpt-5.6-luna* low completed" at 200x60, no "unannounced agent"; inspection "◀ back (Esc)   task: Count the files in the c… · gpt-5.6-luna (requested)", first line "started by orchestrator · Count the files in the current directory and reply with the number only."; plan 6's run keeps /root/list_files and /root/create_fou… (TestSpawnRequestLabel, TestSyntheticScenarios)
PASS 1.4 Mutants: 3 killed, 0 survived (label not emitted; the spawn label over the agent's own; the spawn label refused over "unannounced agent"); mapper.go and reduce.go restored byte for byte
PASS 1.5 Documents: doc/ui-design.md (Zone C label), doc/events-and-state.md (Agent tree), doc/research/codex-cli.md (open question 3, P9a)
PASS 2   Timing step, run alone: TestKeystrokeToFrame p95 1.011 ms; TestDraftKeystrokes p95 2.313 ms; TestLongSession frames p95 3.48 / 3.44 ms, delta-to-frame p95 3.000 ms; TestKeystrokeCost 227.9 µs per key at 10,000 lines, 493.7 µs at 50,000; every p95 under 16 ms, 4 PASS
PASS F   go build ./... and go vet ./... (also -tags live, -tags timing) exit 0; gofmt -l . prints nothing; go test ./... 235 top-level tests PASS in 14 packages (232 at 8dea129 plus TestSpawnLabel, TestRequestedLabel, TestSpawnRequestLabel), 0 FAIL; the changed files have no CR and no BOM; go.mod and go.sum unchanged
Mutation checks: 3 mutants killed, 0 survived.
Not verified: a live run of the label (none in scope; P9a's recording replayed through the whole pipeline instead).
Verdict: DONE
```

## Not verified

- **A live run.** Plan 10's rules exclude one: the label comes from a field
  P9a recorded live on Codex 0.163.0, and the recording goes through the
  mapper, the state and the frame in `TestSpawnRequestLabel`. A Codex
  version that changes `collabAgentToolCall.prompt` would need a new probe.

Observed, not a criterion: the first mutant (label not emitted) left the
dashboard tests passing, because the committed mock file still carries the
labels; `TestMockFiles` catches the mismatch, so the mutant is killed.

## Failures on record

None in the checks. During the work, before scoring: `TestAgentEffort`
failed by design once a prompt-only spawn emitted a label event, and its
expectation was changed to require exactly that event (stricter, not
weaker); `TestMockFiles` failed on codex-plan6-limits.jsonl until the mock
files were rewritten with `-update` (3 lines, labels and descriptions
only, diff reviewed).

## Criteria amended

Criteria 1 and 3, before the checks were scored, recorded in `plan.md`:
"24 cells" is 24 characters (runes; the same for P9a's ASCII prompt); the
expected label `task: Count the files in the …` was a slip for
`task: Count the files in the c…`, which is what a 24-character cut of
"Count the files in the current..." gives; and C's row shows the label as
zone C's label column cuts it (plan 7, item 1), differently at 120 and 200
columns. The intent of each criterion (the task as label, the model and
effort beside it, no "unannounced agent", the whole prompt in the
inspection) is unchanged.
