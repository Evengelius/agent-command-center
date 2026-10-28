package app

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/render"
)

// Plan 60 (brief amendment 62): the file viewer's git blame through the
// application, git's answers from the test double; one test with git itself.

// fakeBlame is the viewer's git over fakeGit: each Blame call recorded with
// its path and bytes and answered by answer, given its number from 1; with
// hold, each call waits until the test releases it.
type fakeBlame struct {
	*fakeGit
	answer   func(path string, content []byte, n int) (input.Blame, error)
	hold     bool
	gates    []chan struct{}
	paths    []string
	contents [][]byte
}

func (f *fakeBlame) Blame(dir, path string, content []byte) (input.Blame, error) {
	f.mu.Lock()
	n := len(f.paths) + 1
	f.paths = append(f.paths, path)
	f.contents = append(f.contents, slices.Clone(content))
	var gate chan struct{}
	if f.hold {
		gate = make(chan struct{})
		f.gates = append(f.gates, gate)
	}
	answer := f.answer
	f.mu.Unlock()
	if gate != nil {
		<-gate
	}
	return answer(path, content, n)
}

// release lets call i (from 0) answer.
func (f *fakeBlame) release(i int) {
	f.mu.Lock()
	g := f.gates[i]
	f.mu.Unlock()
	close(g)
}

// calls is how many Blame calls were made.
func (f *fakeBlame) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.paths)
}

// gitLines counts lines as git does: a last line with no LF counts.
func gitLines(content []byte) int {
	n := bytes.Count(content, []byte("\n"))
	if len(content) > 0 && content[len(content)-1] != '\n' {
		n++
	}
	return n
}

// sampleBlame is Ana Hubaut's commit for the first 20 lines and the rest, Bo
// Li's for lines 21 to 34, lines 35 to 37 not committed; the third commit's
// subject names the read's number.
func sampleBlame(path string, content []byte, n int) (input.Blame, error) {
	cet, est := time.FixedZone("+0100", 3600), time.FixedZone("-0500", -5*3600)
	b := input.Blame{Commits: []input.BlameCommit{
		{Hash: "f52c343d89b0e96f2b3f9a9cbe8a77859e9246c4", Author: "Ana Hubaut", Summary: "Add " + path,
			Time: time.Date(2022, 11, 15, 10, 0, 0, 0, cet)},
		{Hash: "9a1b2c3d4e5f60718293a4b5c6d7e8f901234567", Author: "Bo Li", Summary: "Rework " + path + " (read " + string(rune('0'+n)) + ")",
			Time: time.Date(2023, 3, 2, 8, 30, 0, 0, est)},
		{Hash: strings.Repeat("0", 40), Uncommitted: true, Author: "External file (--contents)"},
	}}
	for i := range gitLines(content) {
		switch {
		case i >= 20 && i < 34:
			b.Lines = append(b.Lines, 1)
		case i >= 34 && i < 37:
			b.Lines = append(b.Lines, 2)
		default:
			b.Lines = append(b.Lines, 0)
		}
	}
	return b, nil
}

// blameRig is the file rig's project (fileview_test.go) with the fake git,
// dates shown at +0100.
func blameRig(t *testing.T) (inputRig, *fakeBlame, *fakeClipboard, string) {
	r, clip, root := fileRig(t)
	f := &fakeBlame{fakeGit: &fakeGit{}, answer: sampleBlame}
	r.h.m.opt.Git = f
	r.h.m.opt.Location = time.FixedZone("+0100", 3600)
	return r, f, clip, root
}

