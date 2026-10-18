package input

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Sessions records which projects this dashboard's sessions run in, so the
// Worktrees view can mark them and refuse to merge into or remove them
// (plan 16). Each session keeps a file open in the sessions folder holding
// its project path; a file nobody holds is a session that ended and is
// deleted at the next read. It knows the dashboard's own sessions only, not
// an agent CLI started in another terminal.
type Sessions struct {
	Dir string // "" is the sessions folder beside the drafts
}

// ErrSessionsUnsupported is Live's error where an open file cannot be told
// from a closed one: outside Windows.
var ErrSessionsUnsupported = errors.New("N/A on this system")

// LiveSession is a running dashboard session and its project.
type LiveSession struct {
	ID      string
	Project string
}

// SessionLock is this session's file, held open until Release.
type SessionLock struct {
	ID   string
	f    *os.File
	path string
}

func (s Sessions) dir() (string, error) {
	if s.Dir != "" {
		return s.Dir, nil
	}
	d, err := DraftDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(d), "sessions"), nil
}

// Register records a session in project and holds its file open.
func (s Sessions) Register(project string) (*SessionLock, error) {
	dir, err := s.dir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	f, err := os.CreateTemp(dir, "session-*.lock")
	if err != nil {
		return nil, err
	}
	if _, err := f.WriteString(project); err != nil {
		f.Close()
		os.Remove(f.Name())
		return nil, err
	}
	return &SessionLock{ID: filepath.Base(f.Name()), f: f, path: f.Name()}, nil
}

// Release closes and deletes the session's file.
func (l *SessionLock) Release() error {
	if l == nil || l.f == nil {
		return nil
	}
	err := l.f.Close()
	l.f = nil
	if rerr := os.Remove(l.path); err == nil {
		err = rerr
	}
	return err
}

// Live is the sessions whose file is held open; the files nobody holds are
// deleted.
func (s Sessions) Live() ([]LiveSession, error) {
	dir, err := s.dir()
	if err != nil {
		return nil, err
	}
	es, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []LiveSession
	for _, e := range es {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".lock") {
			continue
		}
		p := filepath.Join(dir, e.Name())
		live, err := held(p)
		if err != nil {
			return nil, err
		}
		if !live {
			os.Remove(p) // a session that ended without releasing it
			continue
		}
		b, err := os.ReadFile(p)
		if err != nil || len(b) == 0 {
			continue // being written
		}
		out = append(out, LiveSession{ID: e.Name(), Project: string(b)})
	}
	return out, nil
}
