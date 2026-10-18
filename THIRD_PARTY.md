# Third-party modules

Every module of `go list -m all` on 2026-10-18, with Go 1.27.2. The four first
direct dependencies were approved by the owner with plan 3, and Chroma, for the
file viewer's syntax highlighting, with plan 17 (brief amendment 11); the
others come with them.
"Built" means linked into the Windows build (`go list -deps ./...`); the other
modules are only in the module graph and none of their code is in the binary.
The licence is the one in the module's root licence file, read from the module
cache. `internal/licenses` checks this table against both commands and the
licence files, so a module added, removed, upgraded or rebuilt without a row
here fails the tests.

## Patched copies

One module is built from a patched copy kept in this repository, by a
`replace` in `go.mod`; its table row keeps the upstream version it was copied
from, and the licence is read from the copy's own licence file:

- `github.com/charmbracelet/ultraviolet` is replaced by
  `./third_party/ultraviolet`: two decoder fixes for Windows Terminal's
  win32-input-mode (letter case and emoji), described with their source,
  licence and checks in `third_party/ultraviolet/PATCHES.md`.

`internal/licenses` fails if a replaced module has no `PATCHES.md` or is not
named here. No other third-party source code is copied into this repository.

## Modules

| Module | Version | Direct | Built | Licence |
| - | - | - | - | - |
| charm.land/bubbletea/v2 | v2.1.0 | yes | yes | MIT |
| charm.land/lipgloss/v2 | v2.0.6 | yes | yes | MIT |
| dmitri.shuralyov.com/gpu/mtl | v0.0.0-20221208032759-85de2813cf6b | no | no | BSD-3-Clause |
| github.com/alecthomas/assert/v2 | v2.11.0 | no | no | MIT |
| github.com/alecthomas/chroma/v2 | v2.27.0 | yes | yes | MIT |
| github.com/alecthomas/repr | v0.5.2 | no | no | MIT |
| github.com/aymanbagabas/go-udiff | v0.4.1 | no | no | BSD-3-Clause |
| github.com/bits-and-blooms/bitset | v1.24.6 | no | no | BSD-3-Clause |
| github.com/charmbracelet/colorprofile | v0.4.3 | no | yes | MIT |
| github.com/charmbracelet/ultraviolet | v0.0.0-20261008173134-6b8d4baf91b4 | no | yes | MIT |
| github.com/charmbracelet/x/ansi | v0.11.9 | no | yes | MIT |
| github.com/charmbracelet/x/exp/golden | v0.0.0-20250806222409-83e3a29d542f | no | no | MIT |
| github.com/charmbracelet/x/term | v0.2.2 | no | yes | MIT |
| github.com/charmbracelet/x/termios | v0.1.1 | no | no | MIT |
| github.com/charmbracelet/x/windows | v0.2.2 | no | yes | MIT |
| github.com/clipperhouse/displaywidth | v0.11.0 | no | yes | MIT |
| github.com/clipperhouse/stringish | v0.1.1 | no | no | MIT |
| github.com/clipperhouse/uax29/v2 | v2.7.0 | no | yes | MIT |
| github.com/dlclark/regexp2/v2 | v2.2.1 | no | yes | MIT |
| github.com/ebitengine/purego | v0.10.1 | no | no | Apache-2.0 |
| github.com/go-gl/glfw/v3.3/glfw | v0.0.0-20231223183121-56fa3ac82ce7 | no | no | BSD-3-Clause |
| github.com/hexops/gotextdiff | v1.0.3 | no | no | BSD-3-Clause |
| github.com/jezek/xgb | v1.1.1 | no | no | BSD-3-Clause |
| github.com/lucasb-eyer/go-colorful | v1.4.1 | no | yes | MIT |
| github.com/mattn/go-runewidth | v0.0.30 | no | yes | MIT |
| github.com/muesli/cancelreader | v0.2.2 | no | yes | MIT |
| github.com/rivo/uniseg | v0.4.7 | no | yes | MIT |
| github.com/xo/terminfo | v1.0.0 | no | yes | MIT |
| golang.design/x/clipboard | v0.11.0 | yes | yes | MIT |
| golang.design/x/x11 | v0.2.0 | no | no | MIT |
| golang.org/x/exp | v0.0.0-20231006140011-7918f672742d | no | no | BSD-3-Clause |
| golang.org/x/exp/shiny | v0.0.0-20250606033433-dcc06ee1d476 | no | no | BSD-3-Clause |
| golang.org/x/image | v0.28.0 | no | yes | BSD-3-Clause |
| golang.org/x/mobile | v0.0.0-20250606033058-a2a15c67f36f | no | no | BSD-3-Clause |
| golang.org/x/mod | v0.25.0 | no | no | BSD-3-Clause |
| golang.org/x/sync | v0.22.0 | no | yes | BSD-3-Clause |
| golang.org/x/sys | v0.49.0 | yes | yes | BSD-3-Clause |
| golang.org/x/text | v0.26.0 | no | no | BSD-3-Clause |
| golang.org/x/tools | v0.34.0 | no | no | BSD-3-Clause |

Notes:
- **go-gl/glfw**: the Go wrapper is BSD-3-Clause; the GLFW C sources it
  bundles (`glfw/LICENSE.md`, `glfw/src`) are under the zlib licence, which is
  permissive. The module is in the graph through `golang.design/x/clipboard`'s
  non-Windows code and is not built for Windows.
- **aymanbagabas/go-udiff**: dual licence files, `LICENSE-BSD` and
  `LICENSE-MIT`; the BSD-3-Clause one is recorded.
