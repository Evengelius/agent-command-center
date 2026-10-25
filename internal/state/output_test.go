package state

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// TestLiveOutputTail is plan 48's criterion 48.4 (design D2; audit finding
// 7a): a running command's streamed output keeps at most MaxLiveOutput, its
// tail from a line's start, the bytes dropped counted; the completion's
// output replaces it whole.
func TestLiveOutputTail(t *testing.T) {
	b := newBench(t).running()
	b.ev(event.ToolStarted, "", event.Tool{ToolID: "c1", Name: "commandExecution"})
	var last string
	total := 0
	for i := range 200 {
		last = fmt.Sprintf("line %04d %s\n", i, strings.Repeat("x", 1024-11)) // 1 KiB
		total += len(last)
		b.ev(event.ToolProgress, "", event.Tool{ToolID: "c1", Output: last})
	}
	tc := b.s.Tools["c1"]
	if n := len(tc.Output); n > 64<<10 || !strings.HasSuffix(tc.Output, last) || !strings.HasPrefix(tc.Output, "line ") {
		t.Fatalf("output of %d bytes, starting %.12q; want at most 64 KiB from a line's start, ending with the last delta", n, tc.Output)
	}
	if tc.OutputCut+int64(len(tc.Output)) != int64(total) {
		t.Errorf("%d bytes kept and %d dropped, want %d in all", len(tc.Output), tc.OutputCut, total)
	}
	t.Logf("%d of %d bytes kept, %d dropped", len(tc.Output), total, tc.OutputCut)
	b.ev(event.ToolCompleted, "", event.Tool{ToolID: "c1", Name: "commandExecution", Output: "done\n"})
	if tc.Output != "done\n" || tc.OutputCut != 0 || tc.Status != ToolCompleted {
		t.Errorf("after the completion: %q, %d dropped, %s", tc.Output, tc.OutputCut, tc.Status)
	}

	// Where the cut falls: on a line's start as it is, else at the next
	// line, else at a character's start.
	// The cut falls right after "a\n", and another line follows: kept whole.
	exact := "a\n" + strings.Repeat("b", 100) + "\n" + strings.Repeat("c", MaxLiveOutput-101)
	tail := strings.Repeat("日", 30_000) // 3 bytes each, no line break: the cut falls inside one
	for _, c := range []struct {
		name, out, want string
	}{
		{"a line's start", exact, exact[2:]},
		{"the next line", "x" + strings.Repeat("y", 10) + "\n" + strings.Repeat("z", MaxLiveOutput-5), strings.Repeat("z", MaxLiveOutput-5)},
		{"a character's start", tail, strings.Repeat("日", MaxLiveOutput/3)},
	} {
		got, cut := liveTail(c.out)
		if got != c.want || cut != len(c.out)-len(c.want) {
			t.Errorf("%s: %d bytes kept from %.8q, %d dropped; want %d", c.name, len(got), got, cut, len(c.want))
		}
	}
}
