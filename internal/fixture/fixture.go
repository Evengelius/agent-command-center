// Package fixture reads the plan 1 probe recordings in testdata/probes and
// replays them through an adapter's mapper. Driver recordings have one
// {"t_ms","dir","msg"} object per line; one-shot recordings are a CLI's raw
// stdout (doc/research/probes/README.md, Recording format).
package fixture

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// Record is one line of a recording. Dir is in, out, stderr, meta or exit; a
// one-shot recording gives out records with TMs 0.
type Record struct {
	TMs int64           `json:"t_ms"`
	Dir string          `json:"dir"`
	Msg json.RawMessage `json:"msg"`
}

// Read reads a recording. A line with t_ms and dir is a driver record; any
// other line is a raw stdout line.
func Read(path string) ([]Record, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var recs []Record
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 1<<20), 16<<20)
	n := 0
	for sc.Scan() {
		n++
		b := bytes.TrimSpace(sc.Bytes())
		if len(b) == 0 {
			continue
		}
		var r Record
		if json.Unmarshal(b, &r) == nil && r.Dir != "" && r.Msg != nil {
			recs = append(recs, r)
			continue
		}
		recs = append(recs, Record{Dir: "out", Msg: append(json.RawMessage(nil), b...)})
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("%s line %d: %w", path, n+1, err)
	}
	return recs, nil
}

// Mapper is what an adapter's mapper offers: lines read from the CLI and lines
// written to it.
type Mapper interface {
	Out(line []byte) []event.Event
	In(line []byte) []event.Event
}

// Epoch is the time a replay starts at, so replays are deterministic.
var Epoch = time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)

// Replay maps every record and returns the events of each, in order, with
// Seq and At set as the collector would set them.
func Replay(provider string, recs []Record, m Mapper) [][]event.Event {
	out := make([][]event.Event, len(recs))
	var seq uint64
	for i, r := range recs {
		var evs []event.Event
		switch r.Dir {
		case "out":
			evs = m.Out(r.Msg)
		case "in":
			evs = m.In(r.Msg)
		case "meta":
			var s struct {
				Argv []string `json:"argv"`
				Cwd  string   `json:"cwd"`
			}
			_ = json.Unmarshal(r.Msg, &s)
			evs = []event.Event{{Provider: provider, Kind: event.SessionSpawned, Origin: event.Observed, Payload: event.Spawned{Argv: s.Argv, Dir: s.Cwd}, Raw: string(r.Msg)}}
		case "exit":
			var s struct {
				Code *int `json:"code"`
			}
			_ = json.Unmarshal(r.Msg, &s)
			end := event.Ended{Reason: "process exited"}
			if s.Code != nil {
				end.Code, end.HasCode = *s.Code, true
			}
			evs = []event.Event{{Provider: provider, Kind: event.SessionEnded, Origin: event.Observed, Payload: end, Raw: string(r.Msg)}}
		case "stderr":
			var s string
			_ = json.Unmarshal(r.Msg, &s)
			evs = []event.Event{{Provider: provider, Kind: event.Stderr, Origin: event.Observed, Payload: event.Line{Text: s}, Raw: string(r.Msg)}}
		default:
			evs = []event.Event{{Provider: provider, Kind: event.Unrecognised, Origin: event.Observed, Payload: event.Unknown{Type: "record/" + r.Dir}, Raw: string(r.Msg)}}
		}
		for j := range evs {
			seq++
			evs[j].Seq = seq
			evs[j].At = Epoch.Add(time.Duration(r.TMs) * time.Millisecond)
		}
		out[i] = evs
	}
	return out
}

// Flatten joins the per-record events.
func Flatten(per [][]event.Event) []event.Event {
	var all []event.Event
	for _, evs := range per {
		all = append(all, evs...)
	}
	return all
}
