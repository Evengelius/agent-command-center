# Evaluation

How an agent checks its own work after each instruction, before saying it is
done.

Every check is a yes/no check with an exact expected value: a count, a text, an
exit code, a source. There is no score: a criterion passes or fails, and the
work is done only when every criterion passes. What only the owner can judge
(how the screen looks and feels) is listed as not verified, never assumed.

## The loop

1. **Before working**, write down how the result will be checked: one line per
   criterion, each with its exact expected value, taken from the instruction,
   `doc/brief.md` and `AGENTS.md`. Put the criteria of a plan in `plan.md`.
2. **Work.**
3. **Check** every criterion and keep its evidence: a link with a commit, a
   file and line, a command output, an exit code, a diff.
4. **Summarise** in the format below.
5. **If something failed**, give the likely cause and a proposed next step. Do
   not apply it without the owner's approval.

## Kinds of exact checks

| Check | Expected value, for example |
|---|---|
| A research claim | a source: repository URL, the commit inspected, file and line; or a documentation URL and the section |
| A license | the SPDX identifier read from the repository's license file, at that commit |
| A dependency | its name and version, read from `go.mod`, `package.json` or the lock file |
| A documented interface | the documentation page and section that defines it; "undocumented" when there is none |
| A probe (plan 1) | approved by the owner first; run in a temporary folder outside the repository; its command, CLI version and raw output recorded in `doc/research/probes/`, with credentials and personal data removed; no CLI configuration changed (hashes before and after) |
| A document | the sections the plan requires, each present; every brief item covered, with where |
| Working copy | `git status` lists only the intended files |
| Build | `go build ./...` exit 0, no warning |
| Tests | `go test ./...` exit 0, with the count of tests run |
| Vet and format | `go vet ./...` exit 0; `gofmt -l .` prints nothing |
| Telemetry parsing | each fixture parses to the expected values; malformed input gives an error, not a crash |
| Agent state transitions | each transition in a table test, including the invalid ones |
| Rendering | a golden snapshot of a view at a fixed size, from mock data, matches |
| Agent CLI configuration untouched | every file under `~/.claude`, `~/.codex` and `~/.gemini` that the work could touch, and the project settings, have the same hash before and after |
| No credential read | no credential file (`auth.json`, `oauth_creds.json`, `.credentials.json`, any token or key) opened, in research notes or in code |
| Each agent CLI optional | with one CLI's data absent, the views show it as not installed and the others keep working |
| Conversation is real | every message in the conversation state traces to an event from the CLI's session (a recorded stream replayed in a test); none is built from status messages |
| Event mapping | each conceptual event of amendment 2 maps to a documented source, with the docs URL, or is marked unavailable with its fallback |
| Real values or "N/A" | each telemetry field with its source absent renders "N/A", never 0 or a guess; API-equivalent cost never shown as subscription usage |
| Rendering without models | the rendering and state packages make no model or network call (dependency check, or a test with the network unavailable) |
| Mock mode labelled | in mock mode every screen shows that it is mock data; mock events go through the same pipeline as real ones |
| Approval and cancellation | an approval request reaches the screen and the answer reaches the CLI; a cancellation leaves the session resumable (a recorded session in a test) |
| Paste preserved | a bracketed paste of fixtures (code with tabs and spaces, CRLF and LF text, Unicode, trailing whitespace) reaches the CLI byte for byte |
| Paste never submits | a multiline paste ending with a newline leaves the message unsent; only the configured send key sends |
| Large input | a 10,000-line paste stays editable, each keystroke handled within a time budget set in plan 2 |
| Clipboard and selection | copy and paste per terminal and OS listed in a matrix (Windows Terminal, PowerShell, WSL2...): each cell tested, or marked not verified |
| Provider-independent input | the same input tests pass with each agent CLI's adapter |
| Provider detection | for each agent CLI, installed gives its version, missing gives the startup error message (a test with the CLI absent from the PATH) |
| Path validation | table tests: Windows path, `~`, WSL path, relative path, trailing separator, non-existent, a file instead of a directory, no access; each accepted or refused with its message |
| Working directory | the started CLI's working directory equals the chosen project path, read from the process or the CLI's own output |
| Startup gate | the dashboard cannot open before both answers are validated |
| MVP scope | none of the features amendment 4 leaves out exists in the code or the screens |
| MVP acceptance | plan 4 ends with amendment 4's acceptance criteria, each checked with each agent CLI, or listed under not verified |

A check that cannot fail proves nothing. When you write a new check, run it once
on a damaged input and make sure it fails.

## Keystroke timing

A keystroke is checked twice: a deterministic guard in every test run, and a
timing step on the real clock run on its own (owner, 2026-10-16). Timed under
`go test ./...`, the keystroke tests shared the CPU with every other package
and missed the target from load alone (plan 4, step 4: 2 of 6 suites; step 5,
check 8), with no change in the app's per-key cost. This replaces the earlier
procedure of the same day, which reran them alone after a full-suite miss; the
results it produced stay recorded in `plan.md`.

