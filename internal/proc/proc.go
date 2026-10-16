// Package proc starts an agent CLI as a child process with its standard
// streams as pipes (doc/architecture.md, Threads of execution). On Windows each
// child is started suspended, put in a job object that kills the whole tree
// when the dashboard's process ends (ASM-19), then resumed, so no process of
// the tree runs outside the job.
package proc

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"sync"
)

// Spec is what to start.
type Spec struct {
	Path string   // the executable, resolved; never a .cmd shim
	Args []string // without the executable
	Dir  string   // the working directory: the chosen project
	Env  []string // the full environment; nil inherits the dashboard's
}

// Process is a started child.
type Process interface {
	Stdin() io.WriteCloser
	Stdout() io.Reader
	Stderr() io.Reader
	Pid() int
	// Wait waits for the exit; code is -1 when there is none.
	Wait() (code int, err error)
	// Terminate ends the CLI process itself.
	Terminate() error
	// Kill ends the CLI and every process it started.
	Kill() error
	// Release closes the stdout and stderr read ends.
	Release()
}

// Spawner starts processes. Tests replace it; mock mode has none.
type Spawner interface {
	Spawn(Spec) (Process, error)
}

// Exec is the real spawner.
type Exec struct{}

type process struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *os.File
	stderr *os.File
	tree   *tree
	once   sync.Once
	code   int
	err    error
}

// Spawn starts the process in its job object. Stdout and stderr are plain
// pipes, not exec's StdoutPipe: Wait would close those read ends at the exit
// and lose the last lines still in the pipe, so the reader keeps them until
// end of file, which comes once the CLI and everything it started are gone.
func (Exec) Spawn(s Spec) (Process, error) {
	if s.Path == "" {
		return nil, errors.New("no executable")
	}
	cmd := exec.Command(s.Path, s.Args...)
	cmd.Dir, cmd.Env = s.Dir, s.Env
	p := &process{cmd: cmd}
	var err error
	if p.stdin, err = cmd.StdinPipe(); err != nil {
		return nil, err
	}
	outR, outW, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		outR.Close()
		outW.Close()
		return nil, err
	}
	cmd.Stdout, cmd.Stderr = outW, errW
	p.tree, err = start(cmd)
	// The child holds its own copies of the write ends.
	outW.Close()
	errW.Close()
	if err != nil {
		outR.Close()
		errR.Close()
		return nil, err
	}
	p.stdout, p.stderr = outR, errR
	return p, nil
}

func (p *process) Stdin() io.WriteCloser { return p.stdin }
func (p *process) Stdout() io.Reader     { return p.stdout }
func (p *process) Stderr() io.Reader     { return p.stderr }

// Release closes the read ends once the readers are done or given up. On
// Windows a read already blocked on the pipe is not woken by it.
func (p *process) Release() {
	p.stdout.Close()
	p.stderr.Close()
}
func (p *process) Pid() int { return p.cmd.Process.Pid }

func (p *process) Wait() (int, error) {
	p.once.Do(func() {
		p.err = p.cmd.Wait()
		p.code = -1
		if p.cmd.ProcessState != nil {
			p.code = p.cmd.ProcessState.ExitCode()
		}
		var exit *exec.ExitError
		if errors.As(p.err, &exit) {
			p.err = nil
		}
		p.tree.close()
	})
	return p.code, p.err
}

func (p *process) Terminate() error { return p.cmd.Process.Kill() }

func (p *process) Kill() error { return p.tree.kill(p.cmd) }
