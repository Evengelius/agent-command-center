package input

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

// Keys are the editor's send keys and newline keys, the only configurable
// bindings (plan 21, item 5.1; brief amendment 3): each a list of canonical
// Bubble Tea keystroke names, the same strings tea.KeyPressMsg.String gives,
// such as "enter", "ctrl+enter" or "ctrl+j". Every other key and every colour
// is fixed in the code.
type Keys struct {
	Send, Newline []string
}

// DefaultKeys are the keys in force with no file, and with a file that has a
// problem: Enter sends, Ctrl+J, Shift+Enter and Alt+Enter add a newline.
func DefaultKeys() Keys {
	return Keys{Send: []string{"enter"}, Newline: []string{"ctrl+j", "shift+enter", "alt+enter"}}
}

// WithDefaults is k with the default keys for a list that has none: the
// options of a run that gave none use the defaults.
func (k Keys) WithDefaults() Keys {
	def := DefaultKeys()
	if len(k.Send) == 0 {
		k.Send = def.Send
	}
	if len(k.Newline) == 0 {
		k.Newline = def.Newline
	}
	return k
}

// IsSend reports whether name, a keystroke name, is a send key.
func (k Keys) IsSend(name string) bool { return has(k.Send, name) }

// IsNewline reports whether name, a keystroke name, is a newline key.
func (k Keys) IsNewline(name string) bool { return has(k.Newline, name) }

func has(list []string, name string) bool {
	for _, n := range list {
		if n == name {
			return true
		}
	}
	return false
}

// Display is the keys' names as the screen shows them, in the list's order:
// "ctrl+enter" is "Ctrl+Enter", "ctrl+j" is "Ctrl+J".
func (k Keys) Display() (send, newline []string) {
	for _, n := range k.Send {
		send = append(send, DisplayKey(n))
	}
	for _, n := range k.Newline {
		newline = append(newline, DisplayKey(n))
	}
	return send, newline
}

// DisplayKey writes a canonical keystroke name for the screen.
func DisplayKey(name string) string {
	parts := strings.Split(name, "+")
	for i, p := range parts {
		if p != "" {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "+")
}

// ConfigFile is the application's own configuration file: config.json in the
// application's folder under the user's configuration folder, %APPDATA% on
// Windows (plan 21, item 5.1). Nothing writes it.
func ConfigFile() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "agent-command-center", "config.json"), nil
}

// configFile is config.json's content as LoadKeys reads it: nothing else in it
// is read, and a setting this version does not know is a problem. Codex is
// Codex's own section (plan 22, amendment 19): LoadKeys accepts it and does not
// look inside, LoadCodexProxy reads it.
type configFile struct {
	Keys *struct {
		Send    []string `json:"send"`
		Newline []string `json:"newline"`
	} `json:"keys"`
	Codex json.RawMessage `json:"codex"`
}

// keyHelp ends the message of a refused key name.
const keyHelp = "use Enter with Ctrl, Alt or Shift, or Ctrl+J"