// TestBlameSwitch is plan 60, criteria 60.1, 60.2 and 60.4 (designs D1, D2,
// D4 and D5): off at start, no git asked; `b` blames the file shown from
// its bytes, the column and the footer's commit following the cursor; `b`
// again hides them; on, the next file shown is blamed from its own read.
func TestBlameSwitch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, f, _, root := blameRig(t)
		h := r.h
		openPath(h, "main.go")
		if v := h.m.ui.File.Blame; v.On || f.calls() != 0 || !strings.Contains(h.frame(), "b blame") {
			t.Fatalf("at start: on %v, %d blames", v.On, f.calls())
		}
		h.send(typed('b'))
		disk, err := os.ReadFile(filepath.Join(root, "main.go"))
		if err != nil {
			t.Fatal(err)
		}
		if f.calls() != 1 || f.paths[0] != "main.go" || !bytes.Equal(f.contents[0], disk) {
			t.Fatalf("b: %d blames, paths %q, the bytes the file's %v", f.calls(), f.paths, f.calls() > 0 && bytes.Equal(f.contents[0], disk))
		}
		frame := h.frame()
		for _, want := range []string{"15/11/2022 Ana Hubaut      1  // line 1", "02/03/2023 Bo Li          21  // line 21",
			"f52c343 · Ana Hubaut · 15/11/2022 10:00 · Add main.go"} {
			if !strings.Contains(frame, want) {
				t.Errorf("no %q:\n%s", want, frame)
			}
		}
		h.send(typed('g'))
		for _, k := range typedText("21") {
			h.send(k)
		}
		h.send(key(tea.KeyEnter, 0))
		if frame := h.frame(); !strings.Contains(frame, "9a1b2c3 · Bo Li · 02/03/2023 14:30 · Rework main.go (read 1)") {
			t.Errorf("the cursor on line 21: the footer not Bo's commit:\n%s", frame)
		}
		h.send(typed('g'))
		for _, k := range typedText("36") {
			h.send(k)
		}
		h.send(key(tea.KeyEnter, 0))
		if frame := h.frame(); !strings.Contains(frame, "Not committed yet: changed in the working tree") ||
			!strings.Contains(frame, "Not committed             36  // line 36") {
			t.Errorf("the cursor on line 36, not committed:\n%s", frame)
		}
		h.send(typed('b'))
		if v := h.m.ui.File.Blame; v.On || strings.Contains(h.frame(), "Ana Hubaut") || !strings.Contains(h.frame(), "b blame") {
			t.Errorf("b again: on %v", v.On)
		}
		h.send(typed('b'))
		openPath(h, "web/app.ts")
		if v := h.m.ui.File.Blame; !v.On || f.calls() != 3 || f.paths[2] != "web/app.ts" || len(v.Lines) != 1 {
			t.Errorf("the next file: on %v, %d blames, paths %q, %d lines", v.On, f.calls(), f.paths, len(v.Lines))
		}
	})
}

// TestBlameAnswers is criterion 60.7's answers (design D5): an answer is
// dropped when another file was shown during its read, when the viewer was
// closed during it, and when `b` went off and on during it; the latest read
// alone is drawn.
func TestBlameAnswers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, f, _, _ := blameRig(t)
		h := r.h
		f.hold = true
		openPath(h, "main.go")
		h.send(typed('b'))
		if v := h.m.ui.File.Blame; v.Note != "blame…" || v.Lines != nil || !strings.Contains(h.frame(), "blame…") {
			t.Fatalf("while git reads: note %q", v.Note)
		}
		// Another file shown during the read: main.go's answer never shows on it.
		openPath(h, "web/app.ts")
		f.release(0)
		h.settle()
		if v := h.m.ui.File; v.Rel != "web/app.ts" || !v.Blame.On || v.Blame.Lines != nil || v.Blame.Note != "blame…" || !strings.Contains(h.frame(), "blame…") {
			t.Fatalf("main.go's answer on web/app.ts: %q, on %v, lines %v, note %q", v.Rel, v.Blame.On, v.Blame.Lines, v.Blame.Note)
		}
		f.release(1)
		h.settle()
		if v := h.m.ui.File.Blame; len(v.Lines) != 1 || v.Commits[0].Summary != "Add web/app.ts" {
			t.Fatalf("web/app.ts's own answer: %+v", v)
		}

		// The viewer closed during the read: the answer dropped.
		openPath(h, "main.go")
		h.send(keyOf(tea.KeyF3))
		if h.m.ui.File.Open {
			t.Fatal("F3 left the viewer open")
		}
		f.release(2)
		h.settle()
		if v := h.m.ui.File; v.Open || v.Blame.Lines != nil {
			t.Errorf("the viewer closed: an answer applied: %+v", v.Blame)
		}

		// `b` off and on during a read: the first read's answer dropped, the
		// second's drawn.
		openPath(h, "main.go")
		h.send(typed('b'))
		h.send(typed('b'))
		f.release(4)
		h.settle()
		if v := h.m.ui.File.Blame; v.Lines == nil || !strings.HasSuffix(v.Commits[1].Summary, "(read 5)") {
			t.Fatalf("the latest read: %+v", v)
		}
		f.release(3)
		h.settle()
		if v := h.m.ui.File.Blame; !strings.HasSuffix(v.Commits[1].Summary, "(read 5)") {
			t.Errorf("the replaced read's answer drawn: %q", v.Commits[1].Summary)
		}

		// The answer for the file shown before arriving while the next
		// file's read has not come back: dropped by the file's generation.
		openPath(h, "docs/readme.md")
		h.m.openFile(render.FileTarget{Path: "web/app.ts", Opened: "quick open"}) // its read never run
		f.release(5)
		h.settle()
		if v := h.m.ui.File; v.Path != "web/app.ts" || v.Blame.Lines != nil || v.Blame.Note != "" {
			t.Errorf("docs/readme.md's answer while web/app.ts is read: lines %v, note %q", v.Blame.Lines, v.Blame.Note)
		}
	})
}

