# Cascadia Mono NF, bundled for the Explorer's icons

Two font files of Microsoft's Cascadia Code release 2407.24, kept as
released, and its licence. `acc` installs the fonts for the user when
Windows has no font of that name (`scripts/install-font.ps1`; plan 56,
brief amendment 57); the Explorer draws its icons from them in the acc tab
(plan 55). The owner approved bundling them under their licences on
2026-10-27.

| File | Bytes | SHA-256 |
| - | - | - |
| `CascadiaMonoNF.ttf` | 2,526,924 | 5b12cbef13b6de9e773f585917783583d40d944a2578c3dae99b5e4fd913a14e |
| `CascadiaMonoNFItalic.ttf` | 2,336,432 | 0d3729255ba46b06eb292f9eadaddef55a106632d3ead4d1e8ec31d44952f534 |
| `LICENSE` | 4,489 | 51882cd3cdba4e16f220f44ddb08a635c38c44ea6e0975db2574f4be6f958238 |

Source: https://github.com/microsoft/cascadia-code/releases/tag/v2407.24,
whose one asset, `CascadiaCode-2407.24.zip` (150,454,761 bytes, SHA-256
e67a68ee3386db63f48b9054bd196ea752bc6a4ebb4df35adce6733da50c8474), holds
the two fonts as `ttf/CascadiaMonoNF.ttf` and `ttf/CascadiaMonoNFItalic.ttf`;
`LICENSE` is the repository's file at tag v2407.24, its CRLF line endings
kept (`.gitattributes`). `SHA256SUMS` holds the same hashes in
`sha256sum`'s format: `sha256sum -c SHA256SUMS` checks the folder, and the
install script checks the fonts against it before installing them.

## The licence

The fonts are under the SIL Open Font License 1.1 (`LICENSE`):
"Copyright (c) 2019 - Present, Microsoft Corporation, with Reserved Font
Name Cascadia Code". They are redistributed unmodified, with that licence,
as part of this software, and never sold by themselves.

## The Nerd Font symbols in them

Cascadia Code's README (tag v2407.24) describes the NF variants as "a
version of Cascadia that has Nerd Font symbols", the whole set as of April
2024, and names no other licence for them. Nerd Fonts lists each symbol
set's licence in its licence audit and its source in its README
(ryanoasis/nerd-fonts v3.2.1, `license-audit.md` and `readme.md`):

| Symbol set | Licence | Source |
| - | - | - |
| Codicons | CC BY 4.0: (c) Microsoft Corporation; its Git logo by Jason Long, CC BY 3.0 | https://github.com/microsoft/vscode-codicons |
| Font Awesome | CC BY 4.0 (icons): (c) Fonticons, Inc. | https://github.com/FortAwesome/Font-Awesome |
| Seti-UI + Custom | MIT | https://atom.io/themes/seti-ui |
| Devicons | MIT | https://vorillaz.github.io/devicons/ |
| Octicons | MIT | https://github.com/primer/octicons |
| Font Awesome Extension | MIT | https://github.com/AndreLZGava/font-awesome-extension |
| IEC Power Symbols | MIT | https://unicodepowersymbol.com/ |
| Powerline Extra Symbols | MIT | https://github.com/ryanoasis/powerline-extra-symbols |
| Material Design Icons | Apache-2.0 | https://github.com/Templarian/MaterialDesign |
| Pomicons | SIL OFL 1.1 | https://github.com/gabrielelana/pomicons |
| Weather Icons | SIL OFL 1.1 | https://github.com/erikflowers/weather-icons |
| Font Logos | The Unlicense (the audit says "Unlicensed"; the project's own `LICENSE` is The Unlicense) | https://github.com/Lukas-W/font-logos |
| Powerline Symbols | a free licence | Powerline |

The Explorer's icons are Codicons', Seti-UI's and Octicons'. The CC BY 4.0
sets are credited above, by their authors, with their licence:
https://creativecommons.org/licenses/by/4.0/. The symbols are as Microsoft
released them; this repository changes none.
