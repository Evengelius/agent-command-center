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
		// Plan 22, amendment 19: the codex section is accepted and not read.
		{"a codex section alone", `{"codex": {"proxy": "http://127.0.0.1:3128"}}`, def},
		{"keys and a codex section", `{"keys": {"send": ["ctrl+enter"], "newline": ["enter"]}, "codex": {"proxy": "http://127.0.0.1:3128"}}`,
			Keys{Send: []string{"ctrl+enter"}, Newline: []string{"enter"}}},
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

// Plan 22, amendment 19. codexRefused are criterion 19.4's files and the other
// kinds of problem in the codex section, each with the problem LoadCodexProxy
// gives for it, word for word: config.json, the setting and what is wrong, and
// "; no proxy for Codex".
func codexRefused() []struct{ name, json, want string } {
	const (
		ex      = "http://127.0.0.1:3128" // the example the messages give
		noProxy = "; no proxy for Codex"
		mustBe  = "config.json: codex.proxy must be a proxy URL such as " + ex + ", not "
		empty   = "config.json: codex.proxy is empty (give a proxy URL such as " + ex + ")" + noProxy
		object  = `config.json: codex must be an object such as {"proxy": "` + ex + `"}` + noProxy
		such    = " (such as " + ex + ")" + noProxy
		onlyTo  = " must be the proxy's address only, with no path, query or fragment"
		scheme  = " must start with http:// or https://"
	)
	return []struct{ name, json, want string }{
		// Criterion 19.4.
		{"empty", `{"codex": {"proxy": ""}}`, empty},
		{"blank", `{"codex": {"proxy": " \t "}}`, empty},
		{"host and port only", `{"codex": {"proxy": "127.0.0.1:3128"}}`, `config.json: codex.proxy "127.0.0.1:3128"` + scheme + such},
		{"socks5", `{"codex": {"proxy": "socks5://127.0.0.1:1080"}}`, `config.json: codex.proxy "socks5://127.0.0.1:1080"` + scheme + such},
		{"no host", `{"codex": {"proxy": "http://"}}`, `config.json: codex.proxy "http://" has no host` + such},
		{"credentials", `{"codex": {"proxy": "http://user:secret@127.0.0.1:3128"}}`, credsProblem},
		{"a number", `{"codex": {"proxy": 3128}}`, mustBe + "a number" + noProxy},
		{"a list", `{"codex": {"proxy": ["http://127.0.0.1:3128"]}}`, mustBe + "a list" + noProxy},
		{"codex a string", `{"codex": "x"}`, object},
		{"an unknown setting", `{"codex": {"proxy": "http://127.0.0.1:3128", "timeout": 5}}`, "config.json: unknown setting codex.timeout" + noProxy},
		// The other kinds of value, of section and of address.
		{"an object", `{"codex": {"proxy": {}}}`, mustBe + "an object" + noProxy},
		{"true", `{"codex": {"proxy": true}}`, mustBe + "true or false" + noProxy},
		{"false", `{"codex": {"proxy": false}}`, mustBe + "true or false" + noProxy},
		{"null", `{"codex": {"proxy": null}}`, mustBe + "null" + noProxy},
		{"a negative number", `{"codex": {"proxy": -1}}`, mustBe + "a number" + noProxy},
		{"codex a number", `{"codex": 5}`, object},
		{"codex a list", `{"codex": ["http://127.0.0.1:3128"]}`, object},
		{"codex true", `{"codex": true}`, object},
		{"codex null", `{"codex": null}`, object},
		{"two unknown settings, the first by name", `{"codex": {"zzz": 1, "aaa": 2, "proxy": "http://127.0.0.1:3128"}}`,
			"config.json: unknown setting codex.aaa" + noProxy},
		{"an unknown setting and a bad proxy", `{"codex": {"proxy": 5, "timeout": 5}}`, "config.json: unknown setting codex.timeout" + noProxy},
		{"an unknown setting, any case", `{"codex": {"Proxy": "http://127.0.0.1:3128"}}`, "config.json: unknown setting codex.Proxy" + noProxy},
		{"an odd setting name", `{"codex": {"a b": 1}}`, `config.json: unknown setting codex."a b"` + noProxy},
		{"a path", `{"codex": {"proxy": "http://127.0.0.1:3128/proxy"}}`, `config.json: codex.proxy "http://127.0.0.1:3128/proxy"` + onlyTo + such},
		{"a query", `{"codex": {"proxy": "http://127.0.0.1:3128?a=b"}}`, `config.json: codex.proxy "http://127.0.0.1:3128?a=b"` + onlyTo + such},
		{"an empty query", `{"codex": {"proxy": "http://127.0.0.1:3128?"}}`, `config.json: codex.proxy "http://127.0.0.1:3128?"` + onlyTo + such},
		{"a fragment", `{"codex": {"proxy": "http://127.0.0.1:3128#a"}}`, `config.json: codex.proxy "http://127.0.0.1:3128#a"` + onlyTo + such},
		{"an empty fragment", `{"codex": {"proxy": "http://127.0.0.1:3128/#"}}`, `config.json: codex.proxy "http://127.0.0.1:3128/#"` + onlyTo + such},
		{"a port without a host", `{"codex": {"proxy": "http://:3128"}}`, `config.json: codex.proxy "http://:3128" has no host` + such},
		{"no slashes", `{"codex": {"proxy": "http:127.0.0.1:3128"}}`, `config.json: codex.proxy "http:127.0.0.1:3128"` + scheme + such},
		{"a scheme with no text", `{"codex": {"proxy": "://127.0.0.1:3128"}}`, `config.json: codex.proxy "://127.0.0.1:3128"` + scheme + such},
		// The value is quoted as written, so the screen can hide the user's name in a
		// path (backslashes not doubled); a quote or a control character is escaped.
		{"a file path", `{"codex": {"proxy": "C:\\Users\\me\\px.exe"}}`, `config.json: codex.proxy "C:\Users\me\px.exe"` + scheme + such},
		{"a quote in it", `{"codex": {"proxy": "socks5://a\"b"}}`, `config.json: codex.proxy "socks5://a\"b"` + scheme + such},
		{"a control character in it", `{"codex": {"proxy": "http://127.0.0.1:3128\u0007x"}}`,
			`config.json: codex.proxy "http://127.0.0.1:3128\ax" is not a valid URL` + such},
		{"a space in the host", `{"codex": {"proxy": "http://127.0.0.1 :3128"}}`, `config.json: codex.proxy "http://127.0.0.1 :3128" is not a valid URL` + such},
		{"a port that is not a number", `{"codex": {"proxy": "http://127.0.0.1:port"}}`, `config.json: codex.proxy "http://127.0.0.1:port" is not a valid URL` + such},
	}
}

