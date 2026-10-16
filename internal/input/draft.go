package input

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Drafts keeps the editor's text on disk, one file per project, so a crash,
// a panic or a closed window loses no draft (proposal step 4, item 4;
// ASM-23). The directory is the user's own (under %LOCALAPPDATA% on
// Windows), created readable by its owner only; a draft is deleted only once
// its message was accepted.
type Drafts struct {
	Dir string // "" is DraftDir()
}

// DraftDir is where drafts live: agent-command-center\drafts in the user's
// cache directory, %LOCALAPPDATA% on Windows.
func DraftDir() (string, error) {
	d, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "agent-command-center", "drafts"), nil
}

func (d Drafts) dir() (string, error) {
	if d.Dir != "" {
		return d.Dir, nil
	}
	return DraftDir()
}

// Path is the file a project's draft is kept in: the SHA-256 of the cleaned
// project path, lower-cased on Windows where paths ignore case. The project
// path itself is not written anywhere.
func (d Drafts) Path(project string) (string, error) {
	dir, err := d.dir()
	if err != nil {
		return "", err
	}
	p := filepath.Clean(project)
	if runtime.GOOS == "windows" {
		p = strings.ToLower(p)
	}
	sum := sha256.Sum256([]byte(p))
	return filepath.Join(dir, hex.EncodeToString(sum[:])+".txt"), nil
}

// Load reads a project's draft; no draft is "" without an error.
func (d Drafts) Load(project string) (string, error) {
	p, err := d.Path(project)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	return string(b), err
}

// Save writes a project's draft byte for byte, through a temporary file
// renamed over the old one, so a crash mid-write leaves the previous draft.
// An empty text deletes the draft.
func (d Drafts) Save(project, text string) error {
	if text == "" {
		return d.Delete(project)
	}
	p, err := d.Path(project)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(p), ".draft-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	_, werr := f.WriteString(text)
	if err := errors.Join(werr, f.Sync(), f.Close()); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, p); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// Delete removes a project's draft; no draft is not an error.
func (d Drafts) Delete(project string) error {
	p, err := d.Path(project)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
