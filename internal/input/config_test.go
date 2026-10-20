package input

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// Plan 21, item 5.1, criterion (i): no file, the default keys and no problem.
func TestLoadKeysNoFile(t *testing.T) {
	k, problem := LoadKeys(filepath.Join(t.TempDir(), "absent", "config.json"))
	if problem != "" {
		t.Errorf("problem %q for no file", problem)
	}
	if want := (Keys{Send: []string{"enter"}, Newline: []string{"ctrl+j", "shift+enter", "alt+enter"}}); !reflect.DeepEqual(k, want) {
		t.Errorf("keys %v, want %v", k, want)
	}
}

// Criterion (ii)'s file, a missing list keeping its default, and the spellings.
func TestLoadKeysValid(t *testing.T) {
	def := DefaultKeys()
	for _, c := range []struct {
		name, json string
		want       Keys
	}{
		{"criterion ii", `{"keys": {"send": ["ctrl+enter"], "newline": ["enter"]}}`,
			Keys{Send: []string{"ctrl+enter"}, Newline: []string{"enter"}}},
		{"send only", `{"keys": {"send": ["ctrl+enter"]}}`, Keys{Send: []string{"ctrl+enter"}, Newline: def.Newline}},
		{"newline only", `{"keys": {"newline": ["ctrl+j"]}}`, Keys{Send: def.Send, Newline: []string{"ctrl+j"}}},
		{"empty object", `{}`, def},
		{"empty keys", `{"keys": {}}`, def},
		{"null lists", `{"keys": {"send": null, "newline": null}}`, def},
		{"byte order mark", "\xef\xbb\xbf" + `{"keys": {"send": ["ctrl+enter"], "newline": ["enter"]}}`,
			Keys{Send: []string{"ctrl+enter"}, Newline: []string{"enter"}}},
		{"two keys", `{"keys": {"send": ["enter", "ctrl+enter"], "newline": ["shift+enter", "ctrl+j"]}}`,
			Keys{Send: []string{"enter", "ctrl+enter"}, Newline: []string{"shift+enter", "ctrl+j"}}},
	} {
		k, problem := LoadKeys(writeConfig(t, c.json))
		if problem != "" {
			t.Errorf("%s: problem %q", c.name, problem)
		}
		if !reflect.DeepEqual(k, c.want) {
			t.Errorf("%s: keys %v, want %v", c.name, k, c.want)
		}
	}
}

// Criterion (v): each kind of problem names config.json and the problem, and
// every default applies.
func TestLoadKeysProblems(t *testing.T) {
	for _, c := range []struct{ name, json, want string }{
		{"invalid JSON", `{"keys": `, `config.json: invalid JSON (unexpected EOF); default keys used`},
		{"not JSON", `send = "enter"`, `config.json: invalid JSON (invalid character 's' looking for beginning of value); default keys used`},
		{"empty file", ``, `config.json: is empty (expected a JSON object, such as {}); default keys used`},
		{"two values", `{} {}`, `config.json: invalid JSON (more than one value); default keys used`},
		{"unknown setting", `{"colours": {}}`, `config.json: unknown setting "colours"; default keys used`},
		{"unknown key setting", `{"keys": {"send": ["enter"], "quit": ["ctrl+q"]}}`, `config.json: unknown setting "quit"; default keys used`},
		{"a string, not a list", `{"keys": {"send": "enter"}}`, `config.json: keys.send must be a list of key names, not a string; default keys used`},
		{"refused key", `{"keys": {"send": ["Ctrl+S"]}}`,
			`config.json: "Ctrl+S" is not a send or newline key (use Enter with Ctrl, Alt or Shift, or Ctrl+J); default keys used`},
		{"another letter with ctrl", `{"keys": {"newline": ["ctrl+k"]}}`,
			`config.json: "ctrl+k" is not a send or newline key (use Enter with Ctrl, Alt or Shift, or Ctrl+J); default keys used`},
		{"unknown modifier", `{"keys": {"send": ["super+enter"]}}`,
			`config.json: "super+enter" is not a send or newline key (use Enter with Ctrl, Alt or Shift, or Ctrl+J); default keys used`},
		{"j with another modifier", `{"keys": {"send": ["ctrl+alt+j"]}}`,
			`config.json: "ctrl+alt+j" is not a send or newline key (use Enter with Ctrl, Alt or Shift, or Ctrl+J); default keys used`},
		{"empty name", `{"keys": {"send": [""]}}`,
			`config.json: "" is not a send or newline key (use Enter with Ctrl, Alt or Shift, or Ctrl+J); default keys used`},
		{"empty send list", `{"keys": {"send": []}}`, `config.json: keys.send is empty (list at least one key); default keys used`},
		{"empty newline list", `{"keys": {"send": ["ctrl+enter"], "newline": []}}`,
			`config.json: keys.newline is empty (list at least one key); default keys used`},
		{"a key in both lists", `{"keys": {"send": ["enter"], "newline": ["Enter"]}}`,
			`config.json: "Enter" is in both keys.send and keys.newline; default keys used`},
		{"a key in both, through a default", `{"keys": {"send": ["ctrl+j"]}}`,
			`config.json: "Ctrl+J" is in both keys.send and keys.newline; default keys used`},
		{"a key twice", `{"keys": {"send": ["ctrl+enter", "Ctrl + Enter"]}}`,
			`config.json: "Ctrl + Enter" is listed twice in keys.send; default keys used`},
	} {
		k, problem := LoadKeys(writeConfig(t, c.json))
		if problem != c.want {
			t.Errorf("%s: problem\n got %q\nwant %q", c.name, problem, c.want)
		}
		if !reflect.DeepEqual(k, DefaultKeys()) {
			t.Errorf("%s: keys %v, want the defaults", c.name, k)
		}
	}
}

