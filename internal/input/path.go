package input

import (
	"path"
	"strings"
)

// PathEnv is what resolving a project path needs from the system, passed in
// so resolution does no I/O and runs on every keystroke (ui-design.md,
// Questions).
type PathEnv struct {
	Windows bool   // the dashboard runs on Windows: the Windows column of INT paths
	Home    string // os.UserHomeDir
	Cwd     string // the dashboard's current directory, absolute
}

// ResolvedPath is a typed path after steps 1 to 3 of INT paths, "Validation,
// in order": trimmed, ~ expanded, WSL forms translated, classified, made
// absolute and cleaned. Exactly one of Path and Problem is set.
type ResolvedPath struct {
	Path       string // what the agent CLI would receive as its working directory
	Problem    string // why the form is refused (ui-design.md, Validation and messages)
	Warning    string // accepted, with a caution shown in amber
	Translated bool   // from a WSL drive mount, /mnt/<letter>/…
}

// Messages (ui-design.md, Validation and messages; Warnings).
const (
	msgEmpty        = "Type a project directory"
	msgOtherHome    = "Another user's home is not expanded. Type the full path"
	msgNoHome       = "~ cannot be expanded: the home directory is unknown. Type the full path"
	msgDevice       = "Not a directory path"
	msgLinuxOnWin   = `A Linux path. On Windows use C:\…, or /mnt/c/… for a Windows drive`
	msgWindowsOnNix = "A Windows path"
	warnUNC         = "a network share can be slow or disappear"
	warnWSL         = "files in Linux, used by a Windows agent CLI: slow, and unverified"
)

// ResolvePath resolves what was typed in the project field (INT paths,
// "Forms and rules"). Empty uses the current directory (amendment 4).
func ResolvePath(raw string, env PathEnv) ResolvedPath {
	s := CleanPath(raw)
	if s == "" {
		s = env.Cwd
	}
	if s == "" {
		return ResolvedPath{Problem: msgEmpty}
	}
	if strings.HasPrefix(s, "~") {
		rest := s[1:]
		if rest != "" && rest[0] != '/' && !(env.Windows && rest[0] == '\\') {
			return ResolvedPath{Problem: msgOtherHome}
		}
		if env.Home == "" {
			return ResolvedPath{Problem: msgNoHome}
		}
		s = env.Home + rest
	}
	if env.Windows {
		return resolveWindows(s, env.Cwd, true)
	}
	return resolvePOSIX(s, env.Cwd)
}

// resolveWindows follows the Windows column. relOK allows one join with the
// current directory, which must itself be absolute.
func resolveWindows(s, cwd string, relOK bool) ResolvedPath {
	var r ResolvedPath
	// A WSL drive mount: /mnt/c/dev/app is C:\dev\app.
	if len(s) >= 6 && strings.HasPrefix(s, "/mnt/") && isLetter(s[5]) && (len(s) == 6 || s[6] == '/') {
		s = strings.ToUpper(s[5:6]) + `:\` + s[min(len(s), 7):]
		r.Translated = true
	} else if s[0] == '/' && !strings.HasPrefix(s, "//") {
		return ResolvedPath{Problem: msgLinuxOnWin}
	}
	s = strings.ReplaceAll(s, "/", `\`)

	// Device paths name the same file without the prefix: \\?\C:\x is C:\x,
	// \\?\UNC\srv\share is \\srv\share; any other device is refused.
	if strings.HasPrefix(s, `\\?\`) || strings.HasPrefix(s, `\\.\`) {
		rest := s[4:]
		switch {
		case len(rest) >= 4 && strings.EqualFold(rest[:4], `UNC\`):
			s = `\\` + rest[4:]
		case isDrive(rest) && (len(rest) == 2 || rest[2] == '\\'):
			s = rest
		default:
			return ResolvedPath{Problem: msgDevice}
		}
	}

	var vol, tail string
	switch {
	case strings.HasPrefix(s, `\\`):
		parts := strings.SplitN(s[2:], `\`, 3)
		if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
			return ResolvedPath{Problem: msgDevice}
		}
		vol = `\\` + parts[0] + `\` + parts[1]
		if len(parts) == 3 {
			tail = parts[2]
		}
		switch strings.ToLower(parts[0]) {
		case "wsl$", "wsl.localhost":
			r.Warning = warnWSL
		default:
			r.Warning = warnUNC
		}
	case isDrive(s):
		if len(s) == 2 || s[2] != '\\' {
			return ResolvedPath{Problem: "Drive-relative path. Type a full path, for example " + s[:2] + `\` + s[2:]}
		}
		vol, tail = s[:2], s[3:]
	case s[0] == '\\':
		drive := "C:"
		if isDrive(cwd) {
			drive = cwd[:2]
		}
		return ResolvedPath{Problem: "Root-relative path. Type a full path with its drive, for example " + drive + s}
	default:
		if !relOK || cwd == "" {
			return ResolvedPath{Problem: msgEmpty}
		}
		return resolveWindows(strings.TrimRight(cwd, `\/`)+`\`+s, "", false)
	}
	r.Path = vol + `\` + cleanSegments(tail)
	return r
}

// cleanSegments evaluates "." and ".." (never above the root) and trims the
// trailing periods and spaces Windows drops from a segment ([MS:paths],
// "Normalize paths"), so the path shown is the one the agent receives.
func cleanSegments(tail string) string {
	var out []string
	for _, seg := range strings.Split(tail, `\`) {
		switch seg {
		case "", ".":
			continue
		case "..":
			if len(out) > 0 {
				out = out[:len(out)-1]
			}
			continue
		}
		if t := strings.TrimRight(seg, ". "); t != "" {
			seg = t
		}
		out = append(out, seg)
	}
	return strings.Join(out, `\`)
}

// resolvePOSIX follows the Linux, macOS or WSL column. Translating a Windows
// path inside WSL is inferred in INT paths and not built in the MVP.
func resolvePOSIX(s, cwd string) ResolvedPath {
	if isDrive(s) || strings.HasPrefix(s, `\\`) {
		return ResolvedPath{Problem: msgWindowsOnNix}
	}
	if !strings.HasPrefix(s, "/") {
		if cwd == "" {
			return ResolvedPath{Problem: msgEmpty}
		}
		s = cwd + "/" + s
	}
	return ResolvedPath{Path: path.Clean(s)}
}

func isLetter(c byte) bool { return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' }

func isDrive(s string) bool { return len(s) >= 2 && isLetter(s[0]) && s[1] == ':' }
