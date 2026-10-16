// Package mock is the mock provider (doc/reliability.md, Mock mode): it replays
// a file of normalised events through the same reducer and rendering as a live
// session. It has no process, no stdin and no network; every event it emits
// has origin mock, whatever the file says.
package mock

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
)

// ProviderID is the mock provider's id, reachable only through --mock.
const ProviderID = "mock"

// line is one event of a mock file. TMs is the time from the file's start.
type line struct {
	TMs      int64           `json:"t_ms"`
	Kind     event.Kind      `json:"kind"`
	Provider string          `json:"provider,omitempty"`
	Session  string          `json:"session,omitempty"`
	Agent    string          `json:"agent,omitempty"`
	Turn     string          `json:"turn,omitempty"`
	Item     string          `json:"item,omitempty"`
	Type     string          `json:"type,omitempty"`
	Payload  json.RawMessage `json:"payload,omitempty"`
}

// payloads are the payload types a mock file may name.
var payloads = map[string]reflect.Type{}

func init() {
	for _, p := range []any{
		event.Spawned{}, event.Ready{}, event.Started{}, event.Ended{}, event.User{}, event.Delta{},
		event.Marker{}, event.Completed{}, event.Tool{}, event.Agent{}, event.AgentKeys{}, event.Approval{},
		event.Metrics{}, event.TurnEnd{}, event.Diff{}, event.Problem{}, event.Status{}, event.Write{},
		event.Line{}, event.Unknown{},
	} {
		t := reflect.TypeOf(p)
		payloads[t.Name()] = t
	}
}

// Timed is one event and its time from the start of the file.
type Timed struct {
	After time.Duration
	Event event.Event
}

// Encode writes events as a mock file; at is the time of the first event.
func Encode(w io.Writer, evs []event.Event) error {
	if len(evs) == 0 {
		return nil
	}
	start := evs[0].At
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	for _, ev := range evs {
		l := line{TMs: ev.At.Sub(start).Milliseconds(), Kind: ev.Kind, Provider: ev.Provider, Session: ev.Session,
			Agent: ev.Agent, Turn: ev.Turn, Item: ev.Item}
		if ev.Payload != nil {
			t := reflect.TypeOf(ev.Payload)
			if payloads[t.Name()] != t {
				return fmt.Errorf("event %d: payload type %s cannot be written", ev.Seq, t)
			}
			b, err := json.Marshal(ev.Payload)
			if err != nil {
				return err
			}
			l.Type, l.Payload = t.Name(), b
		}
		if err := enc.Encode(l); err != nil {
			return err
		}
	}
	return nil
}

// Decode reads a mock file. Every event gets origin mock.
func Decode(r io.Reader) ([]Timed, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 1<<20), 64<<20)
	var out []Timed
	n := 0
	for sc.Scan() {
		n++
		b := bytes.TrimSpace(sc.Bytes())
		if len(b) == 0 {
			continue
		}
		var l line
		if err := json.Unmarshal(b, &l); err != nil {
			return nil, fmt.Errorf("line %d: %w", n, err)
		}
		if l.Kind == "" {
			return nil, fmt.Errorf("line %d: no kind", n)
		}
		ev := event.Event{Provider: l.Provider, Session: l.Session, Kind: l.Kind, Origin: event.Mock,
			Agent: l.Agent, Turn: l.Turn, Item: l.Item, Raw: string(b)}
		if l.Type != "" {
			t, ok := payloads[l.Type]
			if !ok {
				return nil, fmt.Errorf("line %d: unknown payload type %q", n, l.Type)
			}
			v := reflect.New(t)
			if err := json.Unmarshal(l.Payload, v.Interface()); err != nil {
				return nil, fmt.Errorf("line %d: %w", n, err)
			}
			ev.Payload = v.Elem().Interface()
		}
		out = append(out, Timed{After: time.Duration(l.TMs) * time.Millisecond, Event: ev})
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("line %d: %w", n+1, err)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no events")
	}
	return out, nil
}

// Load reads a mock file from disk.
func Load(path string) ([]Timed, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	evs, err := Decode(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return evs, nil
}
