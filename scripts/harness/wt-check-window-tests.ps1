# wt-check-window-tests.ps1: the tests of wt-check.ps1 that need a screen, in a real
# Windows Terminal, on a recording, with no CLI and no quota:
#   powershell -NoProfile -ExecutionPolicy Bypass -File scripts\harness\wt-check-window-tests.ps1 -exe "$env:TEMP\acc-check\acc.exe"
# Its keys and clicks are real: nothing else uses the machine during a run, about five
# minutes at both sizes, and the owner is told first. An assertion that passes must exit
# 0; one that fails must exit 1 and keep its screen and its log. It also commits and
# pushes through the dashboard on a disposable repository with a local bare remote, hooks
# added after the harness's checks, and reads the result with git. All of it happens in a
# new folder, %TEMP%\acc-wt-window-tests-<random>, with TEMP pointing inside it for the
# harness; it deletes nothing. Prints PASS, FAIL or NOT RUN for each case and the counts;
# exits 1 on any FAIL. -only is a pattern on the cases' names, -modes a comma-separated list.
# ASCII only: Windows PowerShell 5.1 reads a BOM-less script as ANSI.
[CmdletBinding(PositionalBinding = $false)]
param([string]$exe = '', [string]$recording = '', [string]$modes = 'max,90x30', [string]$wtProfile = 'Agent Command Center', [string]$only = '')
$ErrorActionPreference = 'Stop'
if (-not $exe) { throw '-exe: the path of a build of the dashboard' }
if (-not $recording) { $recording = Join-Path $PSScriptRoot '..\..\testdata\mock\codex-commands.jsonl' }
$exe = [IO.Path]::GetFullPath($exe); $recording = [IO.Path]::GetFullPath($recording)
$harness = Join-Path $PSScriptRoot 'wt-check.ps1'
$root = Join-Path ([IO.Path]::GetFullPath($env:TEMP)) ('acc-wt-window-tests-' + [guid]::NewGuid().ToString('N').Substring(0, 8))
$temp = "$root\tmp"
[void][IO.Directory]::CreateDirectory($temp)
$script:passed = 0; $script:failed = 0
$missing = 'no such text anywhere'

