package render

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"charm.land/lipgloss/v2"
)

// Plan 55, design D4: each name to its icon and colour, the expected code
// points and colours written out here rather than read from the table.
func TestExpIconFor(t *testing.T) {
	for _, c := range []struct {
		name      string
		dir, open bool
		cp        rune
		c         colour
	}{
		{"internal", true, false, 0xEAC4, cIconCode},
		{"cmd", true, true, 0xEA85, cIconGreen},
		{"scripts", true, false, 0xEA85, cIconGreen},
		{"doc", true, false, 0xEAA4, cIconTeal},
		{"testdata", true, false, 0xEA79, cIconTest},
		{"references", true, false, 0xEAA5, cIconBookmark},
		{"third_party", true, false, 0xEB29, cIconPackage},
		{".git", true, false, 0xE5FB, cIconGit},
		{".GitHub", true, true, 0xE5FD, cIconGitHub},
		{"node_modules", true, false, 0xE5FA, cIconNpm},
		{".vscode", true, false, 0xE5FC, cIconVSCode},
		{".claude", true, false, 0xF51B, cIconClaude},
		{"anything", true, false, 0xE5FF, cExpFolder},
		{"anything", true, true, 0xE5FE, cExpFolder},
		{"main.go", false, false, 0xE627, cIconGo},
		{"main_test.go", false, false, 0xE627, cIconGo},
		{"go.mod", false, false, 0xE627, cIconGo},
		{"go.sum", false, false, 0xE627, cIconGo},
		{"AGENTS.md", false, false, 0xE609, cIconBlue},
		{"acc.ps1", false, false, 0xE683, cIconPowerShell},
		{"acc.cmd", false, false, 0xEBC4, cIconGreen},
		{"install.sh", false, false, 0xE691, cIconGreen},
		{"agent-command-center.exe", false, false, 0xF471, cExpFile},
		{"config.json", false, false, 0xE60B, cIconYellow},
		{"events.jsonl", false, false, 0xE60B, cIconYellow},
		{"compose.yaml", false, false, 0xE650, cIconDocker},
		{"ci.yml", false, false, 0xE6A8, cIconPurple},
		{"settings.toml", false, false, 0xE615, cIconConfig},
		{".env", false, false, 0xE615, cIconConfig},
		{".env.local", false, false, 0xE615, cIconConfig},
		{"pom.xml", false, false, 0xE674, cIconMaven},
		{"layout.xml", false, false, 0xE619, cIconOrange},
		{"build.gradle", false, false, 0xE660, cIconTeal},
		{"Main.java", false, false, 0xE66D, cIconOrange},
		{"app.py", false, false, 0xE606, cIconPython},
		{"query.sql", false, false, 0xE64D, cIconAmber},
		{"data.csv", false, false, 0xE64A, cIconGreen},
		{"Dockerfile", false, false, 0xE650, cIconDocker},
		{"Dockerfile.dev", false, false, 0xE650, cIconDocker},
		{"Makefile", false, false, 0xE673, cIconOrange},
		{"16-home-screen.webp", false, false, 0xE60D, cIconPurple},
		{"SCREENSHOT.PNG", false, false, 0xE60D, cIconPurple},
		{"report.pdf", false, false, 0xE67D, cIconRed},
		{"archive.zip", false, false, 0xE6AA, cIconAmber},
		{"yarn.lock", false, false, 0xE672, cIconLock},
		{"LICENSE", false, false, 0xE60A, cIconLicense},
		{"LICENSE.md", false, false, 0xE60A, cIconLicense},
		{".gitignore", false, false, 0xE65D, cIconGit},
		{".gitattributes", false, false, 0xE65D, cIconGit},
		{"package.json", false, false, 0xE616, cIconNpm},
		{"index.html", false, false, 0xE60E, cIconRed},
		{"styles.css", false, false, 0xE614, cIconBlue},
		{"app.js", false, false, 0xE60C, cIconYellow},
		{"app.ts", false, false, 0xE628, cIconPowerShell},
		{"notes.txt", false, false, 0xE64E, cExpFile},
		{"server.log", false, false, 0xE64E, cExpFile},
		{"no-extension", false, false, 0xE64E, cExpFile},
	} {
		i := expIconFor(c.name, c.dir, c.open)
		if i.glyph != string(c.cp) || i.c != c.c {
			t.Errorf("%q (folder %v, open %v): %U in %v, want %U in %v", c.name, c.dir, c.open, []rune(i.glyph)[0], i.c, c.cp, c.c)
		}
	}
}

// Plan 55, design D4: every icon one cell by the width table, each chevron
// with its space two, as the glyphs and the spaces they replace.
func TestExpIconWidths(t *testing.T) {
	for _, i := range expIcons {
		if w := textWidth(i.glyph); w != 1 {
			t.Errorf("%U is %d cells wide", []rune(i.glyph)[0], w)
		}
	}
	for _, s := range []string{nfClosed, nfOpen} {
		if textWidth(s) != textWidth(expClosed) {
			t.Errorf("the chevron %q is %d cells wide, %q %d", s, textWidth(s), expClosed, textWidth(expClosed))
		}
	}
}