// LoadKeys reads the keys in force from the file at path and returns a
// problem, "" when there is none. No file is the defaults and no problem. A
// list the file leaves out keeps its default. On any problem, the file cannot
// be read, is not valid JSON, sets something unknown, names a key that is not
// allowed, leaves a list empty or repeats a key, all the defaults apply and
// the problem names config.json and what is wrong. The file is only read.
func LoadKeys(path string) (Keys, string) {
	def := DefaultKeys()
	fail := func(format string, a ...any) (Keys, string) {
		return def, "config.json: " + fmt.Sprintf(format, a...) + "; default keys used"
	}
	fi, err := os.Stat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return def, ""
	case err != nil:
		return fail("cannot be read (%s)", readReason(err))
	case fi.IsDir():
		return fail("cannot be read (%s is a folder, not a file)", filepath.Base(path))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fail("cannot be read (%s)", readReason(err))
	}
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")) // Notepad's byte order mark
	var cf configFile
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cf); err != nil {
		var te *json.UnmarshalTypeError
		switch {
		case errors.Is(err, io.EOF):
			return fail("is empty (expected a JSON object, such as {})")
		case errors.As(err, &te) && te.Field != "":
			return fail("%s must be a list of key names, not a %s", te.Field, te.Value)
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			return fail("unknown setting %s", strings.TrimPrefix(err.Error(), "json: unknown field "))
		}
		return fail("invalid JSON (%s)", strings.TrimPrefix(err.Error(), "json: "))
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return fail("invalid JSON (more than one value)")
	}

	keys := def
	lists := []struct {
		name string
		set  *[]string
		got  []string
	}{{"keys.send", &keys.Send, nil}, {"keys.newline", &keys.Newline, nil}}
	if cf.Keys != nil {
		lists[0].got, lists[1].got = cf.Keys.Send, cf.Keys.Newline
	}
	for _, l := range lists {
		if l.got == nil {
			continue // missing: the default stays
		}
		if len(l.got) == 0 {
			return fail("%s is empty (list at least one key)", l.name)
		}
		names := make([]string, 0, len(l.got))
		for _, raw := range l.got {
			n, ok := NormaliseKey(raw)
			if !ok {
				return fail("%q is not a send or newline key (%s)", strings.TrimSpace(raw), keyHelp)
			}
			if has(names, n) {
				return fail("%q is listed twice in %s", strings.TrimSpace(raw), l.name)
			}
			names = append(names, n)
		}
		*l.set = names
	}
	for _, n := range keys.Send {
		if has(keys.Newline, n) {
			return fail("%q is in both keys.send and keys.newline", DisplayKey(n))
		}
	}
	return keys, ""
}

// readReason is why a file could not be read, without the path (the screen shows
// the file's name, config.json).
func readReason(err error) string {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Err.Error()
	}
	return err.Error()
}

// proxyExample is the address the messages about Codex's proxy give as an
// example: Px, the local proxy that signs in to the corporate proxy for the
// owner (plan 22, amendment 19).
const proxyExample = "http://127.0.0.1:3128"

// LoadCodexProxy reads the proxy Codex is started through from the file at
// path (plan 22, amendment 19): the string setting codex.proxy, an http:// or
// https:// address such as http://127.0.0.1:3128. It is independent of the
// keys: a problem in them does not stop the proxy, nor one in codex the keys.
//
// proxy is "" when none is in force. No file, no codex section or no proxy in
// it is no proxy and no problem. A problem in the codex section, a value that
// is not an object, an unknown setting in it, a proxy that is not a string or
// not such an address (a user name or password in it is refused and never
// repeated), sets no proxy and is returned as problem: config.json, what is
// wrong and "; no proxy for Codex". A problem with the file as a whole, which
// cannot be read, is empty, is not JSON or sets an unknown setting at its top
// level, also sets no proxy but returns no problem: LoadKeys names it, and the
// screen has one line for it. The file is only read.
func LoadCodexProxy(path string) (proxy, problem string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "" // no file, or one LoadKeys says cannot be read
	}
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")) // Notepad's byte order mark
	// The top level, with LoadKeys's own rules and names (configFile's: keep them
	// in step), so the two never disagree on what the file is; the keys are not
	// looked into, and every name is matched as encoding/json does, in any case.
	var top struct {
		Keys  json.RawMessage `json:"keys"`
		Codex json.RawMessage `json:"codex"`
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if dec.Decode(&top) != nil {
		return "", "" // empty, not JSON, not an object, or an unknown setting
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return "", "" // more than one value
	}
	if top.Codex == nil {
		return "", ""
	}

	fail := func(format string, a ...any) (string, string) {
		return "", "config.json: " + fmt.Sprintf(format, a...) + "; no proxy for Codex"
	}
	var section map[string]json.RawMessage
	if len(top.Codex) == 0 || top.Codex[0] != '{' || json.Unmarshal(top.Codex, &section) != nil {
		return fail(`codex must be an object such as {"proxy": %q}`, proxyExample)
	}
	for _, name := range slices.Sorted(maps.Keys(section)) {
		if name != "proxy" {
			return fail("unknown setting codex.%s", settingName(name))
		}
	}
	raw, ok := section["proxy"]
	if !ok {
		return "", ""
	}
	var value string
	if len(raw) == 0 || raw[0] != '"' || json.Unmarshal(raw, &value) != nil {
		return fail("codex.proxy must be a proxy URL such as %s, not %s", proxyExample, jsonKind(raw))
	}
	value = strings.TrimSpace(value)
	if why := proxyProblem(value); why != "" {
		return fail("%s", why)
	}
	return value, ""
}

