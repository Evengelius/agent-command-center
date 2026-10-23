# Plan 33 evaluation (2026-10-23)

The check of plan 33, unsent drafts and images through crashes and
restarts (brief amendment 33: the owner asked whether unsent image
attachments survive a crash, a forced termination, a restart and a
resumed session, then approved the investigation's nine corrections,
"pasted images kept on disk until the message is accepted"). It covers
the criteria written in [`plan.md`, Plan 33](../plan.md) before the code,
with the corrections recorded there. Nothing is committed: the owner
reviews the work.

How it was checked:
- Deterministic tests: `internal/input` (TestDraftSlots, TestDraftKey,
  TestImageOnlySave, TestDraftPasted, TestDamagedList, TestDraftSweep,
  TestSaveWhileRead, TestDraftImages), `internal/app` (TestTwoDashboards,
  TestBackForgetsAttachments, TestAttachSavesNow, TestDraftWhileWaiting,
  TestDamagedImagesList, TestChangedFile, TestPastedKept,
  TestDraftSaveRetried, TestCodexRestart; TestDraftAcceptedDeletes and
  TestPastedDraft changed as D4 and D8 say). A crash in them drops a
  model as a killed process would: its draft's lock closed, nothing
  written after.
- The recovery matrix (TestRecovery): the real program (`Run`) in a child
  process, the fake Claude Code, a draft of text, three generated files of
  different bytes (one named with spaces, one in Unicode) and a pasted
  image, ended by a panic, by the quit Bubble Tea makes of SIGTERM, and by
  TerminateProcess; a new run of the project then reports what it
  restored, names and SHA-256.
- Windows Terminal 1.23: the executable built from this tree with drafts
  on in mock mode (`scratchpad\term\acc-p33b.exe`, one condition removed
  in a copy of the tree), driven by `scratchpad\term\p33b.ps1` (the
  investigation's harness with plan 32's clipboard steps; run with
  `-STA`, the owner's clipboard saved before the steps and restored after,
  4 formats each time), maximised (156x44), then at 120x40: three
  generated files through the Open dialog, a generated JPEG copied as a
  bitmap and pasted with Ctrl+V, text typed; the window closed, or the
  process killed, about 200 ms after a last word was typed; then a
  restart in the same folders. No terminal-hosted Claude Code or Codex
  session was open, so none was closed.
- Build, vet, gofmt, the whole suite, the race detector on the changed
  packages, the timing step alone; an independent review of the diff
  (the `verifier` agent).