**The guard, in `go test ./...`.** `TestKeystrokeAllocs` (`internal/app`)
types, moves, deletes and selects in a 10,000-line draft, at its top and at its
bottom. It fails when a keystroke, from `Update` to the frame's string, takes
more than `keystrokeAllocs` (1,200) allocations on average, or when the editor
draws from more than four screen heights of the draft. Allocations do not
depend on the machine's load, so the guard gives the same answer every run;
drawing the whole draft, the whole screen afresh, or the editor one span per
cell breaks it (each checked by mutation). `TestFrameReuse` checks that a
frame built from the one before it equals the frame drawn afresh, escape codes
included.

**The timing step, on its own.** The timed tests carry the `timing` build tag,
so `go test ./...` never runs them. They run as a separate step, under the
owner's normal working conditions (owner, 2026-10-16), since the target is
about real use:

```text
go test -tags timing ./internal/app -run '^(TestKeystrokeToFrame|TestDraftKeystrokes|TestKeystrokeCost)$' -count=1 -v
```

- Allowed: the owner's usual applications open and idle (VS Code, browsers,
  RanorexStudio).
- Required: the laptop on mains power, and no other heavy job at the same time:
  no other `go` build or test, no Ranorex test execution, no install, virus
  scan or large copy.
- Recorded with each run, not a gate: the power state, the CPU load just
  before the run, and the applications open.

- `TestKeystrokeToFrame` (10,000-line draft) and `TestDraftKeystrokes`
  (50,000 lines, saved while typing) each send 50 untimed warm-up keys, then
  time every key to its frame on the wall clock, the garbage collector on.
- The target is p95 ≤ 16 ms (`keystrokeTarget`); it never changes to make a
  run pass. p50, p95 and the maximum are recorded in `plan.md` with the
  conditions.
- `TestKeystrokeCost` judges nothing: it reports a key's mean time, bytes and
  allocations, the before and after of a change to the drawing.
- A miss is a FAIL: do not rerun until it passes. Investigate the cause (a CPU
  profile, the same test on the last commit) and report it as the loop above
  says.

## Summary format

```text
Evaluation - <the instruction in a few words> - <n> criteria: <p> PASS, <f> FAIL
PASS 1  <criterion>: <evidence>
FAIL 2  <criterion>: expected <...>, found <...>
Not verified: <what could not be checked, and why>
Verdict: DONE | NOT DONE
Likely cause: <for each FAIL>
Proposed next step: <for each FAIL> - waiting for approval
```

## Examples

### Research one reference project (plan 1)

Instruction: "Inspect jedarden/ccdash."

```text
1. doc/research/ccdash.md exists, with the sections of plan 1
2. Commit inspected recorded (full SHA) and its date
3. License: SPDX id from its LICENSE file, and what it allows (fork, copy, link)
4. Language, UI library and every direct dependency with its version, from go.mod
5. Each data source it reads (files, endpoints, hooks), with file:line where it reads it
6. Each source marked documented (with the docs URL) or undocumented
7. Agent CLIs it supports (Claude Code, Codex CLI, others), and how
8. Reusable parts listed, each with its license consequence
9. No claim without a source
Not verified: whether it runs on this machine (nothing is installed in plan 1)
```

### Parse one agent CLI's session telemetry (an implementation plan)

Instruction: "Parse token usage from Codex CLI session files."

```text
1. The Codex collector has a parser with table tests over fixtures in testdata/,
   and fills the common model only
2. Fixtures cover: a normal session, a session with parallel work, an empty
   file, a truncated last line, an unknown event type
3. go test ./... exit 0, n tests; the truncated-line test expects an error, not a panic
4. The parser never writes: no file under ~/.codex changes (hashes before/after);
   auth.json never opened
5. With no Codex data at all, the views show Codex as not installed; the Claude
   Code views unchanged
6. go vet ./... exit 0; gofmt -l . empty; go build ./... exit 0
Not verified: real values against the vendor's usage page (needs the owner's account)
```

## Known limits

- **The look is the owner's call.** A golden snapshot proves a view did not
  change, not that it looks right. Show the owner a screenshot or a mock-data
  run, and list "visual review" under not verified.
- **Quota figures are only as good as their source.** Each vendor counts and
  resets usage its own way (Claude's 5-hour and 7-day windows have no exact
  equivalent elsewhere). If a figure comes from an undocumented source, it can
  change without notice; say so wherever it is shown.
- **Windows is the host.** The owner works on Windows 11; tmux only runs under
  WSL there. Anything that needs tmux, a Unix shell or a POSIX terminal must say
  how it runs on this machine.
- **The network goes through a proxy.** `curl` fails the TLS handshake here;
  PowerShell's web client with the system proxy works (see AGENTS.md).
