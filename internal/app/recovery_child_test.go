package app

// Criterion 33.11 (plan 33): the real program (Run) in a child process,
// with the fake Claude Code, a draft store in a temporary folder, a draft
// of text, three generated files (one named in Unicode, one with spaces)
// and a pasted image, ended by a panic, by the quit Bubble Tea makes of
// SIGTERM (which Go makes of a closed console, CTRL_CLOSE_EVENT), and by
// TerminateProcess; then a new run of the project restores the draft.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Evengelius/agent-command-center/internal/input"
	"github.com/Evengelius/agent-command-center/internal/input/imagetest"
	"github.com/Evengelius/agent-command-center/internal/provider"
	"github.com/Evengelius/agent-command-center/internal/provider/claude"
	"github.com/Evengelius/agent-command-center/internal/state"
)

const recoveryText = "unsent text with images"

type recoveryPoll struct{}

// childPNG is the pasted image, the same bytes in the child and the
// parent: a blue rectangle, unlike the files' drawing.
func childPNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 80, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 80; x++ {
			img.Set(x, y, color.RGBA{B: 200, G: uint8(x), A: 255})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		panic(err)
	}
	return b.Bytes()
}

// TestRecoveryChild is the child process of TestRecovery.
func TestRecoveryChild(t *testing.T) {
	mode := os.Getenv("ACC_RC_CHILD")
	if mode == "" {
		t.Skip("started by TestRecovery")
	}
	recoveryChild(mode)
}

func recoveryChild(mode string) {
	files := strings.Split(os.Getenv("ACC_RC_FILES"), "|")
	mark, drafts, project := os.Getenv("ACC_RC_MARK"), os.Getenv("ACC_RC_DRAFTS"), os.Getenv("ACC_RC_PROJECT")
	w := &wireClaude{}
	p := &picker{answers: [][]string{files}}
	m := New(Options{
		Providers:  []provider.Provider{installedClaude{&claude.Provider{Spawner: w}}},
		Cwd:        project,
		Now:        time.Now,
		Drafts:     &input.Drafts{Dir: drafts},
		MockSpeed:  1,
		PickImages: p.pick,
		Clipboard:  &clip{image: childPNG()},
	})
	stored := func() (string, []string) {
		ds := m.draft
		ds.mu.Lock() // the folder is made by a save, outside Update
		dir := ds.slot.Dir()
		ds.mu.Unlock()
		if dir == "" {
			return "", nil
		}
		text, _ := os.ReadFile(filepath.Join(dir, "draft.txt"))
		var images []string
		if b, err := os.ReadFile(filepath.Join(dir, "images.json")); err == nil {
			_ = json.Unmarshal(b, &images)
		}
		return string(text), images
	}
	say := func(f string, a ...any) { fmt.Fprintf(os.Stderr, "child: "+f+"\n", a...) }
	all := len(files) + 1
	step, polling := 0, false
	var since time.Time
	again := func() []tea.Cmd {
		if polling {
			return nil
		}
		polling = true
		return []tea.Cmd{func() tea.Msg { time.Sleep(20 * time.Millisecond); return recoveryPoll{} }}
	}
	updateHook = func(m *Model, msg tea.Msg) []tea.Cmd {
		if _, ok := msg.(recoveryPoll); ok {
			polling = false
		}
		s := m.s
		switch {
		case step == 0 && s.Startup.Phase == state.Asking:
			step++
			m.ui.Choice = claude.ProviderID
			return []tea.Cmd{func() tea.Msg { return tea.KeyPressMsg{Code: tea.KeyEnter} }}
		case step == 1 && s.Session.Phase == state.Ready && m.draftReady && mode == "restore":
			if since.IsZero() {
				since = time.Now()
			}
			if (len(m.attach) < all || len(m.unread) > 0) && time.Since(since) < 5*time.Second {
				return again()
			}
			step++
			var sums []string
			for _, a := range m.attach {
				sums = append(sums, hex.EncodeToString(a.Sum[:]))
			}
			b, _ := json.Marshal(map[string]any{"text": m.edit.Text(), "names": rcNames(m), "sums": sums, "toast": m.ui.Toast})
			_ = os.WriteFile(mark, b, 0o600)
			return []tea.Cmd{func() tea.Msg { return tea.QuitMsg{} }}
		case step == 1 && s.Session.Phase == state.Ready && m.draftReady:
			step++
			return append(again(), func() tea.Msg { return tea.PasteMsg{Content: recoveryText} })
		case step == 2:
			if text, _ := stored(); text != recoveryText || m.draftTimer {
				return again()
			}
			step++
			say("the text is stored; attaching %d files", len(files))
			return append(again(), m.pickImages()...)
		case step == 3:
			if len(m.attach) < len(files) {
				return again()
			}
			step++
			return append(again(), m.pasteImages(false)...)
		case step == 4:
			if len(m.attach) < all {
				return again()
			}
			step++
			since = time.Now()
			_, images := stored()
			say("%d images attached; %d stored", len(m.attach), len(images))
			switch mode {
			case "panic":
				panic("injected panic")
			case "quit":
				return []tea.Cmd{func() tea.Msg { return tea.QuitMsg{} }}
			case "killnow":
				// Typed after the images: its timer still to fire at the kill.
				return append(again(), func() tea.Msg { return tea.PasteMsg{Content: " LAST"} })
			}
			return again()
		case step == 5:
			_, images := stored()
			typed := strings.HasSuffix(m.edit.Text(), " LAST")
			if len(images) < all || mode == "killsaved" && time.Since(since) < time.Second || mode == "killnow" && !typed {
				return again()
			}
			step++
			say("%d images stored %d ms after the last was attached; text timer pending %v", len(images), time.Since(since).Milliseconds(), m.draftTimer)
			_ = os.WriteFile(mark, nil, 0o600)
		}
		return nil
	}
	code, out := Run(m, tea.WithInput(nil), tea.WithOutput(io.Discard), tea.WithWindowSize(120, 40), tea.WithoutSignals())
	for _, l := range out {
		fmt.Fprintln(os.Stderr, l)
	}
	os.Exit(code)
}

