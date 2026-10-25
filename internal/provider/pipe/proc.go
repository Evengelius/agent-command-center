package pipe

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Evengelius/agent-command-center/internal/event"
	"github.com/Evengelius/agent-command-center/internal/proc"
	"github.com/Evengelius/agent-command-center/internal/provider"
)

// The lifecycle both adapters' sessions run their CLI's process with (plan
// 49, design D8; audit finding 9). What a line means, what it calls for and
// what is written stay the adapter's.

// MaxLine bounds one stdout line; a longer one is skipped with an error event
// and reading continues (ASM-20).
const MaxLine = 64 << 20

// maxStderrLine bounds one stderr line kept for the error tail.
const maxStderrLine = 4 << 10

// CloseStep is each wait of the close escalation: stdin closed, then the CLI
// terminated, then its whole tree killed (reliability.md, Close).
const CloseStep = 5 * time.Second

// drainWait bounds the wait for the readers after the exit, when something
// the CLI started still holds its pipes open.
const drainWait = 2 * time.Second

// Config is what an adapter's session runs its process with.
type Config struct {
	Name     string        // the CLI's name in the messages: "Claude Code", "Codex"
	Provider string        // the provider of the events queued
	Step     time.Duration // the close escalation's step, CloseStep when 0
	Now      func() time.Time
	// Mu is the adapter's lock for its mapper and the order of the events:
	// the events Proc queues take it, as the adapter's mapped ones do.
	Mu *sync.Mutex
	// Line maps one line of stdout and does what it calls for. The reader
	// alone calls it, without Mu, and writes nothing itself: a write the
	// line calls for runs Apart. A panic in it is the session's fatal error.
	Line func(l []byte)
}

// Proc is an agent CLI's process as an adapter's session runs it: its events
// queued in order, its stdout read line by line and its stderr kept, its
// exit reported as the last event once the readers are done, its input
// written a line at a time, the line's events held until it is written, and
// its close escalated.
type Proc struct {
	c     Config
	p     proc.Process
	stdin io.WriteCloser

	wmu      sync.Mutex // one write at a time
	inClosed atomic.Bool

	q       *Queue
	exited  chan struct{}
	closing atomic.Bool
	once    sync.Once
}

