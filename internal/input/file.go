package input

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// The file viewer's reading and listing (plan 17): read only, inside the
// project only, credential files refused by name before they are opened.

// FileText is a file as the file viewer shows it.
type FileText struct {
	Path   string   // absolute, cleaned, as asked (not the link's target)
	Rel    string   // relative to the project, with forward slashes
	Size   int64    // bytes on disk
	Binary bool     // a NUL in the first BinaryProbe bytes: Lines is empty
	Lines  []string // split at LF, a CR before it dropped, valid UTF-8
	// Raw is the bytes read, CRs kept, for git blame (plan 60, design D5);
	// nil for a binary file.
	Raw []byte
}

// Limits (plan 17, Reading a file and Listing). Variables so tests can
// lower them.
var (
	MaxFileSize int64 = 32 << 20
	MaxListed         = 100_000
)

// BinaryProbe is how many leading bytes are searched for a NUL.
const BinaryProbe = 8 << 10

// FileError is why a file is not shown; its text is shown in the viewer.
type FileError struct{ Reason string }

func (e *FileError) Error() string { return e.Reason }

// ErrCredential refuses a credential file (AGENTS.md, Hard constraints:
// never read credential files).
var ErrCredential = &FileError{"a credential file: not opened"}

// ReadFile reads path, absolute or relative to root, the project directory.
// The checks run before the file is opened, in order: a credential name, the
// path and its links' target inside root, not a directory, not over
// MaxFileSize.
func ReadFile(root, path string) (FileText, error) {
	p := path
	if !filepath.IsAbs(p) {
		p = filepath.Join(root, p)
	}
	p = filepath.Clean(p)
	cf, _, err := checkFile(root, "", path, p)
	if err != nil {
		return FileText{}, err
	}
	real, realRoot := cf.real, cf.realRoot
	f, err := os.Open(real)
	if err != nil {
		return FileText{}, &FileError{"cannot be read: " + path}
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxFileSize+1))
	if err != nil {
		return FileText{}, &FileError{"cannot be read: " + path}
	}
	if int64(len(data)) > MaxFileSize {
		return FileText{}, &FileError{fmt.Sprintf("too large to show: more than %s", SizeText(MaxFileSize))}
	}
	rel, err := filepath.Rel(filepath.Clean(root), p)
	if err != nil || strings.HasPrefix(rel, "..") {
		rel, _ = filepath.Rel(realRoot, real)
	}
	ft := FileText{Path: p, Rel: filepath.ToSlash(rel), Size: int64(len(data))}
	if bytes.IndexByte(data[:min(len(data), BinaryProbe)], 0) >= 0 {
		ft.Binary = true
		return ft, nil
	}
	ft.Lines, ft.Raw = SplitLines(string(data)), data
	return ft, nil
}

// checkedFile is a file that passed the viewer's checks.
type checkedFile struct {
	real     string      // its path, every link and junction resolved
	realRoot string      // the project's, resolved
	info     fs.FileInfo // of real
}

// What a file the viewer's checks refuse is called in a few words, in the
// turn review's list of the files it left out (plan 65, design D2); a file
// over MaxFileSize is whyLarge's.
const (
	whyCredential = "a credential file"
	whyOutside    = "outside the project"
	whyDirectory  = "a directory"
	whyUnreadable = "cannot be read"
)

// whyLarge is why a file over MaxFileSize is left out: "over 32.0 MiB".
func whyLarge() string { return "over " + SizeText(MaxFileSize) }