function Report([bool]$ok, [string]$name, [string]$detail) {
    foreach ($form in $root, $root.Replace('\', '/')) { $name = $name.Replace($form, '<root>'); $detail = $detail.Replace($form, '<root>') }
    if ($ok) { $script:passed++; "PASS  $name" } else { $script:failed++; "FAIL  $name : $detail" }
}
function Quote([string]$a) { if ($a -match '[\s"]') { return '"' + $a.Replace('"', '\"') + '"' } else { return $a } }
function Start-Child([string]$file, [string]$arguments, [hashtable]$vars) {
    $psi = New-Object Diagnostics.ProcessStartInfo
    $psi.FileName = $file; $psi.Arguments = $arguments; $psi.WorkingDirectory = $root
    $psi.UseShellExecute = $false; $psi.CreateNoWindow = $true
    $psi.RedirectStandardOutput = $true; $psi.RedirectStandardError = $true
    foreach ($k in @($psi.EnvironmentVariables.Keys)) { if ($k -like 'GIT_*') { $psi.EnvironmentVariables.Remove($k) } }
    foreach ($k in $vars.Keys) { $psi.EnvironmentVariables[$k] = $vars[$k] }
    $p = [Diagnostics.Process]::Start($psi)
    return @{ P = $p; O = $p.StandardOutput.ReadToEndAsync(); E = $p.StandardError.ReadToEndAsync() }
}
function End-Child($c, [int]$ms) {
    if (-not $c.P.WaitForExit($ms)) { $c.P.Kill(); return @{ Exit = -1; Out = ''; Err = "killed after $ms ms" } }
    return @{ Exit = $c.P.ExitCode; Out = $c.O.Result; Err = $c.E.Result }
}
function Start-Harness([string[]]$a, [string]$tempDir) {
    $line = '-NoProfile -ExecutionPolicy Bypass -File ' + (Quote $harness) + ' ' + (($a | ForEach-Object { Quote $_ }) -join ' ')
    return Start-Child 'powershell.exe' $line @{ TEMP = $tempDir; TMP = $tempDir }
}
function Left() {
    $titled = @(Get-Process -ErrorAction SilentlyContinue | Where-Object { $_.MainWindowTitle -like 'ACCWT*' }).Count
    $builds = @(Get-CimInstance Win32_Process | Where-Object { $_.ExecutablePath -eq $exe }).Count
    return "$titled windows, $builds builds"
}
function Wanted([string]$name) { return (-not $only) -or ($name -match $only) }
function Read-Log([string]$results) { if ([IO.File]::Exists("$results\log.txt")) { return [IO.File]::ReadAllText("$results\log.txt") } else { return '' } }

# One run: its exit, its log's lines and files checked against what it must show.
function Run-Case([string]$name, [string[]]$a, [string]$mode, [bool]$passes, [string[]]$logHas, [string[]]$files, [string[]]$notFiles, [string]$errHas = '', [string]$tempDir = $temp) {
    if (-not (Wanted $name)) { return }
    $outName = 'c' + (++$script:counter)
    $r = End-Child (Start-Harness ($a + @('-exe', $exe, '-wtProfile', $wtProfile, '-mode', $mode, '-outName', $outName)) $tempDir) 90000
    $results = "$tempDir\acc-wt\$outName-codex-$mode\results"
    $log = Read-Log $results
    $problems = @()
    if ($passes -and $r.Exit -ne 0) { $problems += "exit $($r.Exit), not 0: $(($r.Err -split "`n")[0..2] -join ' ')" }
    if (-not $passes -and $r.Exit -ne 1) { $problems += "exit $($r.Exit), not 1" }
    foreach ($w in $logHas) { if (-not $log.Contains($w)) { $problems += "the log lacks [$w]" } }
    foreach ($f in $files) { if (-not [IO.File]::Exists("$results\$f")) { $problems += "no $f" } }
    foreach ($f in $notFiles) { if ([IO.File]::Exists("$results\$f")) { $problems += "$f exists: the plan went on" } }
    if ($errHas -and -not ($r.Err -replace '\s+', '').Contains(($errHas -replace '\s+', ''))) { $problems += "stderr lacks [$errHas]" }
    if ((Left) -ne '0 windows, 0 builds') { $problems += "left open: $(Left)" }
    Report ($problems.Count -eq 0) "$mode, $name" ($problems -join '; ')
}

$common = @('-mock', $recording)
$gitCmd = Get-Command git.exe -CommandType Application -ErrorAction SilentlyContinue | Select-Object -First 1
$emptyHome = "$root\emptyhome"; [void][IO.Directory]::CreateDirectory($emptyHome)
$cleanGit = @{ GIT_CONFIG_NOSYSTEM = '1'; HOME = $emptyHome; USERPROFILE = $emptyHome; XDG_CONFIG_HOME = $emptyHome }
function Git-Clean([string]$arguments) {
    $r = End-Child (Start-Child $gitCmd.Source $arguments $cleanGit) 30000
    if ($r.Exit -ne 0) { throw "git $arguments : $($r.Err)" }
    return $r.Out.Trim()
}

foreach ($mode in ($modes -split ',')) {
    "=== $mode ==="
    Run-Case 'expect, the text on screen: exit 0' ($common + @('-plan', 'expect:00-seen:0:to answer;shot:01-after:5000')) $mode $true @('assertion passed 00-seen', 'assertions: 1 passed, 0 failed') @('00-seen.png', '00-seen.txt', '01-after.png', 'log.txt') @()
    Run-Case 'expect, the text absent: exit 1, its screen and log kept, the plan stopped' ($common + @('-plan', "expect:00-missing:0:$missing;shot:01-after:1000")) $mode $false @('ASSERTION FAILED 00-missing', 'STOPPED', 'assertions: 0 passed, 1 failed') @('00-missing.png', '00-missing.txt', 'log.txt') @('01-after.png') 'assertion failed'
    Run-Case 'expectfield, the text on screen: exit 0' ($common + @('-expect', 'LIVE AGENTS', '-plan', 'expect:00-ready:0:to answer;expectfield:01-field:3000')) $mode $true @('assertion passed 01-field', 'assertions: 2 passed, 0 failed') @('01-field.png', '01-field.txt', 'log.txt') @()
    Run-Case 'expectfield, the text absent: exit 1, its screen and log kept, the plan stopped' ($common + @('-expect', $missing, '-plan', 'expect:00-ready:0:to answer;expectfield:01-field:3000;shot:02-after:5000')) $mode $false @('assertion passed 00-ready', 'ASSERTION FAILED 01-field', 'STOPPED', 'assertions: 1 passed, 1 failed') @('01-field.png', '01-field.txt', 'log.txt') @('02-after.png') 'assertion failed'
    Run-Case 'no assertion: exit 0, and the log says what that means' ($common + @('-plan', 'shot:00-only:2000')) $mode $true @('assertions: none') @('00-only.png', 'log.txt') @()
    Run-Case 'a step that cannot act: exit 1' ($common + @('-plan', "shot:00-first:2000;click:01-none:3000:$missing;shot:02-after:5000")) $mode $false @('STOPPED', 'assertions: none') @('00-first.png', 'log.txt') @('02-after.png') 'not on screen'
    $spaced = "$root\with space $mode"; [void][IO.Directory]::CreateDirectory($spaced)
    Run-Case 'a %TEMP% with a space: cmd.exe /c runs the quoted launch.cmd' ($common + @('-plan', 'expect:00-seen:0:to answer')) $mode $true @('assertions: 1 passed, 0 failed') @('00-seen.png') @() '' $spaced

    $case = 'a commit and a push through the dashboard, on a fixture with a local remote'
    if ((Wanted $case) -and $gitCmd) {
        # A fixture as made by hand, hooks added once the harness has passed its checks.
        $tempDir = "$root\repo-$mode"
        $fixtures = "$tempDir\acc-wt-fixtures"; $repo = "$fixtures\repo"; $bare = "$fixtures\repo-remote.git"
        [void][IO.Directory]::CreateDirectory($repo); [void][IO.Directory]::CreateDirectory($bare)
        [void](Git-Clean "init -q `"$repo`""); [void](Git-Clean "init -q --bare `"$bare`"")
        [void](Git-Clean "-C `"$repo`" config --local user.name fixture"); [void](Git-Clean "-C `"$repo`" config --local user.email fixture@example.invalid")
        [IO.File]::WriteAllText("$repo\tracked.txt", "first`n")
        [void](Git-Clean "-C `"$repo`" add tracked.txt"); [void](Git-Clean "-C `"$repo`" commit -q -m first")
        [void](Git-Clean "-C `"$repo`" remote add origin ..\repo-remote.git"); [void](Git-Clean "-C `"$repo`" push -q -u origin HEAD")
        [IO.File]::WriteAllText("$repo\staged.txt", "second`n")
        [void](Git-Clean "-C `"$repo`" add staged.txt")
        $before = Git-Clean "-C `"$repo`" rev-parse HEAD"
        $plan = 'expect:00-end:0:to answer;clickcell:01-source:0:-4:8;expect:02-staged:0:staged.txt;clickcell:03-field:0:-25:7;type:04-message:0:window test commit;clickcell:05-chevron:0:-12:10;clickcell:06-item:0:-20:13;shot:07-done:0'
        $outName = 'repo'
        $c = Start-Harness ($common + @('-exe', $exe, '-wtProfile', $wtProfile, '-mode', $mode, '-outName', $outName, '-fixture', 'repo', '-git', '-plan', $plan)) $tempDir
        $launch = "$tempDir\acc-wt\$outName-codex-$mode\launch.cmd"
        for ($i = 0; $i -lt 300 -and -not [IO.File]::Exists($launch); $i++) { Start-Sleep -Milliseconds 100 }
        $ranRepo = "$root\ran-repo-$mode.txt"; $ranRemote = "$root\ran-remote-$mode.txt"
        [IO.File]::WriteAllText("$repo\.git\hooks\pre-commit", "#!/bin/sh`necho ran > '$($ranRepo.Replace('\', '/'))'`n")
        [IO.File]::WriteAllText("$bare\hooks\post-receive", "#!/bin/sh`necho ran > '$($ranRemote.Replace('\', '/'))'`n")
        $r = End-Child $c 120000
        Start-Sleep -Seconds 2
        $after = Git-Clean "-C `"$repo`" rev-parse HEAD"
        $subject = Git-Clean "-C `"$repo`" log -1 --format=%s"
        $pushed = Git-Clean "-C `"$bare`" rev-parse refs/heads/master"
        $problems = @()
        if ($r.Exit -ne 0) { $problems += "exit $($r.Exit): $(($r.Err -split "`n")[0..2] -join ' ')" }
        if ($after -eq $before -or $subject -ne 'window test commit') { $problems += "no commit: the subject is [$subject]" }
        if ($pushed -ne $after) { $problems += 'the remote did not receive it' }
        if ([IO.File]::Exists($ranRepo)) { $problems += 'the repository''s hook ran' }
        if ([IO.File]::Exists($ranRemote)) { $problems += 'the remote''s hook ran' }
        if ((Left) -ne '0 windows, 0 builds') { $problems += "left open: $(Left)" }
        Report ($problems.Count -eq 0) "$mode, $case, no hook run" ($problems -join '; ')
    } elseif (Wanted $case) {
        "NOT RUN  $mode, $case : no git.exe on the PATH"
    }
}

''
"$($script:passed) passed, $($script:failed) failed"
"folder: %TEMP%\$(Split-Path -Leaf $root)"
if ($script:failed -ne 0) { exit 1 }