// credsProblem is the one message for a value that holds a user name or a
// password; it never repeats the value.
const credsProblem = "config.json: codex.proxy holds a user name or password: give the proxy's address only; no proxy for Codex"

// credentialValues are values that hold a user name or a password, one way or
// another: every one is refused with the same message, which never repeats the
// value (criterion 19.4).
var credentialValues = []string{
	"http://user:secret@127.0.0.1:3128",
	"http://user:secret@127.0.0.1:3128/",
	"https://user:secret@127.0.0.1:3128",
	"HTTP://USER:secret@127.0.0.1:3128",
	"http://:secret@127.0.0.1:3128",
	"http://secret@127.0.0.1:3128",
	"http://user:secret@127.0.0.1:port", // not even a URL
	"http://user:secret@exa mple:3128",
	"http://user:p@secret@127.0.0.1:3128",
	"user:secret@127.0.0.1:3128", // no scheme
	"//user:secret@127.0.0.1:3128",
	"socks5://user:secret@127.0.0.1:1080",
	"http://user:secret@",
}

// Criterion 19.4: each refused value gives no proxy and one problem that names
// config.json, the setting and what is wrong, and ends "; no proxy for Codex".
func TestLoadCodexProxyRefused(t *testing.T) {
	for _, c := range codexRefused() {
		proxy, problem := LoadCodexProxy(writeConfig(t, c.json))
		if proxy != "" {
			t.Errorf("%s: proxy %q, want none", c.name, proxy)
		}
		if problem != c.want {
			t.Errorf("%s: problem\n got %q\nwant %q", c.name, problem, c.want)
		}
		if !strings.HasPrefix(problem, "config.json: ") || !strings.HasSuffix(problem, "; no proxy for Codex") || strings.Contains(problem, "\n") {
			t.Errorf("%s: problem %q is not one line of the form config.json: …; no proxy for Codex", c.name, problem)
		}
		if !strings.Contains(problem, "codex") {
			t.Errorf("%s: problem %q does not name the setting", c.name, problem)
		}
	}
	for _, v := range credentialValues {
		proxy, problem := LoadCodexProxy(writeConfig(t, `{"codex": {"proxy": `+quoteJSON(v)+`}}`))
		if proxy != "" {
			t.Errorf("%q: proxy %q, want none", v, proxy)
		}
		if problem != credsProblem {
			t.Errorf("%q: problem\n got %q\nwant %q", v, problem, credsProblem)
		}
		if strings.Contains(problem, "secret") {
			t.Errorf("%q: problem %q repeats the password", v, problem)
		}
	}
}

