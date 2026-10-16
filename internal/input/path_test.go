package input

import "testing"

// TestResolvePath is the path table of proposal step 1, item 1: every form of
// INT paths, "Forms and rules", accepted with its resolved path or refused
// with the exact message of ui-design.md, Validation and messages. No I/O:
// the existence and access checks are in the app's validation.
func TestResolvePath(t *testing.T) {
	win := PathEnv{Windows: true, Home: `C:\Users\dev`, Cwd: `D:\work\here`}
	nix := PathEnv{Home: "/home/dev", Cwd: "/srv/here"}
	cases := []struct {
		name string
		env  PathEnv
		in   string
		want ResolvedPath
	}{
		// Windows column.
		{"DOS absolute", win, `C:\dev\app`, ResolvedPath{Path: `C:\dev\app`}},
		{"DOS absolute, slashes", win, `C:/dev/app`, ResolvedPath{Path: `C:\dev\app`}},
		{"trailing separator", win, `C:\dev\app\`, ResolvedPath{Path: `C:\dev\app`}},
		{"drive root", win, `C:\`, ResolvedPath{Path: `C:\`}},
		{"dot segments", win, `C:\dev\.\x\..\app`, ResolvedPath{Path: `C:\dev\app`}},
		{"dot-dot above the root", win, `C:\..\..\app`, ResolvedPath{Path: `C:\app`}},
		{"trailing period and space trimmed", win, `C:\dev\app. `, ResolvedPath{Path: `C:\dev\app`}},
		{"inner spaces kept", win, `C:\My Projects\app`, ResolvedPath{Path: `C:\My Projects\app`}},
		{"case kept", win, `c:\Dev\App`, ResolvedPath{Path: `c:\Dev\App`}},
		{"Explorer quotes and newline", win, "  \"C:\\dev\\app\"\r\n", ResolvedPath{Path: `C:\dev\app`}},
		{"empty uses the current directory", win, "", ResolvedPath{Path: `D:\work\here`}},
		{"blank uses the current directory", win, "   ", ResolvedPath{Path: `D:\work\here`}},
		{"relative", win, `app`, ResolvedPath{Path: `D:\work\here\app`}},
		{"relative dot", win, `.\app`, ResolvedPath{Path: `D:\work\here\app`}},
		{"relative parent", win, `../app`, ResolvedPath{Path: `D:\work\app`}},
		{"home", win, `~`, ResolvedPath{Path: `C:\Users\dev`}},
		{"home slash", win, `~/projets/app`, ResolvedPath{Path: `C:\Users\dev\projets\app`}},
		{"home backslash", win, `~\projets\app`, ResolvedPath{Path: `C:\Users\dev\projets\app`}},
		{"another user's home", win, `~bob/app`, ResolvedPath{Problem: "Another user's home is not expanded. Type the full path"}},
		{"home unknown", PathEnv{Windows: true, Cwd: `D:\x`}, `~/app`, ResolvedPath{Problem: "~ cannot be expanded: the home directory is unknown. Type the full path"}},
		{"WSL drive mount", win, `/mnt/c/dev/app`, ResolvedPath{Path: `C:\dev\app`, Translated: true}},
		{"WSL drive mount root", win, `/mnt/d`, ResolvedPath{Path: `D:\`, Translated: true}},
		{"WSL drive mount, trailing slash", win, `/mnt/c/dev/`, ResolvedPath{Path: `C:\dev`, Translated: true}},
		{"POSIX absolute", win, `/home/u/app`, ResolvedPath{Problem: `A Linux path. On Windows use C:\…, or /mnt/c/… for a Windows drive`}},
		{"not a drive mount", win, `/mnt/cdrive/app`, ResolvedPath{Problem: `A Linux path. On Windows use C:\…, or /mnt/c/… for a Windows drive`}},
		{"drive-relative", win, `C:dev\app`, ResolvedPath{Problem: `Drive-relative path. Type a full path, for example C:\dev\app`}},
		{"bare drive", win, `C:`, ResolvedPath{Problem: `Drive-relative path. Type a full path, for example C:\`}},
		{"root-relative", win, `\dev\app`, ResolvedPath{Problem: `Root-relative path. Type a full path with its drive, for example D:\dev\app`}},
		{"UNC", win, `\\server\share\app`, ResolvedPath{Path: `\\server\share\app`, Warning: "a network share can be slow or disappear"}},
		{"UNC share root", win, `\\server\share`, ResolvedPath{Path: `\\server\share\`, Warning: "a network share can be slow or disappear"}},
		{"UNC, slashes", win, `//server/share/app`, ResolvedPath{Path: `\\server\share\app`, Warning: "a network share can be slow or disappear"}},
		{"UNC without a share", win, `\\server`, ResolvedPath{Problem: "Not a directory path"}},
		{"WSL filesystem", win, `\\wsl.localhost\Ubuntu\home\u\app`, ResolvedPath{Path: `\\wsl.localhost\Ubuntu\home\u\app`, Warning: "files in Linux, used by a Windows agent CLI: slow, and unverified"}},
		{"WSL filesystem, wsl$", win, `\\wsl$\Ubuntu\home\u\app`, ResolvedPath{Path: `\\wsl$\Ubuntu\home\u\app`, Warning: "files in Linux, used by a Windows agent CLI: slow, and unverified"}},
		{"device path to a drive", win, `\\?\C:\dev\app`, ResolvedPath{Path: `C:\dev\app`}},
		{"device path, dot form", win, `\\.\C:\dev\app`, ResolvedPath{Path: `C:\dev\app`}},
		{"device path to UNC", win, `\\?\UNC\server\share\app`, ResolvedPath{Path: `\\server\share\app`, Warning: "a network share can be slow or disappear"}},
		{"other device", win, `\\.\PhysicalDrive0`, ResolvedPath{Problem: "Not a directory path"}},
		{"volume GUID", win, `\\?\Volume{0b1c2d3e-0000-0000-0000-000000000000}\`, ResolvedPath{Problem: "Not a directory path"}},

		// Linux, macOS or WSL column.
		{"POSIX absolute on Linux", nix, `/home/u/app/`, ResolvedPath{Path: "/home/u/app"}},
		{"relative on Linux", nix, `../app`, ResolvedPath{Path: "/srv/app"}},
		{"home on Linux", nix, `~/app`, ResolvedPath{Path: "/home/dev/app"}},
		{"Windows path on Linux", nix, `C:\dev\app`, ResolvedPath{Problem: "A Windows path"}},
		{"UNC on Linux", nix, `\\server\share`, ResolvedPath{Problem: "A Windows path"}},
	}
	for _, c := range cases {
		if got := ResolvePath(c.in, c.env); got != c.want {
			t.Errorf("%s: ResolvePath(%q) = %+v, want %+v", c.name, c.in, got, c.want)
		}
	}
}
