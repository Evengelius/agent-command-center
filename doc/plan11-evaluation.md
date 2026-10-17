# Plan 11 evaluation (2026-10-18)

The final check of plan 11, a live approval offering the allow for the
session: its one item of
[`plan.md`, Plan 11](../plan.md#plan-11---a-live-approval-offering-the-allow-for-the-session)
against the yes/no criteria written in the progress list before its work
started, and the plan's rules (the owner's Codex quota, temporary projects
outside the repository, per-process `-c` overrides only, no configuration
file or system setting changed by the work, no new dependency). The
evidence is `plan.md`'s record of the item; the block below is the summary
reported to the owner. The results are not to be rewritten.

```text
Evaluation - Plan 11, a live approval offering the allow for the session - 3 criteria: 3 PASS, 0 FAIL; 1 item not verifiable
PASS C   Configuration check after P11n and P11n-b, against the snapshot taken after the owner's elevated sandbox setup: 24 files identical, 5 new session files, CLI ran (configuration: PASS); the setup itself changed only [windows] sandbox in ~/.codex/config.toml, "unelevated" to "elevated"
PASS D   Documents: plan.md (item 1's findings), doc/research/codex-cli.md (open question 4), doc/research/probes/README.md (P11n, P11n-b, the configuration check); the two recordings sanitized (a scan for the user, host and organisation finds nothing), in doc/research/probes only
PASS F   go build ./... and go vet ./... (also -tags live, -tags timing) exit 0; gofmt -l . prints nothing; go test ./... 235 top-level tests PASS in 14 packages, 0 FAIL; the changed files have no CR and no BOM; go.mod and go.sum unchanged
Not verifiable: item 1, a live approval whose list holds acceptForSession (P11n: with the elevated sandbox, Codex's managed proxy refuses the domain and asks only an ordinary escalation, accept / acceptWithExecpolicyAmendment / cancel; the network approval needs an admin-enforced requirements.toml with a [network] table, which the owner chose not to create).
Not verified: the timing step (no code changed).
Verdict: DONE
```

## Not verifiable

- **Item 1, a live approval whose list holds `acceptForSession`.** None of
  its three criteria was met or attempted past the first:
  1. Probe P11n (Codex 0.163.0, the elevated Windows sandbox,
     `-c features.network_proxy=true -c sandbox_mode="workspace-write"
     -c sandbox_workspace_write.network_access=true`, `curl.exe` to
     example.com twice, every approval answered `acceptForSession`): the
     first run was refused by Codex's managed proxy, "domain is not on the
     allowlist for the current sandbox mode", with no network approval.
     Codex then asked an ordinary escalation, `availableDecisions`
     `["accept", {"acceptWithExecpolicyAmendment": ...}, "cancel"]`, no
     `networkApprovalContext`. The reply `acceptForSession`, not offered,
     was taken, and the second run asked nothing. P11n-b, without network
     access, failed both runs in under 200 ms with no approval of any kind.
  2. and 3. Not attempted: there was no network approval to record, replay
     or show live.

  Why, from openai/codex rust-v0.163.0 read as data: the allowlist-miss
  decider that raises a network approval (the only one to which Codex adds
  `acceptForSession`) is built only when the admin-enforced
  `requirements.toml` has a `[network]` table
  (`core/src/session/session.rs`, `core/src/config/network_proxy_spec.rs`
  `start_proxy`); on Windows that file is
  `%ProgramData%\OpenAI\Codex\requirements.toml`
  (`config/src/loader/mod.rs`), which no `-c` override reaches and which
  does not exist on this machine. Creating it is a machine-wide
  administrator write, enforced on every Codex session until removed. On
  2026-10-18 the owner chose to close the item as not verifiable with this
  evidence and not to pursue it.

What stays true: the dashboard offers "s allow for the session" whenever a
request's list holds it or carries no list (plan 8, replay tests), and
answers it with `acceptForSession`. That a live Codex sends such a list was
not observed.

## Not verified

- **The timing step.** No code changed in this plan.

## Failures on record

None in the checks.

## Left on the machine

The owner's elevated sandbox setup stays as they made it: `[windows] sandbox
= "elevated"` in `~/.codex/config.toml`, the firewall rule
`codex_sandbox_offline_block_inbound`. Undo, if wanted: set the line back to
"unelevated".
