package input

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

// fileAllAccess is FILE_ALL_ACCESS, "FA" in SDDL.
const fileAllAccess = 0x001F01FF

// entriesOf are the access list of path: whether it is protected from the
// parent's, and each entry's type, flags, mask and trustee.
func entriesOf(t *testing.T, path string) (protected bool, aces []struct {
	typ, flags uint8
	mask       uint32
	sid        string
}) {
	t.Helper()
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatalf("access list of %s: %v", path, err)
	}
	control, _, err := sd.Control()
	if err != nil {
		t.Fatal(err)
	}
	acl, _, err := sd.DACL()
	if err != nil {
		t.Fatalf("access list of %s: %v", path, err)
	}
	for i := uint32(0); acl != nil && i < uint32(acl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(acl, i, &ace); err != nil {
			t.Fatal(err)
		}
		aces = append(aces, struct {
			typ, flags uint8
			mask       uint32
			sid        string
		}{ace.Header.AceType, ace.Header.AceFlags, uint32(ace.Mask), (*windows.SID)(unsafe.Pointer(&ace.SidStart)).String()})
	}
	return control&windows.SE_DACL_PROTECTED != 0, aces
}

// TestTurnFolderAccess is criterion 65.3's access list (design D3): the
// store's folder has one entry, the current user's full control, protected
// from the parent's list; what is made in it inherits that entry alone; the
// user makes, reads and removes files and folders in it.
func TestTurnFolderAccess(t *testing.T) {
	t.Parallel()
	dir := turnProject(t, map[string]string{"a.txt": "a\n"})
	s := openTurns(t, dir)
	takeTree(t, s)
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	me := user.User.Sid.String()

	protected, aces := entriesOf(t, s.dir)
	if !protected {
		t.Error("the store's folder inherits from its parent: its list is not protected")
	}
	if len(aces) != 1 {
		t.Fatalf("the store's folder has %d entries, want the user's alone: %+v", len(aces), aces)
	}
	if a := aces[0]; a.typ != windows.ACCESS_ALLOWED_ACE_TYPE || a.sid != me || a.mask != fileAllAccess ||
		a.flags&(windows.OBJECT_INHERIT_ACE|windows.CONTAINER_INHERIT_ACE) != windows.OBJECT_INHERIT_ACE|windows.CONTAINER_INHERIT_ACE ||
		a.flags&windows.INHERITED_ACE != 0 {
		t.Errorf("the store's entry: %+v, want an allow of %#x to %s, inherited by files and folders, not itself inherited", a, fileAllAccess, me)
	}
	// The control: a folder made as os.Mkdir makes one beside it inherits the
	// parent's list, which holds more than the user's entry. (Windows protects a
	// list given at the creation; the "P" of the SDDL says it as well.)
	plain := filepath.Join(filepath.Dir(s.dir), "plain")
	if err := os.Mkdir(plain, 0o700); err != nil {
		t.Fatal(err)
	}
	if protected, aces := entriesOf(t, plain); protected || len(aces) < 2 {
		t.Logf("a plain folder has %d entries, protected %v: this test cannot tell a protected list from an inherited one here", len(aces), protected)
	}
	// What the store holds: the lock, the index, the folders, an object git wrote.
	var object string
	_ = filepath.WalkDir(filepath.Join(s.dir, turnObjects), func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && object == "" {
			object = p
		}
		return nil
	})
	if object == "" {
		t.Fatal("the store holds no object")
	}
	for _, p := range []string{filepath.Join(s.dir, lockName), filepath.Join(s.dir, turnIndex), filepath.Join(s.dir, turnObjects),
		filepath.Join(s.dir, turnHooks), object} {
		if _, aces := entriesOf(t, p); len(aces) != 1 || aces[0].sid != me || aces[0].flags&windows.INHERITED_ACE == 0 {
			t.Errorf("%s has the entries %+v, want the user's, inherited, alone", strings.TrimPrefix(p, s.dir), aces)
		}
	}

	// The user makes, reads and removes in it.
	f, sub := filepath.Join(s.dir, "probe.txt"), filepath.Join(s.dir, "probe")
	if err := os.WriteFile(f, []byte("probe"), 0o600); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(f); err != nil || string(b) != "probe" {
		t.Errorf("reading a file made in the store: %q, %v", b, err)
	}
	if err := os.Mkdir(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{f, sub} {
		if err := os.Remove(p); err != nil {
			t.Errorf("removing %s: %v", filepath.Base(p), err)
		}
	}
}