// proxyProblem is why value, codex.proxy trimmed, is not an address to give
// Codex as its proxy, "" when it is one: http:// or https:// in any case, a host,
// no user name or password, no query or fragment and no path beyond a slash. The
// message quotes the value, except one that may hold a user name or password,
// which no message ever repeats.
func proxyProblem(value string) string {
	if value == "" {
		return "codex.proxy is empty (give a proxy URL such as " + proxyExample + ")"
	}
	u, err := url.Parse(value)
	if (err == nil && u.User != nil) || hasUserinfo(value) {
		return "codex.proxy holds a user name or password: give the proxy's address only"
	}
	var why string
	switch {
	case !hasHTTPScheme(value):
		why = "must start with http:// or https://"
	case err != nil:
		why = "is not a valid URL"
	case u.Hostname() == "":
		why = "has no host"
	case strings.ContainsAny(value, "?#") || (u.Path != "" && u.Path != "/"):
		why = "must be the proxy's address only, with no path, query or fragment"
	default:
		return ""
	}
	return fmt.Sprintf("codex.proxy %s %s (such as %s)", quoted(value), why, proxyExample)
}

// quoted is value in double quotes for a message: as it is when it is made of
// printable characters and holds no quote, so a path in it still reads as one
// (the screen hides the user's name in it, which doubled backslashes would
// defeat), and escaped otherwise.
func quoted(value string) string {
	if strings.IndexFunc(value, func(r rune) bool { return r == '"' || !unicode.IsPrint(r) }) >= 0 {
		return strconv.Quote(value)
	}
	return `"` + value + `"`
}

// hasHTTPScheme reports whether value starts with http:// or https://, in any case.
func hasHTTPScheme(value string) bool {
	scheme, _, ok := strings.Cut(value, "://")
	return ok && (strings.EqualFold(scheme, "http") || strings.EqualFold(scheme, "https"))
}

// hasUserinfo reports whether value, as written, has an "@" in its authority,
// before the first "/", "?" or "#" that follows any "://": what precedes it may
// be a user name or password, even in a value that is not a URL.
func hasUserinfo(value string) bool {
	_, authority, ok := strings.Cut(value, "://")
	if !ok {
		authority = value
	}
	if i := strings.IndexAny(authority, "/?#"); i >= 0 {
		authority = authority[:i]
	}
	return strings.Contains(authority, "@")
}

// jsonKind names the type of a JSON value in a message, from its first byte.
func jsonKind(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "empty"
	}
	switch raw[0] {
	case '"':
		return "a string"
	case '{':
		return "an object"
	case '[':
		return "a list"
	case 't', 'f':
		return "true or false"
	case 'n':
		return "null"
	}
	return "a number"
}

// settingName is a setting's name as a message writes it: as it is when it is
// made of letters, digits, "_" and "-", else quoted.
func settingName(name string) string {
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' {
			return strconv.Quote(name)
		}
	}
	if name == "" {
		return strconv.Quote(name)
	}
	return name
}

// NormaliseKey turns a key name as the owner wrote it into the canonical
// Bubble Tea name: lower case, spaces trimmed around each "+" part, the
// modifiers in the order ctrl, alt, shift. Only Enter with any of those
// modifiers or none, and Ctrl+J, are allowed; ok is false for anything else.
func NormaliseKey(raw string) (name string, ok bool) {
	parts := strings.Split(strings.ToLower(raw), "+")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	base := parts[len(parts)-1]
	var ctrl, alt, shift bool
	for _, p := range parts[:len(parts)-1] {
		var seen *bool
		switch p {
		case "ctrl":
			seen = &ctrl
		case "alt":
			seen = &alt
		case "shift":
			seen = &shift
		default:
			return "", false
		}
		if *seen {
			return "", false // a modifier twice
		}
		*seen = true
	}
	switch {
	case base == "enter":
	case base == "j" && ctrl && !alt && !shift:
	default:
		return "", false
	}
	var out []string
	for _, m := range []struct {
		on   bool
		name string
	}{{ctrl, "ctrl"}, {alt, "alt"}, {shift, "shift"}} {
		if m.on {
			out = append(out, m.name)
		}
	}
	return strings.Join(append(out, base), "+"), true
}
