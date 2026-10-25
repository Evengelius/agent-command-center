package codex

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Evengelius/agent-command-center/internal/fixture"
)

// FuzzMapperOut is plan 48's criterion 48.10 (design D5; audit finding 6):
// no lines from Codex make the mapper panic. An input is a script of lines,
// ">" before one Codex wrote, "<" before one the dashboard wrote, which gives
// the mapper its state; the seeds are every app-server recording's lines,
// sixteen at a time (P5 and P5b are codex exec's). go test runs the seeds;
// the evaluation fuzzes it for a bounded time (doc/evaluation.md, Fuzzing).
func FuzzMapperOut(f *testing.F) {
	names, err := filepath.Glob("../../../testdata/probes/p*-codex-*.jsonl")
	if err != nil || len(names) == 0 {
		f.Fatalf("recordings %v: %v", names, err)
	}
	for _, name := range names {
		if strings.Contains(name, "-codex-exec") {
			continue
		}
		recs, err := fixture.Read(name)
		if err != nil {
			f.Fatal(err)
		}
		var script [][]byte
		for _, r := range recs {
			switch r.Dir {
			case "out":
				script = append(script, append([]byte(">"), r.Msg...))
			case "in":
				script = append(script, append([]byte("<"), r.Msg...))
			}
			if len(script) == 16 {
				f.Add(bytes.Join(script, []byte("\n")))
				script = nil
			}
		}
		if len(script) > 0 {
			f.Add(bytes.Join(script, []byte("\n")))
		}
	}
	f.Fuzz(func(t *testing.T, script []byte) {
		m := NewMapper()
		for _, l := range bytes.Split(script, []byte("\n")) {
			if len(l) > 0 && l[0] == '<' {
				m.In(l[1:])
			} else {
				m.Out(bytes.TrimPrefix(l, []byte(">")))
			}
		}
	})
}
