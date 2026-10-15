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

A check that cannot fail proves nothing. When you write a new check, run it once
on a damaged input and make sure it fails.

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
7. Agent CLIs it supports (Claude Code, Codex CLI, Gemini CLI), and how
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
5. With no Codex data at all, the views show Codex as not installed; Claude Code
   and Gemini CLI views unchanged
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
