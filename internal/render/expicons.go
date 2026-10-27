package render

import (
	"path"
	"strings"

	"charm.land/lipgloss/v2"
)

// The Explorer's Nerd Font icons (plan 55, design D4): Cascadia Mono NF's
// glyphs, drawn when the dashboard runs where the font is (UI.NerdIcons,
// design D3), else the panel's own glyphs. Each glyph is one cell by the
// width table and in the Basic Multilingual Plane; its name in Nerd Fonts
// 3.2.1's glyphnames.json follows it. Each takes the place of today's
// glyph, with the same spaces after it, so the names stay where they are.

// expIcon is an icon and its colour, its styles painted with the palette:
// plain, and on the cursor row's ground.
type expIcon struct {
	glyph   string
	c       colour
	st, row *lipgloss.Style
}

// expIcons are every icon, for the palette to paint.
var expIcons []*expIcon

func newExpIcon(cp rune, c colour) *expIcon {
	i := &expIcon{glyph: string(cp), c: c, st: new(lipgloss.Style), row: new(lipgloss.Style)}
	expIcons = append(expIcons, i)
	return i
}

// paintExpIcons paints the icons' styles, as the panel's own icons are.
func paintExpIcons() {
	for _, i := range expIcons {
		paintStyle(i.st, fg(i.c))
		paintStyle(i.row, fg(i.c).Background(cTreeRow.get()))
	}
}

// The chevrons, as VS Code draws them, and a space after each.
var (
	nfClosed = string(rune(0xEAB6)) + " " // cod-chevron_right
	nfOpen   = string(rune(0xEAB4)) + " " // cod-chevron_down
)

// The folders' icons.
var (
	nfFolder     = newExpIcon(0xE5FF, cExpFolder)    // custom-folder
	nfFolderOpen = newExpIcon(0xE5FE, cExpFolder)    // custom-folder_open
	nfFolderGit  = newExpIcon(0xE5FB, cIconGit)      // custom-folder_git
	nfFolderHub  = newExpIcon(0xE5FD, cIconGitHub)   // custom-folder_github
	nfFolderNpm  = newExpIcon(0xE5FA, cIconNpm)      // custom-folder_npm
	nfFolderConf = newExpIcon(0xE5FC, cExpFolder)    // custom-folder_config
	nfFolderCode = newExpIcon(0xE5FC, cIconVSCode)   // custom-folder_config, VS Code's
	nfClaude     = newExpIcon(0xF51B, cIconClaude)   // oct-sparkle_fill (cod-sparkle is not in Cascadia Mono NF)
	nfTerminal   = newExpIcon(0xEA85, cIconGreen)    // cod-terminal
	nfBook       = newExpIcon(0xEAA4, cIconTeal)     // cod-book
	nfCode       = newExpIcon(0xEAC4, cIconCode)     // cod-code
	nfBeaker     = newExpIcon(0xEA79, cIconTest)     // cod-beaker
	nfBookmark   = newExpIcon(0xEAA5, cIconBookmark) // cod-bookmark
	nfPackage    = newExpIcon(0xEB29, cIconPackage)  // cod-package
)

// The files' icons.
var (
	nfFile       = newExpIcon(0xE64E, cExpFile)        // seti-default
	nfGo         = newExpIcon(0xE627, cIconGo)         // seti-go
	nfMarkdown   = newExpIcon(0xE609, cIconBlue)       // seti-markdown
	nfPowerShell = newExpIcon(0xE683, cIconPowerShell) // seti-powershell
	nfCmd        = newExpIcon(0xEBC4, cIconGreen)      // cod-terminal_cmd
	nfShell      = newExpIcon(0xE691, cIconGreen)      // seti-shell
	nfBinary     = newExpIcon(0xF471, cExpFile)        // oct-file_binary
	nfJSON       = newExpIcon(0xE60B, cIconYellow)     // seti-json
	nfYAML       = newExpIcon(0xE6A8, cIconPurple)     // seti-yml
	nfConfig     = newExpIcon(0xE615, cIconConfig)     // seti-config
	nfXML        = newExpIcon(0xE619, cIconOrange)     // seti-xml
	nfHTML       = newExpIcon(0xE60E, cIconRed)        // seti-html
	nfCSS        = newExpIcon(0xE614, cIconBlue)       // seti-css
	nfJS         = newExpIcon(0xE60C, cIconYellow)     // seti-javascript
	nfTS         = newExpIcon(0xE628, cIconPowerShell) // seti-typescript
	nfJava       = newExpIcon(0xE66D, cIconOrange)     // seti-java
	nfMaven      = newExpIcon(0xE674, cIconMaven)      // seti-maven
	nfGradle     = newExpIcon(0xE660, cIconTeal)       // seti-gradle
	nfPython     = newExpIcon(0xE606, cIconPython)     // seti-python
	nfSQL        = newExpIcon(0xE64D, cIconAmber)      // seti-db
	nfCSV        = newExpIcon(0xE64A, cIconGreen)      // seti-csv
	nfDocker     = newExpIcon(0xE650, cIconDocker)     // seti-docker
	nfMake       = newExpIcon(0xE673, cIconOrange)     // seti-makefile
	nfImage      = newExpIcon(0xE60D, cIconPurple)     // seti-image
	nfPDF        = newExpIcon(0xE67D, cIconRed)        // seti-pdf
	nfArchive    = newExpIcon(0xE6AA, cIconAmber)      // seti-zip
	nfLock       = newExpIcon(0xE672, cIconLock)       // seti-lock
	nfLicense    = newExpIcon(0xE60A, cIconLicense)    // seti-license
	nfGit        = newExpIcon(0xE65D, cIconGit)        // seti-git
	nfNpm        = newExpIcon(0xE616, cIconNpm)        // seti-npm
)

