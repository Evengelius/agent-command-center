package input

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// writeAt creates root/rel with data, and its folders.
func writeAt(t *testing.T, root, rel, data string) string {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// reason is err's text, or "" for none.
func reason(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// TestReadFile covers plan 17's criteria 1.1, 1.2 and 1.4.
func TestReadFile(t *testing.T) {
	root := t.TempDir()
	abs := writeAt(t, root, "src/main.go", "package main\r\n\r\nfunc main() {}\r\n")
	writeAt(t, root, "bad.txt", "ok \xff\xfe end\nlast")
	writeAt(t, root, "empty.txt", "")
	bin := strings.Repeat("a", BinaryProbe-1) + "\x00"
	writeAt(t, root, "img.bin", bin)
	writeAt(t, root, "late.bin", strings.Repeat("a", BinaryProbe)+"\x00")

	// 1.1: absolute and relative, CRLF, invalid UTF-8.
	for _, p := range []string{abs, "src/main.go", filepath.Join("src", "main.go")} {
		ft, err := ReadFile(root, p)
		if err != nil {
			t.Fatalf("ReadFile(%q): %v", p, err)
		}
		if want := []string{"package main", "", "func main() {}"}; !reflect.DeepEqual(ft.Lines, want) {
			t.Errorf("ReadFile(%q) lines %q, want %q", p, ft.Lines, want)
		}
		if ft.Rel != "src/main.go" || ft.Path != abs || ft.Size != 32 || ft.Binary {
			t.Errorf("ReadFile(%q) = rel %q path %q size %d binary %v", p, ft.Rel, ft.Path, ft.Size, ft.Binary)
		}
		// The bytes read, CRs kept, for git blame (plan 60, design D5).
		if want := "package main\r\n\r\nfunc main() {}\r\n"; string(ft.Raw) != want {
			t.Errorf("ReadFile(%q) raw %q, want the bytes on disk %q", p, ft.Raw, want)
		}
	}
	ft, err := ReadFile(root, "bad.txt")
	if err != nil || !reflect.DeepEqual(ft.Lines, []string{"ok � end", "last"}) {
		t.Errorf("bad.txt: %q, %v", ft.Lines, err)
	}
	if ft, err := ReadFile(root, "empty.txt"); err != nil || len(ft.Lines) != 0 || ft.Binary {
		t.Errorf("empty.txt: %q, %v", ft.Lines, err)
	}

	// 1.4: a NUL in the first 8 KiB is binary; after them it is not.
	if ft, err := ReadFile(root, "img.bin"); err != nil || !ft.Binary || ft.Lines != nil || ft.Raw != nil || ft.Size != BinaryProbe {
		t.Errorf("img.bin: binary %v, %d lines, %d bytes kept, size %d, %v", ft.Binary, len(ft.Lines), len(ft.Raw), ft.Size, err)
	}
	if ft, err := ReadFile(root, "late.bin"); err != nil || ft.Binary {
		t.Errorf("late.bin: binary %v, %v", ft.Binary, err)
	}

	// 1.2: refusals, each with its reason.
	outside := writeAt(t, t.TempDir(), "other.go", "package other\n")
	big := writeAt(t, root, "big.txt", strings.Repeat("x", 101))
	defer func(n int64) { MaxFileSize = n }(MaxFileSize)
	MaxFileSize = 100
	for _, c := range []struct{ path, want string }{
		{outside, "outside the project: " + outside},
		{"../escape.go", "outside the project: ../escape.go"},
		{"src", "a directory, not a file"},
		{big, "too large to show: 101 bytes (100 bytes at most)"},
		{"missing.go", "not found: missing.go"},
	} {
		if _, err := ReadFile(root, c.path); reason(err) != c.want {
			t.Errorf("ReadFile(%q) = %q, want %q", c.path, reason(err), c.want)
		}
	}

	// A link inside the project to a file outside it, where links can be made.
	link := filepath.Join(root, "link.go")
	if err := os.Symlink(outside, link); err != nil {
		t.Logf("symbolic link not made, its case is skipped: %v", err)
	} else if _, err := ReadFile(root, "link.go"); reason(err) != "outside the project: link.go" {
		t.Errorf("link to outside: %q", reason(err))
	}
	// A link to a file inside the project reads.
	if err := os.Symlink(abs, filepath.Join(root, "inner.go")); err == nil {
		if ft, err := ReadFile(root, "inner.go"); err != nil || len(ft.Lines) != 3 || ft.Rel != "inner.go" {
			t.Errorf("link inside: %+v, %v", ft, err)
		}
	}
	// A link with an ordinary name to a credential file is refused.
	writeAt(t, root, ".env", "KEY=value\n")
	if err := os.Symlink(filepath.Join(root, ".env"), filepath.Join(root, "settings.txt")); err == nil {
		if _, err := ReadFile(root, "settings.txt"); !errors.Is(err, ErrCredential) {
			t.Errorf("link to .env: %q", reason(err))
		}
	}
}

// TestCredentialName covers plan 17's criterion 1.3: every name of the plan
// is refused, its neighbours are not, and an existing credential file is
// refused before it is opened.
func TestCredentialName(t *testing.T) {
	refused := []string{".credentials.json", "auth.json", "oauth_creds.json", ".git-credentials", ".netrc",
		"_netrc", ".npmrc", ".pypirc", "id_rsa", "id_dsa", "id_ecdsa", "id_ed25519", ".env", ".env.local",
		".env.production", "server.pem", "tls.key", "cert.p12", "cert.pfx", "store.jks", "app.keystore",
		"vault.kdbx", "credentials", "credentials.json", "secret", "secret.yaml", "secrets.yml", "secrets.toml",
		"token", "token.txt", "tokens.json", "secrets.ini", "token.conf", "secret.cfg", "AUTH.JSON", "ID_RSA",
		".ENV", "Credentials.JSON"}
	for _, n := range refused {
		if !CredentialName(n) {
			t.Errorf("CredentialName(%q) = false, want true", n)
		}
	}
	allowed := []string{"token.go", "secrets.go", ".env.example", ".env.sample", ".env.template", "auth.go",
		"id_rsa.pub", "keys.go", "main.go", "tokenizer.json", "README.md", "env.go", "authors.json"}
	for _, n := range allowed {
		if CredentialName(n) {
			t.Errorf("CredentialName(%q) = true, want false", n)
		}
	}

	// An existing credential file is refused, before any open: a file that
	// cannot be opened would give "cannot be read" instead.
	root := t.TempDir()
	writeAt(t, root, "auth.json", `{"never":"read"}`)
	if _, err := ReadFile(root, "auth.json"); !errors.Is(err, ErrCredential) || err.Error() != "a credential file: not opened" {
		t.Errorf("auth.json: %q", reason(err))
	}
	if _, err := ReadFile(root, "missing/.env"); !errors.Is(err, ErrCredential) {
		t.Errorf("a credential name that does not exist is refused by name: %q", reason(err))
	}
}

// TestListFiles covers plan 17's criterion 1.5.
func TestListFiles(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"b.go", "a/z.ts", "a/b/Main.java", ".git/config", "sub/.git/HEAD", ".env",
		"a/id_rsa", "a/.env.example", "README.md"} {
		writeAt(t, root, rel, "x")
	}
	outside := t.TempDir()
	writeAt(t, outside, "far.go", "x")
	linked := os.Symlink(filepath.Join(outside, "far.go"), filepath.Join(root, "far.go")) == nil
	if !linked {
		t.Logf("symbolic links not made: the link case is skipped")
	}
	_ = os.Symlink(outside, filepath.Join(root, "farDir"))

	files, truncated, err := ListFiles(root)
	if err != nil || truncated {
		t.Fatalf("ListFiles: truncated %v, %v", truncated, err)
	}
	want := []string{"README.md", "a/.env.example", "a/b/Main.java", "a/z.ts", "b.go"}
	if !reflect.DeepEqual(files, want) {
		t.Errorf("ListFiles = %q, want %q", files, want)
	}

	defer func(n int) { MaxListed = n }(MaxListed)
	MaxListed = 3
	files, truncated, err = ListFiles(root)
	if err != nil || !truncated || len(files) != 3 {
		t.Errorf("limited: %q, truncated %v, %v", files, truncated, err)
	}
	MaxListed = 5
	if files, truncated, _ = ListFiles(root); truncated || len(files) != 5 {
		t.Errorf("exactly at the limit: %q, truncated %v", files, truncated)
	}

	if _, _, err := ListFiles(filepath.Join(root, "missing")); err == nil {
		t.Errorf("a missing root gives no error")
	}
}
