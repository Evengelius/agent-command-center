package input

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestPasteByteForByte(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"lf", "a\nb\n", "a\nb\n"},
		{"crlf", "a\r\nb\r\n", "a\nb\n"},
		{"lone cr", "a\rb\r", "a\nb\n"},
		{"mixed", "a\r\r\n\nb", "a\n\n\nb"},
		{"tabs", "\ta\t\tb", "\ta\t\tb"},
		{"trailing spaces", "a  \nb \t", "a  \nb \t"},
		{"combining", "é́x", "é́x"},
		{"emoji", "😀 👨‍👩‍👧 🇧🇪", "😀 👨‍👩‍👧 🇧🇪"},
		{"wide", "中文\n日本語", "中文\n日本語"},
		{"c0", "\x00\x01\x1b[0m\x7f", "\x00\x01\x1b[0m\x7f"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var e Editor
			e.Insert("<")
			e.Paste(c.in)
			if got, want := e.Text(), "<"+c.want; got != want {
				t.Fatalf("got %q, want %q", got, want)
			}
			if e.Cursor() != len(e.Text()) {
				t.Fatalf("cursor %d", e.Cursor())
			}
		})
	}
}

func TestPasteReplacesSelection(t *testing.T) {
	var e Editor
	e.SetText("hello world")
	e.Move(WordLeft, true)
	e.Paste("X\r\nY")
	if e.Text() != "hello X\nY" {
		t.Fatalf("%q", e.Text())
	}
	e.SetText("abc")
	e.SelectAll()
	e.Insert("z")
	if e.Text() != "z" {
		t.Fatalf("insert over selection: %q", e.Text())
	}
}

func TestUndoRedo(t *testing.T) {
	var e Editor
	e.Insert("a")
	e.Insert("b")
	e.Insert("c")
	e.Paste("1\n2\n3")
	if !e.Undo() || e.Text() != "abc" {
		t.Fatalf("paste not one step: %q", e.Text())
	}
	if !e.Undo() || e.Text() != "" || e.Cursor() != 0 {
		t.Fatalf("typing not merged: %q", e.Text())
	}
	if e.Undo() {
		t.Fatal("undo past start")
	}
	e.Redo()
	e.Redo()
	if e.Text() != "abc1\n2\n3" || e.Cursor() != len(e.Text()) {
		t.Fatalf("redo: %q", e.Text())
	}
	e.Undo()
	e.Insert("Z") // new edit clears redo
	if e.Redo() {
		t.Fatal("redo not cleared")
	}
	if e.Text() != "abcZ" {
		t.Fatalf("%q", e.Text())
	}
}

func TestUndoTypingBreaks(t *testing.T) {
	var e Editor
	for _, s := range []string{"a", "b", " ", "c"} {
		e.Insert(s)
	}
	e.Undo()
	if e.Text() != "ab " {
		t.Fatalf("after space: %q", e.Text())
	}
	e.Undo()
	if e.Text() != "" {
		t.Fatalf("space joins its word: %q", e.Text())
	}
	e.Insert("x")
	e.Left()
	e.Right()
	e.Insert("y")
	e.Undo()
	if e.Text() != "x" {
		t.Fatalf("cursor move breaks merge: %q", e.Text())
	}
}

func TestUndoDeleteAndSelection(t *testing.T) {
	var e Editor
	e.SetText("hello")
	e.Backspace()
	e.Undo()
	if e.Text() != "hello" || e.Cursor() != 5 {
		t.Fatalf("%q %d", e.Text(), e.Cursor())
	}
	e.SelectAll()
	e.Paste("x")
	e.Undo()
	if e.Text() != "hello" {
		t.Fatalf("%q", e.Text())
	}
	e.SetText("a")
	if e.Undo() {
		t.Fatal("SetText keeps history")
	}
}

func TestSelectionMotions(t *testing.T) {
	const text = "one two\nthree\nfour"
	cases := []struct {
		name   string
		start  int
		m      Motion
		sel    string
		cursor int
	}{
		{"left", 3, Left, "e", 2},
		{"right", 3, Right, " ", 4},
		{"up", 11, Up, " two\nthr", 3},
		{"down", 3, Down, " two\nthr", 11},
		{"linestart", 5, LineStart, "one t", 0},
		{"lineend", 5, LineEnd, "wo", 7},
		{"wordleft", 6, WordLeft, "tw", 4},
		{"wordright", 0, WordRight, "one", 3},
		{"bufstart", 5, BufferStart, "one t", 0},
		{"bufend", 11, BufferEnd, "ee\nfour", len(text)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var e Editor
			e.SetText(text)
			e.cursor, e.anchor = c.start, c.start
			e.Move(c.m, true)
			if e.SelectedText() != c.sel || e.Cursor() != c.cursor {
				t.Fatalf("sel %q cursor %d", e.SelectedText(), e.Cursor())
			}
			if _, _, ok := e.Selection(); !ok {
				t.Fatal("no selection")
			}
			e.Move(c.m, false)
			if _, _, ok := e.Selection(); ok {
				t.Fatal("selection not cleared")
			}
		})
	}
}