// quoteJSON is s as a JSON string.
func quoteJSON(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

// Criterion 19.4: the keys still load from a file whose codex section is
// wrong, and from one with a user name and a password; nothing of the section
// reaches LoadKeys' answer.
func TestLoadKeysWithAWrongCodexSection(t *testing.T) {
	want := Keys{Send: []string{"ctrl+enter"}, Newline: []string{"enter"}}
	const keys = `"keys": {"send": ["ctrl+enter"], "newline": ["enter"]}`
	for _, c := range codexRefused() {
		// The section of each refused file, next to a keys section.
		section := strings.TrimSuffix(strings.TrimPrefix(c.json, "{"), "}")
		k, problem := LoadKeys(writeConfig(t, "{"+keys+", "+section+"}"))
		if problem != "" {
			t.Errorf("%s: LoadKeys problem %q", c.name, problem)
		}
		if !reflect.DeepEqual(k, want) {
			t.Errorf("%s: keys %v, want %v", c.name, k, want)
		}
	}
}

// Criterion 19.1's file and the addresses that are accepted: the trimmed value
// comes back as written.
func TestLoadCodexProxyValid(t *testing.T) {
	for _, c := range []struct{ name, json, want string }{
		{"Px", `{"codex": {"proxy": "http://127.0.0.1:3128"}}`, "http://127.0.0.1:3128"},
		{"spaces around it", `{"codex": {"proxy": "  http://127.0.0.1:3128\t"}}`, "http://127.0.0.1:3128"},
		{"https", `{"codex": {"proxy": "https://proxy.corp.example:8443"}}`, "https://proxy.corp.example:8443"},
		{"any case", `{"codex": {"proxy": "HTTP://Proxy.Example:3128"}}`, "HTTP://Proxy.Example:3128"},
		{"a slash", `{"codex": {"proxy": "http://127.0.0.1:3128/"}}`, "http://127.0.0.1:3128/"},
		{"no port", `{"codex": {"proxy": "http://localhost"}}`, "http://localhost"},
		{"IPv6", `{"codex": {"proxy": "http://[::1]:3128"}}`, "http://[::1]:3128"},
		{"next to the keys", `{"keys": {"send": ["ctrl+enter"], "newline": ["enter"]}, "codex": {"proxy": "http://127.0.0.1:3128"}}`, "http://127.0.0.1:3128"},
		{"keys last", `{"codex": {"proxy": "http://127.0.0.1:3128"}, "keys": {}}`, "http://127.0.0.1:3128"},
		{"byte order mark", "\xef\xbb\xbf" + `{"codex": {"proxy": "http://127.0.0.1:3128"}}`, "http://127.0.0.1:3128"},
		{"set twice, the last", `{"codex": {"proxy": "http://127.0.0.1:1"}, "codex": {"proxy": "http://127.0.0.1:3128"}}`, "http://127.0.0.1:3128"},
		// The keys' problems are not the proxy's: it is independent of them.
		{"a keys problem", `{"keys": {"send": ["nonsense"]}, "codex": {"proxy": "http://127.0.0.1:3128"}}`, "http://127.0.0.1:3128"},
		{"keys of the wrong type", `{"keys": "x", "codex": {"proxy": "http://127.0.0.1:3128"}}`, "http://127.0.0.1:3128"},
		{"an unknown keys setting", `{"keys": {"quit": ["ctrl+q"]}, "codex": {"proxy": "http://127.0.0.1:3128"}}`, "http://127.0.0.1:3128"},
	} {
		proxy, problem := LoadCodexProxy(writeConfig(t, c.json))
		if proxy != c.want || problem != "" {
			t.Errorf("%s: proxy %q, problem %q; want %q and none", c.name, proxy, problem, c.want)
		}
	}
}

// LoadKeys matches the top level's names as encoding/json does, in any case, and
// refuses any other: LoadCodexProxy agrees with it on every name, so a file
// LoadKeys accepts never loses its proxy silently, and one it refuses gives no
// proxy and leaves the problem to it (criterion 19.5).
func TestLoadCodexProxyNamesAgreeWithLoadKeys(t *testing.T) {
	for _, name := range []string{"codex", "Codex", "CODEX", "cOdEx"} {
		p := writeConfig(t, `{"`+name+`": {"proxy": "http://127.0.0.1:3128"}}`)
		if _, problem := LoadKeys(p); problem != "" {
			t.Errorf("%s: LoadKeys problem %q", name, problem)
		}
		if proxy, problem := LoadCodexProxy(p); proxy != "http://127.0.0.1:3128" || problem != "" {
			t.Errorf("%s: proxy %q, problem %q; want the proxy", name, proxy, problem)
		}
		p = writeConfig(t, `{"`+name+`": "x"}`)
		if _, problem := LoadKeys(p); problem != "" {
			t.Errorf("%s: LoadKeys problem %q", name, problem)
		}
		if proxy, problem := LoadCodexProxy(p); proxy != "" || problem == "" {
			t.Errorf("%s: proxy %q, problem %q; want the codex problem", name, proxy, problem)
		}
	}
	for _, name := range []string{"codexx", "code", "codex ", "proxy", "Keys2"} {
		p := writeConfig(t, `{"`+name+`": {"proxy": "http://127.0.0.1:3128"}}`)
		if _, problem := LoadKeys(p); problem == "" {
			t.Errorf("%q: LoadKeys accepts an unknown name", name)
		}
		if proxy, problem := LoadCodexProxy(p); proxy != "" || problem != "" {
			t.Errorf("%q: proxy %q, problem %q; want none of either", name, proxy, problem)
		}
	}
}

// Criterion 19.2: no file, no codex section or no proxy in it is no proxy and no
// problem; so is a file that LoadKeys accepts and sets nothing for Codex.
func TestLoadCodexProxyNone(t *testing.T) {
	if proxy, problem := LoadCodexProxy(filepath.Join(t.TempDir(), "absent", "config.json")); proxy != "" || problem != "" {
		t.Errorf("no file: proxy %q, problem %q", proxy, problem)
	}
	for _, c := range []struct{ name, json string }{
		{"an empty object", `{}`},
		{"null", `null`},
		{"keys only", `{"keys": {"send": ["ctrl+enter"], "newline": ["enter"]}}`},
		{"an empty codex section", `{"codex": {}}`},
		{"codex next to keys", `{"keys": {"send": ["ctrl+enter"]}, "codex": {}}`},
		{"byte order mark", "\xef\xbb\xbf" + `{}`},
	} {
		p := writeConfig(t, c.json)
		if proxy, problem := LoadCodexProxy(p); proxy != "" || problem != "" {
			t.Errorf("%s: proxy %q, problem %q; want none", c.name, proxy, problem)
		}
		if _, problem := LoadKeys(p); problem != "" {
			t.Errorf("%s: LoadKeys problem %q", c.name, problem)
		}
	}
}

// Criterion 19.5: a problem with the file as a whole gives no proxy, even next
// to a good codex section, and no second problem: LoadKeys names it, as in plan
// 21.
func TestLoadCodexProxyFileProblems(t *testing.T) {
	const good = `"codex": {"proxy": "http://127.0.0.1:3128"}`
	for _, c := range []struct{ name, json, keys string }{
		{"empty file", ``, `config.json: is empty (expected a JSON object, such as {}); default keys used`},
		{"blanks only", " \n\t ", `config.json: is empty (expected a JSON object, such as {}); default keys used`},
		{"not JSON", `codex = "x"`, `config.json: invalid JSON (invalid character 'c' looking for beginning of value); default keys used`},
		{"cut short", `{` + good, `config.json: invalid JSON (unexpected EOF); default keys used`},
		{"two values", `{` + good + `} {}`, `config.json: invalid JSON (more than one value); default keys used`},
		{"text after it", `{` + good + `} x`, `config.json: invalid JSON (more than one value); default keys used`},
		{"an unknown setting", `{"colours": {}, ` + good + `}`, `config.json: unknown setting "colours"; default keys used`},
		{"an unknown setting last", `{` + good + `, "colours": {}}`, `config.json: unknown setting "colours"; default keys used`},
		// The decoder's own words, not pinned here: only their shape.
		{"a list", `[{` + good + `}]`, ``},
		{"a string", `"codex"`, ``},
		{"a number", `5`, ``},
	} {
		p := writeConfig(t, c.json)
		_, problem := LoadKeys(p)
		switch {
		case c.keys == "" && (!strings.HasPrefix(problem, "config.json: invalid JSON (") || !strings.HasSuffix(problem, "); default keys used")):
			t.Errorf("%s: LoadKeys problem %q", c.name, problem)
		case c.keys != "" && problem != c.keys:
			t.Errorf("%s: LoadKeys problem\n got %q\nwant %q", c.name, problem, c.keys)
		}
		if proxy, problem := LoadCodexProxy(p); proxy != "" || problem != "" {
			t.Errorf("%s: proxy %q, problem %q; want none of either", c.name, proxy, problem)
		}
	}
	// A folder where the file should be: LoadKeys says it cannot be read.
	dir := t.TempDir()
	if _, problem := LoadKeys(dir); !strings.HasPrefix(problem, "config.json: cannot be read (") {
		t.Errorf("a folder: LoadKeys problem %q", problem)
	}
	if proxy, problem := LoadCodexProxy(dir); proxy != "" || problem != "" {
		t.Errorf("a folder: proxy %q, problem %q; want none of either", proxy, problem)
	}
}

// Criterion 19.8: LoadCodexProxy only reads, whatever the file holds: the file
// is as it was, whatever its content, and no file or folder is created.
func TestLoadCodexProxyOnlyReads(t *testing.T) {
	for _, content := range []string{`{"codex": {"proxy": "nonsense"}}`, `{"codex": {"proxy": "http://127.0.0.1:3128"}}`, `{"codex": `, ``} {
		p := writeConfig(t, content)
		before, _ := os.ReadFile(p)
		info, _ := os.Stat(p)
		LoadCodexProxy(p)
		after, _ := os.ReadFile(p)
		now, _ := os.Stat(p)
		if string(after) != string(before) || now.Size() != info.Size() || !now.ModTime().Equal(info.ModTime()) {
			t.Errorf("file changed for %q: now %q", content, after)
		}
	}
	dir := t.TempDir()
	LoadCodexProxy(filepath.Join(dir, "config.json"))
	LoadCodexProxy(filepath.Join(dir, "absent", "config.json"))
	if es, _ := os.ReadDir(dir); len(es) != 0 {
		t.Errorf("LoadCodexProxy created %d files", len(es))
	}
}