// checkFile is the file viewer's checks (plan 17, Reading a file), made
// before a file is opened and in order: a credential name, the path and its
// links' target inside root, not a directory, not over MaxFileSize. p is the
// file's absolute, cleaned path and path what the refusals call it. realRoot
// is root with its links resolved, "" to resolve it here: ReadFile does, the
// turn review resolves it once for all its files (plan 65, design D2). A
// refusal is the viewer's error and, in a few words, why.
func checkFile(root, realRoot, path, p string) (cf checkedFile, why string, err error) {
	if CredentialName(filepath.Base(p)) {
		return cf, whyCredential, ErrCredential
	}
	if realRoot == "" {
		if realRoot, err = realPath(root); err != nil {
			return cf, whyUnreadable, &FileError{"the project cannot be read: " + root}
		}
	}
	if !inside(realRoot, filepath.Clean(root), p) {
		return cf, whyOutside, &FileError{"outside the project: " + path}
	}
	real, err := realPath(p)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cf, whyUnreadable, &FileError{"not found: " + path}
		}
		return cf, whyUnreadable, &FileError{"cannot be read: " + path}
	}
	if CredentialName(filepath.Base(real)) {
		return cf, whyCredential, ErrCredential
	}
	if !inside(realRoot, realRoot, real) {
		return cf, whyOutside, &FileError{"outside the project: " + path}
	}
	info, err := os.Stat(real)
	if err != nil {
		return cf, whyUnreadable, &FileError{"cannot be read: " + path}
	}
	if info.IsDir() {
		return cf, whyDirectory, &FileError{"a directory, not a file"}
	}
	if info.Size() > MaxFileSize {
		return cf, whyLarge(), &FileError{fmt.Sprintf("too large to show: %s (%s at most)", SizeText(info.Size()), SizeText(MaxFileSize))}
	}
	return checkedFile{real: real, realRoot: realRoot, info: info}, "", nil
}

// SplitLines splits text at LF, drops a CR before it and the empty line
// after a final LF, and replaces invalid UTF-8 with U+FFFD.
func SplitLines(text string) []string {
	text = strings.ToValidUTF8(text, "�")
	if text == "" {
		return nil
	}
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSuffix(l, "\r")
	}
	return lines
}

// inside reports whether p, or the path it gives once root is replaced by
// realRoot, is root or under it: case-insensitive on Windows.
func inside(realRoot, root, p string) bool {
	for _, r := range []string{realRoot, root} {
		rel, err := filepath.Rel(r, p)
		if err != nil {
			continue
		}
		if runtime.GOOS == "windows" && !strings.EqualFold(filepath.VolumeName(r), filepath.VolumeName(p)) {
			continue
		}
		if rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// SizeText is a size as the viewer shows it.
func SizeText(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KiB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d bytes", n)
}

// credentialExact, credentialExt and credentialStem name the files the viewer
// refuses (plan 17, Reading a file); compared without case.
var (
	credentialExact = map[string]bool{".credentials.json": true, "auth.json": true, "oauth_creds.json": true,
		".git-credentials": true, ".netrc": true, "_netrc": true, ".npmrc": true, ".pypirc": true,
		"id_rsa": true, "id_dsa": true, "id_ecdsa": true, "id_ed25519": true, ".env": true}
	credentialExt = map[string]bool{".pem": true, ".key": true, ".p12": true, ".pfx": true, ".jks": true,
		".keystore": true, ".kdbx": true}
	credentialStem    = map[string]bool{"credentials": true, "secret": true, "secrets": true, "token": true, "tokens": true}
	credentialStemExt = map[string]bool{"": true, ".json": true, ".yaml": true, ".yml": true, ".toml": true,
		".txt": true, ".ini": true, ".conf": true, ".cfg": true}
	envExamples = map[string]bool{".env.example": true, ".env.sample": true, ".env.template": true}
)

// CredentialName reports whether a file named name is refused as a
// credential file, by its name alone.
func CredentialName(name string) bool {
	n := strings.ToLower(name)
	if credentialExact[n] {
		return true
	}
	if strings.HasPrefix(n, ".env.") {
		return !envExamples[n]
	}
	ext := filepath.Ext(n)
	if credentialExt[ext] {
		return true
	}
	return credentialStem[strings.TrimSuffix(n, ext)] && credentialStemExt[ext]
}

// ListFiles lists the files under root for the quick open (plan 17,
// Listing): relative paths with forward slashes, sorted, without .git
// folders, credential files or anything behind a link, at most MaxListed;
// truncated says the limit cut it. A folder that cannot be read is left out.
func ListFiles(root string) (files []string, truncated bool, err error) {
	if _, err := os.Stat(root); err != nil {
		return nil, false, err
	}
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == root {
				return err
			}
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if p != root && strings.EqualFold(d.Name(), ".git") {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || CredentialName(d.Name()) {
			return nil
		}
		if len(files) == MaxListed {
			truncated = true
			return fs.SkipAll
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return nil
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(files)
	return files, truncated, err
}