```text
Evaluation - Plan 33, unsent drafts and images through crashes and restarts - 14 criteria: 14 PASS, 0 FAIL
PASS 33.1 TestDraftSlots: no drafts folder before the first save, none made by an empty save; the first save: one folder named the project's SHA-256, "-", 16 hexadecimal digits (no part of the path), holding draft.txt, images.json and lock, no temporary file; the text byte for byte (a tab, CR LF, wide characters, an emoji, an escape, a NUL); held: a second store of the project takes nothing, cannot read the lock, and leaves the folder's files as they were; let go: a save after it changes nothing, the lock holds "claude-code", a Codex store takes it (its folder, text and list unchanged, no other counted), the lock then "codex"; two let go, Claude Code's the newer: Codex takes its own (1 more counted), Claude Code its own; two of Codex: the newer, 1 more counted; a pre-plan-33 <sha>.txt and <sha>.images.json: taken, text and list (a plan 32 pasted entry in it) unchanged, moved into a new folder of the project, none left beside the folders; emptied then let go: the folder deleted. TestDraftKey: the SHA-256 of the cleaned, lower-cased path, the same in another case. TestTakeWhileOpen: a draft whose lock another program held 100 ms taken once it let go; held past the retries: nothing taken, 1 counted; TestDraftInUse: then the toast "a stored draft of this project is in use: the next start takes it", and the next start takes it; TestTakeOwned: a draft another dashboard owns is left at once, neither waited for nor counted (under 300 ms with the look, under 100 ms for the open alone)
PASS 33.2 TestTwoDashboards: a Claude Code and a Codex dashboard on one project (the real adapters over fake CLIs), each with text and an image: two folders; Claude Code's message accepted (probe P2's lines 7 to 9), then new text and a GIF: every file of Codex's folder but its lock unchanged byte for byte; both killed; Claude Code again: "claude after" with the GIF, the toast "draft restored · 1 more stored draft of this project"; Codex again: "codex draft" with the JPEG
PASS 33.3 TestBackForgetsAttachments: the handshake never answered, text, a PNG from the dialog and a pasted image (two thumbnails); the CLI exits, "The session could not start", Esc: the startup screen, no attachment, pasted bytes, thumbnail or text left; the first project's folder: the text, the list [the PNG's path, pasted:pasted-image-1.png#<sha256>], the pasted copy equal to the bytes; a session in another project: Ready, nothing attached, its message written without an image; a new dashboard in the first project: the text, the PNG and pasted-image-1.png once, the toast "draft restored"; Back then Enter in the same project in one dashboard: the text, the JPEG and the pasted image once, "draft restored"
PASS 33.4 TestAttachSavesNow: the draft's timer never firing: after the dialog's two files the list holds both; after Alt+V the pasted entry too, its copy equal to the bytes; after the first chip's ✕ the JPEG and the pasted entry; no timer armed by an image; typed text not stored and one timer armed, stored "typed" once it fires. TestRecovery/killnow: the four images stored 21 ms after the last was attached, the text's timer still pending; TerminateProcess under a millisecond after: the next run restored the four images (names and SHA-256) and the text without the word typed last
PASS 33.5 TestDraftWhileWaiting: while "sent" with A waits for its acceptance, "typed" and A and B attached: stored "sent\ntyped" with A then B, A once; a crash then: a new model restores "sent\ntyped", A and B; the acceptance instead: stored "typed" with A and B; an image-only message given back by a CLI crash after "x" was typed: the editor holds "x", the image back; Enter again while "sent" and a pasted image wait: refused ("turn running: wait, or interrupt"), the state still waiting with "sent" and its image, stored "sent\nsecond" with the pasted copy kept, and a crash then restoring both
PASS 33.6 TestImageOnlySave: images without text: images.json and lock, no draft.txt; text, then no text, with the same images: the list unchanged after each save; no text and no images: lock alone
PASS 33.7 TestDamagedImagesList: images.json holding ["C:\\half: the text restored, nothing attached, the toast "draft restored; its images list was unreadable and was set aside", images.json.unreadable with the same bytes, drafts on ("!" typed: stored "precious text!"). TestDamagedList: the same at the store
PASS 33.8 TestChangedFile: the attached file rewritten with another image: no message written, the notice "the message could not be sent: shot.png: changed since it was attached; attached again as it is now", the text and the chip kept, the toast "shot.png: updated", the thumbnail from the new bytes; Enter: the new bytes sent; attached again by hand: unchanged "shot.png: already attached", changed "shot.png: updated" with one chip and the new thumbnail; rewritten with the same bytes: sent as it is
PASS 33.9 TestPastedKept: Alt+V: <sha256>.png in the dashboard's folder equal to the pasted bytes, the list [pasted:pasted-image-1.png#<sha256>], no other file of the drafts folder holding them raw or in base64; killed, then a new model: pasted-image-1.png and its thumbnail; Enter: the same bytes sent; the copy kept while the message waits, deleted at its acceptance; the next paste "pasted-image-2.png", its copy written, deleted by Ctrl+Shift+X; a third paste killed and its copy altered: "pasted-image-3.png: its stored copy is damaged", not attached; a plan 32 draft: "legacy" and the toast "draft restored; pasted-image-1.png was pasted, not kept". TestDraftPasted: the entry's form, ParsePasted's four cases, a save without the bytes refused, the copy read back, altered "its stored copy is damaged", missing "its stored copy is missing", unlisted deleted. TestPastedDraft: the copy in one file only, the pasted image and the JPEG restored in order, "draft restored". TestRestoreOverLimits: a Codex draft of two pasted images, each under 7.4 MB, both over it, restored in Claude Code (7.4 MB a message): both attached, the toast "the draft's images: more than this session sends in a message: remove some before sending", the send refused ("the images are over 7.4 MB a message"), both copies kept. TestRestoreOverCount: three stored files restored in a session taking two images a message: all three attached and said, the send refused ("3 images, over 2 a message"), sent with two once one is removed
PASS 33.10 TestDraftSweep: in a folder let go, .draft-123 and an unlisted pasted file deleted, its draft and listed copy kept; a held folder's files all kept; a folder holding only its lock deleted; a folder without lock two minutes old deleted, one just made kept; beside the folders, .draft-old (two minutes) deleted, .draft-new kept. TestPruneWhileOpen: a pasted file another program holds when its draft no longer lists it: the save written at once without an error, the next save deletes it
PASS 33.11 TestRecovery: ended by a panic (exit code 2), by Bubble Tea's quit (0), by TerminateProcess one second after the last image (1) and 21 ms after it with the text's timer pending (1): each next run restored acc-image-upload-test.png, acc image upload test.jpg, acc-image-upload-test-éè-日本.gif and pasted-image-1.png, each with the SHA-256 of its bytes, and "unsent text with images" (the word typed before the late kill not stored). TestCodexRestart: a Codex session's text, PNG, JPEG and pasted image restored in order; the turn/start's three images byte for byte
PASS 33.12 Windows Terminal, the final build: before each ending the folder held the text without the last word, 4 images listed and the pasted copy; the window closed 183 ms (maximised) and 180 ms (120x40) after the last word began: the window gone in 51 and 36 ms, the dashboard ended by itself in 133 and 122 ms with exit code 0, the folder then holding the last word too; TerminateProcess 191 ms (maximised) and 187 ms (120x40) after it: the last word not stored, the 4 images and the copy kept; each restart: "draft restored", the text, the chips (+2 more, +3 more) and four thumbnails, the pasted image's among them; no .draft-* file in any dump; the owner's clipboard saved and restored in every run (4 formats). Beyond the criterion, the dashboard's tab closed with Ctrl+Shift+W 197 ms (maximised) and 180 ms (120x40) after the last word, its window kept open by another tab: the dashboard ended by itself in 1083 and 1086 ms with exit code 0, the last word and the file stored, both restarts restoring them
PASS 33.13 Build, vet and gofmt clean; the suite, 15 packages ok, TestGoldenFrames among them (no frame changed); the race detector (WinLibs GCC, CGO_ENABLED=1) on the final code: internal/input, internal/state and internal/app ok, 0 data races; on earlier states of the code, internal/app ok five times more and its process, timing and new tests four rounds over, 0 races each; one run before those failed after 167 s with a goroutine trace whose output was not kept, and was not seen again; the timing step alone on the final code, 7 of 7 within target: keystroke-to-frame p95 1.0 ms (10,000-line draft) and 1.1 ms (draft window), 2.3 ms in a 50,000-line draft with 8 saves, the long session's deltas 4.5 ms, paste-to-frame p50 1.1 ms, stream p95 3.5 ms, the thinking card 4.0 ms (targets 16 ms, 100 ms for the paste)
PASS 33.14 Docs: brief.md (amendment 33, the owner's words), plan.md, architecture.md (ASM-23, the Input and application rows), ui-design.md (the send's changed file, the stored draft, Back, pasted images), reliability.md (the dashboard's own failure), AGENTS.md (status), this file; the 31 changed or new files (26 changed, 5 new) read byte by byte: no BOM, no CR; git status lists them only; nothing committed
Not verified: a Windows logoff or shutdown (Go turns their console events into SIGTERM, but a process that loaded the clipboard's or the dialog's libraries may not receive them); two dashboards kept apart outside Windows (held files are Windows', D1); Ctrl+R resuming a Codex session with images (the application's code is the one Claude Code's tests ran); a real antivirus holding a draft's file (simulated by a reader without delete sharing); a stored draft whose images lie on a path that does not answer (a share offline): nothing is saved until its reading ends, as the review noted
Verdict: DONE
```

