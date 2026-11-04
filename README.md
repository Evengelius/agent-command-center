# Agent Command Center

A personal terminal workspace, a "mission control", for multi-agent coding
with agent CLIs. One persistent screen of three zones: the orchestrator with
its workflow tree and telemetry (quota, tokens, context, costs); a real,
streaming conversation with it, approvals and follow-ups included; and the
live subagents, each inspectable. It drives each CLI through its supported
interfaces: the CLI does the work, the dashboard converses with it, shows it
and relays approvals.

Claude Code is the supported CLI. The OpenAI Codex CLI adapter is paused,
built and tested but not offered by the dashboard
([provider parity](doc/provider-parity.md)). The project is proprietary (all
rights reserved, see [LICENSE](LICENSE)).

## Requirements

- Windows 11 and Windows Terminal (measured with version 1.23).
- Go 1.27.2.
- Claude Code, for a real session. Without it the dashboard shows it as not
  installed; mock mode starts no CLI.
- Git on the PATH, for the git views.
- Cascadia Mono NF: where Windows has none, `acc` installs the copy bundled
  in `third_party/fonts` for the user.

## Build, run, test

From the repository root ([AGENTS.md](AGENTS.md), Commands, has the full
list):

```text
go build ./...
go test ./...
go run ./cmd/agent-command-center --mock testdata/mock/claude-subagent-approval.jsonl
.\acc
```

`acc` (`acc.cmd`) starts the dashboard from this repository with `go run`,
its project field prefilled with the folder it runs in; in Windows Terminal
it opens in a tab of its own at a 9 pt font. Tests never start an agent CLI;
live runs follow [doc/evaluation.md](doc/evaluation.md).

## Documents

- [AGENTS.md](AGENTS.md): the rules for any agent working here, and the
  repository's layout; [CLAUDE.md](CLAUDE.md) adds Claude Code's development
  setup.
- [plan.md](plan.md): the active plan, if any, the owner's standing decisions
  and one entry per closed plan; each plan's results are in
  `doc/planN-evaluation.md`.
- [doc/brief.md](doc/brief.md): the owner's request, word for word, and its
  amendments.
- The design, kept current: [architecture](doc/architecture.md),
  [events and state](doc/events-and-state.md), [UI](doc/ui-design.md),
  [reliability](doc/reliability.md), [provider parity](doc/provider-parity.md)
  and the [reference design](doc/reference-design.md).
- [doc/evaluation.md](doc/evaluation.md): how every change is checked and
  reported.
- [THIRD_PARTY.md](THIRD_PARTY.md): every module and its licence, the patched
  `third_party/ultraviolet` and the bundled font.
