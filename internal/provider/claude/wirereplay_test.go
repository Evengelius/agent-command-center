package claude

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// TestWireReplay replays a logging wrapper's record of a live session (the
// folder named by ACC_WIRE_LOG: in.jsonl and out.jsonl with their
// in-times.txt and out-times.txt) through the mapper, in the order the
// lines went over the wire, and names every line the mapper could not map.
// A tool for live checks (plan 25); skipped when the variable is not set.
func TestWireReplay(t *testing.T) {
	dir := os.Getenv("ACC_WIRE_LOG")
	if dir == "" {
		t.Skip("ACC_WIRE_LOG not set")
	}
	type item struct {
		at   int64
		dir  string
		line string
	}
	read := func(name, times, dir string) []item {
		f, err := os.Open(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		tf, err := os.ReadFile(filepath.Join(dir, times))
		if err != nil {
			t.Fatal(err)
		}
		var at []int64
		for _, l := range strings.Split(strings.TrimSpace(string(tf)), "\n") {
			ms, _ := strconv.ParseInt(strings.Fields(l)[0], 10, 64)
			at = append(at, ms)
		}
		var out []item
		sc := bufio.NewScanner(f)
		sc.Buffer(nil, 64<<20)
		for i := 0; sc.Scan(); i++ {
			if strings.TrimSpace(sc.Text()) == "" {
				continue
			}
			ms := int64(0)
			if i < len(at) {
				ms = at[i]
			}
			out = append(out, item{ms, name, sc.Text()})
		}
		return out
	}
	items := append(read("in.jsonl", "in-times.txt", dir), read("out.jsonl", "out-times.txt", dir)...)
	sort.SliceStable(items, func(i, j int) bool { return items[i].at < items[j].at })
	m := NewMapper()
	kinds := map[event.Kind]int{}
	unknown := 0
	for _, it := range items {
		var evs []event.Event
		switch {
		case it.dir == "in.jsonl" && strings.Contains(it.line, `"content":"/usage"`):
			evs = m.InInternal([]byte(it.line))
		case it.dir == "in.jsonl":
			evs = m.In([]byte(it.line))
		default:
			evs = m.Out([]byte(it.line))
		}
		for _, ev := range evs {
			kinds[ev.Kind]++
			if ev.Kind == event.Unrecognised {
				unknown++
				u, _ := ev.Payload.(event.Unknown)
				t.Logf("UNRECOGNISED %d at +%d ms: type %q reason %q\n  line: %.400s", unknown, it.at-items[0].at, u.Type, u.Reason, it.line)
			}
			if ev.Kind == event.Error {
				p, _ := ev.Payload.(event.Problem)
				t.Logf("ERROR at +%d ms: %s", it.at-items[0].at, p.Message)
			}
		}
	}
	var names []string
	for k, n := range kinds {
		names = append(names, string(k)+"="+strconv.Itoa(n))
	}
	sort.Strings(names)
	t.Logf("%d lines (%d in, %d out); events: %s", len(items), len(read("in.jsonl", "in-times.txt", dir)), len(read("out.jsonl", "out-times.txt", dir)), strings.Join(names, " "))
	t.Logf("unrecognised: %d", unknown)
}