func TestSelectionCollapse(t *testing.T) {
	var e Editor
	e.SetText("abcdef")
	e.cursor, e.anchor = 2, 2
	e.Move(Right, true)
	e.Move(Right, true)
	if a, b, ok := e.Selection(); !ok || a != 2 || b != 4 {
		t.Fatalf("%d %d %v", a, b, ok)
	}
	e.Move(Left, false)
	if e.Cursor() != 2 || e.hasSel() {
		t.Fatalf("left collapse: %d", e.Cursor())
	}
	e.Move(Right, true)
	e.Move(Right, true)
	e.Move(Right, false)
	if e.Cursor() != 4 || e.hasSel() {
		t.Fatalf("right collapse: %d", e.Cursor())
	}
	e.SelectAll()
	if a, b, _ := e.Selection(); a != 0 || b != 6 {
		t.Fatal("select all")
	}
	e.Backspace()
	if e.Text() != "" {
		t.Fatal("backspace with selection")
	}
	e.SetText("abc")
	e.SelectAll()
	e.Delete()
	if e.Text() != "" {
		t.Fatal("delete with selection")
	}
}

func TestWordDeletesAndCut(t *testing.T) {
	var e Editor
	e.SetText("foo bar  baz")
	e.DeleteWordLeft()
	if e.Text() != "foo bar  " {
		t.Fatalf("%q", e.Text())
	}
	e.DeleteWordLeft()
	if e.Text() != "foo " {
		t.Fatalf("%q", e.Text())
	}
	e.Undo()
	if e.Text() != "foo bar  " {
		t.Fatalf("undo: %q", e.Text())
	}
	e.Move(BufferStart, false)
	e.DeleteWordRight()
	if e.Text() != " bar  " {
		t.Fatalf("%q", e.Text())
	}
	e.SetText("héllo 中文 x")
	e.Move(WordLeft, false)
	e.Move(WordLeft, false)
	e.DeleteWordRight()
	if e.Text() != "héllo  x" {
		t.Fatalf("%q", e.Text())
	}
	e.SetText("cut me")
	e.Move(WordLeft, true)
	if got := e.Cut(); got != "me" || e.Text() != "cut " {
		t.Fatalf("cut %q left %q", got, e.Text())
	}
	e.Undo()
	if e.Text() != "cut me" {
		t.Fatalf("undo cut: %q", e.Text())
	}
	e.SetText("x")
	if e.Cut() != "" {
		t.Fatal("cut without selection")
	}
}

func TestGoalColumn(t *testing.T) {
	var e Editor
	e.SetText("abcdef\nab\n中文字xyz")
	e.cursor, e.anchor = 5, 5
	e.Move(Down, false)
	if e.Cursor() != 9 { // end of "ab"
		t.Fatalf("short line: %d", e.Cursor())
	}
	e.Move(Down, false)
	if got := e.Text()[7+3 : e.Cursor()]; got != "中文字xy" {
		t.Fatalf("goal column in runes: %q", got)
	}
	e.Move(Up, false)
	e.Move(Up, false)
	if e.Cursor() != 5 {
		t.Fatalf("back up: %d", e.Cursor())
	}
	e.Move(Up, false)
	if e.Cursor() != 0 {
		t.Fatalf("up on first line: %d", e.Cursor())
	}
	e.Move(BufferEnd, false)
	e.Move(Down, false)
	if e.Cursor() != len(e.Text()) {
		t.Fatal("down on last line")
	}
}