// TestBlameRead is blameRead's rule itself (design D5), whatever an answer's
// numbers: drawn only for the file and the read it was asked for, the
// viewer open and the blame on.
func TestBlameRead(t *testing.T) {
	answer := blameMsg{gen: 1, seq: 1, b: input.Blame{Lines: []int{0}, Commits: []input.BlameCommit{
		{Hash: "f52c343d89b0e96f2b3f9a9cbe8a77859e9246c4", Author: "Ana Hubaut", Summary: "Add f.txt"}}}}
	for _, c := range []struct {
		name            string
		gen, seq        int
		open, on, drawn bool
	}{
		{"the same file and read", 1, 1, true, true, true},
		{"another file since", 2, 1, true, true, false},
		{"another read since", 1, 2, true, true, false},
		{"the viewer closed", 1, 1, false, true, false},
		{"the blame off", 1, 1, true, false, false},
	} {
		m := &Model{fileGen: c.gen, blameSeq: c.seq, blameOn: c.on}
		m.ui.File = render.FileView{Open: c.open, Lines: []string{"x"}, Blame: render.FileBlame{On: c.on}}
		m.blameRead(answer)
		if drawn := m.ui.File.Blame.Lines != nil; drawn != c.drawn {
			t.Errorf("%s: drawn %v, want %v", c.name, drawn, c.drawn)
		}
	}
}

// TestBlameMessages is criterion 60.5 (design D6): each of git's refusals
// as the footer says it, the file shown meanwhile; a blame whose lines
// differ in number from the viewer's; `b` on a file changed since it was
// shown, which git is not asked about.
func TestBlameMessages(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, f, _, root := blameRig(t)
		h := r.h
		for _, c := range []struct {
			err  error
			want string
		}{
			{input.Refusal{Kind: "untracked", Msg: "not in git yet: no history"}, "not in git yet: no history"},
			{input.Refusal{Kind: "no-commits", Msg: "no commits yet"}, "no commits yet"},
			{input.Refusal{Kind: "timeout", Msg: "blame stopped after 15 s"}, "blame stopped after 15 s"},
			{input.NotRepoError{Dir: root}, "not a git repository"},
			{input.ErrNoGit, "git not found"},
			{errors.New("fatal: bad object HEAD"), "fatal: bad object HEAD"},
		} {
			f.answer = func(string, []byte, int) (input.Blame, error) { return input.Blame{}, c.err }
			openPath(h, "main.go")
			h.send(typed('b'))
			v := h.m.ui.File
			if v.Blame.Note != c.want || !v.Blame.NoteBad || v.State != "" || len(v.Lines) != 100 || !strings.Contains(h.frame(), c.want) {
				t.Errorf("%v: note %q, state %q, %d lines", c.err, v.Blame.Note, v.State, len(v.Lines))
			}
			h.send(typed('b'))
		}
		f.answer = func(string, []byte, int) (input.Blame, error) {
			return input.Blame{Lines: []int{0, 0}, Commits: []input.BlameCommit{{Hash: "f52c343d89b0e96f2b3f9a9cbe8a77859e9246c4"}}}, nil
		}
		openPath(h, "main.go")
		h.send(typed('b'))
		if v := h.m.ui.File.Blame; v.Note != "the file changed after it was read: show it again" || v.Lines != nil {
			t.Errorf("two lines blamed for 100: note %q", v.Note)
		}
		h.send(typed('b'))

		// Changed on disk after it was shown: `b` reads it again and asks git
		// nothing.
		f.answer = sampleBlame
		openPath(h, "main.go")
		if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("changed\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		calls := f.calls()
		h.send(typed('b'))
		if v := h.m.ui.File.Blame; v.Note != "the file changed after it was read: show it again" || f.calls() != calls {
			t.Errorf("a file changed since shown: note %q, %d more blames", v.Note, f.calls()-calls)
		}
	})
}