// Plan 55, design D4: with the icons, a folder's row and a file's carry the
// chevron and the icon of their kind in its colour, on the cursor row its
// ground's; an ignored row's icon is grey and a deleted one's red, as their
// names; the names start where the panel's own glyphs leave them; without
// the icons the row is the panel's own.
func TestExplorerNerdRows(t *testing.T) {
	g := GitView{Ignored: map[string]bool{"build.log": true}}
	for _, c := range []struct {
		row  ExplorerRow
		cur  bool
		head string
		icon rune
		st   *lipgloss.Style
	}{
		{ExplorerRow{Path: "cmd", Name: "cmd", Dir: true}, false, nfClosed, 0xEA85, nfTerminal.st},
		{ExplorerRow{Path: "app", Name: "app", Dir: true, Open: true}, false, nfOpen, 0xE5FE, nfFolderOpen.st},
		{ExplorerRow{Path: "main.go", Name: "main.go"}, false, "  ", 0xE627, nfGo.st},
		{ExplorerRow{Path: "main.go", Name: "main.go"}, true, "  ", 0xE627, nfGo.row},
		{ExplorerRow{Path: "build.log", Name: "build.log"}, false, "  ", 0xE64E, stGrey},
		{ExplorerRow{Path: "old.md", Name: "old.md", Deleted: true}, false, "  ", 0xE609, stFailed},
	} {
		l := explorerRow(g, c.row, c.cur, 40, true)
		if l[1].text != c.head || l[2].text != string(c.icon) || l[2].st != c.st {
			t.Errorf("%s (cursor %v): head %q, icon %q in %p; want %q, %U in %p", c.row.Name, c.cur, l[1].text, l[2].text, l[2].st, c.head, c.icon, c.st)
		}
		own := explorerRow(g, c.row, c.cur, 40, false)
		if at, was := nameColumn(l, c.row.Name), nameColumn(own, c.row.Name); at < 0 || at != was || own[2].text == l[2].text {
			t.Errorf("%s: the name at %d, %d with the panel's own glyphs; icons %q and %q", c.row.Name, at, was, l[2].text, own[2].text)
		}
	}
	if r := rootRow("proj", 30, true); r[0].text != nfOpen || r[1].text != string(rune(0xE5FE)) {
		t.Errorf("the root row: %q %q", r[0].text, r[1].text)
	}
}

// nameColumn is the cell where the span holding name starts, -1 if none.
func nameColumn(l line, name string) int {
	x := 0
	for _, s := range l {
		if s.text == name {
			return x
		}
		x += textWidth(s.text)
	}
	return -1
}

// Plan 55, criterion 55.3: every glyph the Explorer draws is in the
// installed Cascadia Mono NF, read from the font file's character map
// (format 4); skipped where the font is not installed for the user.
func TestExpIconsInFont(t *testing.T) {
	file := filepath.Join(os.Getenv("LOCALAPPDATA"), `Microsoft\Windows\Fonts\CascadiaMonoNF.ttf`)
	data, err := os.ReadFile(file)
	if err != nil {
		t.Skipf("Cascadia Mono NF not installed for the user: %v", err)
	}
	has := cmapFormat4(t, data)
	glyphs := []rune{[]rune(nfClosed)[0], []rune(nfOpen)[0]}
	for _, i := range expIcons {
		glyphs = append(glyphs, []rune(i.glyph)[0])
	}
	for _, r := range glyphs {
		if !has(r) {
			t.Errorf("%U is not in %s", r, file)
		}
	}
}

// cmapFormat4 is a font's Windows Unicode BMP character map (platform 3,
// encoding 1, format 4): whether a code point has a glyph.
func cmapFormat4(t *testing.T, f []byte) func(rune) bool {
	t.Helper()
	u16 := func(o int) int { return int(binary.BigEndian.Uint16(f[o:])) }
	u32 := func(o int) int { return int(binary.BigEndian.Uint32(f[o:])) }
	cmap := -1
	for i := range u16(4) {
		rec := 12 + 16*i
		if string(f[rec:rec+4]) == "cmap" {
			cmap = u32(rec + 8)
		}
	}
	if cmap < 0 {
		t.Fatal("no cmap table")
	}
	sub := -1
	for i := range u16(cmap + 2) {
		rec := cmap + 4 + 8*i
		if u16(rec) == 3 && u16(rec+2) == 1 && u16(cmap+u32(rec+4)) == 4 {
			sub = cmap + u32(rec+4)
		}
	}
	if sub < 0 {
		t.Fatal("no format 4 subtable for Windows Unicode BMP")
	}
	segs := u16(sub+6) / 2
	ends, starts, deltas, offsets := sub+14, sub+16+2*segs, sub+16+4*segs, sub+16+6*segs
	return func(r rune) bool {
		for s := range segs {
			end, start := u16(ends+2*s), u16(starts+2*s)
			if int(r) > end || int(r) < start {
				continue
			}
			ro := u16(offsets + 2*s)
			if ro == 0 {
				return (int(r)+u16(deltas+2*s))&0xFFFF != 0
			}
			g := u16(offsets + 2*s + ro + 2*(int(r)-start))
			return g != 0
		}
		return false
	}
}