**The independent review.** The `verifier` agent read the diff against
the criteria, ran the three packages' tests and its own experiments in a
copy of the tree. Its first pass found four defects (plan.md, Progress):
a second Enter while a message waited replaced that message in the state
and the stored draft and deleted its pasted copy (Medium); a stored
draft's image past the session's limits lost its pasted copy at the next
save; a stored draft whose lock another program held a moment was passed
over in silence; a pasted copy that could not be deleted failed the save
after it was written (each Low). Its second pass confirmed the four
fixes and found a fifth defect (Low), the wait for a held lock also
waiting for a draft another dashboard had just taken; fixed too, with
two of its notes (a second look at a lock met during another start's
sweep; no limits toast for images read before the limits were known).
Every criterion above is reported on the code with all five fixes.

**33.12, the runs.** They were made twice: with the build before the
review's fixes, then with the final build, whose figures are above. In
the first set the maximised kill's restart ran in a window of 120x30,
not maximised, and its draft gained an "a" that no step of the harness
types: input from outside the run reached the window; that run was made
again, clean. The final set had no such event.

**The owner's questions before approving (2026-10-23).** On the "Not
verified" list above, the owner asked to simulate a Codex session resumed
with images and a stored draft on a network share gone offline, and a
logoff or shutdown if that can be done. Checked on the code with the
changes below (plan.md, Progress); the block above stays as reported.

