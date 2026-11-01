# wt-check.ps1: a check of the dashboard in a real Windows Terminal
# (doc/evaluation.md, Windows Terminal checks). Its tests: wt-check-tests.ps1, with no
# terminal, and wt-check-window-tests.ps1, in a real one.
#
# What the code enforces:
# - Every parameter and step is checked first: the first wrong one stops the run
#   before anything is written.
# - It deletes nothing, and no step writes a file a plan chooses. A run makes a new
#   folder, %TEMP%\acc-wt\<outName>-<tag>-<mode>, and writes only there. It stops if
#   that folder exists, or if %TEMP% or %TEMP%\acc-wt is a link. The one exception:
#   a -fixture receives the recording's copy, never over an existing file.
# - The launch is fixed: the build with --mock and that copy, or with no argument
#   under -homeScreen, from a generated launch.cmd that holds only checked paths and
#   that cmd.exe /c runs. -exe must hold the dashboard's main package. No parameter adds
#   an argument or a command, and only -git adds a folder to the PATH.
# - The build gets a temporary home, local and roaming app data and a PATH of
#   System32 alone, every inherited GIT_ setting cleared: it finds no agent CLI, and
#   without -git no git.
#
# Assertions and observations. Only expect and expectfield assert. A failed one keeps
# its screen, logs ASSERTION FAILED, stops the plan, closes the window and exits 1; a
# passed one logs it, and the log's last line counts them. Every other step observes: it
# acts, saves its screen and logs, and says nothing about what the screen shows, so a
# run's exit 0 speaks only for its assertions and for its steps having run. A step that
# cannot act (a click's text not on screen, no dialog) also stops the run with exit 1.
# poll and fieldcheck, which logged NOT found and exited 0, are gone under those names,
# so that no plan changes meaning unseen: a plan naming them is refused.
#
# -git's containment, so that git stays among the fixtures. Before the launch, for the
# fixture's repository and each of its remotes, or the run is refused:
# - No link anywhere inside, and no other .git; the repository's .git is a folder.
# - Its config holds only keys of an allow-list: core's basics, user, a remote's url and
#   fetch, a branch's remote and merge. So no hook path, worktree, include, URL
#   rewrite, push URL, helper, filter, program or extension.
# - No active hook, alternates, commondir, gitdir, config.worktree or modules. A linked
#   worktree must live in the fixtures' folder and lead back to its repository.
# - A remote is a plain local path to a bare repository in the fixtures' folder, which
#   passes the same checks and has no remotes of its own: no scheme, no host.
# In the launch, for every git the build runs:
# - Every inherited GIT_ variable is cleared; no system settings; the home and
#   XDG_CONFIG_HOME are the run's own. So nothing inherited can redirect it.
# - core.hooksPath is an empty folder of the run's, forced over the repository's config
#   and, by the home's config, on a local remote's side of a push, where git clears
#   the forced settings. So no hook runs, even one added after these checks.
# - GIT_ALLOW_PROTOCOL=file refuses every network transport, whatever a config says.
# - The search for a repository stops at the fixtures' folder.
#
# What it does not enforce. This is no sandbox:
# - The build and git run with the owner's rights: only their settings point them at
#   temporary folders. A changed build or git, or a file made to look like one, can do
#   anything. %TEMP% is trusted: folders above it are not checked, and the checks above
#   are not repeated, so another process changing a fixture's config or remotes during
#   a run is not seen. No step of a plan can do so.
# - With -git, a plan's keys and clicks drive the dashboard's real git on the fixture:
#   commit, push to a local remote, branch switch, worktree removal.
# - Keys and mouse events are real. Term.cs checks the window has the focus before
#   each batch and stops otherwise, but cannot see a focus change inside a batch:
#   nothing else uses the machine during a run.
# - Keys can make the dashboard write the owner's clipboard (a double click, y): a
#   rule keeps those steps the owner's, not the code. No clipboard code here, on
#   purpose: the antivirus quarantined a harness that saved and restored the
#   clipboard beside its keys (plan 51).
#
# A run's folder holds results\ (for each step a PNG, the screen's text by UI
# Automation, crops of zones B and C and of the sidebar; log.txt), demo\ (the project,
# unless -fixture), lad\, ad\, home\ and launch.cmd. It stays until someone moves it
# away; what matters of a run goes in the plan's evaluation file.
#
#   -exe         a build of the dashboard, required
#   -mock        the recording (.jsonl), copied into the project as mock.jsonl
#   -homeScreen  the home screen instead of a recording: exactly one of the two
#   -outName     the run's name, required, new for each run
#   -tag         part of the run's name, codex by default
#   -mode        max, or columns x rows as wt --size takes them
#   -wtProfile   a profile for the tab: 'Agent Command Center' is ACC's 9 pt, where
#                -mode 90x30 gives 120x40
#   -plan        the steps, or @file in UTF-8, required
#   -fixture     NAME, with -mock: the project is %TEMP%\acc-wt-fixtures\NAME, a plain
#                folder you made, with no link inside it, no mock.jsonl and no .git file
#   -git         with -fixture holding a repository: the folder of git.exe, Git for
#                Windows' wrapper if it is that git, on the PATH, and the containment
#                above; needs git 2.31 or newer
#   -trace       TEA_TRACE logs to trace.txt in the run's folder
#   -expect      the text expectfield looks for, which may hold a ':'
#   -noLaunch    does all of this but open the terminal; it sends no key
#
# Names (-outName, -tag, -fixture, labels): 1 to 40 letters, digits, _ or -, starting
# with a letter or a digit. Steps, separated by ';', each NAME:LABEL:MS[:ARG...], MS
# from the start, labels unique, no ':' in a text but type's. Each saves its screen
# as LABEL unless said otherwise:
#   expect:TEXT                ASSERTS: TEXT on screen within 25 s, or the run fails
#   expectfield                ASSERTS: the screen holds -expect now, or the run fails
#   shot
#   press:VK[.VK...]           keys, in hex, 150 ms apart
#   chord:MOD[.MOD...].VK      the modifiers held while the key is pressed
#   altgr:VK[.VK...]           AltGr and the key, as a physical press sends it
#   ctrlhold:N                 Ctrl's key-down N times, then its key-up
#   click:TEXT[:DX]            a left click DX cells right of TEXT's first cell
#   dclick:TEXT[:DX]           a double click
#   cclick:TEXT[:DX]           a click with Ctrl held
#   clickrow:TEXT:COL          cell COL of the first row holding TEXT ('at': TEXT's own)
#   clickbelow:TEXT:DROWS:DX   DROWS rows below and DX cells right of TEXT's first cell
#   clickcell:COL:ROW          a cell, anchored on the title row; COL negative from the right
#   hoverat:COL:ROW, hoveroff  the pointer glided there, or off the sidebar
#   homeclick:COL:ROW:ANCOL:ANROW:CW:ANCHOR    the home screen's picture has no text:
#                              a cell from an anchor's own cell, CW pixels a cell
#   wheel:COL:ROW:ANCOL:ANROW:CW:ANCHOR:NOTCHES   the same, the wheel turned, plus up
#   fieldsel:ANCHOR:DX:N:CW    the terminal's own selection with Shift held, never copied
#   drag:DX:DY                 the window's bottom-right corner dragged
#   dialog[:TITLE], type:TEXT, enter, esc   a Windows file dialog, saved as LABEL-dialog
# ASCII only: Windows PowerShell 5.1 reads a BOM-less script as ANSI.
[CmdletBinding(PositionalBinding = $false)]
param([string]$exe = '', [string]$mock = '', [switch]$homeScreen, [string]$outName = '', [string]$tag = 'codex', [string]$mode = 'max', [string]$wtProfile = '', [string]$plan = '', [string]$fixture = '', [switch]$git, [switch]$trace, [string]$expect = '', [switch]$noLaunch)
$ErrorActionPreference = 'Stop'
$dir = $PSScriptRoot