// A directory at the path is a problem, not a crash and not a silent default.
func TestLoadKeysDirectory(t *testing.T) {
	k, problem := LoadKeys(t.TempDir())
	if !strings.HasPrefix(problem, "config.json: cannot be read (") || !strings.HasSuffix(problem, "; default keys used") {
		t.Errorf("problem %q for a directory", problem)
	}
	if strings.Contains(problem, os.TempDir()) {
		t.Errorf("problem %q holds the path", problem)
	}
	if !reflect.DeepEqual(k, DefaultKeys()) {
		t.Errorf("keys %v, want the defaults", k)
	}
}

// LoadKeys only reads: the file is as it was, and no file is created.
func TestLoadKeysOnlyReads(t *testing.T) {
	p := writeConfig(t, `{"keys": {"send": ["nonsense"]}}`)
	before, _ := os.ReadFile(p)
	LoadKeys(p)
	if after, _ := os.ReadFile(p); string(after) != string(before) {
		t.Errorf("file changed: %q", after)
	}
	dir := t.TempDir()
	LoadKeys(filepath.Join(dir, "config.json"))
	if es, _ := os.ReadDir(dir); len(es) != 0 {
		t.Errorf("LoadKeys created %d files", len(es))
	}
}

func TestNormaliseKey(t *testing.T) {
	for in, want := range map[string]string{
		"enter":                "enter",
		"Enter":                "enter",
		"Shift+Ctrl+Enter":     "ctrl+shift+enter",
		" ctrl + enter ":       "ctrl+enter",
		"ALT+SHIFT+CTRL+ENTER": "ctrl+alt+shift+enter",
		"shift+alt+enter":      "alt+shift+enter",
		"Ctrl+J":               "ctrl+j",
		"ctrl+ j":              "ctrl+j",
	} {
		if got, ok := NormaliseKey(in); !ok || got != want {
			t.Errorf("NormaliseKey(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}
	for _, in := range []string{"", "+", "j", "ctrl+", "ctrl+ctrl+enter", "ctrl+shift+j", "alt+j", "tab", "ctrl+enter+j", "meta+enter", "ctrl+space"} {
		if got, ok := NormaliseKey(in); ok {
			t.Errorf("NormaliseKey(%q) = %q, want it refused", in, got)
		}
	}
}

func TestDisplayKey(t *testing.T) {
	for in, want := range map[string]string{
		"enter": "Enter", "ctrl+enter": "Ctrl+Enter", "ctrl+j": "Ctrl+J",
		"ctrl+alt+shift+enter": "Ctrl+Alt+Shift+Enter", "shift+enter": "Shift+Enter",
	} {
		if got := DisplayKey(in); got != want {
			t.Errorf("DisplayKey(%q) = %q, want %q", in, got, want)
		}
	}
	send, newline := DefaultKeys().Display()
	if !reflect.DeepEqual(send, []string{"Enter"}) || !reflect.DeepEqual(newline, []string{"Ctrl+J", "Shift+Enter", "Alt+Enter"}) {
		t.Errorf("default display %v, %v", send, newline)
	}
}

func TestConfigFile(t *testing.T) {
	p, err := ConfigFile()
	if err != nil {
		t.Skip("no configuration folder:", err)
	}
	if filepath.Base(p) != "config.json" || filepath.Base(filepath.Dir(p)) != "agent-command-center" {
		t.Errorf("ConfigFile() = %q", p)
	}
}
