package render

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/state"
)

// Plan 72's command model and zone B's commands. The scenes are the approved
// frames' (plan.md, Plan 72, Reference frames), built from synthetic events,
// each field one the CLIs send.

// posixQuote is s in single quotes, a quote in it closed, escaped and opened
// again: one of the POSIX quotings Codex's command field uses (V1).
func posixQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'" }

// psExe is the launcher Codex runs a command with on Windows, as its
// command field writes it: double-quoted, its backslashes doubled.
const psExe = `"C:\\WINDOWS\\System32\\WindowsPowerShell\\v1.0\\powershell.exe"`

// todoScript is the reference frames' long command.
const todoScript = `$files = Get-ChildItem -Recurse -Filter *.go -File; $todos = $files | Select-String -Pattern 'TODO|FIXME' | Sort-Object Path, LineNumber; $todos | ForEach-Object { "- $($_.Path):$($_.LineNumber) $($_.Line.Trim())" } | Set-Content -LiteralPath 'TODO.md' -Encoding utf8; '{0} items in {1} files' -f $todos.Count, ($todos | Select-Object -Unique Path).Count`

// codexItem is a commandExecution item's JSON, Codex's call input.
func codexItem(t testing.TB, id, script string) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"type": "commandExecution", "id": id, "command": psExe + " -Command " + posixQuote(script), "cwd": `C:\work\acc-demo`, "status": "inProgress"})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// itemEv applies an event naming item, as Codex's review names its target.
func (b *convBench) itemEv(k event.Kind, item string, p any) {
	b.seq++
	b.s.Apply(event.Event{Seq: b.seq, At: lookAt.Add(time.Duration(b.seq) * time.Second), Kind: k, Item: item, Origin: event.Observed, Payload: p})
}

// codexScene is the frames' Codex session: two short commands, the long one
// waiting for the owner, then, with allow, allowed and completed; review
// puts Codex's auto-review on the second.
func codexScene(t testing.TB, allow, review bool) *convBench {
	b := newConvBench("codex", "gpt-5.6-sol")
	b.s.Context.Project = `C:\work\acc-demo`
	b.prompt("Collect the TODO and FIXME comments of the Go sources into TODO.md.")
	b.msg("m1", "I'll list the Go files, then write their TODO and FIXME comments to TODO.md.")
	for _, c := range []struct{ id, script string }{{"c1", "git status --short"}, {"c2", "Get-ChildItem -Recurse -Filter *.go -File | Measure-Object"}} {
		b.ev(event.ToolStarted, event.Tool{ToolID: c.id, Name: "commandExecution", Input: codexItem(t, c.id, c.script)})
		if review && c.id == "c2" {
			b.itemEv(event.ReviewUpdated, "c2", event.Review{ReviewID: "rv1", Status: "approved", Risk: "low", Authorization: "unknown", Rationale: "Counts the Go files of the project and reads nothing else; a read-only check of the working tree."})
		}
		b.ev(event.ToolCompleted, event.Tool{ToolID: c.id, Name: "commandExecution", Output: "ok"})
	}
	b.ev(event.ToolStarted, event.Tool{ToolID: "c3", Name: "commandExecution", Input: codexItem(t, "c3", todoScript)})
	req, _ := json.Marshal(map[string]any{"threadId": "th1", "turnId": "tu1", "itemId": "c3", "reason": "Create TODO.md from the TODO and FIXME comments of the Go files",
		"command": psExe + " -Command " + posixQuote(todoScript), "cwd": `C:\work\acc-demo`})
	b.ev(event.ApprovalRequest, event.Approval{ID: "r3", Tool: "commandExecution", ToolID: "c3", Input: req, Summary: "Create TODO.md from the TODO and FIXME comments of the Go files"})
	if allow {
		b.ev(event.ApprovalResolved, event.Approval{ID: "r3", Decision: event.Allow})
		b.ev(event.ToolCompleted, event.Tool{ToolID: "c3", Name: "commandExecution", Output: "14 items in 9 files", ExitCode: 0, HasExitCode: true})
	}
	return b
}

// claudeScene is the frames' Claude session: a failed go test, then an update
// denied by the owner, which Claude reports failed after the denial.
func claudeScene() *convBench {
	b := newConvBench("claude-code", "claude-opus-5-5")
	b.prompt("Run the tests, and push to origin if they pass.")
	b.call("t1", "Bash", map[string]string{"command": "go test ./...", "description": "Run the tests"}, false, false)
	b.ev(event.ToolCompleted, event.Tool{ToolID: "t1", IsError: true, Output: "--- FAIL: TestMarkdownCacheBytes (0.65s)\nFAIL\nFAIL\tgithub.com/x/render\t1.2s"})
	b.msg("m1", "TestMarkdownCacheBytes fails. I will update from origin and run the tests again.")
	cmd := "git stash && git pull --rebase origin main && go test ./..."
	b.call("t2", "Bash", map[string]string{"command": cmd, "description": "Update from origin and test again"}, false, false)
	in, _ := json.Marshal(map[string]string{"command": cmd, "description": "Update from origin and test again"})
	b.ev(event.ApprovalRequest, event.Approval{ID: "a2", Tool: "Bash", ToolID: "t2", Input: in, Summary: "Update from origin and test again"})
	b.ev(event.ApprovalResolved, event.Approval{ID: "a2", Decision: event.Deny})
	b.ev(event.ToolCompleted, event.Tool{ToolID: "t2", IsError: true, Output: "The user doesn't want to proceed with this tool use."})
	b.msg("m2", "The tests fail on TestMarkdownCacheBytes, and you denied the update: nothing was pushed.")
	b.end("completed")
	return b
}

// entryOf is the 1-based entry of call or request id.
func entryOf(s *state.State, role, id string) int {
	for i, e := range s.Conversation {
		if e.Role == role && e.MessageID == id {
			return i + 1
		}
	}
	return 0
}

// TestCommandScenesShow writes the scenes' rows, with -v, for a look.
func TestCommandScenesShow(t *testing.T) {
	if !testing.Verbose() {
		t.Skip("-v shows the scenes")
	}
	for _, w := range []int{74, 134} {
		for _, c := range []struct {
			name string
			b    *convBench
			ui   UI
		}{
			{"codex pending", codexScene(t, false, true), UI{}},
			{"codex allowed", codexScene(t, true, true), UI{}},
			{"codex allowed, unfolded", codexScene(t, true, true), UI{Unfolded: map[string]bool{CommandKey("c3"): true, CommandKey("c2"): true}}},
			{"claude", claudeScene(), UI{}},
		} {
			t.Logf("%s at %d:\n%s", c.name, w, strings.Join(rowTexts(bRows(c.b.s, c.ui, w)), "\n"))
		}
	}
	b := codexScene(t, true, true)
	t.Logf("full view:\n%s", texts(Draw(b.s, UI{Width: 120, Height: 40, ViewEntry: entryOf(b.s, "tool", "c3"), InspectHold: true}).lines))
}