function Fail([string]$why) { throw "wt-check: $why" }
# A name becomes part of a file's name. An allow-list, so that no separator, dot-dot,
# drive, wildcard or shell character gets through, and no device name.
function Assert-Name([string]$what, [string]$value) {
    if ($value -cnotmatch '\A[A-Za-z0-9][A-Za-z0-9_-]{0,39}\z') { Fail "$what '$value': 1 to 40 letters, digits, _ or -, starting with a letter or a digit" }
    if ($value -match '\A(con|prn|aux|nul|com[0-9]|lpt[0-9]|log)\z') { Fail "$what '$value': a reserved name" }
}
# A path that reaches launch.cmd or wt.exe's command line: whole, local, and free of
# every character cmd.exe or wt.exe would act on. Quoting alone is not relied on.
function Full([string]$what, [string]$path) {
    try { $f = [IO.Path]::GetFullPath($path) } catch { Fail "$what '$path': not a path" }
    $f = $f.TrimEnd('\')
    if ($f -cnotmatch '\A[A-Za-z]:\\[A-Za-z0-9 _.~\\-]+\z') { Fail "$what '$f': only a local path of letters, digits, spaces and _ . ~ - \" }
    return $f
}
function Test-Link([string]$path) { return ([int][IO.File]::GetAttributes($path) -band 0x400) -ne 0 }
function Assert-PlainDir([string]$what, [string]$path) {
    if (-not [IO.Directory]::Exists($path)) { Fail "$what '$path': no such folder" }
    if (Test-Link $path) { Fail "$what '$path': a link, not a plain folder" }
}
# No link anywhere under a folder, and no .git below $top: a gitfile, or a repository
# with settings of its own. $top is '' for a remote, which holds no .git at all.
function Assert-Tree([string]$what, [string]$path, [string]$top) {
    foreach ($e in (New-Object IO.DirectoryInfo $path).GetFileSystemInfos()) {
        if (([int]$e.Attributes -band 0x400) -ne 0) { Fail "${what}: a link inside it: $($e.FullName)" }
        if ($e.Name -ieq '.git' -and $path -ine $top) { Fail "${what}: a repository inside it: $($e.FullName)" }
        if ($e -is [IO.DirectoryInfo]) { Assert-Tree $what $e.FullName $top }
    }
}
# A folder among the fixtures, each folder on the way plain, so that no link leads away.
function Assert-Inside([string]$what, [string]$path) {
    if (-not $path.StartsWith("$fixtures\", [StringComparison]::OrdinalIgnoreCase)) { Fail "$what '$path': outside %TEMP%\acc-wt-fixtures" }
    $at = $fixtures
    foreach ($part in $path.Substring($fixtures.Length + 1).Split('\')) { $at = "$at\$part"; Assert-PlainDir $what $at }
}
# git itself, only to read its version and one config file, with no inherited GIT_ setting.
function Read-Git([string]$arguments) {
    $psi = New-Object Diagnostics.ProcessStartInfo
    $psi.FileName = $gitPath; $psi.Arguments = $arguments
    $psi.UseShellExecute = $false; $psi.CreateNoWindow = $true
    $psi.RedirectStandardOutput = $true; $psi.RedirectStandardError = $true
    foreach ($k in @($psi.EnvironmentVariables.Keys)) { if ($k -like 'GIT_*') { $psi.EnvironmentVariables.Remove($k) } }
    $p = [Diagnostics.Process]::Start($psi)
    $o = $p.StandardOutput.ReadToEndAsync(); $e = $p.StandardError.ReadToEndAsync()
    if (-not $p.WaitForExit(15000)) { $p.Kill(); Fail 'git did not answer' }
    if ($p.ExitCode -ne 0) { return $null }
    return $o.Result
}
# -git's containment (the head says why): what a repository's own files could redirect,
# run or reach. git reads the config: --file reads that file alone and follows no
# include, and whatever is not on the allow-list stops the run.
$plainConfig = '\A(core\.(repositoryformatversion|filemode|bare|logallrefupdates|ignorecase|precomposeunicode|symlinks|autocrlf|eol|safecrlf)|user\.(name|email))\z'
function Assert-Repository([string]$what, [string]$gitdir, [string]$top, [bool]$isRemote) {
    foreach ($f in 'commondir', 'gitdir', 'config.worktree', 'modules', 'objects\info\alternates', 'objects\info\http-alternates') {
        if (Test-Path -LiteralPath "$gitdir\$f") { Fail "${what}: its $f can lead outside it" }
    }
    if ([IO.Directory]::Exists("$gitdir\hooks")) {
        foreach ($h in [IO.Directory]::GetFiles("$gitdir\hooks")) { if (-not $h.EndsWith('.sample')) { Fail "${what}: an active hook: $h" } }
    }
    if ([IO.Directory]::Exists("$gitdir\worktrees")) {
        foreach ($t in [IO.Directory]::GetDirectories("$gitdir\worktrees")) {
            $file = [IO.Path]::GetFullPath([IO.Path]::Combine($t, [IO.File]::ReadAllText("$t\gitdir").Trim()))
            $folder = [IO.Path]::GetDirectoryName($file)
            Assert-Inside "$what, a linked worktree" $folder
            Assert-Tree "$what, a linked worktree" $folder $folder
            $back = [IO.File]::ReadAllText($file).Trim() -replace '\Agitdir: ', ''
            if ([IO.Path]::GetFullPath([IO.Path]::Combine($folder, $back)) -ine $t -or [IO.File]::ReadAllText("$t\commondir").Trim() -ne '../..') { Fail "${what}: a linked worktree leads elsewhere: $t" }
        }
    }
    $config = Read-Git "config --file `"$gitdir\config`" --list"
    if ($null -eq $config) { Fail "${what}: git cannot read its config" }
    $remotes = @{}; $urls = @(); $branches = @(); $bare = $false
    foreach ($line in @($config -split "`r?`n" | Where-Object { $_ })) {
        $key, $value = $line -split '=', 2
        if ($key -eq 'core.bare' -and $value -eq 'true') { $bare = $true }
        if ($key -match $plainConfig) { continue }
        if (-not $isRemote -and $key -match '\Aremote\.(.+)\.(url|fetch)\z') {
            $remotes[$Matches[1]] = $true
            if ($Matches[2] -eq 'url') { $urls += [string]$value }
            continue
        }
        if (-not $isRemote -and $key -match '\Abranch\..+\.merge\z') { continue }
        if (-not $isRemote -and $key -match '\Abranch\..+\.remote\z') { $branches += [string]$value; continue }
        Fail "${what}: its config sets '$key', which this harness does not let through"
    }
    if ($isRemote -and -not $bare) { Fail "${what}: not a bare repository" }
    foreach ($b in $branches) { if ($b -ne '.' -and -not $remotes.ContainsKey($b)) { Fail "${what}: a branch's remote '$b' names no remote of its own" } }
    foreach ($u in $urls) { Assert-Remote $what $u $top }
}
# A remote's URL: a plain local path, no scheme, host or option, then where it leads.
function Assert-Remote([string]$what, [string]$url, [string]$top) {
    if ($url -cnotmatch '\A(?:[A-Za-z]:[\\/])?[A-Za-z0-9_.~ \\/-]+\z' -or $url.StartsWith('-')) { Fail "${what}, remote '$url': only a local path, no URL" }
    $path = [IO.Path]::GetFullPath([IO.Path]::Combine($top, $url)).TrimEnd('\')
    Assert-Inside "$what, remote" $path
    Assert-Tree "$what, remote" $path ''
    Assert-Repository "$what, remote" $path $path $true
}
# Each step's arguments: t a text, n a number, k keys in hex, b the same of one byte,
# c a number or 'at'; ? optional, + all the rest.
$stepArgs = @{
    shot = ''; hoveroff = ''; expectfield = ''; enter = ''; esc = ''
    expect = 't'; press = 'k'; chord = 'k'; altgr = 'b'; ctrlhold = 'n'
    click = 't n?'; dclick = 't n?'; cclick = 't n?'
    hoverat = 'n n'; clickcell = 'n n'; drag = 'n n'
    clickrow = 't c'; clickbelow = 't n n'
    homeclick = 'n n n n n t'; wheel = 'n n n n n t n'; fieldsel = 't n n n'
    dialog = 't?'; type = 't+'
}
$argForm = @{
    t = '\A.+\z'; n = '\A-?[0-9]+(\.[0-9]+)?\z'; c = '\A(at|-?[0-9]+)\z'
    k = '\A[0-9A-Fa-f]{1,4}(\.[0-9A-Fa-f]{1,4})*\z'; b = '\A[0-9A-Fa-f]{1,2}(\.[0-9A-Fa-f]{1,2})*\z'
}
function Assert-Plan([string]$text) {
    $steps = @($text -split ';' | Where-Object { $_ -ne '' })
    if ($steps.Count -eq 0) { Fail '-plan: no step' }
    $labels = @{}
    foreach ($step in $steps) {
        $p = $step -split ':'
        if (-not $stepArgs.ContainsKey($p[0])) { Fail "step '$step': no such step" }
        if ($p.Count -lt 3) { Fail "step '$step': NAME:LABEL:MS first" }
        Assert-Name "step '$step': the label" $p[1]
        if ($labels.ContainsKey($p[1])) { Fail "step '$step': this label is already used" }
        $labels[$p[1]] = $true
        if ($p[2] -cnotmatch '\A[0-9]{1,7}\z') { Fail "step '$step': MS, a whole number" }
        if ($p[0] -eq 'expectfield' -and -not $expect) { Fail "step '$step': needs -expect" }
        $given = @(if ($p.Count -gt 3) { $p[3..($p.Count - 1)] })
        $n = 0
        foreach ($w in @($stepArgs[$p[0]] -split ' ' | Where-Object { $_ })) {
            if ($n -ge $given.Count) {
                if ($w.EndsWith('?')) { continue }
                Fail "step '$step': an argument is missing"
            }
            if ($given[$n] -cnotmatch $argForm[[string]$w[0]]) { Fail "step '$step': argument '$($given[$n])' has the wrong form" }
            $n++
            if ($w.EndsWith('+')) { $n = $given.Count }
        }
        if ($n -lt $given.Count) { Fail "step '$step': too many arguments" }
    }
}

Assert-Name '-outName' $outName
Assert-Name '-tag' $tag
if ($mode -cnotmatch '\A(max|[1-9][0-9]{1,2}x[1-9][0-9]{1,2})\z') { Fail "-mode '$mode': max, or columns x rows such as 90x30" }
if ($wtProfile -cnotmatch '\A[A-Za-z0-9 ._-]{0,64}\z') { Fail "-wtProfile '$wtProfile': letters, digits, spaces and . _ - only" }
if ($plan.StartsWith('@')) { $plan = [IO.File]::ReadAllText($plan.Substring(1), [Text.Encoding]::UTF8).Trim() }
Assert-Plan $plan
if ([bool]$homeScreen -eq [bool]$mock) { Fail 'exactly one of -mock and -homeScreen' }
if ($fixture -and -not $mock) { Fail '-fixture: only with -mock' }
if ($git -and -not $fixture) { Fail '-git: only with -fixture' }
if (-not $exe) { Fail '-exe: the path of a build of the dashboard' }
$exe = Full '-exe' $exe
if (-not [IO.File]::Exists($exe) -or [IO.Path]::GetExtension($exe) -ine '.exe') { Fail "-exe '$exe': no such .exe" }
if (Test-Link $exe) { Fail "-exe '$exe': a link" }
# Go writes a build's main package into it (go version -m): no agent CLI, and no other
# program, holds this line by accident.
$image = [Text.Encoding]::GetEncoding(28591).GetString([IO.File]::ReadAllBytes($exe))
if (-not $image.Contains("path`tgithub.com/Evengelius/agent-command-center/cmd/agent-command-center")) { Fail "-exe '$exe': not a build of the dashboard" }
if ($mock) {
    try { $mock = [IO.Path]::GetFullPath($mock) } catch { Fail "-mock '$mock': not a path" }
    if (-not [IO.File]::Exists($mock) -or [IO.Path]::GetExtension($mock) -ine '.jsonl') { Fail "-mock '$mock': no such .jsonl" }
    if (Test-Link $mock) { Fail "-mock '$mock': a link" }
    # The first line, as the mock provider reads every line: JSON with a kind.
    $kind = $null
    try { $kind = (Get-Content -LiteralPath $mock -TotalCount 1 -Encoding UTF8 | ConvertFrom-Json).kind } catch { }
    if (-not $kind) { Fail "-mock '$mock': its first line is no event of a mock file" }
}
$temp = Full '%TEMP%' $env:TEMP
Assert-PlainDir '%TEMP%' $temp
$root = Join-Path $temp 'acc-wt'
if (Test-Path -LiteralPath $root) { Assert-PlainDir '%TEMP%\acc-wt' $root }
$name = "$outName-$tag-$mode"
$base = Join-Path $root $name
if (Test-Path -LiteralPath $base) { Fail "the run's folder '$base' already exists: this harness deletes nothing; give another -outName" }
$proj = "$base\demo"
if ($fixture) {
    Assert-Name '-fixture' $fixture
    $fixtures = Join-Path $temp 'acc-wt-fixtures'
    Assert-PlainDir '%TEMP%\acc-wt-fixtures' $fixtures
    $proj = Join-Path $fixtures $fixture
    Assert-PlainDir '-fixture' $proj
    Assert-Tree '-fixture' $proj $proj
    if (Test-Path -LiteralPath "$proj\mock.jsonl") { Fail "-fixture '$proj': it already holds a mock.jsonl, which is not replaced" }
    if (Test-Path -LiteralPath "$proj\.git" -PathType Leaf) { Fail "-fixture '$proj': its .git is a file: a worktree of a repository elsewhere" }
}
if ($git) {
    $gitExe = Get-Command git.exe -CommandType Application -ErrorAction SilentlyContinue | Select-Object -First 1
    if (-not $gitExe) { Fail '-git: no git.exe on the PATH' }
    # From Git Bash the PATH finds the git behind Git for Windows' wrapper, which crashed
    # on a local push under the launch's PATH: the wrapper sets up what it needs.
    $gitPath = $gitExe.Source -replace '\\(ucrt64|mingw64|mingw32|clangarm64)\\bin\\git\.exe\z', '\cmd\git.exe'
    if (-not [IO.File]::Exists($gitPath)) { Fail "-git: no git.exe at '$gitPath'" }
    $gitDir = Full 'the folder of git.exe' ([IO.Path]::GetDirectoryName($gitPath))
    # GIT_CONFIG_COUNT, which forces the settings of the launch, came in git 2.31.
    if ((Read-Git '--version') -notmatch 'git version ([0-9]+)\.([0-9]+)' -or [int]$Matches[1] -lt 2 -or ([int]$Matches[1] -eq 2 -and [int]$Matches[2] -lt 31)) { Fail '-git: needs git 2.31 or newer' }
    if (-not [IO.Directory]::Exists("$proj\.git")) { Fail "-git: '$proj' holds no repository" }
    Assert-Repository "-fixture '$fixture'" "$proj\.git" $proj $false
}
if (-not $noLaunch -and (Get-Process -Name LogonUI -ErrorAction SilentlyContinue)) { Fail 'the workstation is locked' }

# All is valid. New-Item without -Force fails if the folder exists, so all under the
# run's folder is this run's own.
if (-not (Test-Path -LiteralPath $root)) { New-Item -ItemType Directory -Path $root | Out-Null }
New-Item -ItemType Directory -Path $base | Out-Null
$out = "$base\results"
New-Item -ItemType Directory -Path $out, "$base\lad", "$base\ad", "$base\home" | Out-Null
if (-not $fixture) { New-Item -ItemType Directory -Path $proj | Out-Null }
if ($mock) { [IO.File]::Copy($mock, "$proj\mock.jsonl", $false) }
$launch = "$base\launch.cmd"
$cmdLines = @(
    '@echo off',
    'for /f "delims==" %%V in (''set GIT_ 2^>nul'') do set "%%V="',
    "set `"LOCALAPPDATA=$base\lad`"",
    "set `"APPDATA=$base\ad`"",
    "set `"USERPROFILE=$base\home`"",
    "set `"HOME=$base\home`"",
    "set `"XDG_CONFIG_HOME=$base\home\.config`"",
    $(if ($git) { "set `"PATH=C:\Windows\System32;$gitDir`"" } else { 'set "PATH=C:\Windows\System32"' }))
if ($git) {
    # Forced over a repository's config. Git clears these on a local remote's side of a
    # push, so the run's home says the same about hooks there.
    New-Item -ItemType Directory -Path "$base\nohooks" | Out-Null
    Set-Content -LiteralPath "$base\home\.gitconfig" -Encoding ascii -Value @('[core]', "`thooksPath = $($base.Replace('\', '/'))/nohooks")
    $cmdLines += @(
        "set `"GIT_CEILING_DIRECTORIES=$fixtures`"",
        'set "GIT_CONFIG_NOSYSTEM=1"',
        'set "GIT_ALLOW_PROTOCOL=file"',
        'set "GIT_CONFIG_COUNT=2"',
        'set "GIT_CONFIG_KEY_0=core.hooksPath"',
        "set `"GIT_CONFIG_VALUE_0=$base\nohooks`"",
        'set "GIT_CONFIG_KEY_1=core.fsmonitor"',
        'set "GIT_CONFIG_VALUE_1=false"')
}
if ($trace) { $cmdLines += "set `"TEA_TRACE=$base\trace.txt`"" }
$cmdLines += @('set NO_COLOR=', $(if ($homeScreen) { "`"$exe`"" } else { "`"$exe`" --mock `"$proj\mock.jsonl`"" }))
Set-Content -LiteralPath $launch -Encoding ascii -Value $cmdLines
"run folder: %TEMP%\acc-wt\$name"
if ($noLaunch) { return }

Add-Type -AssemblyName UIAutomationClient, UIAutomationTypes, WindowsBase, System.Drawing, System.Windows.Forms
Add-Type -Path "$dir\Term.cs", "$dir\Shot.cs" -ReferencedAssemblies UIAutomationClient, UIAutomationTypes, WindowsBase, System.Drawing
[Term]::Dpi()
Add-Type -TypeDefinition @"
using System;
using System.Runtime.InteropServices;
public static class WinMove40 {
    [DllImport("user32.dll")] static extern bool SetWindowPos(IntPtr h, IntPtr after, int x, int y, int cx, int cy, uint flags);
    public static void TopLeft(IntPtr h) { SetWindowPos(h, IntPtr.Zero, 0, 0, 0, 0, 0x0001 | 0x0004 | 0x0040); }
}
public static class AltGr67 {
    [DllImport("user32.dll")] static extern void keybd_event(byte vk, byte scan, uint flags, UIntPtr extra);
    public static void Press(byte vk) {
        keybd_event(0xA5, 0x38, 1, UIntPtr.Zero); System.Threading.Thread.Sleep(30);
        keybd_event(vk, 0, 0, UIntPtr.Zero); System.Threading.Thread.Sleep(30);
        keybd_event(vk, 0, 2, UIntPtr.Zero); System.Threading.Thread.Sleep(30);
        keybd_event(0xA5, 0x38, 1 | 2, UIntPtr.Zero); System.Threading.Thread.Sleep(60);
    }
}
"@
$log = @()
$bar = [string][char]0x2502 # a frame's vertical line
$corners = @([string][char]0x2570, [string][char]0x2514) # a box's bottom-left corner, rounded or square
function Rows() {
    $t = $null; try { $t = [Term]::VisibleText($script:h) } catch { $t = $null }
    if ($t -eq $null) { return @() }
    $rows = $t -split "`r?`n"
    if ($rows.Count -gt 0 -and $rows[-1] -eq '') { $rows = $rows[0..($rows.Count - 2)] }
    return ,$rows
}
function Text() { return ((Rows) -join "`n") }
function Cell() {
    $cols = [Math]::Max(((Rows) | ForEach-Object { $_.Length } | Measure-Object -Maximum).Maximum, 1)
    return ([Term]::Rect($script:h).R - [Term]::Rect($script:h).L) / $cols
}
function Crop([int]$x0, [int]$y0, [int]$w, [int]$ht, [int]$k, [string]$path) {
    $src = New-Object System.Drawing.Bitmap($w, $ht)
    $g = [System.Drawing.Graphics]::FromImage($src); $g.CopyFromScreen($x0, $y0, 0, 0, (New-Object System.Drawing.Size($w, $ht))); $g.Dispose()
    $src.Save("$path.png", [System.Drawing.Imaging.ImageFormat]::Png)
    if ($k -gt 1) {
        $big = New-Object System.Drawing.Bitmap(($w * $k), ($ht * $k))
        $g = [System.Drawing.Graphics]::FromImage($big)
        $g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::NearestNeighbor
        $g.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::Half
        $g.DrawImage($src, 0, 0, ($w * $k), ($ht * $k)); $g.Dispose()
        $big.Save("$path-x$k.png", [System.Drawing.Imaging.ImageFormat]::Png); $big.Dispose()
    }
    $src.Dispose()
}
# SaveZoneC: zone C's box, from its header ("C . LIVE AGENTS", inside the
# frame): its left edge 6 cells left of "LIVE AGENTS", its right edge the
# first frame line after it on that row, its top one row above, its bottom
# the first row below with a corner at the left edge.
function SaveZoneC([string]$label) {
    $hm = [Term]::Find($script:h, 'LIVE AGENTS')
    if ($hm.Count -eq 0) { return }
    $a = $hm[$hm.Count - 1]
    $rows = Rows
    $r0 = -1; for ($i = 0; $i -lt $rows.Count; $i++) { if ($rows[$i].Contains('LIVE AGENTS')) { $r0 = $i } }
    if ($r0 -lt 0) { return }
    $c = $rows[$r0].LastIndexOf('LIVE AGENTS')
    $left = $c - 6
    $right = $rows[$r0].IndexOf($bar, $c)
    if ($left -lt 0 -or $right -lt 0) { return }
    $r1 = -1
    for ($i = $r0 + 1; $i -lt $rows.Count -and $r1 -lt 0; $i++) { if ($rows[$i].Length -gt $left -and $corners -contains $rows[$i].Substring($left, 1)) { $r1 = $i } }
    if ($r1 -lt 0) { $r1 = $rows.Count - 1 }
    $cell = Cell
    $x0 = [int]($a.X - 6 * $cell); $x1 = [int]($a.X + ($right - $c + 1) * $cell)
    $y0 = [int]($a.Y - $a.Height); $y1 = [int]($a.Y + ($r1 - $r0 + 1) * $a.Height)
    Crop $x0 $y0 ($x1 - $x0) ($y1 - $y0) 2 "$out\$label-zonec"
    $script:log += "zone C $label`: rows $($r0 - 1) to $r1, cells $left to $right, at $x0,$y0, $($x1 - $x0) x $($y1 - $y0) px"
}
# SaveZoneB: zone B's box, from its title in the top edge
# ("B . CONVERSATION"): its left edge 7 cells left of "CONVERSATION", its
# right edge the first top-right corner after it on that row, its bottom the
# first row below with a corner at the left edge.
function SaveZoneB([string]$label) {
    $hm = [Term]::Find($script:h, 'CONVERSATION')
    if ($hm.Count -eq 0) { return }
    $a = $hm[0]
    $rows = Rows
    $r0 = -1; for ($i = 0; $i -lt $rows.Count -and $r0 -lt 0; $i++) { if ($rows[$i].Contains('CONVERSATION')) { $r0 = $i } }
    if ($r0 -lt 0) { return }
    $c = $rows[$r0].IndexOf('CONVERSATION')
    $left = $c - 7
    $right = $rows[$r0].IndexOf([string][char]0x256E, $c)
    if ($left -lt 0 -or $right -lt 0) { return }
    $r1 = -1
    for ($i = $r0 + 1; $i -lt $rows.Count -and $r1 -lt 0; $i++) { if ($rows[$i].Length -gt $left -and $corners -contains $rows[$i].Substring($left, 1)) { $r1 = $i } }
    if ($r1 -lt 0) { $r1 = $rows.Count - 1 }
    $cell = Cell
    $x0 = [int]($a.X - 7 * $cell); $x1 = [int]($a.X + ($right - $c + 1) * $cell)
    $y0 = [int]$a.Y; $y1 = [int]($a.Y + ($r1 - $r0 + 1) * $a.Height)
    Crop $x0 $y0 ($x1 - $x0) ($y1 - $y0) 2 "$out\$label-zoneb"
    $script:log += "zone B $($label): rows $r0 to $r1, cells $left to $right, at $x0,$y0, $($x1 - $x0) x $($y1 - $y0) px"
}
function SaveShot([string]$label) {
    [Shot]::Save($script:h, "$out\$label.png")
    Set-Content -LiteralPath "$out\$label.txt" -Value ((Rows) -join "`n") -Encoding utf8
    SaveZoneC $label
    SaveZoneB $label
    try { SaveSidebar $label } catch { }
}
# Spot is the screen point of the cell DX cells right of the first cell of
# the first occurrence of text, and the row's height.
function Spot([string]$text, [double]$dx) {
    $rects = [Term]::Find($script:h, $text)
    if ($rects.Count -eq 0) { throw "'$text' not on screen" }
    $rc = $rects[0]
    $cell = Cell
    return @([int]($rc.X + ($dx + 0.5) * $cell), [int]($rc.Y + $rc.Height / 2), $rc.Height)
}
# CellPoint is the screen point of the middle of grid cell (col, row),
# anchored on the title row's "AGENT COMMAND CENTER"; a negative col counts
# from the right edge.
function CellPoint([int]$col, [int]$row) {
    $rects = [Term]::Find($script:h, 'AGENT COMMAND CENTER')
    if ($rects.Count -eq 0) { throw 'no title row' }
    $rc = $rects[0]
    $rows = Rows
    $c0 = $rows[0].IndexOf('AGENT COMMAND CENTER')
    $cols = ($rows | ForEach-Object { $_.Length } | Measure-Object -Maximum).Maximum
    if ($col -lt 0) { $col = $cols + $col }
    # The cell from two texts of the title row whose columns are known: UI
    # Automation gives their X right but their widths about 3.5% wide.
    $far = [Term]::Find($script:h, 'not a live session') | Sort-Object Y | Select-Object -First 1
    $c2 = $rows[0].IndexOf('not a live session')
    $cell = ($far.X - $rc.X) / ($c2 - $c0)
    return @([int]($rc.X + ($col - $c0 + 0.5) * $cell), [int]($rc.Y + ($row + 0.5) * $rc.Height), $rc.Height)
}
function GlideTo([int]$tx, [int]$ty) {
    $from = [System.Windows.Forms.Cursor]::Position
    for ($k = 1; $k -le 10; $k++) {
        [Term]::Glide([int]($from.X + ($tx - $from.X) * $k / 10), [int]($from.Y + ($ty - $from.Y) * $k / 10))
        Start-Sleep -Milliseconds 25
    }
}
function SaveSidebar([string]$label) {
    $rows = Rows
    $cols = ($rows | ForEach-Object { $_.Length } | Measure-Object -Maximum).Maximum
    $a = CellPoint ($cols - 12) 1
    $b = CellPoint ($cols - 1) 14
    $cell = ($b[0] - $a[0]) / 11
    $x0 = [int]($a[0] - $cell / 2); $y0 = [int]($a[1] - $a[2] / 2)
    $x1 = [int]($b[0] + $cell / 2); $y1 = [int]($b[1] + $a[2] / 2)
    Crop $x0 $y0 ($x1 - $x0) ($y1 - $y0) 4 "$out\$label-sidebar"
}
function Alive() { return [bool]([Term]::Windows() | Where-Object { $_.H -eq $script:h }) }
# An assertion, after its step has saved its screen: a failure stops the plan and the run
# exits 1, the screen and the log kept.
$assertionsPassed = 0; $assertionsFailed = 0
function Assert-Screen([bool]$held, [string]$label, [string]$claim) {
    if ($held) {
        $script:assertionsPassed++
        $script:log += "assertion passed $label at $($sw.ElapsedMilliseconds) ms: $claim"
        return
    }
    $script:assertionsFailed++
    $script:log += "ASSERTION FAILED $label at $($sw.ElapsedMilliseconds) ms: $claim"
    Fail "assertion failed, step $($label): $claim; its screen and log.txt are in %TEMP%\acc-wt\$name\results"
}
$title = "ACCWT$name"
$wtArgs = @('-w', 'new')
switch ($mode) {
    'max' { $wtArgs += '--maximized' }
    default { $wh = $mode -split 'x'; $wtArgs += @('--size', "$($wh[0]),$($wh[1])") }
}
$wtArgs += @('new-tab')
# Start-Process joins its arguments with spaces, unquoted: what may hold one is quoted here.
if ($wtProfile) { $wtArgs += @('--profile', "`"$wtProfile`"") }
$wtArgs += @('--title', $title, '--suppressApplicationTitle', '-d', "`"$base`"", 'cmd.exe', '/c', "`"$launch`"")
Start-Process wt.exe -ArgumentList $wtArgs
$sw = [System.Diagnostics.Stopwatch]::StartNew()
$end = (Get-Date).AddSeconds(20); $w = $null
while (-not $w -and (Get-Date) -lt $end) { Start-Sleep -Milliseconds 50; $w = [Term]::Windows() | Where-Object { $_.Title -eq $title -and $_.Class -eq 'CASCADIA_HOSTING_WINDOW_CLASS' } | Select-Object -First 1 }
if (-not $w) { throw 'no window' }
$script:h = $w.H; [Term]::Target = $w.H
if ($mode -ne 'max') { [WinMove40]::TopLeft($script:h); Start-Sleep -Milliseconds 300 }
[void][Term]::Focus()
$log += "window found at $($sw.ElapsedMilliseconds) ms"
try {
foreach ($step in ($plan -split ';' | Where-Object { $_ -ne '' })) {
    $p = $step -split ':'
    $at = [int]$p[2]
    $wait = $at - $sw.ElapsedMilliseconds
    if ($wait -gt 0) { Start-Sleep -Milliseconds $wait }
    switch ($p[0]) {
        'shot' { SaveShot $p[1]; $log += "shot $($p[1]) at $($sw.ElapsedMilliseconds) ms" }
        'expect' {
            $found = $false; $limit = $sw.ElapsedMilliseconds + 25000
            while (-not $found -and $sw.ElapsedMilliseconds -lt $limit) {
                foreach ($r in (Rows)) { if ($r.Contains($p[3])) { $found = $true } }
                if (-not $found) { Start-Sleep -Milliseconds 150 }
            }
            Start-Sleep -Milliseconds 800
            SaveShot $p[1]
            Assert-Screen $found $p[1] "'$($p[3])' $(if ($found) { 'found' } else { 'not found within 25 s' })"
        }
        'click' {
            $dx = 0; if ($p.Count -gt 4) { $dx = [double]$p[4] }
            $xy = Spot $p[3] $dx
            [void][Term]::Focus()
            [Term]::Click($xy[0], $xy[1])
            Start-Sleep -Milliseconds 700
            SaveShot $p[1]
            $log += "click $($p[1]) at $($sw.ElapsedMilliseconds) ms: '$($p[3])' + $dx = $($xy[0]),$($xy[1])"
        }
        'dclick' {
            $dx = 0; if ($p.Count -gt 4) { $dx = [double]$p[4] }
            $xy = Spot $p[3] $dx
            [void][Term]::Focus()
            [Term]::Click($xy[0], $xy[1])
            Start-Sleep -Milliseconds 120
            [Term]::Click($xy[0], $xy[1])
            Start-Sleep -Milliseconds 700
            SaveShot $p[1]
            $log += "dclick $($p[1]) at $($sw.ElapsedMilliseconds) ms: '$($p[3])' + $dx = $($xy[0]),$($xy[1])"
        }
        'hoverat' {
            $xy = CellPoint ([int]$p[3]) ([int]$p[4])
            GlideTo $xy[0] $xy[1]
            Start-Sleep -Milliseconds 700
            SaveShot $p[1]
            $log += "hoverat $($p[1]) at $($sw.ElapsedMilliseconds) ms: cell $($p[3]),$($p[4]) = $($xy[0]),$($xy[1])"
        }
        'clickcell' {
            $xy = CellPoint ([int]$p[3]) ([int]$p[4])
            [void][Term]::Focus()
            [Term]::Click($xy[0], $xy[1])
            Start-Sleep -Milliseconds 700
            SaveShot $p[1]
            $log += "clickcell $($p[1]) at $($sw.ElapsedMilliseconds) ms: cell $($p[3]),$($p[4]) = $($xy[0]),$($xy[1])"
        }
        'hoveroff' {
            $xy = CellPoint 10 10
            GlideTo $xy[0] $xy[1]
            Start-Sleep -Milliseconds 700
            SaveShot $p[1]
            $log += "hoveroff $($p[1]) at $($sw.ElapsedMilliseconds) ms"
        }
        'chord' {
            [void][Term]::Focus()
            $ks = $p[3] -split '\.'
            $mods = [uint16[]]@($ks[0..($ks.Count - 2)] | ForEach-Object { [Convert]::ToUInt16($_, 16) })
            [Term]::Chord($mods, [Convert]::ToUInt16($ks[$ks.Count - 1], 16))
            Start-Sleep -Milliseconds 700
            SaveShot $p[1]
            $log += "chord $($p[1]) at $($sw.ElapsedMilliseconds) ms: $($p[3])"
        }
        'drag' {
            $rc = [Term]::Rect($script:h)
            $x0 = $rc.R - 3; $y0 = $rc.B - 3
            [void][Term]::Focus()
            [Term]::Drag($x0, $y0, $x0 + [int]$p[3], $y0 + [int]$p[4])
            Start-Sleep -Milliseconds 1200
            $r2 = [Term]::Rect($script:h)
            SaveShot $p[1]
            $log += "drag $($p[1]) at $($sw.ElapsedMilliseconds) ms: corner $x0,$y0 by $($p[3]),$($p[4]): window $($rc.R - $rc.L)x$($rc.B - $rc.T) px to $($r2.R - $r2.L)x$($r2.B - $r2.T) px"
        }
        'homeclick' {
            # The home screen's picture has no text but the field's: a cell
            # anchored on a text of known cell. p[3] col, p[4] row; p[5] and
            # p[6] the anchor's col and row; p[7] the cell's width; p[8] the
            # anchor's text (no colon).
            $rects = [Term]::Find($script:h, $p[8])
            if ($rects.Count -eq 0) { throw "homeclick: '$($p[8])' not on screen" }
            $rc = $rects[0]; $cw = [double]$p[7]
            $x = [int]($rc.X + ([int]$p[3] - [int]$p[5] + 0.5) * $cw)
            $y = [int]($rc.Y + ([int]$p[4] - [int]$p[6] + 0.5) * $rc.Height)
            [void][Term]::Focus()
            [Term]::Click($x, $y)
            Start-Sleep -Milliseconds 700
            SaveShot $p[1]
            $log += "homeclick $($p[1]) at $($sw.ElapsedMilliseconds) ms: cell $($p[3]),$($p[4]) = $x,$y"
        }
        'wheel' {
            # WHEEL:LABEL:MS:COL:ROW:ANCOL:ANROW:CW:ANCHOR:NOTCHES, as HOMECLICK
            # but the wheel turned NOTCHES, plus up, minus down (plan 54).
            $rects = [Term]::Find($script:h, $p[8])
            if ($rects.Count -eq 0) { throw "wheel: '$($p[8])' not on screen" }
            $rc = $rects[0]; $cw = [double]$p[7]
            $x = [int]($rc.X + ([int]$p[3] - [int]$p[5] + 0.5) * $cw)
            $y = [int]($rc.Y + ([int]$p[4] - [int]$p[6] + 0.5) * $rc.Height)
            [void][Term]::Focus()
            [Term]::Wheel($x, $y, [int]$p[9])
            Start-Sleep -Milliseconds 700
            SaveShot $p[1]
            $log += "wheel $($p[1]) at $($sw.ElapsedMilliseconds) ms: cell $($p[3]),$($p[4]) = $x,$y, $($p[9]) notches"
        }
        'fieldsel' {
            # FIELDSEL:LABEL:MS:ANCHOR:DX:N:CW: the terminal's own selection, a
            # drag with Shift held over N cells from DX cells right of the
            # anchor's first cell, CW pixels a cell (plan 51: the field's
            # path is terminal text). The selection is shot, never copied.
            $rects = [Term]::Find($script:h, $p[3])
            if ($rects.Count -eq 0) { throw "fieldsel: '$($p[3])' not on screen" }
            $rc = $rects[0]; $cw = [double]$p[6]
            $x0 = [int]($rc.X + ([double]$p[4] + 0.5) * $cw); $y = [int]($rc.Y + $rc.Height / 2)
            $x1 = [int]($x0 + ([int]$p[5] - 1) * $cw)
            [void][Term]::Focus()
            [Term]::KeyDown(0x10)
            [Term]::Drag($x0, $y, $x1, $y)
            [Term]::KeyUp(0x10)
            Start-Sleep -Milliseconds 500
            SaveShot $p[1]
            $log += "fieldsel $($p[1]) at $($sw.ElapsedMilliseconds) ms: $x0,$y to $x1,$y"
        }
        'expectfield' {
            # EXPECTFIELD:LABEL:MS: the screen holds -expect, not named in the log: it may
            # be a path.
            Start-Sleep -Milliseconds 300
            SaveShot $p[1]
            $held = (Text).Contains($expect)
            Assert-Screen $held $p[1] "the expected text $(if ($held) { 'on screen' } else { 'not on screen' })"
        }
        'clickbelow' {
            $rects = [Term]::Find($script:h, $p[3])
            if ($rects.Count -eq 0) { throw "clickbelow: '$($p[3])' not on screen" }
            $rc = $rects[0]; $cw = Cell
            $x = [int]($rc.X + ([double]$p[5] + 0.5) * $cw)
            $y = [int]($rc.Y + ([int]$p[4] + 0.5) * $rc.Height)
            [void][Term]::Focus()
            [Term]::Click($x, $y)
            Start-Sleep -Milliseconds 700
            SaveShot $p[1]
            $log += "clickbelow $($p[1]) at $($sw.ElapsedMilliseconds) ms: '$($p[3])' + $($p[4]) rows, $($p[5]) cells = $x,$y"
        }
        'clickrow' {
            $rows = Rows
            $ri = -1; for ($i = 0; $i -lt $rows.Count -and $ri -lt 0; $i++) { if ($rows[$i].Contains($p[3])) { $ri = $i } }
            if ($ri -lt 0) { throw "clickrow: '$($p[3])' not on screen" }
            # 'at': the text's own column on its row (no wide character before it)
            $col = if ($p[4] -eq 'at') { $rows[$ri].IndexOf($p[3]) } else { [int]$p[4] }
            $xy = CellPoint $col $ri
            [void][Term]::Focus()
            [Term]::Click($xy[0], $xy[1])
            Start-Sleep -Milliseconds 700
            SaveShot $p[1]
            $log += "clickrow $($p[1]) at $($sw.ElapsedMilliseconds) ms: '$($p[3])' row $ri col $col = $($xy[0]),$($xy[1])"
        }
        'dialog' {
            # DIALOG:LABEL:MS[:TITLE], the upload dialog's title by default
            # (plan 52: the folder dialog's).
            $want = if ($p.Count -gt 3) { $p[3] } else { 'Upload from computer' }
            $dw = $null; $until = (Get-Date).AddSeconds(15)
            while (-not $dw -and (Get-Date) -lt $until) { Start-Sleep -Milliseconds 100; $dw = [Term]::Windows() | Where-Object { $_.Class -eq '#32770' -and $_.Title -eq $want } | Select-Object -First 1 }
            if (-not $dw) { throw "dialog: no dialog '$want'" }
            Start-Sleep -Milliseconds 700
            [Shot]::Save($dw.H, "$out\$($p[1])-dialog.png")
            $script:dlg = $dw.H; [Term]::Target = $dw.H
            $log += "dialog $($p[1]) at $($sw.ElapsedMilliseconds) ms: $($dw)"
        }
        'type' {
            $text = ($p[3..($p.Count - 1)] -join ':')
            if (-not $script:dlg) { [void][Term]::Focus() }
            [Term]::Type($text)
            Start-Sleep -Milliseconds 300
            if ($script:dlg) { [Shot]::Save($script:dlg, "$out\$($p[1])-dialog.png") }
            $log += "type $($p[1]) at $($sw.ElapsedMilliseconds) ms: $($text.Length) characters"
        }
        'enter' {
            [Term]::Press(0x0D)
            Start-Sleep -Milliseconds 800
            $open = [Term]::Windows() | Where-Object { $_.H -eq $script:dlg }
            if (-not $open) { $script:dlg = $null; [Term]::Target = $script:h; Start-Sleep -Milliseconds 700 }
            SaveShot $p[1]
            $log += "enter $($p[1]) at $($sw.ElapsedMilliseconds) ms: the dialog $(if ($open) { 'still open' } else { 'closed' })"
        }
        'esc' {
            # ESC:LABEL:MS: Esc to the dialog, as ENTER (plan 52: cancel).
            [Term]::Press(0x1B)
            Start-Sleep -Milliseconds 800
            $open = [Term]::Windows() | Where-Object { $_.H -eq $script:dlg }
            if (-not $open) { $script:dlg = $null; [Term]::Target = $script:h; Start-Sleep -Milliseconds 700 }
            SaveShot $p[1]
            $log += "esc $($p[1]) at $($sw.ElapsedMilliseconds) ms: the dialog $(if ($open) { 'still open' } else { 'closed' })"
        }
        'cclick' {
            $dx = 0; if ($p.Count -gt 4) { $dx = [double]$p[4] }
            $xy = Spot $p[3] $dx
            [void][Term]::Focus()
            [Term]::CtrlClick($xy[0], $xy[1])
            Start-Sleep -Milliseconds 700
            SaveShot $p[1]
            $log += "cclick $($p[1]) at $($sw.ElapsedMilliseconds) ms: '$($p[3])' + $dx = $($xy[0]),$($xy[1])"
        }
        'ctrlhold' {
            [void][Term]::Focus()
            [Term]::CtrlHold([int]$p[3])
            Start-Sleep -Milliseconds 700
            SaveShot $p[1]
            $log += "ctrlhold $($p[1]) at $($sw.ElapsedMilliseconds) ms: $($p[3]) key-downs"
        }
        'altgr' {
            [void][Term]::Focus()
            foreach ($vk in ($p[3] -split '\.')) { [AltGr67]::Press([Convert]::ToByte($vk, 16)); Start-Sleep -Milliseconds 150 }
            Start-Sleep -Milliseconds 700
            SaveShot $p[1]
            $log += "altgr $($p[1]) at $($sw.ElapsedMilliseconds) ms: $($p[3])"
        }
        'press' {
            [void][Term]::Focus()
            foreach ($vk in ($p[3] -split '\.')) { [Term]::Press([Convert]::ToUInt16($vk, 16)); Start-Sleep -Milliseconds 150 }
            Start-Sleep -Milliseconds 700
            SaveShot $p[1]
            $log += "press $($p[1]) at $($sw.ElapsedMilliseconds) ms: $($p[3])"
        }
        default { Fail "step '$step': no such step" }
    }
}
} catch {
    $log += "STOPPED: $($_.Exception.Message)"
    throw
} finally {
    if (Alive) {
        $rows = Rows
        $log += "grid: $($rows.Count) rows, $(($rows | Measure-Object -Maximum Length).Maximum) columns at most"
        [void][Term]::Focus()
        [Term]::Chord([uint16[]]@(0x11), 0x51); Start-Sleep -Milliseconds 500
        if (Alive) { if ((Text).Contains('quit?')) { [void][Term]::Focus(); [Term]::Chord([uint16[]]@(0x11), 0x51) } }
        $end = (Get-Date).AddSeconds(10)
        while ((Get-Date) -lt $end -and (Alive)) { Start-Sleep -Milliseconds 200 }
        if (Alive) { [Term]::Close($script:h); $log += 'window closed by message' }
    }
    $log += $(if ($assertionsPassed + $assertionsFailed -eq 0) { 'assertions: none, so this exit code says only that every step ran' } else { "assertions: $assertionsPassed passed, $assertionsFailed failed" })
    Set-Content -LiteralPath "$out\log.txt" -Value $log -Encoding utf8
    $log | ForEach-Object { Write-Output $_ }
}
