// Package arch checks the separation of the five parts (proposal step 0,
// item 4; test 7). It has no code of its own.
package arch

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const module = "github.com/Evengelius/agent-command-center"

// pkg is one module package as its non-test files declare it.
type pkg struct {
	imports []string
	clock   []string // calls that read or wait on the clock, file:line
}

// clockCalls read or wait on the clock; the reducer may use time's types only.
var clockCalls = map[string]bool{"Now": true, "Since": true, "Until": true, "Sleep": true, "After": true,
	"AfterFunc": true, "Tick": true, "NewTimer": true, "NewTicker": true}

// load parses every package under root (module path module), non-test files
// only, whatever their build tags.
func load(t *testing.T, root string) map[string]*pkg {
	t.Helper()
	pkgs := map[string]*pkg{}
	err := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if n := d.Name(); p != root && (n == "testdata" || n == "doc" || n == "references" || strings.HasPrefix(n, ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(root, filepath.Dir(p))
		path := module
		if rel != "." {
			path += "/" + filepath.ToSlash(rel)
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, p, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		k := pkgs[path]
		if k == nil {
			k = &pkg{}
			pkgs[path] = k
		}
		timeName := ""
		for _, im := range f.Imports {
			ip, _ := strconv.Unquote(im.Path.Value)
			k.imports = append(k.imports, ip)
			if ip == "time" {
				timeName = "time"
				if im.Name != nil {
					timeName = im.Name.Name
				}
			}
		}
		if timeName != "" {
			ast.Inspect(f, func(n ast.Node) bool {
				if sel, ok := n.(*ast.SelectorExpr); ok {
					if x, ok := sel.X.(*ast.Ident); ok && x.Name == timeName && clockCalls[sel.Sel.Name] {
						k.clock = append(k.clock, filepath.Base(p)+":"+strconv.Itoa(fset.Position(sel.Pos()).Line)+" time."+sel.Sel.Name)
					}
				}
				return true
			})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return pkgs
}

// closure is p and every module package it reaches.
func closure(pkgs map[string]*pkg, p string) []string {
	seen := map[string]bool{}
	var walk func(string)
	walk = func(q string) {
		if seen[q] || pkgs[q] == nil {
			return
		}
		seen[q] = true
		for _, im := range pkgs[q].imports {
			walk(im)
		}
	}
	walk(p)
	var out []string
	for q := range seen {
		out = append(out, q)
	}
	sort.Strings(out)
	return out
}

func under(p, dir string) bool {
	return p == module+"/"+dir || strings.HasPrefix(p, module+"/"+dir+"/")
}

// forbiddenIO are the I/O, process, network and system packages the reducer
// may not import (directly, or through a module package it reaches).
func forbiddenIO(ip string) bool {
	switch ip {
	case "os", "os/exec", "os/signal", "io", "io/fs", "io/ioutil", "bufio", "net", "syscall", "path/filepath", "unsafe":
		return true
	}
	return strings.HasPrefix(ip, "net/") || strings.HasPrefix(ip, "golang.org/x/sys/")
}

// violations checks the three rules of test 7.
func violations(pkgs map[string]*pkg) []string {
	var out []string
	rule := func(part, dir string, bad func(string) bool, why string) {
		root := module + "/" + dir
		if pkgs[root] == nil {
			out = append(out, part+": package "+dir+" not found")
			return
		}
		for _, q := range closure(pkgs, root) {
			if q != root && bad(q) {
				out = append(out, part+" reaches "+strings.TrimPrefix(q, module+"/")+" ("+why+")")
			}
		}
	}
	// Rendering imports no adapter and no process package.
	rule("render", "internal/render", func(q string) bool {
		return under(q, "internal/provider") || under(q, "internal/proc") || under(q, "internal/app")
	}, "an adapter or process package")
	// The editor imports no provider package.
	rule("input", "internal/input", func(q string) bool {
		return under(q, "internal/provider") || under(q, "internal/proc") || under(q, "internal/app")
	}, "a provider package")
	// The reducer: no I/O, process, network or clock, in any module package
	// it reaches.
	rule("state", "internal/state", func(q string) bool {
		return under(q, "internal/provider") || under(q, "internal/proc") || under(q, "internal/app") || under(q, "internal/render")
	}, "an effectful package")
	if pkgs[module+"/internal/state"] != nil {
		for _, q := range closure(pkgs, module+"/internal/state") {
			name := strings.TrimPrefix(q, module+"/")
			for _, ip := range pkgs[q].imports {
				if forbiddenIO(ip) {
					out = append(out, "state: "+name+" imports "+ip+" (I/O, process or network)")
				}
			}
			for _, c := range pkgs[q].clock {
				out = append(out, "state: "+name+" reads the clock: "+c)
			}
		}
	}
	sort.Strings(out)
	return out
}

func TestSeparation(t *testing.T) {
	pkgs := load(t, "../..")
	for _, v := range violations(pkgs) {
		t.Error(v)
	}
	t.Logf("%d module packages; render reaches %d, input %d, state %d", len(pkgs),
		len(closure(pkgs, module+"/internal/render")), len(closure(pkgs, module+"/internal/input")),
		len(closure(pkgs, module+"/internal/state")))
}

// TestSeparationCatchesPlantedFaults plants one fault of each kind, direct
// and transitive, and checks every one is reported.
func TestSeparationCatchesPlantedFaults(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"internal/proc/p.go":              "package proc\n",
		"internal/provider/claude/c.go":   "package claude\n",
		"internal/event/e.go":             "package event\nimport \"os\"\nvar _ = os.Args\n",
		"internal/helper/h.go":            "package helper\nimport _ \"" + module + "/internal/proc\"\n",
		"internal/render/r.go":            "package render\nimport _ \"" + module + "/internal/helper\"\n",
		"internal/input/i.go":             "package input\nimport _ \"" + module + "/internal/provider/claude\"\n",
		"internal/state/s.go":             "package state\nimport (\n\tclk \"time\"\n\t_ \"" + module + "/internal/event\"\n)\nvar _ = clk.Now()\n",
		"internal/state/s_test.go":        "package state\nimport \"os\"\nvar _ = os.Args\n",
		"internal/state/net.go":           "package state\nimport _ \"net/http\"\n",
		"internal/state/ok.go":            "package state\nimport \"time\"\nvar _ time.Duration\n",
		"testdata/ignored/x.go":           "package x\nimport _ \"os\"\n",
		"internal/provider/claude/doc.go": "package claude\n",
	}
	for name, src := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := strings.Join(violations(load(t, root)), "\n")
	want := []string{
		"render reaches internal/proc",
		"input reaches internal/provider/claude",
		"state: internal/event imports os",
		"state: internal/state imports net/http",
		"state: internal/state reads the clock: s.go:6 time.Now",
	}
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("planted fault not reported: %q\nreported:\n%s", w, got)
		}
	}
	if strings.Contains(got, "s_test.go") || strings.Contains(got, "ok.go") || strings.Contains(got, "testdata") {
		t.Errorf("reported a file the rules allow:\n%s", got)
	}
	if n := len(strings.Split(got, "\n")); n != len(want) {
		t.Errorf("%d violations, want %d:\n%s", n, len(want), got)
	}
}