// New is the process pr the adapter spawned, not yet run: Run starts it, once
// the session holds it.
func New(c Config, pr proc.Process) *Proc {
	if c.Step == 0 {
		c.Step = CloseStep
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	return &Proc{c: c, p: pr, stdin: pr.Stdin(), q: NewQueue(c.Now), exited: make(chan struct{})}
}

// Run queues session.spawned with the command line and its folder, then reads
// the process's output and waits for its exit; ctx ending kills it, the
// application's forced kill.
func (x *Proc) Run(ctx context.Context, argv []string, dir string) {
	x.q.Push(event.Event{Provider: x.c.Provider, Kind: event.SessionSpawned, Origin: event.Observed,
		Payload: event.Spawned{Argv: argv, Dir: dir}})
	go x.q.Forward()

	out, errs := make(chan struct{}), make(chan struct{})
	go func() { defer close(out); x.read(x.p.Stdout()) }()
	go func() { defer close(errs); x.readStderr(x.p.Stderr()) }()
	go x.wait(out, errs)
	go func() {
		select {
		case <-ctx.Done():
			_ = x.p.Kill()
		case <-x.exited:
		}
	}()
}

// Queue is the session's event queue, for the adapter's mapped events, which
// it queues holding Mu.
func (x *Proc) Queue() *Queue { return x.q }

// Events are the session's events; the channel closes after session.ended.
func (x *Proc) Events() <-chan event.Event { return x.q.Out() }

// Push queues one event of the adapter's own, in order with the mapped ones.
func (x *Proc) Push(ev event.Event) {
	ev.Provider = x.c.Provider
	x.c.Mu.Lock()
	x.q.Push(ev)
	x.c.Mu.Unlock()
}

// Problem queues an error of the adapter's own.
func (x *Proc) Problem(msg string) {
	x.Push(event.Event{Kind: event.Error, Origin: event.Observed, Payload: event.Problem{Message: msg}})
}

// Apart runs f in a goroutine of its own, as the reader's writes are run: the
// reader never waits on stdin, which the CLI may not be reading while its
// stdout is full (plan 48, design D3). A panic in f is the session's fatal
// error, never the end of the program (plan 48, design D5).
func (x *Proc) Apart(what string, f func()) {
	go func() {
		defer func() {
			if v := recover(); v != nil {
				x.Fatal(what+" failed", v)
			}
		}()
		f()
	}()
}

// Fatal queues a recovered panic as the session's fatal error: the dashboard
// shows it as it ends the session and closes the CLI (plan 48, design D5).
func (x *Proc) Fatal(what string, v any) {
	x.Push(event.Event{Kind: event.Error, Origin: event.Observed, Payload: event.Problem{Message: what + ": " + provider.Recovered(v), Fatal: true}})
}

// Closing says the session is closing: Close was called.
func (x *Proc) Closing() bool { return x.closing.Load() }

// InputClosed says the CLI's stdin is closed: nothing more is written.
func (x *Proc) InputClosed() bool { return x.inClosed.Load() }

// read maps stdout line by line until end of file. A panic in the mapping is
// the session's fatal error, and the rest of stdout is read and dropped, so
// the CLI never blocks on it before the dashboard closes it (plan 48, design
// D5; audit finding 6).
func (x *Proc) read(r io.Reader) {
	br := bufio.NewReaderSize(r, 64<<10)
	defer func() {
		if v := recover(); v != nil {
			x.Fatal("reading "+x.c.Name+"'s output failed", v)
			_, _ = io.Copy(io.Discard, br)
		}
	}()
	for {
		l, size, over, err := ReadLine(br, MaxLine)
		switch {
		case over:
			x.Problem(fmt.Sprintf("a line of %d bytes from %s was skipped (limit %d MiB)", size, x.c.Name, MaxLine>>20))
		case len(bytes.TrimSpace(l)) > 0:
			x.c.Line(l)
		}
		if err != nil {
			if !errors.Is(err, io.EOF) && !x.closing.Load() {
				x.Problem("reading " + x.c.Name + "'s output: " + err.Error())
			}
			return
		}
	}
}

func (x *Proc) readStderr(r io.Reader) {
	br := bufio.NewReaderSize(r, 4<<10)
	for {
		l, size, over, err := ReadLine(br, maxStderrLine)
		if over {
			l = []byte(fmt.Sprintf("(a stderr line of %d bytes, not kept)", size))
		}
		if len(bytes.TrimSpace(l)) > 0 {
			x.Push(event.Event{Kind: event.Stderr, Origin: event.Observed, Payload: event.Line{Text: string(l)}, Raw: string(l)})
		}
		if err != nil {
			return
		}
	}
}

// wait reports the exit as session.ended, the last event, once the readers
// are done. A process the CLI started and the job object did not end could
// hold the pipes open; a blocked read on Windows does not return when its
// file is closed, so after drainWait the session ends without the readers,
// saying so, and anything they read later is dropped.
func (x *Proc) wait(out, errs <-chan struct{}) {
	code, err := x.p.Wait()
	close(x.exited)
	timer := time.NewTimer(drainWait)
	for out != nil || errs != nil {
		select {
		case <-out:
			out = nil
		case <-errs:
			errs = nil
		case <-timer.C:
			x.Problem(x.c.Name + "'s output stayed open after its exit: a process it started still holds it")
			out, errs = nil, nil
		}
	}
	timer.Stop()
	x.p.Release()
	end := event.Ended{Code: code, HasCode: code >= 0}
	switch {
	case err != nil:
		end.Reason = x.c.Name + " ended: " + err.Error()
	case x.closing.Load():
		end.Reason = x.c.Name + " closed (exit code " + strconv.Itoa(code) + ")"
	default:
		end.Reason = x.c.Name + " exited with code " + strconv.Itoa(code)
	}
	x.Push(event.Event{Kind: event.SessionEnded, Origin: event.Observed, Payload: end})
	x.q.Close()
}

// Write sends one line: msg as JSON, HTML characters as they are. hold maps
// the line's own events under Mu and holds them (Queue.Hold): mapped before
// the write, a reply can never be mapped first, and they are released once
// the line is written, or dropped when it is not.
func (x *Proc) Write(msg any, hold func(l []byte) []*Item) error {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(msg); err != nil {
		return err
	}
	x.wmu.Lock()
	defer x.wmu.Unlock()
	if x.inClosed.Load() {
		return errors.New(x.c.Name + "'s input is closed")
	}
	held := hold(bytes.TrimRight(b.Bytes(), "\n"))
	_, err := x.stdin.Write(b.Bytes())
	x.q.Release(held, err == nil)
	if err != nil {
		return fmt.Errorf("not delivered to %s: %w", x.c.Name, err)
	}
	return nil
}

// Close closes stdin, then terminates the CLI, then kills its tree, each
// after the step; it returns when the process has exited or ctx ends, and
// the escalation goes on in the background.
func (x *Proc) Close(ctx context.Context) error {
	x.once.Do(func() {
		x.closing.Store(true)
		go x.escalate()
	})
	select {
	case <-x.exited:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (x *Proc) escalate() {
	// A write blocked on a full pipe holds the file: closing it waits for the
	// write, so it runs apart from the escalation.
	go func() { x.inClosed.Store(true); _ = x.stdin.Close() }()
	if x.waitExit() {
		return
	}
	_ = x.p.Terminate()
	if x.waitExit() {
		return
	}
	_ = x.p.Kill()
}

func (x *Proc) waitExit() bool {
	t := time.NewTimer(x.c.Step)
	defer t.Stop()
	select {
	case <-x.exited:
		return true
	case <-t.C:
		return false
	}
}
