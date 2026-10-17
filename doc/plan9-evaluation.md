# Plan 9 evaluation (2026-10-18)

The final check of plan 9, plan 8's items left not verified: its three
items of [`plan.md`, Plan 9](../plan.md#plan-9---plan-8s-items-left-not-verified)
against the yes/no criteria written in the progress list before their work
started, and the plan's rules (the owner's Codex quota, temporary projects
outside the repository, per-process `-c` overrides through a test-only
wrapper, no configuration file or system setting changed, no new
dependency). The evidence is `plan.md`'s record of each item; the block
below is the summary reported to the owner. The results are not to be
rewritten.

```text
Evaluation - Plan 9, plan 8's items left not verified - 6 criteria: 6 PASS, 0 FAIL (one harness run, 2 FAIL, on record); 1 item not verifiable
PASS 1.1 Probe P9a (Codex 0.163.0, the dashboard's handshake, -c model="gpt-5.6-luna", the catalogue's multi-agent v1): the spawnAgent item carries model gpt-5.6-luna and reasoningEffort low; reply "2", exit 0 after 12 s; configuration check PASS before and after
PASS 1.2 p9a-codex-spawn-request.jsonl (63 lines) identical in testdata/probes and doc/research/probes; TestFixturesMatchRecordings counts it; TestSpawnRequestRecorded replays it: the child gets gpt-5.6-luna and low, source "spawn request", on AgentProgress, its parent the orchestrator
PASS 1.3 Live, the dashboard through the wrapper (acc-p9live-235645, turn 27.0 s, no approval asked), at 156 columns maximised and at 120 by 30: C row "unannounced agent | gpt-5.6-luna* | low", C header "* requested model · 0 ACTIVE · 1 DONE", inspection "gpt-5.6-luna (requested) · ... · effort low (requested)"; the wrapper's log holds spawnAgent model gpt-5.6-luna, effort low
PASS 2   Same run: the wrapper's log holds git status completed with exitCode 1, last line "fatal: not a git repository (or any of the parent directories): .git"; B shows "✗ commandExecution · exit 1 · fatal: not a git repository (or any of the parent directories): .git" at 156 columns, the same cut with "…" at 120; Ctrl+Q 0.21 s, 0 of 3 started processes left (wrapper included); configuration check PASS
PASS D   Documents: doc/research/codex-cli.md (open question 3, question 4), doc/research/probes/README.md (P9n, P9a, the configuration checks), doc/ui-design.md, doc/reliability.md; the probe driver in doc/research/probes/drivers/p9-probe.mjs
PASS F   go build ./... and go vet ./... (also -tags live, -tags timing) exit 0; gofmt -l . prints nothing; go test ./... 232 top-level tests PASS in 14 packages (231 at 1daeec1 plus TestSpawnRequestRecorded), 0 FAIL; the changed files have no CR and no BOM; go.mod and go.sum unchanged, 4 direct modules
Mutation checks: 4 mutants killed, 0 survived (spawn emission disabled; ModelRequested false; the parent set to the child; the effort dropped).
Not verifiable: item 3, a live approval whose list holds acceptForSession (P9n: managed networking needs the elevated Windows sandbox, which the owner declined).
Not verified: the timing step (no input or stream path changed).
Verdict: DONE
```

## Not verified

What no check of this plan settles, and why. None of it is reported as PASS.

- **Item 3, a live approval whose list holds `acceptForSession`.** Codex
  0.163.0 adds it only to a network approval (`default_available_decisions`).
  Probe P9n (2026-10-17, `features.network_proxy=true`, workspace-write with
  network access, `curl.exe` to example.com) asked for no approval: the
  command never started, stderr "managed networking requires the elevated
  Windows sandbox backend". Raising one needs
  `codex sandbox setup --elevated --current-user`, which creates the local
  accounts CodexSandboxOffline and CodexSandboxOnline, firewall rules and
  ACLs, and writes `windows.sandbox = "elevated"` to `~/.codex/config.toml`
  (`cli/src/sandbox_setup.rs` 86-124). The owner chose on 2026-10-18 to
  record the item as not verifiable rather than make that change. The case
  stays covered by `TestOfferedDecisions` and `TestCodexSessionAllow`
  (synthetic lists).
- **The keystroke timing step.** Plan 9 added a recording, a mapper test
  and documents; no input, stream or frame-loop path changed, so the step
  was not run.

Observed, not a criterion: on the multi-agent v1 backend C labels the
child "unannounced agent"; no plan 9 criterion covers its name.

## Failures on record

- **Harness, run phase** (23:58:46): the turn passed its checks, then the
  harness failed reading the wrapper's log, held open by the wrapper:
  `[ERROR] FAIL  Exception calling "ReadAllLines" with "1" argument(s):
  "The process cannot access the file '...\wrap\log-p9live\...'`. Fixed in
  the harness (read with `FileShare.ReadWrite`); the dashboard was not
  touched and kept running for the check phase.
- **Harness, first check run** (23:58:56 to 23:59:10): 2 FAIL of 12, B's
  failed row at each size. Verbatim:

  ```text
  23:58:58 [2 max B's failed row matches Codex's exit code and last line] FAIL  B '✗ commandExecution · exit 1 · fatal: not a git repository (or any of the parent directories): .git          observed'
  23:59:05 [2 default B's failed row matches Codex's exit code and last line] FAIL  B '✗ commandExecution · exit 1 · fatal: not a git repository (or any of th…observed'
  23:59:10 [END] FAIL  2 FAIL
  ```

  Cause: the harness's pattern took B's provenance tag "observed", which
  ends every event row, as part of the command's last line. The rows shown
  are the expected ones. Fixed in the harness: the tag is stripped, the line
  must equal the wrapper's last line when whole, and be its prefix only
  when cut by "…". The re-run on the same frames (23:59:31 to 23:59:45)
  gave 12 of 12 PASS; the quit phase 3 of 3.
- **Harness, first launch attempt:** Windows Terminal split the command line
  at the `;` of the PATH it set, opened a window titled "%PATH%&&" that
  started nothing, and the harness found no dashboard window. Fixed by
  escaping it as `\;`; the stray window was closed. No check had run.

## Criteria amended

None. The harness's comparison was corrected to match the criterion as
written (the last non-blank line of the output); the criterion was not
changed.