// TestBlameUnchanged is criterion 60.6 (design D7): with the blame off, git
// is never asked; on, a `b` typed in the search goes to the search; a drag
// copies the code alone; full screen draws the column; plan 58's click
// outside still closes the viewer.
func TestBlameUnchanged(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, f, clip, _ := blameRig(t)
		h := r.h
		for _, p := range []string{"main.go", "web/app.ts", "src/Main.java", "docs/readme.md"} {
			openPath(h, p)
		}
		if f.calls() != 0 {
			t.Fatalf("the blame off: %d blames", f.calls())
		}
		openPath(h, "main.go")
		h.send(typed('b'))
		h.send(typed('/'))
		h.send(typed('b'))
		if v := h.m.ui.File; v.Field != '/' || v.Typed != "b" || !v.Blame.On {
			t.Errorf("b in the search: field %q, typed %q, on %v", v.Field, v.Typed, v.Blame.On)
		}
		h.send(keyOf(tea.KeyEscape))

		frame := strings.Split(h.frame(), "\n")
		y2, y4 := -1, -1
		for i, l := range frame {
			switch {
			case strings.Contains(l, "// line 2") && y2 < 0:
				y2 = i
			case strings.Contains(l, "// line 4") && y4 < 0:
				y4 = i
			}
		}
		if y2 < 0 || y4 < 0 {
			t.Fatal("no lines 2 and 4 on screen")
		}
		x0 := strings.Index(frame[y2], "15/11/2022")
		x1 := lastOn(t, h, y4, "4") + 1
		dragCopy(t, h, x0, y2, x1, y4)
		if got := clip.writes[len(clip.writes)-1]; strings.Contains(got, "2022") || strings.Contains(got, "Ana") || !strings.Contains(got, "// line 3") {
			t.Errorf("the drag copied %q", got)
		}

		h.send(typed('f'))
		if !strings.Contains(h.frame(), "15/11/2022 Ana Hubaut") {
			t.Error("full screen: no column")
		}
		h.send(typed('f'))
		mouseClick(t, h, 1, 0)
		if h.m.ui.File.Open {
			t.Error("a click outside left the viewer open")
		}
	})
}

// TestBlameCommits is criterion 60.2's texts (design D2): the dates in the
// zone given, the short hash, the initials, a line not committed with no
// author of git's shown.
func TestBlameCommits(t *testing.T) {
	m := &Model{opt: Options{Location: time.FixedZone("+0100", 3600)}}
	b, _ := sampleBlame("f.txt", []byte("x\n"), 1)
	cs := m.blameCommits(b.Commits)
	if c := cs[0]; c != (render.BlameCommit{Short: "f52c343", Author: "Ana Hubaut", Initials: "AH", Summary: "Add f.txt",
		Date: "15/11/2022", DateShort: "15/11/22", When: "15/11/2022 10:00"}) {
		t.Errorf("Ana's %+v", c)
	}
	if c := cs[1]; c.When != "02/03/2023 14:30" || c.Initials != "BL" {
		t.Errorf("Bo's, at +0100: %+v", c)
	}
	if c := cs[2]; c != (render.BlameCommit{Uncommitted: true}) {
		t.Errorf("not committed: %+v", c)
	}
	for name, want := range map[string]string{"Jean Paul Sartre Dupont": "JPS", "évariste galois": "ÉG", "solo": "S", "": ""} {
		if got := initials(name); got != want {
			t.Errorf("initials(%q) %q, want %q", name, got, want)
		}
	}
}