func TestLineRangeAndCursorLine(t *testing.T) {
	var e Editor
	e.SetText("a\nb\n\nd")
	if e.CursorLine() != 3 {
		t.Fatalf("cursor line %d", e.CursorLine())
	}
	e.Move(BufferStart, false)
	if e.CursorLine() != 0 {
		t.Fatal("line 0")
	}
	cases := []struct {
		first, n int
		want     []string
	}{
		{0, 2, []string{"a", "b"}},
		{1, 3, []string{"b", "", "d"}},
		{2, 10, []string{"", "d"}},
		{3, 1, []string{"d"}},
		{4, 1, nil},
		{0, 0, []string{}},
	}
	for _, c := range cases {
		if got := e.LineRange(c.first, c.n); !reflect.DeepEqual(got, c.want) {
			t.Errorf("LineRange(%d,%d) = %q, want %q", c.first, c.n, got, c.want)
		}
	}
	e.SetText("a\n")
	if got := e.LineRange(0, 5); !reflect.DeepEqual(got, []string{"a", ""}) {
		t.Fatalf("%q", got)
	}
}

func TestOSC52AndTmuxWrap(t *testing.T) {
	if got, want := OSC52("hi"), "\x1b]52;c;aGk=\x07"; got != want {
		t.Fatalf("%q", got)
	}
	if got, want := TmuxWrap("\x1b]52;c;aGk=\x07"), "\x1bPtmux;\x1b\x1b]52;c;aGk=\x07\x1b\\"; got != want {
		t.Fatalf("%q", got)
	}
}

type fakeClip struct {
	wrote string
	err   error
}

func (f *fakeClip) Read() (string, error) { return f.wrote, nil }
func (f *fakeClip) Write(s string) error {
	if f.err != nil {
		return f.err
	}
	f.wrote = s
	return nil
}

func env(kv ...string) func(string) string {
	return func(k string) string {
		for i := 0; i+1 < len(kv); i += 2 {
			if kv[i] == k {
				return kv[i+1]
			}
		}
		return ""
	}
}

func TestCopier(t *testing.T) {
	osc := OSC52("a\nb")
	cases := []struct {
		name   string
		native *fakeClip
		nilNat bool
		env    func(string) string
		text   string
		seq    string
		toast  string
		wrote  string
	}{
		{"native", &fakeClip{}, false, env(), "a\nb", osc, "copied 2 lines · native", "a\nb"},
		{"one line", &fakeClip{}, false, env(), "a", OSC52("a"), "copied 1 line · native", "a"},
		{"trailing newline", &fakeClip{}, false, env(), "a\n", OSC52("a\n"), "copied 1 line · native", "a\n"},
		{"failure", &fakeClip{err: errors.New("busy")}, false, env(), "a\nb", osc, "copied · OSC 52 (terminal must allow it)", ""},
		{"nil", nil, true, env(), "a\nb", osc, "copied · OSC 52 (terminal must allow it)", ""},
		{"ssh", &fakeClip{}, false, env("SSH_CONNECTION", "1 2 3 4"), "a\nb", osc, "copied · OSC 52 (terminal must allow it)", ""},
		{"ssh tty", &fakeClip{}, false, env("SSH_TTY", "/dev/pts/1"), "a\nb", osc, "copied · OSC 52 (terminal must allow it)", ""},
		{"tmux", &fakeClip{}, false, env("TMUX", "/tmp/tmux-1/default,1,0"), "a\nb", TmuxWrap(osc), "copied 2 lines · native", "a\nb"},
		{"empty", &fakeClip{}, false, env(), "", "", "nothing to copy", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cp := Copier{Getenv: c.env}
			if !c.nilNat {
				cp.Native = c.native
			}
			seq, toast, err := cp.Copy(c.text)
			if err != nil || seq != c.seq || toast != c.toast {
				t.Fatalf("seq %q toast %q err %v", seq, toast, err)
			}
			if !c.nilNat && c.native.wrote != c.wrote {
				t.Fatalf("wrote %q", c.native.wrote)
			}
		})
	}
}

func buildText(lines int) string {
	line := strings.Repeat("abcdefghi ", 6)
	return strings.Repeat(line+"\n", lines-1) + line
}

func BenchmarkKeystroke10k(b *testing.B) {
	var e Editor
	e.SetText(buildText(10000))
	e.cursor = len(e.text) / 2
	e.anchor = e.cursor
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		e.Insert("x")
		_ = e.CursorLine()
		_ = e.LineRange(e.CursorLine()-10, 20)
		if i%1000 == 999 { // keep the history bounded like a real session
			e.undo = e.undo[:0]
		}
	}
}

func BenchmarkPaste50k(b *testing.B) {
	s := strings.ReplaceAll(buildText(50000), "\n", "\r\n")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var e Editor
		e.Paste(s)
		_ = e.CursorLine()
		_ = e.LineRange(e.CursorLine()-19, 20)
	}
}