type recoveryRun struct {
	code   int
	out    string
	killed time.Duration // from the mark to the kill
}

// runRecovery runs the child in mode for base's project; a "kill" mode is
// ended by TerminateProcess as soon as its mark appears.
func runRecovery(t *testing.T, mode, base string, files []string) recoveryRun {
	t.Helper()
	mark := filepath.Join(base, "mark-"+mode)
	project := filepath.Join(base, "project")
	if err := os.MkdirAll(project, 0o700); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestRecoveryChild$", "-test.count=1")
	cmd.Env = append(os.Environ(), "ACC_RC_CHILD="+mode, "ACC_RC_DRAFTS="+filepath.Join(base, "drafts"), "ACC_RC_PROJECT="+project, "ACC_RC_MARK="+mark, "ACC_RC_FILES="+strings.Join(files, "|"))
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	var r recoveryRun
	var err error
	if strings.HasPrefix(mode, "kill") {
		deadline := time.Now().Add(30 * time.Second)
	wait:
		for {
			if _, e := os.Stat(mark); e == nil {
				seen := time.Now()
				_ = cmd.Process.Kill() // TerminateProcess: no handler runs, nothing deferred
				r.killed = time.Since(seen)
				break
			}
			select {
			case err = <-done:
				t.Fatalf("the child ended before its mark (%v):\n%s", err, out.String())
			case <-time.After(2 * time.Millisecond):
			}
			if time.Now().After(deadline) {
				_ = cmd.Process.Kill()
				break wait
			}
		}
	}
	select {
	case err = <-done:
	case <-time.After(60 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatalf("the child did not end:\n%s", out.String())
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		r.code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	r.out = out.String()
	return r
}

// TestRecovery is criterion 33.11's child processes.
func TestRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("child processes")
	}
	pasted := childPNG()
	for _, c := range []struct {
		mode string
		code int
	}{
		{"panic", ExitPanic},
		{"quit", ExitOK},
		{"killsaved", 1},
		{"killnow", 1},
	} {
		t.Run(c.mode, func(t *testing.T) {
			base := t.TempDir()
			images := filepath.Join(base, "images")
			if err := os.MkdirAll(images, 0o700); err != nil {
				t.Fatal(err)
			}
			all, err := imagetest.WriteAll(images)
			if err != nil {
				t.Fatal(err)
			}
			// Three files of different bytes: a PNG, a JPEG named with
			// spaces, a GIF named in Unicode.
			spaces, unicode := filepath.Join(images, "acc image upload test.jpg"), filepath.Join(images, "acc-image-upload-test-éè-日本.gif")
			if err := os.WriteFile(spaces, rcRead(t, all["jpg"]), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(unicode, rcRead(t, all["gif"]), 0o644); err != nil {
				t.Fatal(err)
			}
			files := []string{all["png"], spaces, unicode}
			r := runRecovery(t, c.mode, base, files)
			var log []string
			for _, l := range strings.Split(r.out, "\n") {
				if strings.HasPrefix(l, "child:") || strings.Contains(l, "stopped on") || strings.Contains(l, "could not be saved") {
					log = append(log, strings.TrimSpace(l))
				}
			}
			t.Logf("exit %d (killed %v after the mark)\n  %s", r.code, r.killed, strings.Join(log, "\n  "))
			if r.code != c.code {
				t.Errorf("exit code %d, want %d", r.code, c.code)
			}
			rr := runRecovery(t, "restore", base, files)
			b, err := os.ReadFile(filepath.Join(base, "mark-restore"))
			if err != nil {
				t.Fatalf("the next run wrote nothing: %v\n%s", err, rr.out)
			}
			var got struct {
				Text  string   `json:"text"`
				Names []string `json:"names"`
				Sums  []string `json:"sums"`
				Toast string   `json:"toast"`
			}
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatal(err)
			}
			t.Logf("restored by the next run: %s", b)
			var names, sums []string
			for _, f := range files {
				sum := sha256.Sum256(rcRead(t, f))
				names, sums = append(names, filepath.Base(f)), append(sums, hex.EncodeToString(sum[:]))
			}
			sum := sha256.Sum256(pasted)
			names, sums = append(names, "pasted-image-1.png"), append(sums, hex.EncodeToString(sum[:]))
			if !slices.Equal(got.Names, names) || !slices.Equal(got.Sums, sums) {
				t.Errorf("restored images %q (%q); want %q, the same bytes", got.Names, got.Sums, names)
			}
			if got.Text != recoveryText {
				t.Errorf("restored text %q, want %q", got.Text, recoveryText)
			}
		})
	}
}