```text
Follow-up - the owner's questions before approving - 4 checks: 3 PASS, 1 not run
PASS F1 Codex resumed with images: TestCodexResumeImages, the real Codex adapter over a fake app-server: a PNG and a JPEG from the dialog and a pasted image, the CLI crashed: all three kept, with their thumbnails and the pasted bytes; Ctrl+R: the same thread resumed (thread/resume, a second spawn), Enter: the resumed thread's turn/start carries the three images byte for byte; a message of the three waiting for its acceptance when the CLI crashed: given back to the editor in order, resent after Ctrl+R byte for byte
PASS F2 A path that does not answer: TestRestoreSlowPath, a stored draft of a PNG, a JPEG and a pasted image, the JPEG's read held by the test (no network): text and a GIF added meanwhile, then a crash, a quit or Back. On the code evaluated above the text and the GIF were lost in all three (stored "before" with 3 images); saves now go on during the read: stored "before and after" with 4 images, the next run restoring the text and the four images in order after each of the three. TestRestoreSlowPathAnswers: the read answering while the dashboard runs puts the stored draft's images first, the GIF after. TestRestoreSlowPathWaiting: with a message waiting meanwhile, the order is its image, then the stored draft's, before and after the read and in the editor once the message fails; the stored pasted image pasted again: "pasted-image-1.png: already attached"
PASS F3 A share that stays offline: TestRestoreShareOffline, the JPEG's read failing at last as Windows reports a server it cannot find ("the file is gone"), its share's root unreachable: kept, its chip "acc-image-upload-test.jpg · not reachable", the toast "the draft's images: acc-image-upload-test.jpg: its folder cannot be reached; kept in the draft", still listed in the stored draft; Enter: refused, "the message could not be sent: acc-image-upload-test.jpg: its folder cannot be reached", nothing written; the share answering again: the three images sent, the JPEG's bytes as read then
NOT RUN F4 A Windows logoff or shutdown: not simulated; a real one ends the owner's session, this one included, and a second Windows account is not ours to create. What decides it was checked: a running dashboard has user32.dll and gdi32.dll loaded from its start (18 modules, 1.4 s after launch, before any clipboard or dialog use), so Windows treats it as a windowed program and sends it no logoff or shutdown console event; Windows ends it by closing its console, as a closed window does (below: ended by itself in 160 ms, exit code 0, the last word stored), or by ending the process (below: everything but what was typed in the last 500 ms)
Checks after these changes: build, vet, gofmt; the suite, 15 packages ok; the race detector on internal/input, internal/state and internal/app, 0 data races; the timing step alone, 7 of 7 within target (keystroke p95 1.0 ms, 2.3 ms in a 50,000-line draft with 8 saves); Windows Terminal, the final build, maximised and at 120x40: the window closed 187 and 171 ms after the last word (ended by itself in 160 ms, exit code 0, the last word stored), the process killed 184 and 177 ms after it (the four images and the text before it kept), the tab closed 187 and 174 ms after it (ended by itself in about 1.1 s, exit code 0), each restart restoring them; the owner's clipboard saved and restored in every run (7 formats)
The independent review's third pass, on F2's change: no loss found; its two findings fixed (F3, and F2's order and second paste); its note on a restore writing the stored draft again unchanged, fixed too
Not verified: a real network share and SMB's own time-outs (a held read and a failing read stand in); a real logoff or shutdown (F4)
```
