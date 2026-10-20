package input

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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

// configFile is config.json's content: nothing else in it is read, and a
// setting this version does not know is a problem.
type configFile struct {
	Keys *struct {
		Send    []string `json:"send"`
		Newline []string `json:"newline"`
	} `json:"keys"`
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