// TestBlameGit is criteria 60.2 and 60.7 with git itself: on a repository
// in the project, the column names the real commits and authors, the line
// changed since "Not committed"; git called once, with the file's bytes on
// its standard input.
func TestBlameGit(t *testing.T) {
	isolateGit(t)
	r, _, root := fileRig(t)
	h := r.h
	run := func(name, date string, args ...string) {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME="+name, "GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_NAME="+name, "GIT_COMMITTER_DATE="+date)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("Ana Hubaut", "2022-11-15T10:00:00+01:00", "init", "-q", "-b", "main")
	run("Ana Hubaut", "2022-11-15T10:00:00+01:00", "add", "main.go")
	run("Ana Hubaut", "2022-11-15T10:00:00+01:00", "commit", "-q", "-m", "Add main.go")
	text, _ := os.ReadFile(filepath.Join(root, "main.go"))
	if err := os.WriteFile(filepath.Join(root, "main.go"), bytes.Replace(text, []byte("// line 3\n"), []byte("// line three\n"), 1), 0o644); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var calls [][]string
	h.m.opt.Git = input.GitCLI{Trace: func(a []string) { mu.Lock(); calls = append(calls, a); mu.Unlock() }}
	h.m.opt.Location = time.FixedZone("+0100", 3600)
	openPath(h, "main.go")
	h.send(typed('b'))
	for i := 0; i < 100 && h.m.ui.File.Blame.Lines == nil && h.m.ui.File.Blame.Note == "blame…"; i++ {
		h.settle()
	}
	frame := h.frame()
	for _, want := range []string{"15/11/2022 Ana Hubaut      1  // line 1", "Not committed              3  // line three"} {
		if !strings.Contains(frame, want) {
			t.Errorf("no %q (note %q):\n%s", want, h.m.ui.File.Blame.Note, frame)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	var blames []string
	for _, c := range calls {
		if s := strings.Join(c, " "); strings.Contains(s, " blame ") {
			blames = append(blames, s)
		}
	}
	if len(blames) != 1 || !strings.HasSuffix(blames[0], "blame --porcelain --contents - -- main.go") {
		t.Errorf("the blame calls %q", blames)
	}
}

// TestBlameFrameCost is criterion 60.9's timing (design D8): a key to its
// frame in the viewer with the blame on, on a 10,000-line file with a commit
// every ten lines, at 200x60; its 95th percentile within the keystroke's
// target, judged in the timing step.
func TestBlameFrameCost(t *testing.T) {
	r, f, _, root := blameRig(t)
	h := r.h
	var text strings.Builder
	for i := range 10_000 {
		text.WriteString("\tline " + strings.Repeat("x", i%7) + ": the quick brown fox jumps over the lazy dog\n")
	}
	if err := os.WriteFile(filepath.Join(root, "big.go"), []byte(text.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	f.answer = func(path string, content []byte, n int) (input.Blame, error) {
		var b input.Blame
		for i := range 1000 {
			b.Commits = append(b.Commits, input.BlameCommit{Hash: strings.Repeat("1", 36) + string(rune('a'+i%26)) + "bcd",
				Author: "Author " + strings.Repeat("y", i%9), Summary: "a commit", Time: time.Date(2022, 1, 1, 9, 0, 0, 0, time.UTC)})
		}
		for i := range gitLines(content) {
			b.Lines = append(b.Lines, i/10)
		}
		return b, nil
	}
	h.send(tea.WindowSizeMsg{Width: 200, Height: 60})
	openPath(h, "big.go")
	h.send(typed('b'))
	if v := h.m.ui.File.Blame; len(v.Lines) != 10_000 || len(v.Commits) != 1000 {
		t.Fatalf("the blame: %d lines, %d commits, note %q", len(v.Lines), len(v.Commits), v.Note)
	}
	var ds []time.Duration
	for i := range 400 {
		k := keyOf(tea.KeyPgDown)
		if i%100 >= 50 {
			k = keyOf(tea.KeyUp)
		}
		ds = append(ds, frameTime(h, k))
	}
	p50, p95 := percentile(ds, 0.50), percentile(ds, 0.95)
	t.Logf("a key to its frame, 10,000 lines blamed: p50 %v, p95 %v (target p95 ≤ %v)", p50, p95, keystrokeTarget)
	if p95 > keystrokeTarget {
		overTarget(t, "p95 %v over the %v target", p95, keystrokeTarget)
	}
}