// TestShellWords is D2's reading of an invocation: POSIX's quoting, and no
// word where a shell would read more than words.
func TestShellWords(t *testing.T) {
	for _, c := range []struct {
		in    string
		words []string
		ok    bool
	}{
		{"git status --short", []string{"git", "status", "--short"}, true},
		{"  a\t b  ", []string{"a", "b"}, true},
		{"'a b' c", []string{"a b", "c"}, true},
		{`"a \"b\" \\c \$d \e"`, []string{`a "b" \c $d \e`}, true},
		{`'it'"'"'s'`, []string{"it's"}, true},
		{`'$p = '"'found.txt'; "'$e'`, []string{`$p = 'found.txt'; $e`}, true},
		{`a\ b c\\d`, []string{"a b", `c\d`}, true},
		{"''", []string{""}, true},
		{"x='y z'", []string{"x=y z"}, true},
		{"é,ü", []string{"é,ü"}, true},
		{"a\\\nb", []string{"ab"}, true},
		{"'unclosed", nil, false},
		{`"unclosed`, nil, false},
		{`ok "$HOME"`, []string{"ok"}, false},
		{"\"`id`\"", nil, false},
		{"a;b", nil, false},
		{"a | b", []string{"a"}, false},
		{"a && b", []string{"a"}, false},
		{"*.go", nil, false},
		{"~/x", nil, false},
		{"a#b", nil, false},
		{"$x", nil, false},
		{`trailing\`, nil, false},
	} {
		words, ok := shellWords(c.in)
		if ok != c.ok || !slices.Equal(words, c.words) {
			t.Errorf("shellWords(%q) = %q, %v; want %q, %v", c.in, words, ok, c.words, c.ok)
		}
	}
}

// TestUnwrap is D1 and D2: a known launcher's script, exactly, else the
// invocation whole with its executable's shell.
func TestUnwrap(t *testing.T) {
	for _, c := range []struct{ in, shell, script string }{
		{psExe + ` -Command 'git status'`, "PowerShell", "git status"},
		{psExe + ` -Command "Get-Content -LiteralPath '..\\x.txt'"`, "PowerShell", `Get-Content -LiteralPath '..\x.txt'`},
		{`pwsh -NoProfile -NonInteractive -Command 'Get-Date'`, "PowerShell", "Get-Date"},
		{`powershell.exe -c "dir"`, "PowerShell", "dir"},
		{`/bin/bash -lc 'ls -la | wc -l'`, "bash", "ls -la | wc -l"},
		{`bash -c 'echo hi'`, "bash", "echo hi"},
		{`sh -c 'echo hi'`, "sh", "echo hi"},
		{`zsh -c 'echo hi'`, "zsh", "echo hi"},
		{`C:\\Windows\\system32\\cmd.exe /c "dir /b"`, "cmd", "dir /b"},
		{`cmd /C "type a.txt"`, "cmd", "type a.txt"},
		{`git status`, "", ""},
		{`powershell.exe -File x.ps1`, "PowerShell", ""},
		{`powershell.exe -ExecutionPolicy Bypass -Command 'x'`, "PowerShell", ""},
		{`powershell.exe -Command 'a' extra`, "PowerShell", ""},
		{`bash -x -c 'a'`, "bash", ""},
		{`bash -lc 'a' 'b'`, "bash", ""},
		{psExe + ` -Command "$x"`, "PowerShell", ""},
		{`"unclosed -Command 'x'`, "", ""},
	} {
		if shell, script := unwrap(c.in); shell != c.shell || script != c.script {
			t.Errorf("unwrap(%q) = %q, %q; want %q, %q", c.in, shell, script, c.shell, c.script)
		}
	}
	// What zone B shows: the shell's name, the script out of the launcher,
	// coloured by its shell's lexer; the invocation whole and plain else.
	for _, c := range []struct {
		tool, sent, shell, text, lexer string
		unwrapped                      bool
	}{
		{"Bash", "go test ./...", "Bash", "go test ./...", "bash", false},
		{"PowerShell", "Get-ChildItem", "PowerShell", "Get-ChildItem", "powershell", false},
		{"commandExecution", psExe + " -Command 'git status'", "PowerShell", "git status", "powershell", true},
		{"commandExecution", "/bin/zsh -lc 'ls'", "zsh", "ls", "bash", true},
		{"commandExecution", "cmd /c dir", "cmd", "dir", "batch", true},
		{"commandExecution", "git status", "command", "git status", "", false},
		{"commandExecution", psExe + ` -Command "$x"`, "PowerShell", psExe + ` -Command "$x"`, "", false},
		{"Read", "x", "", "", "", false},
	} {
		in, _ := json.Marshal(map[string]string{"command": c.sent})
		got := commandOf(c.tool, readCmd(in))
		if got.shell != c.shell || got.text != c.text || got.lexer != c.lexer || got.unwrapped != c.unwrapped || got.ok && got.sent != c.sent {
			t.Errorf("%s %q: %+v; want %s %q %s %v", c.tool, c.sent, got, c.shell, c.text, c.lexer, c.unwrapped)
		}
	}
}

// recordedCommands are the commands of the probes' recordings: Codex's
// command fields, each with Codex's own reading of its script when its
// commandActions give one apart from the invocation, and Claude's Bash and
// PowerShell commands (V1, V2).
func recordedCommands(t *testing.T) (codex map[string]string, claude []string) {
	t.Helper()
	codex = map[string]string{}
	seen := map[string]bool{}
	var walk func(v any)
	walk = func(v any) {
		switch v := v.(type) {
		case []any:
			for _, x := range v {
				walk(x)
			}
		case map[string]any:
			if c, ok := v["command"].(string); ok && (v["type"] == "commandExecution" || v["commandActions"] != nil || v["reason"] != nil) {
				if _, ok := codex[c]; !ok {
					codex[c] = ""
				}
				if as, ok := v["commandActions"].([]any); ok && len(as) == 1 {
					if a, ok := as[0].(map[string]any); ok {
						if s, ok := a["command"].(string); ok && s != c {
							codex[c] = s
						}
					}
				}
			}
			if name, ok := v["name"].(string); ok && (name == "Bash" || name == "PowerShell") {
				if in, ok := v["input"].(map[string]any); ok {
					if c, ok := in["command"].(string); ok && !seen[c] {
						seen[c] = true
						claude = append(claude, c)
					}
				}
			}
			for _, x := range v {
				walk(x)
			}
		}
	}
	for _, dir := range []string{filepath.Join("..", "..", "testdata", "probes"), filepath.Join("..", "..", "doc", "research", "probes")} {
		files, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range files {
			fh, err := os.Open(f)
			if err != nil {
				t.Fatal(err)
			}
			sc := bufio.NewScanner(fh)
			sc.Buffer(make([]byte, 1<<20), 64<<20)
			for sc.Scan() {
				var v any
				if json.Unmarshal(sc.Bytes(), &v) == nil {
					walk(v)
				}
			}
			fh.Close()
		}
	}
	return codex, claude
}

// TestCommandRecordings is V1 and criterion 72.2: every Codex invocation the
// recordings hold is a PowerShell launcher's, taken out of it exactly: where
// Codex's own commandActions read its script, the same script byte for byte.
func TestCommandRecordings(t *testing.T) {
	codex, claude := recordedCommands(t)
	if len(codex) < 20 {
		t.Fatalf("%d recorded Codex invocations, want 20 or more", len(codex))
	}
	compared := 0
	for sent, theirs := range codex {
		shell, script := unwrap(sent)
		if shell != "PowerShell" || script == "" {
			t.Errorf("%q: shell %q, script %q; want PowerShell's", sent, shell, script)
			continue
		}
		words, _ := shellWords(sent)
		if words[len(words)-1] != script || !strings.HasPrefix(sent, `"`) {
			t.Errorf("%q: script %q is not its last word", sent, script)
		}
		if theirs != "" {
			compared++
			if script != theirs {
				t.Errorf("%q: script %q, Codex's own reading %q", sent, script, theirs)
			}
		}
	}
	t.Logf("%d Codex invocations, %d compared with Codex's own reading; %d Claude commands", len(codex), compared, len(claude))
	if compared < 10 {
		t.Errorf("%d compared with Codex's reading, want 10 or more", compared)
	}
	if len(claude) == 0 {
		t.Error("no Claude command in the recordings")
	}
}

// checkLayout is criterion 72.3 on one command: its rows, in order, give it
// back with only blanks and its own line breaks between them; none wider
// than w but a lone unit; a break only after a top-level ; (cmd's &), before
// a top-level && || or |, at a blank, or at the width.
func checkLayout(t *testing.T, text, rules string, w int) {
	t.Helper()
	rows := cmdLayout(text, rules, w)
	if len(rows) == 0 {
		t.Fatalf("%q at %d: no row", text, w)
	}
	at := 0
	for k, r := range rows {
		gap := text[at:r.from]
		if strings.Trim(gap, " \t\n") != "" || strings.Count(gap, "\n") > 1 || r.to < r.from {
			t.Errorf("%q at %d: row %d %q after %q", text, w, k, text[r.from:r.to], gap)
			return
		}
		if c := r.ind + cellsOf(text, r.from, r.to); c > w {
			if n, _ := cmdCellSpan(text, r.from); r.from+n != r.to || r.ind >= w {
				t.Errorf("%q at %d: row %d %q is %d cells", text, w, k, text[r.from:r.to], c)
			}
		}
		if k > 0 && !strings.Contains(gap, "\n") {
			prev, next := text[rows[k-1].from:rows[k-1].to], text[r.from:r.to]
			_, unit := cmdCellSpan(text, r.from)
			ok := strings.HasSuffix(prev, ";") || rules == "cmd" && strings.HasSuffix(prev, "&") ||
				strings.HasPrefix(next, "&&") || strings.HasPrefix(next, "||") || strings.HasPrefix(next, "|") ||
				gap != "" || rows[k-1].ind+cellsOf(text, rows[k-1].from, rows[k-1].to)+unit > w // at the width
			if !ok {
				t.Errorf("%q at %d: a break between %q and %q", text, w, prev, next)
			}
		}
		at = r.to
	}
	if rest := text[at:]; strings.Trim(rest, " \t\n") != "" {
		t.Errorf("%q at %d: %q left after the rows", text, w, rest)
	}
}

// TestCmdLayout is D3 and criterion 72.3, on the recordings' commands and
// on commands made to break it, at the widths of zone B's blocks and cards
// at 80, 120, 200 and 208 columns, and narrower.
func TestCmdLayout(t *testing.T) {
	codex, claude := recordedCommands(t)
	type cmd struct{ text, rules string }
	var all []cmd
	for sent := range codex {
		_, script := unwrap(sent)
		all = append(all, cmd{script, "PowerShell"}, cmd{sent, ""})
	}
	for _, c := range claude {
		all = append(all, cmd{c, "bash"})
	}
	all = append(all,
		cmd{todoScript, "PowerShell"},
		cmd{"git stash && git pull --rebase origin main && go test ./...", "bash"},
		cmd{"cd src & dir /b & echo done", "cmd"},
		cmd{"echo 'a; b && c | d' \"e; f\" && (cd x; ls | wc -l) || echo $((1 | 2))", "bash"},
		cmd{"Write-Output `\"quoted`\"; Get-Date", "PowerShell"},
		cmd{"echo " + strings.Repeat("x", 300), "bash"},
		cmd{"printf 'a\\tb'\t|\tcat; echo  two  spaces", "bash"},
		cmd{"$s = @'\nline one; still one\n  indented | kept\n'@\nWrite-Output $s", "PowerShell"},
		cmd{"echo 日本語のテキスト; echo 👨\u200d👩\u200d👧 | cat", "bash"},
		cmd{"echo safe\u202etxt.exe\u200b; ls", "bash"},
		cmd{"a | b | c | d | e | f | g | h | i | j | k | l", "bash"},
		cmd{"", "bash"},
	)
	for _, c := range all {
		for _, w := range []int{8, 12, 20, 40, 67, 69, 71, 73, 74, 76, 78, 127, 130, 134, 136, 140} {
			checkLayout(t, c.text, c.rules, w)
		}
	}
	t.Logf("%d commands laid out at 16 widths", len(all))
	// The frames' layouts (plan.md, Plan 72, Reference frames), at zone B's
	// block width at 120x40.
	rows := cmdLayout(todoScript, "PowerShell", 67)
	var got []string
	for _, r := range rows {
		got = append(got, strings.Repeat(" ", r.ind)+todoScript[r.from:r.to])
	}
	want := []string{
		"$files = Get-ChildItem -Recurse -Filter *.go -File;",
		"$todos = $files | Select-String -Pattern 'TODO|FIXME'",
		"  | Sort-Object Path, LineNumber;",
		"$todos",
		"  | ForEach-Object",
		"      { \"- $($_.Path):$($_.LineNumber) $($_.Line.Trim())\" }",
		"  | Set-Content -LiteralPath 'TODO.md' -Encoding utf8;",
		"'{0} items in {1} files' -f $todos.Count,",
		"    ($todos | Select-Object -Unique Path).Count",
	}
	if !slices.Equal(got, want) {
		t.Errorf("the frames' command at 67:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	and := "git stash && git pull --rebase origin main && go test ./..."
	got = nil
	for _, r := range cmdLayout(and, "bash", 67) {
		got = append(got, strings.Repeat(" ", r.ind)+and[r.from:r.to])
	}
	if want := []string{"git stash", "  && git pull --rebase origin main", "  && go test ./..."}; !slices.Equal(got, want) {
		t.Errorf("the denied command: %q, want %q", got, want)
	}
}

// codexRows are the Codex scene's rows w cells wide, with ui.
func codexRows(t *testing.T, allow, review bool, ui UI, w int) (*convBench, []string) {
	b := codexScene(t, allow, review)
	return b, rowTexts(bRows(b.s, ui, w))
}

// hasRows says rows hold want, in order, one after the other.
func hasRows(rows, want []string) bool {
	for i := range rows {
		if len(rows)-i < len(want) {
			return false
		}
		if slices.Equal(rows[i:i+len(want)], want) {
			return true
		}
	}
	return false
}

// TestCommandBlocks is criteria 72.4, 72.5, 72.10 and 72.12 on the frames'
// scenes at 120x40 (zone B's conversation 74 cells wide), then at 200x60.
func TestCommandBlocks(t *testing.T) {
	_, rows := codexRows(t, true, true, UI{}, 74)
	frame := []string{
		"▎  ✓ PowerShell  git status --short                   11:30:05  observed",
		"▎  ▸ ✓ PowerShell           approved by auto-review · 11:30:08  observed",
		"▎     Get-ChildItem -Recurse -Filter *.go -File | Measure-Object",
		"▎     ◈ risk low: Counts the Go files of the project and reads nothing …",
		"▎  ▸ ✓ PowerShell                    allowed by you · 11:30:12  observed",
		"▎     $files = Get-ChildItem -Recurse -Filter *.go -File;",
		"▎     $todos = $files | Select-String -Pattern 'TODO|FIXME'",
		"▎     … 7 more lines",
	}
	if !hasRows(rows, frame) {
		t.Errorf("folded, 74 cells:\n%s\nwant\n%s", strings.Join(rows, "\n"), strings.Join(frame, "\n"))
	}
	open := UI{Unfolded: map[string]bool{CommandKey("c2"): true, CommandKey("c3"): true}}
	_, rows = codexRows(t, true, true, open, 74)
	frame = []string{
		"▎  ▾ ✓ PowerShell           approved by auto-review · 11:30:08  observed",
		"▎     Get-ChildItem -Recurse -Filter *.go -File | Measure-Object",
		"▎     ◈ risk low: Counts the Go files of the project and reads nothing",
		"▎       else; a read-only check of the working tree.",
		"▎  ▾ ✓ PowerShell                    allowed by you · 11:30:12  observed",
		"▎     $files = Get-ChildItem -Recurse -Filter *.go -File;",
		"▎     $todos = $files | Select-String -Pattern 'TODO|FIXME'",
		"▎       | Sort-Object Path, LineNumber;",
		"▎     $todos",
		"▎       | ForEach-Object",
		"▎           { \"- $($_.Path):$($_.LineNumber) $($_.Line.Trim())\" }",
		"▎       | Set-Content -LiteralPath 'TODO.md' -Encoding utf8;",
		"▎     '{0} items in {1} files' -f $todos.Count,",
		"▎         ($todos | Select-Object -Unique Path).Count",
	}
	if !hasRows(rows, frame) {
		t.Errorf("unfolded, 74 cells:\n%s\nwant\n%s", strings.Join(rows, "\n"), strings.Join(frame, "\n"))
	}
	// Pending: no row of the call while its card shows (D5).
	_, rows = codexRows(t, false, true, UI{}, 74)
	if all := strings.Join(rows, "\n"); strings.Contains(all, "waiting for you") || !strings.Contains(all, "! APPROVAL · orchestrator · PowerShell") {
		t.Errorf("pending:\n%s", all)
	}
	// Claude's frame: the error under its row; the denial on its row, no ✓,
	// no failure, no error, whatever Claude reports after it (D5, V5).
	c := claudeScene()
	rows = rowTexts(bRows(c.s, UI{}, 74))
	for _, want := range [][]string{
		{"▎  ✗ Bash  go test ./...                       failed 11:30:04  observed", "▎     --- FAIL: TestMarkdownCacheBytes (0.65s)"},
		{"▎  ✗ Bash                               denied by you 11:30:08  observed", "▎     git stash", "▎       && git pull --rebase origin main", "▎       && go test ./..."},
	} {
		if !hasRows(rows, want) {
			t.Errorf("Claude's frame lacks\n%s\nin\n%s", strings.Join(want, "\n"), strings.Join(rows, "\n"))
		}
	}
	if all := strings.Join(rows, "\n"); strings.Contains(all, "doesn't want") || strings.Contains(all, "orchestrator · Bash") {
		t.Errorf("a denial's output or a request's row:\n%s", all)
	}
	// At 200x60 the short commands fit their rows, the review's part too:
	// nothing hidden, no mark (72.4).
	_, rows = codexRows(t, true, true, UI{}, 134)
	if !hasRows(rows, []string{"▎  ✓ PowerShell  Get-ChildItem -Recurse -Filter *.go -File | Measure-Object             approved by auto-review · 11:30:08  observed",
		"▎     ◈ risk low: Counts the Go files of the project and reads nothing else; a read-only check of the working tree.",
		"▎  ▸ ✓ PowerShell                                                                                allowed by you · 11:30:12  observed"}) {
		t.Errorf("at 134:\n%s", strings.Join(rows, "\n"))
	}
	// The other cases of D5 and D6, each on its call's row.
	for _, c := range []struct {
		name string
		do   func(b *convBench)
		want string
	}{
		{"allowed for the session, running", func(b *convBench) {
			b.ev(event.ApprovalRequest, event.Approval{ID: "a", Tool: "Bash", ToolID: "x"})
			b.ev(event.ApprovalResolved, event.Approval{ID: "a", Decision: event.AllowSession})
			b.ev(event.ToolProgress, event.Tool{ToolID: "x", Output: "…"})
		}, "allowed for the session by you · running "},
		{"allowed, failed", func(b *convBench) {
			b.ev(event.ApprovalRequest, event.Approval{ID: "a", Tool: "Bash", ToolID: "x"})
			b.ev(event.ApprovalResolved, event.Approval{ID: "a", Decision: event.Allow})
			b.ev(event.ToolCompleted, event.Tool{ToolID: "x", IsError: true, Output: "boom"})
		}, "allowed by you · failed "},
		{"expired, not run", func(b *convBench) {
			b.ev(event.ApprovalRequest, event.Approval{ID: "a", Tool: "Bash", ToolID: "x"})
			b.ev(event.ApprovalResolved, event.Approval{ID: "a", Decision: event.Withdrawn})
		}, "expired, not answered  observed"},
		{"expired, then completed", func(b *convBench) {
			b.ev(event.ApprovalRequest, event.Approval{ID: "a", Tool: "Bash", ToolID: "x"})
			b.ev(event.ApprovalResolved, event.Approval{ID: "a", Decision: event.Withdrawn})
			b.ev(event.ToolCompleted, event.Tool{ToolID: "x", Output: "ok"})
		}, "expired, not answered · "},
		{"Claude's classifier, approved", func(b *convBench) {
			b.ev(event.ToolDecision, event.Verdict{ToolID: "x", ReasonType: "classifier", Decision: event.Allow})
		}, "approved by auto-review · running "},
		{"Claude's classifier, denied", func(b *convBench) {
			b.ev(event.ToolDecision, event.Verdict{ToolID: "x", ReasonType: "classifier", Decision: event.Deny})
			b.ev(event.ToolCompleted, event.Tool{ToolID: "x", IsError: true, Output: "denied"})
		}, "denied by auto-review "},
	} {
		b := newConvBench("claude-code", "claude-opus-5-5")
		b.prompt("go")
		b.call("x", "Bash", map[string]string{"command": "make all"}, false, false)
		c.do(b)
		all := strings.Join(rowTexts(bRows(b.s, UI{}, 74)), "\n")
		if !strings.Contains(all, c.want) || strings.Contains(all, "orchestrator · Bash") {
			t.Errorf("%s: no %q, or a request's row:\n%s", c.name, c.want, all)
		}
		if strings.HasSuffix(c.name, "denied") && (strings.Contains(all, "✓") || strings.Contains(all, "failed") || strings.Contains(all, "▎     denied")) {
			t.Errorf("%s: shown as run:\n%s", c.name, all)
		}
	}
	// Codex's reviews: denied, in progress, timed out; and one naming a
	// subagent's call, or no call, keeps its entry (D6).
	for _, c := range []struct {
		status, want string
	}{{"denied", "denied by auto-review "}, {"inProgress", "auto-review in progress · "}, {"timedOut", "auto-review timed out · "}} {
		b := newConvBench("codex", "gpt-5.6-sol")
		b.prompt("go")
		b.ev(event.ToolStarted, event.Tool{ToolID: "c1", Name: "commandExecution", Input: codexItem(t, "c1", "Remove-Item x")})
		b.itemEv(event.ReviewUpdated, "c1", event.Review{ReviewID: "rv", Status: c.status, Risk: "high", Authorization: "low", Rationale: "deletes a file"})
		all := strings.Join(rowTexts(bRows(b.s, UI{}, 74)), "\n")
		if !strings.Contains(all, c.want) || strings.Contains(all, "◈ auto-review") {
			t.Errorf("review %s: no %q, or its entry:\n%s", c.status, c.want, all)
		}
		if c.status != "inProgress" && !strings.Contains(all, "◈ risk high · authorization low: deletes a file") {
			t.Errorf("review %s: no rationale with its authorization:\n%s", c.status, all)
		}
	}
	b := newConvBench("codex", "gpt-5.6-sol")
	b.prompt("go")
	b.itemEv(event.ReviewUpdated, "nowhere", event.Review{ReviewID: "rv", Status: "approved", Risk: "low", Rationale: "fine"})
	if all := strings.Join(rowTexts(bRows(b.s, UI{}, 74)), "\n"); !strings.Contains(all, "◈ auto-review approved · risk low: fine") {
		t.Errorf("a review naming no call drawn: its entry gone:\n%s", all)
	}
}

// TestCommandCursor is V6 and K1: at each width, the entry cursor on a
// block, folded and unfolded, shows its hint on the block's row and leaves
// zone B's rows as many as without it; the hint names → or ←.
func TestCommandCursor(t *testing.T) {
	b := codexScene(t, true, true)
	n := entryOf(b.s, "tool", "c3")
	for _, w := range []int{74, 76, 134, 140} {
		for _, open := range []bool{false, true} {
			ui := UI{Focus: "B"}
			if open {
				ui.Unfolded = map[string]bool{CommandKey("c3"): true}
			}
			_, rows, _, _ := conversation(b.s, ui, w, 40)
			ui.Entry = n
			view, with, _, _ := conversation(b.s, ui, w, 1<<20)
			if with != rows {
				t.Errorf("%d, unfolded %v: %d rows with the cursor, %d without", w, open, with, rows)
			}
			hint := "→ unfold · Enter full view"
			if open {
				hint = "← fold · Enter full view"
			}
			found := false
			for _, c := range view {
				if r := rowText(c.draw()); strings.Contains(r, "PowerShell") && strings.Contains(r, hint) {
					found = true
				}
			}
			if !found {
				t.Errorf("%d, unfolded %v: no row with %q", w, open, hint)
			}
		}
	}
}

// TestCommandRequestsKept is criterion 72.10's other half: a subagent's
// request, a request naming no call and an unreadable one keep their rows.
func TestCommandRequestsKept(t *testing.T) {
	b := newConvBench("claude-code", "claude-opus-5-5")
	b.prompt("go")
	b.agent("ag1", false, false)
	in, _ := json.Marshal(map[string]string{"command": "ls"})
	b.ev(event.ToolStarted, event.Tool{ToolID: "sub", Name: "Bash", Input: in})
	b.s.Tools["sub"].Agent = "ag1" // a subagent's call: zone C's (V8)
	b.s.Conversation = slices.DeleteFunc(b.s.Conversation, func(e state.Entry) bool { return e.MessageID == "sub" })
	b.seq++
	b.s.Apply(event.Event{Seq: b.seq, At: lookAt.Add(time.Duration(b.seq) * time.Second), Kind: event.ApprovalRequest, Agent: "ag1", Origin: event.Observed, Payload: event.Approval{ID: "s1", Tool: "Bash", ToolID: "sub", Input: in}})
	b.ev(event.ApprovalResolved, event.Approval{ID: "s1", Decision: event.Allow})
	b.ev(event.ApprovalRequest, event.Approval{ID: "n1", Tool: "Bash", ToolID: "none", Input: in})
	b.ev(event.ApprovalResolved, event.Approval{ID: "n1", Decision: event.Deny})
	b.ev(event.ApprovalRequest, event.Approval{ID: "", Tool: "Bash", Problem: "unreadable request: x"})
	all := strings.Join(rowTexts(bRows(b.s, UI{}, 74)), "\n")
	for _, want := range [][]string{{"general-purpose · Bash", "ls", "allowed by you"}, {"orchestrator · Bash", "ls", "denied by you"}, {"orchestrator · Bash", "denied automatically: request not readable"}} {
		found := false
		for _, r := range strings.Split(all, "\n") {
			found = found || strings.Contains(r, want[0]) && strings.Contains(r, want[1]) && strings.Contains(r, want[len(want)-1])
		}
		if !found {
			t.Errorf("no row of %q:\n%s", want, all)
		}
	}
}

// TestCommandCounts is P1 and criterion 72.5: each entry's rows, counted
// from the cache, are the rows a fresh frame draws, through what changes a
// block: a fold, a call's end and failure, a review, a decision, a longer
// name in its group, a resize.
func TestCommandCounts(t *testing.T) {
	b := codexScene(t, false, false)
	ui := UI{Wrap: NewWrapCache(), Memo: NewMemo()}
	check := func(step string) {
		t.Helper()
		// Each step from the width the one before ended at: the cache holds
		// its counts there, so a key that misses a change shows.
		for _, w := range []int{74, 76, 134, 140, 74} {
			lk, _ := lookAll(b.s, ui)
			starts, rows := convStarts(b.s, ui, w, 40, lk)
			fresh, _, _, _ := conversation(b.s, UI{Unfolded: ui.Unfolded}, w, 1<<20)
			var n int
			for _, c := range fresh {
				if c.solid() || true {
					n++
				}
			}
			if rows != n-len(sessionState(b.s, UI{}, w)) {
				t.Errorf("%s at %d: %d rows counted, %d drawn", step, w, rows, n)
			}
			for i := range b.s.Conversation {
				drawn := len(entryRows(b.s, UI{Unfolded: ui.Unfolded}, nil, i, b.s.Conversation[i], w, lk[i]))
				if got := starts[i+1] - starts[i]; got != drawn {
					t.Errorf("%s at %d: entry %d counted %d, drawn %d", step, w, i, got, drawn)
				}
			}
			ui.Wrap.sweep()
			ui.Memo.sweep()
		}
	}
	check("pending")
	b.ev(event.ApprovalResolved, event.Approval{ID: "r3", Decision: event.Allow})
	check("allowed")
	ui.Unfolded = map[string]bool{CommandKey("c3"): true}
	check("unfolded")
	b.ev(event.ToolCompleted, event.Tool{ToolID: "c3", Name: "commandExecution", IsError: true, Output: "a long failure that will not fit on one row of the block, so that its row is cut and its block can unfold it", ExitCode: 1, HasExitCode: true})
	check("failed, unfolded")
	ui.Unfolded = nil
	check("failed, folded")
	// Output after the failure, which the reducer keeps: the error's rows
	// change, the call's status does not.
	ui.Unfolded = map[string]bool{CommandKey("c3"): true}
	check("failed, unfolded again")
	b.ev(event.ToolProgress, event.Tool{ToolID: "c3", Output: "\n" + strings.Repeat("a late line of output, long enough to wrap the error over more rows ", 4)})
	check("a late output")
	ui.Unfolded = nil
	b.itemEv(event.ReviewUpdated, "c1", event.Review{ReviewID: "rv", Status: "approved", Risk: "low", Rationale: strings.Repeat("a rationale long enough to be cut ", 4)})
	check("reviewed")
	b.ev(event.ToolStarted, event.Tool{ToolID: "c4", Name: "aVeryLongToolNameIndeed", Input: json.RawMessage(`{}`)})
	check("a longer name")
}

// TestCommandCard is D7 and criterion 72.9: the command laid out, whole up to
// seven lines, else six and a counter; no target row; the context the CLI
// sent, each part only when sent, the folder only outside the project; the
// pill.
func TestCommandCard(t *testing.T) {
	card := func(b *convBench, w int) string {
		rows := rowTexts(bRows(b.s, UI{}, w))
		for i, r := range rows {
			if strings.Contains(r, "! APPROVAL") {
				for j := i; j < len(rows); j++ {
					if strings.Contains(rows[j], "▎ ╰") {
						return strings.Join(rows[i:j+1], "\n")
					}
				}
			}
		}
		return ""
	}
	request := func(lines int, ctx map[string]string) *convBench {
		b := newConvBench("codex", "gpt-5.6-sol")
		b.s.Context.Project = `C:\work\acc-demo`
		b.prompt("go")
		var ls []string
		for i := 1; i <= lines; i++ {
			ls = append(ls, fmt.Sprintf("Write-Output %d", i))
		}
		script := strings.Join(ls, "\n")
		b.ev(event.ToolStarted, event.Tool{ToolID: "c1", Name: "commandExecution", Input: codexItem(t, "c1", script)})
		p := map[string]any{"itemId": "c1", "command": psExe + " -Command " + posixQuote(script)}
		for k, v := range ctx {
			p[k] = v
		}
		raw, _ := json.Marshal(p)
		b.ev(event.ApprovalRequest, event.Approval{ID: "r1", Tool: "commandExecution", ToolID: "c1", Input: raw, Summary: ctx["reason"]})
		return b
	}
	for _, w := range []int{74, 134} {
		seven := card(request(7, nil), w)
		if !strings.Contains(seven, "Write-Output 7") || strings.Contains(seven, "more lines") || strings.Contains(seven, "powershell.exe") {
			t.Errorf("seven lines at %d:\n%s", w, seven)
		}
		eight := card(request(8, map[string]string{"reason": "Prints eight numbers", "cwd": `C:\work\other`}), w)
		for _, want := range []string{"Write-Output 6", "… 2 more lines", "reason  Prints eight numbers", `in      C:\work\other`, "▐ F2 to answer ▌"} {
			if !strings.Contains(eight, want) {
				t.Errorf("eight lines at %d: no %q:\n%s", w, want, eight)
			}
		}
		if strings.Contains(eight, "Write-Output 7") {
			t.Errorf("eight lines at %d: a seventh shown:\n%s", w, eight)
		}
		inside := card(request(1, map[string]string{"cwd": `C:\work\acc-demo`}), w)
		if strings.Contains(inside, " in ") || strings.Contains(inside, "reason") {
			t.Errorf("the project's folder or an unsent reason at %d:\n%s", w, inside)
		}
		if nested := w >= cardContent(cardAI, nestFrom); nested != strings.Contains(eight, "│ ╭") {
			t.Errorf("at %d: the grey box %v", w, !nested)
		}
	}
	// Claude's description, under its own word.
	b := newConvBench("claude-code", "claude-opus-5-5")
	b.prompt("go")
	b.call("t1", "Bash", map[string]string{"command": "go test ./...", "description": "Run the tests"}, false, false)
	in, _ := json.Marshal(map[string]string{"command": "go test ./...", "description": "Run the tests"})
	b.ev(event.ApprovalRequest, event.Approval{ID: "a1", Tool: "Bash", ToolID: "t1", Input: in, Summary: "Run the tests"})
	if c := card(b, 74); !strings.Contains(c, "description  Run the tests") || !strings.Contains(c, "▎ │   go test ./...") || strings.Count(c, "go test") != 1 {
		t.Errorf("Claude's card:\n%s", c)
	}
}

// entryFull is the full view of entry n, w by h, and the screen drawn.
func entryFull(s *state.State, ui UI, n int) (*Screen, string) {
	ui.ViewEntry, ui.InspectHold = n, true
	if ui.Width == 0 {
		ui.Width, ui.Height = 120, 60
	}
	sc := Draw(s, ui)
	return sc, texts(sc.lines)
}

// TestCommandView is D9 and criterion 72.11: the full view's head and
// sections for each case of 72.10, a denied call "not run".
func TestCommandView(t *testing.T) {
	b := codexScene(t, true, true)
	_, v := entryFull(b.s, UI{}, entryOf(b.s, "tool", "c3"))
	row := func(label, text string) string { return fmt.Sprintf("%-*s%s", histCol, "  "+label, text) }
	for _, want := range []string{"✓ PowerShell  completed · 3s · exit 0", "COMMAND, laid out for reading", "AS SENT, verbatim (Codex's command field)",
		`IN         C:\work\acc-demo`, "APPROVAL", row("requested", "11:30:10  by the orchestrator"), row("reason", "Create TODO.md"), row("decision", "11:30:11  allowed by you, once"),
		row("ran", "11:30:11 to 11:30:12"), "OUTPUT", "14 items in 9 files", "y copy the command · "} {
		if !strings.Contains(v, want) {
			t.Errorf("Codex's view: no %q:\n%s", want, v)
		}
	}
	_, v = entryFull(b.s, UI{}, entryOf(b.s, "tool", "c2"))
	for _, want := range []string{"auto-review approved · risk low: Counts the Go files", "observed from Codex's item/autoApprovalReview, a notification marked unstable in its schema"} {
		if !strings.Contains(v, want) {
			t.Errorf("the reviewed call's view: no %q:\n%s", want, v)
		}
	}
	c := claudeScene()
	_, v = entryFull(c.s, UI{}, entryOf(c.s, "tool", "t2"))
	for _, want := range []string{"✗ Bash  denied · ", row("decision", clockSec(c.s.Approvals["a2"].AnsweredAt)+"  denied by you"), row("ran", "not run"), row("description", "Update from origin and test again")} {
		if !strings.Contains(v, want) {
			t.Errorf("Claude's denied view: no %q:\n%s", want, v)
		}
	}
	if strings.Contains(v, "AS SENT") || strings.Contains(v, "exit ") || strings.Contains(v, "IN    ") {
		t.Errorf("Claude's denied view: AS SENT, an exit code or a folder:\n%s", v)
	}
	// Claude's classifier, its note of an undocumented field.
	k := newConvBench("claude-code", "claude-opus-5-5")
	k.prompt("go")
	k.call("x", "Bash", map[string]string{"command": "make"}, true, false)
	k.ev(event.ToolDecision, event.Verdict{ToolID: "x", ReasonType: "classifier", Decision: event.Allow})
	if _, v := entryFull(k.s, UI{}, entryOf(k.s, "tool", "x")); !strings.Contains(v, "approved by auto-review, Claude's classifier") || !strings.Contains(v, classifierNote) {
		t.Errorf("the classifier's view:\n%s", v)
	}
}

// selectSection selects in sc every row of the section headed by head, from
// the first unit of its first row to the end of its last, and gives what
// the copy takes.
func selectSection(t *testing.T, sc *Screen, s *state.State, head string) string {
	t.Helper()
	y0, y1 := -1, -1
	for y, l := range sc.lines {
		txt := texts([]line{l})
		switch {
		case y0 < 0 && strings.Contains(txt, head):
			y0 = y + 1
		case y0 >= 0 && y1 < 0 && strings.TrimSpace(txt) == "":
			y1 = y - 1
		}
	}
	if y0 < 0 || y1 < y0 {
		t.Fatalf("no section %q", head)
	}
	from, _, ok := sc.HitIn('I', 0, y0, false)
	_, to, ok2 := sc.HitIn('I', 1<<20, y1, true)
	if !ok || !ok2 {
		t.Fatalf("section %q: no content", head)
	}
	return sc.Text(s, Selection{From: from, To: to})
}

// TestCommandCopy is C2 and criterion 72.13: the full view's COMMAND copies
// the command's own text and AS SENT the invocation, byte for byte, its
// wraps and indents in neither; a word or a line selected in them, theirs;
// another full view copies its rows as before.
func TestCommandCopy(t *testing.T) {
	script := "$a = 1;\t$b  =  2; Get-ChildItem -Recurse | Where-Object { $_.Length -gt 1MB } | Select-Object Name, Length | Sort-Object Length -Descending; Write-Output \"done\"\n\tWrite-Output 'second line'"
	b := newConvBench("codex", "gpt-5.6-sol")
	b.prompt("go")
	b.ev(event.ToolStarted, event.Tool{ToolID: "c1", Name: "commandExecution", Input: codexItem(t, "c1", script)})
	b.ev(event.ToolCompleted, event.Tool{ToolID: "c1", Name: "commandExecution", Output: "ok"})
	n := entryOf(b.s, "tool", "c1")
	for _, w := range []int{80, 120, 200} {
		sc, v := entryFull(b.s, UI{Width: w, Height: 60}, n)
		if got := selectSection(t, sc, b.s, "COMMAND,"); got != script {
			t.Errorf("COMMAND at %d copies %q, want %q:\n%s", w, got, script, v)
		}
		sent := psExe + " -Command " + posixQuote(script)
		if got := selectSection(t, sc, b.s, "AS SENT,"); got != sent {
			t.Errorf("AS SENT at %d copies %q, want %q", w, got, sent)
		}
		if got, ok := CommandSent(b.s, UI{ViewEntry: n}); !ok || got != sent {
			t.Errorf("CommandSent: %q, %v", got, ok)
		}
		// A row's line selected (LineText) is the source the offsets are in.
		for y, l := range sc.lines {
			if strings.Contains(texts([]line{l}), "Where-Object") {
				p, _, _ := sc.HitIn('I', 4, y, false)
				if got := sc.LineText(b.s, p); got != script {
					t.Errorf("LineText at %d: %q", w, got)
				}
				break
			}
		}
	}
	// A message's full view copies its rows as before: wrapped rows joined
	// by line breaks.
	m := newConvBench("claude-code", "claude-opus-5-5")
	m.prompt(strings.Repeat("word ", 40))
	sc, _ := entryFull(m.s, UI{Width: 80, Height: 30}, 1)
	from, _, _ := sc.HitIn('I', 0, 3, false)
	_, to, _ := sc.HitIn('I', 1<<20, 6, true)
	if got := sc.Text(m.s, Selection{From: from, To: to}); !strings.Contains(got, "\n") || strings.Contains(got, "\n\n") {
		t.Errorf("a message's view copies %q", got)
	}
}

// TestCommandZoneBCopy is C1: a selection across zone B's blocks and cards
// copies the conversation as before, no command line in it.
func TestCommandZoneBCopy(t *testing.T) {
	for _, b := range []*convBench{codexScene(t, true, true), codexScene(t, false, true), claudeScene()} {
		sc := Draw(b.s, UI{Width: 120, Height: 60, Focus: "B"})
		top, bottom := -1, -1
		for y, l := range sc.lines {
			for _, x := range l {
				if x.ref.panel == 'B' && (x.ref.kind == refText || x.ref.kind == refAtom) {
					if top < 0 {
						top = y
					}
					bottom = y
				}
				if x.ref.panel == 'B' && (x.ref.kind == refText || x.ref.kind == refAtom) && x.ref.line == 0 && strings.Contains(x.text, "Get-ChildItem") {
					t.Errorf("a command drawn as selectable content: %q", x.text)
				}
			}
		}
		from, _, _ := sc.HitIn('B', 0, top, false)
		_, to, _ := sc.HitIn('B', 1<<20, bottom, true)
		got := sc.Text(b.s, Selection{From: from, To: to})
		for _, not := range []string{"Get-ChildItem", "git status --short", "git stash", "powershell.exe"} {
			if strings.Contains(got, not) {
				t.Errorf("the copy holds a command, %q:\n%s", not, got)
			}
		}
		if !strings.Contains(got, "PowerShell") && !strings.Contains(got, "Bash") {
			t.Errorf("the copy lacks a call's name:\n%s", got)
		}
	}
}

// TestCommandSearch is C4 and criterion 72.14: a command is found as its
// call's target, out of Codex's launcher; a match in what a folded block
// hides unfolds it, and a card's hidden lines light its counter row.
func TestCommandSearch(t *testing.T) {
	b := codexScene(t, true, true)
	ms, _ := ConvMatches(b.s, UI{}, "Set-Content", nil)
	if len(ms) != 1 || !ms[0].Target || ms[0].Item != entryOf(b.s, "tool", "c3")-1 {
		t.Fatalf("Set-Content: %+v", ms)
	}
	if got, _ := ConvMatches(b.s, UI{}, "powershell.exe", nil); len(got) != 0 {
		t.Errorf("the launcher's path matched: %+v", got)
	}
	ui := UI{}
	if keys := MatchFolds(b.s, ui, 74, ms[0]); !slices.Equal(keys, []string{CommandKey("c3")}) {
		t.Errorf("a hidden match's folds: %q", keys)
	}
	vis, _ := ConvMatches(b.s, UI{}, "$files", nil)
	if keys := MatchFolds(b.s, ui, 74, vis[0]); len(keys) != 0 {
		t.Errorf("a shown match unfolds %q", keys)
	}
	ui.Unfolded = map[string]bool{CommandKey("c3"): true}
	row, _, ok := ConvMatchRow(b.s, ui, 74, 40, ms[0])
	rows := rowTexts(bRows(b.s, ui, 74))
	if !ok || !strings.Contains(rows[row], "Set-Content") {
		t.Errorf("the match's row %d: %q", row, rows[min(row, len(rows)-1)])
	}
	// The review's part: searched on its call, its entry not twice.
	rv, _ := ConvMatches(b.s, UI{}, "read-only check", nil)
	if len(rv) != 1 || rv[0].Line != 2 || !rv[0].Target {
		t.Fatalf("the rationale: %+v", rv)
	}
	if keys := MatchFolds(b.s, UI{}, 74, rv[0]); !slices.Equal(keys, []string{CommandKey("c2")}) {
		t.Errorf("a cut rationale's match unfolds %q", keys)
	}
	// The card's hidden lines: its counter row lit.
	p := codexScene(t, false, true)
	pm, _ := ConvMatches(p.s, UI{}, "items in", nil)
	if len(pm) != 1 {
		t.Fatalf("the card's match: %+v", pm)
	}
	sc := Draw(p.s, UI{Width: 120, Height: 60, Search: SearchView{Open: true, Query: "items in", Matches: pm, Cur: 0}})
	lit := false
	for _, l := range sc.lines {
		for _, x := range l {
			if strings.Contains(x.text, "more lines") && x.st == stMatchCur {
				lit = true
			}
		}
	}
	if !lit {
		t.Errorf("the card's counter row not lit:\n%s", texts(sc.lines))
	}
}

// TestCommandColours is D8, V2, V3 and criterion 72.15: Chroma's tokens of
// every recorded command rejoin into it; the classes are the file viewer's;
// a block, a card and a full view draw the same text with and without
// colours; on 16 colours, plain.
func TestCommandColours(t *testing.T) {
	codex, claude := recordedCommands(t)
	first := map[string]time.Duration{}
	coloured := 0 // the spans in a class other than the text's
	check := func(lexer, text string) {
		start := time.Now()
		c := ColourCommand(CmdWant{lexer, text})
		if _, ok := first[lexer]; !ok {
			first[lexer] = time.Since(start)
		}
		if !c.ok {
			t.Errorf("%s: %q: the tokens do not rejoin", lexer, text)
		}
		coloured += len(c.spans)
		at := 0
		for _, s := range c.spans {
			if s.from < at || s.to <= s.from || s.to > len(text) || s.class == SynText {
				t.Errorf("%s: %q: span %+v", lexer, text, s)
			}
			at = s.to
		}
	}
	for sent := range codex {
		_, script := unwrap(sent)
		check("powershell", script)
	}
	for _, c := range claude {
		check("bash", c)
	}
	check("powershell", todoScript)
	check("batch", "dir /b & echo done")
	t.Logf("first tokenisation: %v; %d coloured spans", first, coloured)
	if coloured == 0 {
		t.Error("no recorded command coloured")
	}
	if c := ColourCommand(CmdWant{"no-such-lexer", "x"}); len(c.spans) != 0 || c.ok {
		t.Errorf("an unknown lexer coloured: %+v", c)
	}
	// Drawn with its colours, the same text and rows.
	b := codexScene(t, false, true)
	colors := NewCmdColors()
	plainUI := UI{Colors: NewCmdColors()}
	rowsPlain := rowTexts(bRows(b.s, plainUI, 74))
	ws := plainUI.Colors.Wants()
	if len(ws) == 0 {
		t.Fatal("a frame drew commands plain and asked for none")
	}
	for _, w := range colors.Ask(ws) {
		colors.Put(ColourCommand(w))
	}
	ui := UI{Colors: colors}
	ls := bRows(b.s, ui, 74)
	if got := rowTexts(ls); !slices.Equal(got, rowsPlain) {
		t.Errorf("colours changed the rows:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(rowsPlain, "\n"))
	}
	drawn := false
	for _, l := range ls {
		for _, x := range l {
			if strings.Contains(x.text, "Get-ChildItem") && x.st != stText && x.st != tint(stMuted) && x.st != stMuted {
				drawn = true
			}
		}
	}
	if !drawn {
		t.Error("no command coloured once its colours came")
	}
	// Each span of a call's block in its own token's class: the colours are
	// the script's, the text drawn, never the invocation's.
	scripts := map[string]string{"c1": "git status --short", "c2": "Get-ChildItem -Recurse -Filter *.go -File | Measure-Object"}
	checked := 0
	for _, l := range ls {
		for _, x := range l {
			if x.ref.kind != refTarget || x.ref.line != 0 || x.text == "" || b.s.Conversation[x.ref.item].Role != "tool" {
				continue
			}
			script := scripts[b.s.Conversation[x.ref.item].MessageID]
			if !strings.HasPrefix(script[x.ref.off:], x.text) {
				t.Fatalf("%q at %d is not the script's %q", x.text, x.ref.off, script)
			}
			want := stMuted
			for _, s := range ColourCommand(CmdWant{"powershell", script}).spans {
				if s.from <= x.ref.off && x.ref.off < s.to {
					want = s.class.style()
				}
			}
			if x.st != tint(want) { // on the tool rows' ground
				t.Errorf("%q at %d of %q: not its token's class", x.text, x.ref.off, script)
			}
			checked++
		}
	}
	if checked == 0 {
		t.Error("no block's command span checked")
	}
	if len(colors.Wants()) != 0 {
		t.Error("a coloured frame asked again")
	}
	UsePalette(pickANSI)
	for _, l := range bRows(b.s, ui, 74) {
		for _, x := range l {
			if x.ref.kind == refTarget && x.st != stText && x.st != stMuted {
				t.Errorf("16 colours: %q coloured", x.text)
			}
		}
	}
	UsePalette(pickTrue)
}

// TestCmdColorsStore is P3's store: asked once, wanted when drawn plain,
// kept, bounded.
func TestCmdColorsStore(t *testing.T) {
	c := NewCmdColors()
	w := CmdWant{"bash", "ls"}
	if got := c.Ask([]CmdWant{w, w, {"", "plain"}}); len(got) != 1 {
		t.Errorf("asked %v", got)
	}
	if _, ok := c.colours(w); ok || len(c.Wants()) != 0 {
		t.Error("a command asked for is wanted again")
	}
	c.Put(Coloured{want: w, spans: []hlSpan{{0, 2, SynBuiltin}}})
	if sp, ok := c.colours(w); !ok || len(sp) != 1 {
		t.Errorf("kept %v %v", sp, ok)
	}
	if _, ok := c.colours(CmdWant{"bash", "pwd"}); ok || len(c.Wants()) != 1 {
		t.Error("a miss not wanted")
	}
	for i := range 3 * maxCmdColors {
		c.Put(Coloured{want: CmdWant{"bash", fmt.Sprint(i)}})
	}
	if n := len(c.got) + len(c.old); n > maxCmdColors {
		t.Errorf("%d kept, the bound %d", n, maxCmdColors)
	}
}

// brokenLexer gives a token the text does not hold before the text itself:
// a stream that does not rejoin its source, which no lexer of Chroma's
// gives, for the rejoin check (D8).
type brokenLexer struct{}

func (brokenLexer) Config() *chroma.Config { return &chroma.Config{Name: "p72-broken"} }
func (brokenLexer) Tokenise(_ *chroma.TokeniseOptions, text string) (chroma.Iterator, error) {
	return chroma.Literator(chroma.Token{Type: chroma.Keyword, Value: "x"}, chroma.Token{Type: chroma.Text, Value: text}), nil
}
func (l brokenLexer) SetRegistry(*chroma.LexerRegistry) chroma.Lexer { return l }
func (l brokenLexer) SetAnalyser(func(string) float32) chroma.Lexer  { return l }
func (brokenLexer) AnalyseText(string) float32                       { return 0 }

// TestCommandColourRejoin is D8's rejoin check: tokens that do not give the
// text back byte for byte, a stray one among them, leave the command plain.
func TestCommandColourRejoin(t *testing.T) {
	lexers.Register(brokenLexer{})
	if c := ColourCommand(CmdWant{"p72-broken", "abc"}); c.ok || len(c.spans) != 0 {
		t.Errorf("a stream that does not rejoin coloured: %+v", c)
	}
}

// TestCommandP8Card is criterion 72.1's card: plan 8's live command request
// (plan8-evaluation.md, 4c: note.txt "one"; no recording of it is in the
// repository), its shape P12x's recorded request's, reads
// "! APPROVAL · orchestrator · PowerShell" and its command out of the
// launcher.
func TestCommandP8Card(t *testing.T) {
	b := newConvBench("codex", "gpt-5.6-sol")
	b.prompt("Append one to note.txt.")
	sent := psExe + ` -Command "Add-Content -Path note.txt -Value one"`
	item, _ := json.Marshal(map[string]any{"type": "commandExecution", "id": "c8", "command": sent, "status": "inProgress"})
	b.ev(event.ToolStarted, event.Tool{ToolID: "c8", Name: "commandExecution", Input: item})
	req, _ := json.Marshal(map[string]any{"kind": "command", "itemId": "c8", "command": sent, "commandActions": []any{map[string]any{"type": "unknown", "command": "Add-Content -Path note.txt -Value one"}}})
	b.ev(event.ApprovalRequest, event.Approval{ID: "r8", Tool: "commandExecution", ToolID: "c8", Input: req, Summary: sent})
	for _, w := range []int{74, 134} {
		rows := strings.Join(rowTexts(bRows(b.s, UI{}, w)), "\n")
		if !strings.Contains(rows, "! APPROVAL · orchestrator · PowerShell") || !strings.Contains(rows, "  Add-Content -Path note.txt -Value one") || strings.Contains(rows, "powershell.exe") {
			t.Errorf("at %d:\n%s", w, rows)
		}
	}
}

// TestCommandMemo is P2 and criterion 72.18's memos: frames drawn again read
// no input again, lay a command out once a width, and keep each command's
// facts, its layouts with them, between frames.
func TestCommandMemo(t *testing.T) {
	b := codexScene(t, true, true)
	ui := UI{Width: 120, Height: 40, Focus: "B", Wrap: NewWrapCache(), Memo: NewMemo(), Unfolded: map[string]bool{CommandKey("c3"): true}}
	Draw(b.s, ui)
	misses := ui.Memo.misses
	f := ui.Memo.command("commandExecution", b.s.Tools["c3"].Input)
	lays := len(f.lays)
	for range 3 {
		Draw(b.s, ui)
	}
	if ui.Memo.misses != misses {
		t.Errorf("inputs read again: %d misses, then %d", misses, ui.Memo.misses)
	}
	if g := ui.Memo.command("commandExecution", b.s.Tools["c3"].Input); g != f || len(g.lays) != lays || lays != 1 {
		t.Errorf("the command's facts: kept %v, %d layouts then %d, want 1", g == f, lays, len(g.lays))
	}
	ui.Width = 200
	Draw(b.s, ui)
	if len(f.lays) != 2 {
		t.Errorf("a resize: %d layouts, want the two widths'", len(f.lays))
	}
}

// TestCommandQuestionKept is D5 as built: an answered question keeps its
// row, its answers under it, its call drawn too and left out only while its
// card shows.
func TestCommandQuestionKept(t *testing.T) {
	b := newConvBench("claude-code", "claude-opus-5-5")
	b.prompt("go")
	qin := json.RawMessage(`{"questions":[{"question":"Which one?","header":"Pick","options":[{"label":"this","description":"the first"},{"label":"that","description":"the second"}]}]}`)
	b.ev(event.ToolStarted, event.Tool{ToolID: "q1", Name: "AskUserQuestion", Input: qin})
	b.ev(event.Question, event.Approval{ID: "aq", Tool: "AskUserQuestion", ToolID: "q1", Input: qin})
	if rows := strings.Join(rowTexts(bRows(b.s, UI{}, 74)), "\n"); !strings.Contains(rows, "? QUESTION") || strings.Contains(rows, "AskUserQuestion  ") && strings.Contains(rows, "waiting for you") {
		t.Errorf("the question waiting: its card, no row of its call:\n%s", rows)
	}
	ap := b.s.Approvals["aq"]
	ap.Status, ap.Answers, ap.AnsweredAt = state.Allowed, map[string]string{"Which one?": "this"}, lookAt
	rows := strings.Join(rowTexts(bRows(b.s, UI{}, 74)), "\n")
	if !strings.Contains(rows, "orchestrator · AskUserQuestion") || !strings.Contains(rows, "answered by you") || !strings.Contains(rows, "Which one? → this") {
		t.Errorf("the question answered: its row and its answers gone:\n%s", rows)
	}
}

// TestCommandStaysPut is D4's room: a command keeps to its row by the room
// its decision's longest outcome leaves, not by the outcome drawn, so that
// it neither moves while the call runs and ends nor is cut on its row: at 74
// cells, one of 24 under its row allowed (17 cells left), on it undecided
// (34).
func TestCommandStaysPut(t *testing.T) {
	cmd := "go test -run TestLay ./x"
	for _, allow := range []bool{false, true} {
		b := newConvBench("claude-code", "claude-opus-5-5")
		b.prompt("go")
		b.call("x", "Bash", map[string]string{"command": cmd}, false, false)
		if allow {
			in, _ := json.Marshal(map[string]string{"command": cmd})
			b.ev(event.ApprovalRequest, event.Approval{ID: "a", Tool: "Bash", ToolID: "x", Input: in})
			b.ev(event.ApprovalResolved, event.Approval{ID: "a", Decision: event.Allow})
		}
		under := func(when string) bool {
			t.Helper()
			for _, r := range rowTexts(bRows(b.s, UI{}, 74)) {
				if strings.Contains(r, cmd) {
					if strings.Contains(r, "…") {
						t.Errorf("allowed %v, %s: the command cut: %q", allow, when, r)
					}
					return !strings.Contains(r, "Bash")
				}
			}
			t.Fatalf("allowed %v, %s: no row of the command", allow, when)
			return false
		}
		running := under("running")
		b.ev(event.ToolCompleted, event.Tool{ToolID: "x", Output: "ok"})
		if done := under("completed"); done != running || done != allow {
			t.Errorf("allowed %v: under its row running %v, completed %v; want %v both", allow, running, done, allow)
		}
	}
}

// TestCommandReviewOfFile is D6 as built: Codex's review of an item that
// runs no command, a file change drawn in zone B, keeps its ◈ entry, and
// the call's row no verdict.
func TestCommandReviewOfFile(t *testing.T) {
	b := newConvBench("codex", "gpt-5.6-sol")
	b.prompt("go")
	in, _ := json.Marshal(map[string]any{"type": "fileChange", "id": "f1", "changes": []any{}})
	b.ev(event.ToolStarted, event.Tool{ToolID: "f1", Name: "fileChange", Input: in})
	b.itemEv(event.ReviewUpdated, "f1", event.Review{ReviewID: "rv", Status: "approved", Risk: "low", Rationale: "writes one file"})
	rows := strings.Join(rowTexts(bRows(b.s, UI{}, 74)), "\n")
	if !strings.Contains(rows, "◈ auto-review approved · risk low: writes one file") || strings.Contains(rows, "approved by auto-review") {
		t.Errorf("a file change's review:\n%s", rows)
	}
}

// TestCommandAnswering is D5 while the answer is sent: the card, "sending
// answer…", is still the call's, its row left out until the answer lands.
func TestCommandAnswering(t *testing.T) {
	b := codexScene(t, false, false)
	b.s.Approvals["r3"].Status = state.Answering
	rows := strings.Join(rowTexts(bRows(b.s, UI{}, 74)), "\n")
	if !strings.Contains(rows, "sending answer…") || strings.Count(rows, "PowerShell") != 3 {
		t.Errorf("answering: the card, and the call's row left out:\n%s", rows)
	}
}

// TestCommandDeniedSymbol is D5's symbol: a call denied before it ran, by
// Codex's review, shows ✗ though the CLI reported it started, never its
// running symbol; one whose request expired before it ran, the grey ·.
func TestCommandDeniedSymbol(t *testing.T) {
	b := newConvBench("codex", "gpt-5.6-sol")
	b.prompt("go")
	b.ev(event.ToolStarted, event.Tool{ToolID: "c1", Name: "commandExecution", Input: codexItem(t, "c1", "Remove-Item x")})
	b.itemEv(event.ReviewUpdated, "c1", event.Review{ReviewID: "rv", Status: "denied", Risk: "high", Rationale: "deletes a file"})
	b.ev(event.ToolStarted, event.Tool{ToolID: "c2", Name: "commandExecution", Input: codexItem(t, "c2", "Remove-Item y")})
	b.ev(event.ApprovalRequest, event.Approval{ID: "r2", Tool: "commandExecution", ToolID: "c2"})
	b.ev(event.ApprovalResolved, event.Approval{ID: "r2", Decision: event.Withdrawn})
	rows := rowTexts(bRows(b.s, UI{Spinner: 1}, 74))
	for _, want := range []string{"▎  ✗ PowerShell  Remove-Item x", "▎  · PowerShell "} {
		found := false
		for _, r := range rows {
			found = found || strings.HasPrefix(r, want)
		}
		if !found {
			t.Errorf("no row %q:\n%s", want, strings.Join(rows, "\n"))
		}
	}
}

// TestCommandFirstEntry: a block unfolds as well on the conversation's
// first entry, a turn the CLI opened itself before any prompt.
func TestCommandFirstEntry(t *testing.T) {
	b := newConvBench("codex", "gpt-5.6-sol")
	b.ev(event.ToolStarted, event.Tool{ToolID: "c1", Name: "commandExecution", Input: codexItem(t, "c1", todoScript)})
	if b.s.Conversation[0].MessageID != "c1" {
		t.Fatalf("the first entry %+v", b.s.Conversation[0])
	}
	rows := strings.Join(rowTexts(bRows(b.s, UI{Unfolded: map[string]bool{CommandKey("c1"): true}}, 74)), "\n")
	if !strings.Contains(rows, "▾ ") || !strings.Contains(rows, "($todos | Select-Object -Unique Path).Count") {
		t.Errorf("the first entry's block unfolded:\n%s", rows)
	}
}

// TestCommandCardNoRepeat is D7's context: a description that is the
// command itself is not shown again under it.
func TestCommandCardNoRepeat(t *testing.T) {
	b := newConvBench("claude-code", "claude-opus-5-5")
	b.prompt("go")
	b.call("t1", "Bash", map[string]string{"command": "go vet ./..."}, false, false)
	in, _ := json.Marshal(map[string]string{"command": "go vet ./..."})
	b.ev(event.ApprovalRequest, event.Approval{ID: "a1", Tool: "Bash", ToolID: "t1", Input: in, Summary: "go vet ./..."})
	rows := strings.Join(rowTexts(bRows(b.s, UI{}, 74)), "\n")
	if strings.Contains(rows, "description") || strings.Count(rows, "go vet ./...") != 1 {
		t.Errorf("the command repeated as its description:\n%s", rows)
	}
}

// TestCommandAsSentRows is D9's AS SENT drawn: its rows break at a blank,
// the blank in neither row, so that no row of it starts with a space past
// its indent.
func TestCommandAsSentRows(t *testing.T) {
	b := codexScene(t, true, false)
	sc, _ := entryFull(b.s, UI{Width: 80, Height: 60}, entryOf(b.s, "tool", "c3"))
	in := false
	rows := 0
	for _, l := range sc.lines {
		txt := texts([]line{l})
		switch {
		case strings.Contains(txt, "AS SENT,"):
			in = true
		case in && strings.TrimSpace(txt) == "":
			in = false
		case in:
			rows++
			if strings.HasPrefix(txt, "    ") || !strings.HasPrefix(txt, "   ") {
				t.Errorf("an AS SENT row past its indent: %q", txt)
			}
		}
	}
	if rows < 3 {
		t.Errorf("AS SENT on %d rows at 80 columns", rows)
	}
}

// TestCommandCopyPart is C2 on a part: a selection from a section's first
// row to a word's end in a later one copies the source's bytes between the
// two places, no more.
func TestCommandCopyPart(t *testing.T) {
	b := codexScene(t, true, false)
	sc, _ := entryFull(b.s, UI{Width: 80, Height: 60}, entryOf(b.s, "tool", "c3"))
	var y0, y1, x1 int = -1, -1, 0
	for y, l := range sc.lines {
		txt := texts([]line{l})
		if y0 < 0 && strings.Contains(txt, "$files = Get-ChildItem") {
			y0 = y
		}
		if i := strings.Index(txt, "Sort-Object"); y0 >= 0 && y1 < 0 && i >= 0 {
			y1, x1 = y, lipglossWidth(txt[:i])+len("Sort-Object")-1
		}
	}
	if y0 < 0 || y1 < 0 {
		t.Fatal("no COMMAND rows")
	}
	from, _, _ := sc.HitIn('I', 0, y0, false)
	_, to, _ := sc.HitIn('I', x1, y1, false)
	got := sc.Text(b.s, Selection{From: from, To: to})
	if want := todoScript[:strings.Index(todoScript, "Sort-Object")+len("Sort-Object")]; got != want {
		t.Errorf("the part copies %q, want %q", got, want)
	}
}

// lipglossWidth is s's width in cells, as the screen draws it.
func lipglossWidth(s string) int { return textWidth(s) }

// TestCommandSearchLit is C4's lighting: a match in a review's part lights
// the review's row, never the command's bytes at the same offsets.
func TestCommandSearchLit(t *testing.T) {
	b := codexScene(t, true, true)
	ms, _ := ConvMatches(b.s, UI{}, "Counts", nil)
	if len(ms) != 1 || ms[0].Line != 2 {
		t.Fatalf("the rationale's match: %+v", ms)
	}
	sc := Draw(b.s, UI{Width: 200, Height: 60, Search: SearchView{Open: true, Query: "Counts", Matches: ms, Cur: 0}})
	lit := map[string]bool{}
	for _, l := range sc.lines {
		txt := texts([]line{l})
		for _, x := range l {
			if x.st == stMatch || x.st == stMatchCur {
				switch {
				case strings.Contains(txt, "◈ risk low"):
					lit["review"] = true
				case strings.Contains(txt, "Get-ChildItem"):
					lit["command"] = true
				}
			}
		}
	}
	if !lit["review"] || lit["command"] {
		t.Errorf("lit %v, want the review's row only:\n%s", lit, texts(sc.lines))
	}
}

// TestCommandRanTimes is D9's "ran": from the owner's answer, the run
// after it; from the call's report when Claude Code's classifier decided,
// whose verdict comes with the result; none while Codex's review is in
// progress, the call not known to run.
func TestCommandRanTimes(t *testing.T) {
	row := func(label, text string) string { return fmt.Sprintf("%-*s%s", histCol, "  "+label, text) }
	k := newConvBench("claude-code", "claude-opus-5-5")
	k.prompt("go")
	k.call("x", "Bash", map[string]string{"command": "make"}, false, false)
	started := k.s.Tools["x"].StartedAt
	k.ev(event.ToolCompleted, event.Tool{ToolID: "x", Output: "ok"})
	k.ev(event.ToolDecision, event.Verdict{ToolID: "x", ReasonType: "classifier", Decision: event.Allow})
	if _, v := entryFull(k.s, UI{}, entryOf(k.s, "tool", "x")); !strings.Contains(v, row("ran", clockSec(started)+" to "+clockSec(k.s.Tools["x"].EndedAt))) {
		t.Errorf("the classifier's call: ran from its report:\n%s", v)
	}
	c := newConvBench("codex", "gpt-5.6-sol")
	c.prompt("go")
	c.ev(event.ToolStarted, event.Tool{ToolID: "c1", Name: "commandExecution", Input: codexItem(t, "c1", "Remove-Item x")})
	c.itemEv(event.ReviewUpdated, "c1", event.Review{ReviewID: "rv", Status: "inProgress"})
	if _, v := entryFull(c.s, UI{}, entryOf(c.s, "tool", "c1")); strings.Contains(v, "  ran ") || !strings.Contains(v, "auto-review inProgress") {
		t.Errorf("a review in progress: a run shown, or no review:\n%s", v)
	}
	c.itemEv(event.ReviewUpdated, "c1", event.Review{ReviewID: "rv", Status: "approved", Risk: "low"})
	approved := c.s.Tools["c1"].Review.At
	c.ev(event.ToolCompleted, event.Tool{ToolID: "c1", Name: "commandExecution", Output: "ok"})
	if _, v := entryFull(c.s, UI{}, entryOf(c.s, "tool", "c1")); !strings.Contains(v, row("ran", clockSec(approved)+" to "+clockSec(c.s.Tools["c1"].EndedAt))) {
		t.Errorf("approved by the review: ran from its approval:\n%s", v)
	}
}
