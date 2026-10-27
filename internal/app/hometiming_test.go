//go:build timing && windows

package app

import (
	"testing"
	"time"
	"unsafe"

	tea "charm.land/bubbletea/v2"
	"golang.org/x/sys/windows"

	"github.com/Evengelius/agent-command-center/internal/input"
)

// The home screen's timing (plan 51, criterion 51.7), in the timing step:
// at 208x59 with Sixel, the banner running. A banner frame is timed on the
// performance counter, Go's clock stepping by 0.5 ms here (measured
// 2026-10-27), too coarse for its 2 ms target.
const (
	bannerTarget = 2 * time.Millisecond
	// pictureTarget is the picture's time on entering, from the size report
	// to its sequence written, the terminal's 40 ms to draw it (probe
	// P51, the whole screen at 208x59) apart: 300 ms in all.
	pictureTarget = 260 * time.Millisecond
)

var (
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	qpc      = kernel32.NewProc("QueryPerformanceCounter")
	qpf      = kernel32.NewProc("QueryPerformanceFrequency")
)

// perfCount is the performance counter's count.
func perfCount() int64 {
	var c int64
	qpc.Call(uintptr(unsafe.Pointer(&c)))
	return c
}

// perfSince is the time since the count start.
func perfSince(start int64) time.Duration {
	var f int64
	qpf.Call(uintptr(unsafe.Pointer(&f)))
	return time.Duration((perfCount() - start) * int64(time.Second) / f)
}

// homeTimed is the startup screen at 208x59 with Sixel, its picture drawn
// whole, the window focused: the banner running.
func homeTimed(t *testing.T) *harness {
	t.Helper()
	if _, err := input.Glyph('H', 20, false); err != nil {
		t.Skipf("GDI: %v", err)
	}
	h := projectsRig(demoProjects(5), -1)(t)
	h.send(tea.WindowSizeMsg{Width: 208, Height: 59})
	h.send(PrimaryDeviceAttributesEvent{61, 4, 6, 7, 14})
	screenNow(h)
	h.m.thumbSequence()
	h.send(tea.FocusMsg{})
	if h.m.home.banner == nil {
		t.Fatal("208x59: no banner")
	}
	return h
}

// TestBannerToFrame: each banner frame from its message through Update to
// its frame, as the program runs it, the scene's sequence built: p95
// within 2 ms (bannerTarget); and the bytes a frame sends.
func TestBannerToFrame(t *testing.T) {
	h := homeTimed(t)
	for range warmUp {
		h.m.Update(bannerMsg{})
		h.m.View()
	}
	var ds []time.Duration
	sent := 0
	for range 1000 {
		start := perfCount()
		_, cmd := h.m.Update(bannerMsg{})
		h.m.View()
		ds = append(ds, perfSince(start))
		for _, c := range batch(cmd) {
			if raw, ok := c().(tea.RawMsg); ok {
				sent += len(raw.Msg.(string))
			}
		}
	}
	p50, p95, worst := percentile(ds, 0.50), percentile(ds, 0.95), percentile(ds, 1)
	t.Logf("banner frame to frame, 208x59, %d frames after %d warm-up: p50 %v, p95 %v, max %v (target p95 ≤ %v); %d bytes a frame", len(ds), warmUp, p50, p95, worst, bannerTarget, sent/len(ds))
	if p95 > bannerTarget {
		overTarget(t, "p95 %v over the %v target", p95, bannerTarget)
	}
}

// TestHomeKeysToFrame: typing in the project field with the banner
// running, a banner frame between two keys as at the program's pace, each
// key timed from Update to its frame: p95 within 16 ms
// (keystrokeTarget).
func TestHomeKeysToFrame(t *testing.T) {
	h := homeTimed(t)
	h.send(key(tea.KeyTab, 0)) // the field
	if h.m.ui.Field != 1 {
		t.Fatalf("field %d, want the project field", h.m.ui.Field)
	}
	keys := []tea.KeyPressMsg{key('a', 0), key('b', 0), key('c', 0), key(tea.KeyBackspace, 0)}
	for i := range warmUp {
		frameTime(h, keys[i%len(keys)])
		h.m.Update(bannerMsg{})
	}
	var ds []time.Duration
	for i := range 400 {
		start := perfCount()
		h.m.Update(keys[i%len(keys)])
		h.m.View()
		ds = append(ds, perfSince(start))
		h.m.Update(bannerMsg{})
		h.m.View()
		if i%10 == 9 {
			h.m.thumbSequence() // the picture's drawing after the frames
		}
	}
	p50, p95, worst := percentile(ds, 0.50), percentile(ds, 0.95), percentile(ds, 1)
	t.Logf("key to frame in the project field, banner running, 208x59, %d keys: p50 %v, p95 %v, max %v (target p95 ≤ %v)", len(ds), p50, p95, worst, keystrokeTarget)
	if p95 > keystrokeTarget {
		overTarget(t, "p95 %v over the %v target", p95, keystrokeTarget)
	}
}

// TestHomePictureTime: entering the screen at 208x59, ten times, the
// picture's drawing whole once the size settled, timed: the wait for the
// size to settle (homeQuiet) and the drawing within 260 ms
// (pictureTarget).
func TestHomePictureTime(t *testing.T) {
	h := homeTimed(t)
	var ds []time.Duration
	for range 10 {
		h.send(tea.WindowSizeMsg{Width: 208, Height: 59})
		screenNow(h)
		h.m.home.banner = nil // entering: no scene yet
		start := perfCount()
		seq := h.m.thumbSequence()
		ds = append(ds, homeQuiet+perfSince(start))
		if len(homeImages(seq)) == 0 || h.m.home.banner == nil {
			t.Fatal("the picture not drawn whole")
		}
	}
	p50, worst := percentile(ds, 0.50), percentile(ds, 1)
	t.Logf("the picture on entering, 208x59, the size's %v wait included: p50 %v, max %v (target ≤ %v, the terminal's 40 ms apart)", homeQuiet, p50, worst, pictureTarget)
	if worst > pictureTarget {
		overTarget(t, "the worst %v over the %v target", worst, pictureTarget)
	}
}