// expDirIcons are notable folders' icons by their name in lower case.
var expDirIcons = map[string]*expIcon{
	".git": nfFolderGit, ".github": nfFolderHub, "node_modules": nfFolderNpm,
	".vscode": nfFolderCode, ".idea": nfFolderConf, ".config": nfFolderConf, ".codex": nfFolderConf,
	".claude": nfClaude,
	"cmd":     nfTerminal, "scripts": nfTerminal, "bin": nfTerminal,
	"doc": nfBook, "docs": nfBook,
	"internal": nfCode, "src": nfCode, "lib": nfCode, "pkg": nfCode,
	"test": nfBeaker, "tests": nfBeaker, "testdata": nfBeaker, "__tests__": nfBeaker,
	"references":  nfBookmark,
	"third_party": nfPackage, "vendor": nfPackage,
}

// expNameIcons are files' icons by their whole name in lower case, before
// their extension's.
var expNameIcons = map[string]*expIcon{
	"go.mod": nfGo, "go.sum": nfGo, "go.work": nfGo,
	"license": nfLicense, "licence": nfLicense, "copying": nfLicense,
	".gitignore": nfGit, ".gitattributes": nfGit, ".gitmodules": nfGit, ".gitkeep": nfGit,
	"dockerfile": nfDocker, ".dockerignore": nfDocker, "compose.yaml": nfDocker, "compose.yml": nfDocker,
	"docker-compose.yaml": nfDocker, "docker-compose.yml": nfDocker,
	"makefile": nfMake, "gnumakefile": nfMake,
	"pom.xml": nfMaven, "gradlew": nfGradle, "gradlew.bat": nfGradle,
	"package.json": nfNpm, "package-lock.json": nfLock,
	".env": nfConfig, ".editorconfig": nfConfig,
}

// expExtIcons are files' icons by their extension in lower case.
var expExtIcons = map[string]*expIcon{
	".go": nfGo, ".md": nfMarkdown, ".markdown": nfMarkdown,
	".ps1": nfPowerShell, ".psm1": nfPowerShell, ".psd1": nfPowerShell,
	".cmd": nfCmd, ".bat": nfCmd, ".sh": nfShell, ".bash": nfShell, ".zsh": nfShell,
	".exe": nfBinary, ".dll": nfBinary, ".so": nfBinary,
	".json": nfJSON, ".jsonl": nfJSON, ".jsonc": nfJSON,
	".yaml": nfYAML, ".yml": nfYAML,
	".toml": nfConfig, ".ini": nfConfig, ".cfg": nfConfig, ".conf": nfConfig, ".properties": nfConfig,
	".xml": nfXML, ".html": nfHTML, ".htm": nfHTML, ".css": nfCSS, ".scss": nfCSS,
	".js": nfJS, ".mjs": nfJS, ".cjs": nfJS, ".ts": nfTS, ".tsx": nfTS,
	".java": nfJava, ".gradle": nfGradle, ".py": nfPython,
	".sql": nfSQL, ".db": nfSQL, ".sqlite": nfSQL, ".csv": nfCSV, ".tsv": nfCSV,
	".png": nfImage, ".jpg": nfImage, ".jpeg": nfImage, ".gif": nfImage, ".webp": nfImage,
	".bmp": nfImage, ".ico": nfImage, ".svg": nfImage, ".tif": nfImage, ".tiff": nfImage,
	".pdf": nfPDF,
	".zip": nfArchive, ".tar": nfArchive, ".gz": nfArchive, ".tgz": nfArchive, ".7z": nfArchive,
	".rar": nfArchive, ".xz": nfArchive, ".bz2": nfArchive,
	".lock": nfLock,
}

// expIconFor is an entry's icon: a notable folder's, else a folder's, open
// or closed; a file's by its name, then by its kind of name (a licence, an
// environment file, a Dockerfile), then by its extension, else a plain
// file's.
func expIconFor(name string, dir, open bool) *expIcon {
	l := strings.ToLower(name)
	if dir {
		if i := expDirIcons[l]; i != nil {
			return i
		}
		if open {
			return nfFolderOpen
		}
		return nfFolder
	}
	if i := expNameIcons[l]; i != nil {
		return i
	}
	switch {
	case strings.HasPrefix(l, "license.") || strings.HasPrefix(l, "licence."):
		return nfLicense
	case strings.HasPrefix(l, ".env."):
		return nfConfig
	case strings.HasPrefix(l, "dockerfile."):
		return nfDocker
	}
	if i := expExtIcons[path.Ext(l)]; i != nil {
		return i
	}
	return nfFile
}
