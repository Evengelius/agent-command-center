// Package licenses checks THIRD_PARTY.md against the module graph (proposal
// step 0, item 3). It has no code of its own.
package licenses

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

const root = "../.."

// permissive are the licences AGENTS.md allows.
var permissive = map[string]bool{"MIT": true, "BSD-2-Clause": true, "BSD-3-Clause": true, "Apache-2.0": true, "ISC": true}

// copyTests are the tests of a patched copy that must run and pass in
// TestThirdPartyTable (PATCHES.md, Checks): go test ./... skips the copy, a
// module of its own, and a test file left out by a build constraint, or a
// test skipped, would pass unseen there (plan 68, design D6). The console
// test is Windows only.
func copyTests(path string) []string {
	if path != "github.com/charmbracelet/ultraviolet" {
		return nil
	}
	names := []string{"TestPatchedWin32Input", "TestPatchedDriftRows"}
	if runtime.GOOS == "windows" {
		names = append(names, "TestConsoleReader")
	}
	return names
}

type module struct {
	Path, Version, Dir string
	Main, Indirect     bool
	Replace            *struct{ Path string }
}

func goCmd(t *testing.T, args ...string) []byte {
	t.Helper()
	cmd := exec.Command("go", args...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOPROXY=off", "GOFLAGS=-mod=readonly")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go %s: %v", strings.Join(args, " "), err)
	}
	return out
}

func modules(t *testing.T) []module {
	dec := json.NewDecoder(bytes.NewReader(goCmd(t, "list", "-m", "-json", "all")))
	var mods []module
	for {
		var m module
		if err := dec.Decode(&m); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		if !m.Main {
			mods = append(mods, m)
		}
	}
	return mods
}

// licenceOf classifies each licence file in the module's root folder.
func licenceOf(t *testing.T, dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("module not in the cache: %v", err)
	}
	var kinds []string
	for _, e := range entries {
		if !regexp.MustCompile(`(?i)^(licen[cs]e|copying)`).MatchString(e.Name()) {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		s := string(b)
		switch {
		case strings.Contains(s, "Apache License"):
			kinds = append(kinds, "Apache-2.0")
		case strings.Contains(s, "Permission is hereby granted, free of charge"):
			kinds = append(kinds, "MIT")
		case strings.Contains(s, "Redistribution and use") && strings.Contains(s, "Neither the name"):
			kinds = append(kinds, "BSD-3-Clause")
		case strings.Contains(s, "Redistribution and use"):
			kinds = append(kinds, "BSD-2-Clause")
		case regexp.MustCompile(`Permission to use, copy, modify, and/or distribute`).MatchString(s):
			kinds = append(kinds, "ISC")
		default:
			kinds = append(kinds, "unrecognised:"+e.Name())
		}
	}
	return kinds
}

type row struct{ version, direct, built, licence string }

func table(t *testing.T) map[string]row {
	b, err := os.ReadFile(filepath.Join(root, "THIRD_PARTY.md"))
	if err != nil {
		t.Fatal(err)
	}
	rows := map[string]row{}
	for _, line := range strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n") {
		cells := strings.Split(strings.Trim(line, "| "), " | ")
		if !strings.HasPrefix(line, "| ") || len(cells) != 5 || cells[0] == "Module" || cells[0] == "-" {
			continue
		}
		if _, dup := rows[cells[0]]; dup {
			t.Errorf("THIRD_PARTY.md: %s listed twice", cells[0])
		}
		rows[cells[0]] = row{cells[1], cells[2], cells[3], cells[4]}
	}
	return rows
}

func yes(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func TestThirdPartyTable(t *testing.T) {
	built := map[string]bool{}
	for _, p := range strings.Fields(string(goCmd(t, "list", "-deps", "-f", "{{if .Module}}{{.Module.Path}}{{end}}", "./..."))) {
		built[p] = true
	}
	rows := table(t)
	b, err := os.ReadFile(filepath.Join(root, "THIRD_PARTY.md"))
	if err != nil {
		t.Fatal(err)
	}
	doc := string(b)
	mods := modules(t)
	for _, m := range mods {
		r, ok := rows[m.Path]
		if !ok {
			t.Errorf("THIRD_PARTY.md: no row for %s %s", m.Path, m.Version)
			continue
		}
		delete(rows, m.Path)
		if r.version != m.Version {
			t.Errorf("%s: version %s in the table, %s in the graph", m.Path, r.version, m.Version)
		}
		if r.direct != yes(!m.Indirect) {
			t.Errorf("%s: direct %s in the table, %s in go.mod", m.Path, r.direct, yes(!m.Indirect))
		}
		if r.built != yes(built[m.Path]) {
			t.Errorf("%s: built %s in the table, %s by go list -deps", m.Path, r.built, yes(built[m.Path]))
		}
		if !permissive[r.licence] {
			t.Errorf("%s: licence %q is not one AGENTS.md allows", m.Path, r.licence)
		}
		if m.Dir == "" {
			// go.sum keeps only the go.mod hash of a module that provides
			// no built package, so go list gives no folder: ask the module
			// cache directly (offline; go.sum is not changed).
			var d struct{ Dir string }
			if err := json.Unmarshal(goCmd(t, "mod", "download", "-json", m.Path+"@"+m.Version), &d); err != nil {
				t.Fatal(err)
			}
			m.Dir = d.Dir
		}
		if m.Replace != nil {
			// A copy kept in the repository (THIRD_PARTY.md, Patched copies):
			// its changes are recorded beside it and the table names it.
			if _, err := os.Stat(filepath.Join(m.Dir, "PATCHES.md")); err != nil {
				t.Errorf("%s: replaced by %s without a PATCHES.md", m.Path, m.Replace.Path)
			}
			if !strings.Contains(doc, "`"+m.Replace.Path+"`") {
				t.Errorf("THIRD_PARTY.md: %s is replaced by %s, which it does not name", m.Path, m.Replace.Path)
			}
			// The copy is a module of its own, which go test ./... skips:
			// its tests of the patches (PATCHES.md, Checks) run here, each
			// shown to have passed and none skipped (plan 68, design D6).
			cmd := exec.Command("go", "test", "-count=1", "-v", ".")
			cmd.Dir = m.Dir
			cmd.Env = append(os.Environ(), "GOPROXY=off", "GOFLAGS=-mod=readonly")
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("%s: the patched copy's tests fail: %v\n%s", m.Replace.Path, err, out)
			}
			for _, name := range copyTests(m.Path) {
				if !bytes.Contains(out, []byte("--- PASS: "+name+" (")) {
					t.Errorf("%s: %s did not run and pass\n%s", m.Replace.Path, name, out)
				}
			}
			if bytes.Contains(out, []byte("--- SKIP: ")) {
				t.Errorf("%s: the patched copy skipped a test\n%s", m.Replace.Path, out)
			}
		}
		found := licenceOf(t, m.Dir)
		if !contains(found, r.licence) {
			t.Errorf("%s: licence %s in the table, licence files say %v", m.Path, r.licence, found)
		}
	}
	for p := range rows {
		t.Errorf("THIRD_PARTY.md: %s is not in the module graph", p)
	}
	t.Logf("%d modules, %d built", len(mods), len(built))
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
