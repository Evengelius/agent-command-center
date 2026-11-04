# Provider parity

What differs between the agent CLIs ACC drives, feature by feature. The aim is
that a CLI paused, or one added later, can reach the same function, not
necessarily the same implementation (brief, amendment 96).

- **When it changes.** Only when a plan introduces a meaningful functional
  difference between the CLIs, as one of that plan's files. Earlier plans are
  not audited (amendment 97).
- **What it holds.** One row per difference, pointing to where the feature is
  specified. The specifications' own "Claude Code | Codex" tables stay the
  reference for detailed behaviour: `events-and-state.md`, `ui-design.md`,
  `reliability.md`, `research/integration.md` and
  `research/telemetry-sources.md`.

## Codex: paused

Since plan 76 (amendment 96), Claude Code comes first: a new capability is
designed, built and validated for it. Codex's adapter, tests, probes and mock
files stay, and its tests run in `go test ./...`. No dedicated Codex
validation campaign runs meanwhile (amendment 97).

**What the pause does.** `cmd/agent-command-center/main.go` leaves Codex's
adapter out of its provider list, so:
- the home screen keeps Codex's row, greyed and not choosable, reading
  "paused";
- the session history keeps its entries and resumes none, saying "OpenAI
  Codex is paused";
- `--provider codex` chooses Claude Code and says why;
- the home screen shows neither Codex's proxy from `config.json` nor a
  problem in its `codex` setting. Both come back with the adapter.

Codex's recorded projects and sessions stay in `projects.json`, within ACC's
limits (plan 75): 20 sessions per project, counting every CLI, and the 10 most
recent projects that are not favourites. Codex's own session store is never
touched.

**To reactivate Codex:**
1. Put the adapter back in `main.go`'s provider list.
2. Run the whole suite: Codex's tests never stopped.
3. Compare `codex --version` with the last probed version
   (`research/probes/README.md`). If it is newer, run `evaluation.md`, After a
   CLI update, for Codex.
4. Run one live run on Codex, as plan 75's L75.
5. Go through the rows below and decide, in a plan, which to close.

## Differences

| Feature | Claude Code | Codex | What parity needs | Where |
|---|---|---|---|---|
| The read-only Reviewer of ACC's own development (plan 74) | a native subagent whose only tools are `Read`, `Grep` and `Glob`, besides the CLI's own report and `advisor` tools | none: a Codex subagent reads files through its shell, which a definition cannot take away. A step led by Codex is reported NOT REVIEWED, and Codex's own `/review` gives findings, never a PASS | a Codex agent without a shell, or a later review by Claude Code | `AGENTS.md`, Review cycle; brief, amendment 88 |
| Convergence and execution supervision (plan 76) | the review cycle's evidence-based stops; deadlines set with Claude Code's Monitor tool | not defined: with no Reviewer, there is no review cycle to converge | first the Reviewer, then a way for Codex's orchestrator to set a deadline | `evaluation.md`, The Reviewer, Convergence; Execution supervision |
| **[Nice to have]** Earlier turns after a resume from the home screen (plan 69, design D1) | not drawn: Claude Code's `--resume` replays no earlier message | not drawn, though `thread/turns/list` and `thread/items/list` could page them | Codex's pages drawn, with their rendering cost, their tool events and very long sessions; until then both CLIs behave the same | `reliability.md`, Resumption; `ui-design.md`, the resumed session's note |