// TestTurnJunction is criterion 65.2's junction: git lists the files behind a
// junction to a folder outside the project, so the file viewer's checks are
// what keeps them out; a junction needs no privilege, unlike a symbolic link.
// The file behind it is held open with no sharing: nothing opens it.
func TestTurnJunction(t *testing.T) {
	t.Parallel()
	dir := turnProject(t, map[string]string{"near.txt": "near\n"})
	outside := t.TempDir()
	notes := writeAt(t, outside, "notes.txt", "MARKER-outside-junction\n")
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", filepath.Join(dir, "link-dir"), outside).CombinedOutput(); err != nil {
		t.Fatalf("mklink /J: %v: %s", err, out)
	}
	held, err := openExclusive(notes, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { held.Close() })

	s := openTurns(t, dir)
	entries, err := s.list(turnCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(entries, turnEntry{path: "link-dir/notes.txt", untracked: true}) {
		t.Fatalf("git does not list the file behind the junction (%+v): the checks keep nothing out here", entries)
	}
	tt := takeTree(t, s)
	info, err := os.Lstat(notes)
	if err != nil {
		t.Fatal(err)
	}
	if len(tt.Left) != 1 || tt.Left[0].Path != "link-dir/notes.txt" || tt.Left[0].Why != "outside the project" ||
		tt.Left[0].Size != info.Size() || !tt.Left[0].ModTime.Equal(info.ModTime()) {
		t.Errorf("left out %+v, want link-dir/notes.txt outside the project, %d bytes", tt.Left, info.Size())
	}
	if strings.Contains(storeObjects(t, s), "MARKER-outside-junction") {
		t.Error("the file behind the junction reached the store")
	}
	if paths := pathsIn(t, s, tt.Tree); !slices.Equal(paths, []string{"near.txt"}) {
		t.Errorf("the tree holds %v", paths)
	}
	if got, err := s.Changing(turnCtx(t)); err != nil || len(got) != 0 {
		t.Errorf("Changing lists %v, %v: a file left out and unchanged", got, err)
	}
}

// TestTurnHardLink is the review of 2026-10-29 on criterion 65.2: a
// credential file outside the project, linked inside under a harmless name,
// and a file of the project with a second name in it. The checks see a hard
// link's name alone, so a file with more than one name is left out, opened
// for its attributes alone: the credential file is held open without
// sharing, and nothing fails. Their markers are in no object the store
// holds; an ordinary file's is, so that the search proves something.
func TestTurnHardLink(t *testing.T) {
	t.Parallel()
	dir := turnProject(t, map[string]string{"a.txt": "a\n"})
	s := openTurns(t, dir)
	takeTree(t, s)
	secret := writeAt(t, t.TempDir(), ".env", "MARKER-hardlink-secret\n")
	if !CredentialName(".env") || CredentialName("notes.txt") || CredentialName("twin.txt") {
		t.Fatal("the names do not test what they should")
	}
	link := func(from, to string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Link(from, to); err != nil {
			t.Fatalf("a hard link: %v", err)
		}
	}
	link(secret, filepath.Join(dir, "notes.txt"))
	link(writeAt(t, dir, "twin.txt", "MARKER-hardlink-twin\n"), filepath.Join(dir, "sub", "twin.txt"))
	writeAt(t, dir, "plain.txt", "MARKER-hardlink-plain\n")
	held, err := openExclusive(secret, false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { held.Close() })

	tt := takeTree(t, s)
	var left []string
	for _, l := range tt.Left {
		left = append(left, l.Path+": "+l.Why)
	}
	if want := []string{"notes.txt: a hard link", "sub/twin.txt: a hard link", "twin.txt: a hard link"}; !slices.Equal(left, want) {
		t.Errorf("left out %q, want %q", left, want)
	}
	objects := storeObjects(t, s)
	for _, marker := range []string{"MARKER-hardlink-secret", "MARKER-hardlink-twin"} {
		if strings.Contains(objects, marker) {
			t.Errorf("%s reached the store", marker)
		}
	}
	if !strings.Contains(objects, "MARKER-hardlink-plain") {
		t.Error("the ordinary file is not in the store either: the search proves nothing")
	}
	if paths := pathsIn(t, s, tt.Tree); !slices.Equal(paths, []string{"a.txt", "plain.txt"}) {
		t.Errorf("the tree holds %v", paths)
	}
}

// TestTurnHeldCredential is criterion 65.2's last case: credential files held
// open with no sharing, a tracked one and an untracked one, make nothing
// fail, and are not read.
func TestTurnHeldCredential(t *testing.T) {
	t.Parallel()
	dir := turnProject(t, map[string]string{"normal.txt": "n\n", "config/auth.json": "old\n"})
	s := openTurns(t, dir)
	takeTree(t, s)
	writeAt(t, dir, "held/credentials.json", "MARKER-held-credential\n")
	for _, p := range []string{"held/credentials.json", "config/auth.json"} {
		f, err := openExclusive(filepath.Join(dir, filepath.FromSlash(p)), false)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { f.Close() })
	}
	writeAt(t, dir, "after.txt", "a\n")
	tt, err := s.Take(turnCtx(t))
	if err != nil {
		t.Fatalf("Take with credential files held open: %v", err)
	}
	if len(tt.Left) != 0 || !slices.Contains(pathsIn(t, s, tt.Tree), "after.txt") {
		t.Errorf("left out %+v; the tree holds %v", tt.Left, pathsIn(t, s, tt.Tree))
	}
	if strings.Contains(storeObjects(t, s), "MARKER-held-credential") {
		t.Error("a credential file reached the store")
	}
	if got, err := s.Changing(turnCtx(t)); err != nil || len(got) != 0 {
		t.Errorf("Changing: %v, %v", got, err)
	}
}
