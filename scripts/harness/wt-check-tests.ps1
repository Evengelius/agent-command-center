# wt-check-tests.ps1: the tests of wt-check.ps1, to run after any change to it:
#   powershell -NoProfile -ExecutionPolicy Bypass -File scripts\harness\wt-check-tests.ps1
# Every call passes -noLaunch, so no terminal opens and no key is sent even if a check
# regresses. All of it happens in a new folder, %TEMP%\acc-wt-tests-<random>, with TEMP
# pointing inside it for the harness. A snapshot of that whole folder (each path, size,
# attributes and SHA-256, links not followed) proves that a refused call changed
# nothing, so the files placed outside the harness's folders stay as they were. A
# refusal must also give its expected reason, so a crash does not pass for a check. It
# deletes nothing: the folder stays, to go with a plan's temporary files. Prints PASS,
# FAIL or NOT RUN for each case and the counts; exits 1 on any FAIL.
# Its git runs only on repositories it makes in that folder, under a hostile environment
# for -git's containment. Its network URLs name 127.0.0.1 port 9, where nothing listens:
# git must refuse them before connecting. Assertions need a screen: they are in
# wt-check-window-tests.ps1.
# ASCII only: Windows PowerShell 5.1 reads a BOM-less script as ANSI.
$ErrorActionPreference = 'Stop'
$harness = Join-Path $PSScriptRoot 'wt-check.ps1'
$root = Join-Path ([IO.Path]::GetFullPath($env:TEMP)) ('acc-wt-tests-' + [guid]::NewGuid().ToString('N').Substring(0, 8))
New-Item -ItemType Directory -Path $root | Out-Null
$script:passed = 0; $script:failed = 0; $script:notRun = 0
$sha = [Security.Cryptography.SHA256]::Create()

function Report([bool]$ok, [string]$name, [string]$detail) {
    foreach ($form in $root, $root.Replace('\', '/')) { $name = $name.Replace($form, '<root>'); $detail = $detail.Replace($form, '<root>') }
    if ($ok) { $script:passed++; "PASS  $name" } else { $script:failed++; "FAIL  $name : $detail" }
}
function NotRun([string]$name, [string]$why) { $script:notRun++; "NOT RUN  $name : $why" }
function MakeDir([string]$path) { [void][IO.Directory]::CreateDirectory($path); return $path }
function Put([string]$path, [string]$text) { [void][IO.Directory]::CreateDirectory([IO.Path]::GetDirectoryName($path)); [IO.File]::WriteAllText($path, $text); return $path }
function Junction([string]$link, [string]$target) { New-Item -ItemType Junction -Path $link -Target $target | Out-Null }
function Walk([string]$path) {
    foreach ($e in (New-Object IO.DirectoryInfo $path).GetFileSystemInfos()) {
        if ($e.Name -like '__PSScriptPolicyTest_*') { continue } # PowerShell's own, at its start, in %TEMP%
        if (([int]$e.Attributes -band 0x400) -ne 0) { "$($e.FullName)|link"; continue }
        if ($e -is [IO.DirectoryInfo]) { "$($e.FullName)|dir"; Walk $e.FullName; continue }
        "$($e.FullName)|$($e.Length)|$([int]$e.Attributes)|$([BitConverter]::ToString($sha.ComputeHash([IO.File]::ReadAllBytes($e.FullName))))"
    }
}
function Snapshot() { return (@(Walk $root) | Sort-Object) -join "`n" }
function Quote([string]$a) {
    if ($a -eq '') { return '""' }
    if ($a -notmatch '[\s"]') { return $a }
    return '"' + (($a -replace '(\\*)"', '$1$1\"') -replace '(\\+)$', '$1$1') + '"'
}
# Run-Process starts a program with no shell in between; TEMP as given, removed if
# '<none>'; no GIT_ setting of this shell's, only those in $vars.
function Run-Process([string]$file, [string]$arguments, [string]$tempDir, [string]$cwd, [hashtable]$vars = @{}) {
    $psi = New-Object Diagnostics.ProcessStartInfo
    $psi.FileName = $file; $psi.Arguments = $arguments; $psi.WorkingDirectory = $cwd
    $psi.UseShellExecute = $false; $psi.CreateNoWindow = $true
    $psi.RedirectStandardOutput = $true; $psi.RedirectStandardError = $true
    foreach ($k in @($psi.EnvironmentVariables.Keys)) { if ($k -like 'GIT_*') { $psi.EnvironmentVariables.Remove($k) } }
    foreach ($v in 'TEMP', 'TMP') {
        if ($tempDir -eq '<none>') { [void]$psi.EnvironmentVariables.Remove($v) } else { $psi.EnvironmentVariables[$v] = $tempDir }
    }
    foreach ($k in $vars.Keys) { $psi.EnvironmentVariables[$k] = $vars[$k] }
    $p = [Diagnostics.Process]::Start($psi)
    $o = $p.StandardOutput.ReadToEndAsync(); $e = $p.StandardError.ReadToEndAsync()
    if (-not $p.WaitForExit(60000)) { $p.Kill(); return @{ Exit = -1; Out = ''; Err = 'killed after 60 s' } }
    return @{ Exit = $p.ExitCode; Out = $o.Result; Err = $e.Result }
}
function Run-Harness([string[]]$a, [string]$tempDir, [string]$cwd = $root, [hashtable]$vars = @{}) {
    $line = '-NoProfile -ExecutionPolicy Bypass -File ' + (Quote $harness) + ' -noLaunch ' + (($a | ForEach-Object { Quote $_ }) -join ' ')
    return Run-Process 'powershell.exe' $line $tempDir $cwd $vars
}
function Squeeze([string]$s) { return ($s -replace '\s+', '') }
# Refused: the call fails, for the reason given, and nothing anywhere under $root changes.
function Refused([string]$name, [string]$why, [string[]]$a, [string]$tempDir = $temp, [hashtable]$vars = @{}) {
    $r = Run-Harness $a $tempDir $root $vars
    $now = Snapshot
    $problems = @()
    if ($r.Exit -eq 0) { $problems += 'it was accepted' }
    if ($why -and -not (Squeeze $r.Err).Contains((Squeeze $why))) { $problems += "not the expected reason [$why]: $(($r.Err -split "`n")[0..3] -join ' ')" }
    if ($now -cne $script:state) { $problems += 'files changed'; $script:state = $now }
    Report ($problems.Count -eq 0) $name ($problems -join '; ')
}
# Accepted: the call passes and adds exactly the run's folder; the rest stays as it was.
function Accepted([string]$name, [string]$run, [string[]]$a, [string]$tempDir = $temp, [string]$cwd = $root, [string[]]$also = @(), [hashtable]$vars = @{}) {
    $before = @($script:state -split "`n")
    $r = Run-Harness $a $tempDir $cwd $vars
    $script:state = Snapshot
    $after = @($script:state -split "`n")
    $problems = @()
    if ($r.Exit -ne 0) { $problems += "exit $($r.Exit): $(($r.Err -split "`n")[0..3] -join ' ')" }
    if (-not $r.Out.Contains("run folder: %TEMP%\acc-wt\$(Split-Path -Leaf $run)")) { $problems += 'it did not name the run folder' }
    $gone = @($before | Where-Object { $_ -notin $after })
    if ($gone.Count -ne 0) { $problems += "changed or gone: $($gone -join ', ')" }
    $added = @($after | Where-Object { $_ -notin $before })
    $stray = @($added | Where-Object { -not $_.StartsWith("$run\") -and -not $_.StartsWith("$run|") -and -not $_.StartsWith("$tempDir\acc-wt|") -and ($also | Where-Object { $_ }) -notcontains ($_ -split '\|')[0] })
    if ($stray.Count -ne 0) { $problems += "written outside the run's folder: $($stray -join ', ')" }
    Report ($problems.Count -eq 0) $name ($problems -join '; ')
}

# The fixtures.
$temp = MakeDir "$root\tmp"
$acc = Put "$root\bin\acc.exe" "a build`npath`tgithub.com/Evengelius/agent-command-center/cmd/agent-command-center`nmod`n"
$other = Put "$root\bin\other.exe" "another program`n"
$accTxt = Put "$root\bin\acc.txt" "path`tgithub.com/Evengelius/agent-command-center/cmd/agent-command-center`n"
$accAmp = Put "$root\b&in\acc.exe" "path`tgithub.com/Evengelius/agent-command-center/cmd/agent-command-center`n"
$accPct = Put "$root\b%TEMP%in\acc.exe" "path`tgithub.com/Evengelius/agent-command-center/cmd/agent-command-center`n"
$accPar = Put "$root\b(x86)\acc.exe" "path`tgithub.com/Evengelius/agent-command-center/cmd/agent-command-center`n"
$rec = Put "$root\data\ok.jsonl" "{`"t_ms`":0,`"kind`":`"session.spawned`",`"provider`":`"codex`"}`n"
$recTxt = Put "$root\data\ok.txt" "{`"kind`":`"session.spawned`"}`n"
$recBad = Put "$root\data\notjson.jsonl" "not json`n"
$recNoKind = Put "$root\data\nokind.jsonl" "{`"t_ms`":0}`n"
$recEmpty = Put "$root\data\empty.jsonl" ''
$planFile = Put "$root\data\plan.txt" 'shot:fromfile:0'
$planBad = Put "$root\data\planbad.txt" 'rm:a:0:..\outside\keep.txt'
# What a slip would destroy: files the old harness reached in its 47-case investigation.
[void](Put "$root\outside\keep.txt" "keep`n")
[void](Put "$root\outside\sub\keep2.txt" "keep`n")
[void](Put "$root\victim\precious.txt" "keep`n")
[void](Put "$root\victim\codex-max\precious.txt" "keep`n")
[void](Put "$root\victim-max\precious.txt" "keep`n")
[void](Put "$root\tmp\victim\precious.txt" "keep`n")
[void](Put "$root\tmp\victim-codex-max\precious.txt" "keep`n")
$readOnly = Put "$root\outside\readonly.txt" "keep`n"
[IO.File]::SetAttributes($readOnly, [IO.FileAttributes]::ReadOnly)
$canSymlink = $true
try { New-Item -ItemType SymbolicLink -Path "$root\symlink-probe" -Target "$root\outside" -ErrorAction Stop | Out-Null } catch { $canSymlink = $false }
$base = @('-exe', $acc, '-mock', $rec, '-plan', 'shot:a:0')
$script:state = Snapshot

'=== names of the run: -outName, -tag ==='
$badNames = @('', '.', '..', '..\..\victim', '../victim', '..\victim', 'a\b', 'a/b', 'C:\x', 'C:', 'C:x', '\x', '\\server\share', 'a&b', 'x&md.', 'a&&b', 'a|b', 'a>b', 'a<b', 'a^b', 'a%TEMP%b', 'a!b', 'a"b', "a'b", 'a b', ' a', 'a ', 'a;b', 'a,b', 'a=b', 'a*', 'a?', 'a[1]', 'a.b', 'a.', '_a', 'a`b', 'a$b', 'a(b)', "a`nb", 'CON', 'nul', 'COM1', 'LPT9', 'log', ('x' * 41), ('caf' + [char]0xE9))
foreach ($v in $badNames) { Refused "-outName [$v]" '-outName' ($base + @('-outName', $v)) }
foreach ($v in @('', '..', '..\..\..\victim', '../victim', 'a\b', 'a/b', 'C:\x', 'a&b', 'a*', 'a%TEMP%b')) { Refused "-tag [$v]" '-tag' ($base + @('-outName', 'n', '-tag', $v)) }
Refused '-outName missing' '-outName' $base
Refused '-outName, a value like a parameter' '' ($base + @('-outName', '-lead'))

'=== -mode ==='
foreach ($v in @('', 'MAX', '.', '..', '..\..\..\..\..\victim', '90', '90x', 'x30', '90x30x1', '9x9', '0x0', '1000x30', '90x30&calc', 'max;calc', 'max&calc', 'max ', ' max', '90,30', '90X30')) { Refused "-mode [$v]" '-mode' ($base + @('-outName', 'n', '-mode', $v)) }

'=== the steps the harness no longer has: rm, touch, run, poll, fieldcheck ==='
foreach ($s in @('poll', 'poll:a:0:text', 'fieldcheck', 'fieldcheck:a:0', 'rm', 'rm:a:0', 'rm:a:0:', 'rm:a:0:.', 'rm:a:0:..', 'rm:a:0:file.txt', 'rm:a:0:..\outside\keep.txt', 'rm:a:0:../outside/keep.txt', 'rm:a:0:C:\x', 'rm:a:0:\x', 'rm:a:0:\\server\share\x', 'rm:a:0:junc\keep.txt', 'rm:a:0:file.txt:stream', 'rm:a:0:*.txt', 'touch', 'touch:a:0', 'touch:a:0:', 'touch:a:0:new.txt', 'touch:a:0:..\outside\keep.txt', 'touch:a:0:../outside/keep.txt', 'touch:a:0:C:\x', 'run', 'run:a:0:x.ps1', 'run:a:0:..\x.ps1', 'nosuch:a:0', 'SHOT2:a:0')) {
    Refused "step [$s]" 'no such step' @('-exe', $acc, '-mock', $rec, '-outName', 'n', '-plan', $s)
}
Refused 'a removed step after a good one' 'no such step' @('-exe', $acc, '-mock', $rec, '-outName', 'n', '-plan', 'shot:a:0;rm:b:0:x')
Refused 'a removed step from a plan file' 'no such step' @('-exe', $acc, '-mock', $rec, '-outName', 'n', '-plan', "@$planBad")

'=== labels ==='
foreach ($s in @('shot:..\x:0', 'shot:../x:0', 'shot:..:0', 'shot:.:0', 'shot:a\b:0', 'shot:a/b:0', 'shot::0', 'shot:a&b:0', 'shot:a*:0', 'shot:a b:0', 'shot:a.b:0', 'shot:CON:0', 'shot:NUL:0', 'shot:log:0', 'shot:..\..\outside\keep:0', 'dialog:..\x:0', 'type:..\x:0:t', 'enter:..\x:0')) {
    Refused "step [$s]" 'the label' @('-exe', $acc, '-mock', $rec, '-outName', 'n', '-plan', $s)
}
Refused 'a label used twice' 'already used' @('-exe', $acc, '-mock', $rec, '-outName', 'n', '-plan', 'shot:a:0;press:a:10:26')
Refused 'a drive in a label: the colon leaves no time' 'MS' @('-exe', $acc, '-mock', $rec, '-outName', 'n', '-plan', 'shot:C:\x:0')

'=== malformed steps ==='
foreach ($s in @('shot', 'shot:a')) { Refused "step [$s]" 'NAME:LABEL:MS' @('-exe', $acc, '-mock', $rec, '-outName', 'n', '-plan', $s) }
foreach ($s in @('shot:a:', 'shot:a:later', 'shot:a:-1', 'shot:a:1.5', 'shot:a:99999999')) { Refused "step [$s]" 'MS' @('-exe', $acc, '-mock', $rec, '-outName', 'n', '-plan', $s) }
foreach ($s in @('shot:a:0:extra', 'hoveroff:a:0:x', 'enter:a:0:x', 'esc:a:0:x', 'click:a:0:t:1:2', 'dialog:a:0:t:extra', 'press:a:0:26:27', 'expect:a:0:t:extra')) { Refused "step [$s]" 'too many' @('-exe', $acc, '-mock', $rec, '-outName', 'n', '-plan', $s) }
foreach ($s in @('expect:a:0', 'press:a:0', 'chord:a:0', 'altgr:a:0', 'ctrlhold:a:0', 'click:a:0', 'hoverat:a:0:1', 'clickcell:a:0', 'drag:a:0:1', 'clickrow:a:0:t', 'clickbelow:a:0:t:1', 'homeclick:a:0:1:2', 'wheel:a:0:1:2:3:4:5:t', 'fieldsel:a:0:t:1:2', 'type:a:0')) { Refused "step [$s]" 'missing' @('-exe', $acc, '-mock', $rec, '-outName', 'n', '-plan', $s) }
foreach ($s in @('expect:a:0:', 'press:a:0:', 'press:a:0:xyz', 'press:a:0:26.', 'press:a:0:12345', 'chord:a:0:G', 'altgr:a:0:100', 'ctrlhold:a:0:x', 'click:a:0:t:x', 'hoverat:a:0:1:x', 'drag:a:0:x:1', 'clickrow:a:0:t:x', 'clickbelow:a:0:t:1:x', 'homeclick:a:0:1:2:3:4:x:t', 'wheel:a:0:1:2:3:4:5:t:x', 'fieldsel:a:0:t:1:x:3')) { Refused "step [$s]" 'wrong form' @('-exe', $acc, '-mock', $rec, '-outName', 'n', '-plan', $s) }
Refused 'expectfield without -expect' 'needs -expect' @('-exe', $acc, '-mock', $rec, '-outName', 'n', '-plan', 'expectfield:a:0')
Refused 'expectfield with an empty -expect' 'needs -expect' @('-exe', $acc, '-mock', $rec, '-outName', 'n', '-expect', '', '-plan', 'expectfield:a:0')
Refused 'expectfield with an argument' 'too many' @('-exe', $acc, '-mock', $rec, '-outName', 'n', '-expect', 'x', '-plan', 'expectfield:a:0:x')
Refused 'a space before a step' 'no such step' @('-exe', $acc, '-mock', $rec, '-outName', 'n', '-plan', 'shot:a:0; shot:b:1')
Refused 'a newline in a text' 'wrong form' @('-exe', $acc, '-mock', $rec, '-outName', 'n', '-plan', "expect:a:0:one`ntwo")
Refused '-plan empty' 'no step' @('-exe', $acc, '-mock', $rec, '-outName', 'n', '-plan', '')
Refused '-plan missing' 'no step' @('-exe', $acc, '-mock', $rec, '-outName', 'n')
Refused '-plan only separators' 'no step' @('-exe', $acc, '-mock', $rec, '-outName', 'n', '-plan', ';;')
Refused '-plan, a missing file' '' @('-exe', $acc, '-mock', $rec, '-outName', 'n', '-plan', "@$root\data\nosuch.txt")

'=== -exe ==='
Refused '-exe missing' '-exe' @('-mock', $rec, '-outName', 'n', '-plan', 'shot:a:0')
Refused '-exe empty' '-exe' @('-exe', '', '-mock', $rec, '-outName', 'n', '-plan', 'shot:a:0')
Refused '-exe, no such file' '-exe' @('-exe', "$root\bin\nosuch.exe", '-mock', $rec, '-outName', 'n', '-plan', 'shot:a:0')
Refused '-exe, a folder' '-exe' @('-exe', "$root\bin", '-mock', $rec, '-outName', 'n', '-plan', 'shot:a:0')
Refused '-exe, not an .exe' '-exe' @('-exe', $accTxt, '-mock', $rec, '-outName', 'n', '-plan', 'shot:a:0')
Refused '-exe, another program' 'not a build of the dashboard' @('-exe', $other, '-mock', $rec, '-outName', 'n', '-plan', 'shot:a:0')
Refused '-exe, cmd.exe' 'not a build of the dashboard' @('-exe', "$env:SystemRoot\System32\cmd.exe", '-mock', $rec, '-outName', 'n', '-plan', 'shot:a:0')
Refused '-exe, & in its path' '-exe' @('-exe', $accAmp, '-mock', $rec, '-outName', 'n', '-plan', 'shot:a:0')
Refused '-exe, % in its path' '-exe' @('-exe', $accPct, '-mock', $rec, '-outName', 'n', '-plan', 'shot:a:0')
Refused '-exe, a bracket in its path' '-exe' @('-exe', $accPar, '-mock', $rec, '-outName', 'n', '-plan', 'shot:a:0')
Refused '-exe, a network path' '-exe' @('-exe', '\\server\share\acc.exe', '-mock', $rec, '-outName', 'n', '-plan', 'shot:a:0')
Refused '-exe, a wildcard' '-exe' @('-exe', "$root\bin\*.exe", '-mock', $rec, '-outName', 'n', '-plan', 'shot:a:0')

'=== the recording: missing or wrong ==='
Refused 'neither -mock nor -homeScreen' 'exactly one' @('-exe', $acc, '-outName', 'n', '-plan', 'shot:a:0')
Refused 'both -mock and -homeScreen' 'exactly one' ($base + @('-outName', 'n', '-homeScreen'))
Refused '-mock empty' 'exactly one' @('-exe', $acc, '-mock', '', '-outName', 'n', '-plan', 'shot:a:0')
Refused '-mock, no such file' '-mock' @('-exe', $acc, '-mock', "$root\data\nosuch.jsonl", '-outName', 'n', '-plan', 'shot:a:0')
Refused '-mock, a folder' '-mock' @('-exe', $acc, '-mock', "$root\data", '-outName', 'n', '-plan', 'shot:a:0')
Refused '-mock, not a .jsonl' '-mock' @('-exe', $acc, '-mock', $recTxt, '-outName', 'n', '-plan', 'shot:a:0')
Refused '-mock, not JSON' 'no event' @('-exe', $acc, '-mock', $recBad, '-outName', 'n', '-plan', 'shot:a:0')
Refused '-mock, no kind' 'no event' @('-exe', $acc, '-mock', $recNoKind, '-outName', 'n', '-plan', 'shot:a:0')
Refused '-mock, an empty file' 'no event' @('-exe', $acc, '-mock', $recEmpty, '-outName', 'n', '-plan', 'shot:a:0')
Refused '-mock, a wildcard' '-mock' @('-exe', $acc, '-mock', "$root\data\*.jsonl", '-outName', 'n', '-plan', 'shot:a:0')

'=== the options the harness no longer has ==='
foreach ($o in @('-projDir', '-gitDir', '-exeArgs', '-extraPath', '-setup', '-envLine', '-runDir')) { Refused "$o" '' ($base + @('-outName', 'n', $o, "$root\outside")) }
Refused '-trace with a path: it is a switch now' '' ($base + @('-outName', 'n', '-trace', "$root\outside\keep.txt"))
Refused 'a stray argument' '' ($base + @('-outName', 'n', 'stray'))
Refused '-git without -fixture' '-git: only with -fixture' ($base + @('-outName', 'n', '-git'))
Refused '-fixture without -mock' '-fixture: only with -mock' @('-exe', $acc, '-homeScreen', '-fixture', 'f', '-outName', 'n', '-plan', 'shot:a:0')
Refused '-git on the home screen' '-git: only with -fixture' @('-exe', $acc, '-homeScreen', '-git', '-outName', 'n', '-plan', 'shot:a:0')
foreach ($v in @('x;new-tab cmd', 'a"b', 'a&b', 'a|b', 'a%b', ('x' * 65))) { Refused "-wtProfile [$v]" '-wtProfile' ($base + @('-outName', 'n', '-wtProfile', $v)) }

'=== -fixture: the project of a run on a repository ==='
$tf = MakeDir "$root\tf"
$fixtures = MakeDir "$tf\acc-wt-fixtures"
[void](Put "$fixtures\clean\file.txt" "keep`n")
[void](Put "$fixtures\hasmock\mock.jsonl" "the fixture's own`n")
[void](Put "$fixtures\linked\.git" "gitdir: C:/elsewhere/.git/worktrees/x`n")
[void](Put "$tf\real\project.txt" "keep`n")
[void](Put "$fixtures\inner\sub\file.txt" "keep`n")
Junction "$fixtures\junction" "$tf\real"
Junction "$fixtures\inner\sub\out" "$tf\real"
$script:state = Snapshot
foreach ($v in @('.', '..', '..\real', '../real', '..\..\outside', 'a\b', 'a/b', 'C:\x', "$tf\real", '\\server\share', 'a&b', 'a*')) { Refused "-fixture [$v]" '-fixture' ($base + @('-outName', 'n', '-fixture', $v)) $tf }
Refused '-fixture, no such one' 'no such folder' ($base + @('-outName', 'n', '-fixture', 'nosuch')) $tf
Refused '-fixture, a junction to a real project' 'a link' ($base + @('-outName', 'n', '-fixture', 'junction')) $tf
Refused '-fixture, a junction inside it' 'a link inside it' ($base + @('-outName', 'n', '-fixture', 'inner')) $tf
Refused '-fixture, its own mock.jsonl' 'not replaced' ($base + @('-outName', 'n', '-fixture', 'hasmock')) $tf
Refused '-fixture, a .git file' 'a worktree of a repository elsewhere' ($base + @('-outName', 'n', '-fixture', 'linked')) $tf
Refused '-fixture, no fixtures folder' 'no such folder' ($base + @('-outName', 'n', '-fixture', 'clean'))
$tj = MakeDir "$root\tj"
Junction "$tj\acc-wt-fixtures" "$tf\acc-wt-fixtures"
$script:state = Snapshot
Refused 'the fixtures folder is a junction' 'a link' ($base + @('-outName', 'n', '-fixture', 'clean')) $tj
if ($canSymlink) {
    $ts = MakeDir "$root\ts"; $sf = MakeDir "$ts\acc-wt-fixtures"
    New-Item -ItemType SymbolicLink -Path "$sf\link" -Target "$tf\real" | Out-Null
    [void](Put "$sf\inner\file.txt" "keep`n"); New-Item -ItemType SymbolicLink -Path "$sf\inner\out" -Target "$tf\real" | Out-Null
    New-Item -ItemType SymbolicLink -Path "$ts\acc-wt" -Target "$root\outside" | Out-Null
    $script:state = Snapshot
    Refused '-fixture, a symbolic link' 'a link' ($base + @('-outName', 'n', '-fixture', 'link')) $ts
    Refused '-fixture, a symbolic link inside it' 'a link inside it' ($base + @('-outName', 'n', '-fixture', 'inner')) $ts
    Refused '%TEMP%\acc-wt, a symbolic link' 'a link' ($base + @('-outName', 'n')) $ts
} else {
    NotRun 'symbolic links as a fixture, inside one, and as %TEMP%\acc-wt' 'this account cannot create symbolic links; the check reads the reparse-point attribute junctions also have'
}

'=== the run''s folder and what leads to it ==='
$t1 = MakeDir "$root\t1"
[void](Put "$t1\acc-wt\exists-codex-max\old-result.txt" "keep`n")
[void](Put "$t1\acc-wt\afile-codex-max" "keep`n")
Junction "$t1\acc-wt\junc-codex-max" "$root\outside"
[void](MakeDir "$root\gone1"); Junction "$t1\acc-wt\dangling-codex-max" "$root\gone1"; [IO.Directory]::Move("$root\gone1", "$root\gone1moved")
$t2 = MakeDir "$root\t2"; Junction "$t2\acc-wt" "$root\outside"
$t3 = MakeDir "$root\t3"; [void](Put "$t3\acc-wt" "keep`n")
$t4 = MakeDir "$root\t4"; [void](MakeDir "$root\gone4"); Junction "$t4\acc-wt" "$root\gone4"; [IO.Directory]::Move("$root\gone4", "$root\gone4moved")
Junction "$root\t5" "$root\outside"
$t6 = MakeDir "$root\t&6"
$t7 = MakeDir "$root\t%TEMP%7"
$t8 = MakeDir "$root\t(8)"
$script:state = Snapshot
Refused 'the run''s folder exists' 'already exists' ($base + @('-outName', 'exists')) $t1
Refused 'the run''s folder is a file' 'already exists' ($base + @('-outName', 'afile')) $t1
Refused 'the run''s folder is a junction' 'already exists' ($base + @('-outName', 'junc')) $t1
$r = Run-Harness ($base + @('-outName', 'dangling')) $t1; $now = Snapshot
Report ($r.Exit -ne 0 -and $now -ceq $script:state) 'the run''s folder is a dangling junction' "exit $($r.Exit); same files: $($now -ceq $script:state)"
Refused '%TEMP%\acc-wt is a junction' 'a link' ($base + @('-outName', 'n')) $t2
Refused '%TEMP%\acc-wt is a file' 'no such folder' ($base + @('-outName', 'n')) $t3
$r = Run-Harness ($base + @('-outName', 'n')) $t4; $now = Snapshot
Report ($r.Exit -ne 0 -and $now -ceq $script:state) '%TEMP%\acc-wt is a dangling junction' "exit $($r.Exit); same files: $($now -ceq $script:state)"
Refused '%TEMP% is a junction' 'a link' ($base + @('-outName', 'n')) "$root\t5"
Refused '%TEMP% has an &' '%TEMP%' ($base + @('-outName', 'n')) $t6
Refused '%TEMP% has a %' '%TEMP%' ($base + @('-outName', 'n')) $t7
Refused '%TEMP% has a bracket' '%TEMP%' ($base + @('-outName', 'n')) $t8
# PowerShell itself makes a missing %TEMP% as it starts, so the run stays in that new folder.
Accepted '%TEMP% missing: PowerShell makes it, and the run stays inside' "$root\nosuch\acc-wt\n-codex-max" ($base + @('-outName', 'n')) "$root\nosuch" $root @("$root\nosuch")
Refused '%TEMP% not set' '%TEMP%' ($base + @('-outName', 'n')) '<none>'

'=== what it still does, with no terminal ==='
$runA = "$temp\acc-wt\ok-codex-max"
Accepted 'a recording: the run''s folder' $runA ($base + @('-outName', 'ok'))
$clearGit = 'for /f "delims==" %%V in (''set GIT_ 2^>nul'') do set "%%V="'
$want = @('@echo off', $clearGit, "set `"LOCALAPPDATA=$runA\lad`"", "set `"APPDATA=$runA\ad`"", "set `"USERPROFILE=$runA\home`"", "set `"HOME=$runA\home`"", "set `"XDG_CONFIG_HOME=$runA\home\.config`"", 'set "PATH=C:\Windows\System32"', 'set NO_COLOR=', "`"$acc`" --mock `"$runA\demo\mock.jsonl`"") -join "`n"
$launch = ([IO.File]::ReadAllText("$runA\launch.cmd") -replace "`r`n", "`n").TrimEnd()
Report ($launch -ceq $want) 'its launch.cmd, exactly: --mock, a temporary home, System32 alone' $launch
Report ([IO.File]::ReadAllText("$runA\demo\mock.jsonl") -ceq [IO.File]::ReadAllText($rec)) 'the recording''s copy is the recording' ''
foreach ($d in 'results', 'demo', 'lad', 'ad', 'home') { Report ([IO.Directory]::Exists("$runA\$d")) "its folder $d" '' }
Refused 'the same run again: refused, the first one untouched' 'already exists' ($base + @('-outName', 'ok'))
$runB = "$temp\acc-wt\home-codex-max"
Accepted 'the home screen' $runB @('-exe', $acc, '-homeScreen', '-outName', 'home', '-plan', 'shot:a:0')
$lastLine = (([IO.File]::ReadAllText("$runB\launch.cmd") -replace "`r`n", "`n").TrimEnd() -split "`n")[-1]
Report ($lastLine -ceq "`"$acc`"") 'its launch has no argument' $lastLine
Report (-not [IO.Directory]::GetFiles($runB, 'mock.jsonl', 'AllDirectories')) 'and no recording' ''
$runC = "$temp\acc-wt\sized-other-90x30"
Accepted 'a size, a tag and a profile' $runC ($base + @('-outName', 'sized', '-tag', 'other', '-mode', '90x30', '-wtProfile', 'Agent Command Center'))
$runD = "$temp\acc-wt\relative-codex-max"
Accepted 'relative paths' $runD @('-exe', 'bin\acc.exe', '-mock', 'data\ok.jsonl', '-outName', 'relative', '-plan', 'shot:a:0')
$runE = "$temp\acc-wt\planfile-codex-max"
Accepted 'a plan from a file' $runE @('-exe', $acc, '-mock', $rec, '-outName', 'planfile', '-plan', "@$planFile")
$runF = "$temp\acc-wt\traced-codex-max"
Accepted '-trace' $runF ($base + @('-outName', 'traced', '-trace'))
Report ([IO.File]::ReadAllText("$runF\launch.cmd").Contains("set `"TEA_TRACE=$runF\trace.txt`"")) 'its trace stays in the run''s folder' ''
$every = 'shot:s1:0;expect:s2:1:text;press:s3:2:26.27;chord:s4:3:11.51;altgr:s5:4:45;ctrlhold:s6:5:3;click:s7:6:t;click:s8:7:t:2;dclick:s9:8:t;cclick:s10:9:t:1.5;clickrow:s11:10:t:at;clickrow:s12:11:t:5;clickbelow:s13:12:t:1:2;clickcell:s14:13:-4:8;hoverat:s15:14:1:2;hoveroff:s16:15;homeclick:s17:16:1:2:3:4:9.5:t;wheel:s18:17:1:2:3:4:9.5:t:-2;fieldsel:s19:18:t:1:2:9;expectfield:s20:19;drag:s21:20:-5:5;dialog:s22:21;dialog:s23:22:Title;type:s24:23:C:\x\y.png;enter:s25:24;esc:s26:25'
$runG = "$temp\acc-wt\everystep-codex-max"
Accepted 'every step, in every form' $runG @('-exe', $acc, '-mock', $rec, '-outName', 'everystep', '-expect', 'x', '-plan', $every)
$gitCmd = Get-Command git.exe -CommandType Application -ErrorAction SilentlyContinue | Select-Object -First 1
$runH = "$tf\acc-wt\fix-codex-max"
Accepted 'a fixture: its copy of the recording' $runH ($base + @('-outName', 'fix', '-fixture', 'clean')) $tf $root @("$fixtures\clean\mock.jsonl")
Report ([IO.File]::ReadAllText("$fixtures\clean\mock.jsonl") -ceq [IO.File]::ReadAllText($rec)) 'the fixture holds the recording''s copy' ''
$lines = ([IO.File]::ReadAllText("$runH\launch.cmd") -replace "`r`n", "`n").TrimEnd() -split "`n"
Report ($lines[-1] -ceq "`"$acc`" --mock `"$fixtures\clean\mock.jsonl`"" -and $lines -contains 'set "PATH=C:\Windows\System32"') 'its launch: --mock on the fixture, no git without -git' ($lines -join ' ')

if ($gitCmd) {
    '=== -git: what a repository may hold, before the launch ==='
    $tg = MakeDir "$root\tg"; $gf = MakeDir "$tg\acc-wt-fixtures"; $emptyHome = MakeDir "$root\emptyhome"
    $evil = MakeDir "$root\evil"; $victim = MakeDir "$root\victim-repo"; $outsideBare = MakeDir "$root\outside-real.git"
    $cleanGit = @{ GIT_CONFIG_NOSYSTEM = '1'; HOME = $emptyHome; USERPROFILE = $emptyHome; XDG_CONFIG_HOME = $emptyHome }
    function Git-Clean([string]$arguments) { return Run-Process $gitCmd.Source $arguments $temp $root $cleanGit }
    function Must-Git([string]$arguments) {
        $r = Git-Clean $arguments
        if ($r.Exit -ne 0) { throw "git $arguments : $($r.Err)" }
        return $r.Out.Trim()
    }
    function New-Repo([string]$dir, [string]$kind = '') { [void](MakeDir $dir); [void](Must-Git "init -q --template= $kind `"$dir`"") }
    function Seed-Repo([string]$dir) {
        [void](Must-Git "-C `"$dir`" config --local user.name fixture")
        [void](Must-Git "-C `"$dir`" config --local user.email fixture@example.invalid")
        [void](Put "$dir\tracked.txt" "first`n")
        [void](Must-Git "-C `"$dir`" add tracked.txt")
        [void](Must-Git "-C `"$dir`" commit -q -m first")
    }
    New-Repo "$gf\repo"
    New-Repo $victim
    New-Repo $outsideBare '--bare'
    Seed-Repo $victim
    $gitBase = @('-exe', $acc, '-mock', $rec, '-plan', 'shot:a:0', '-git', '-outName', 'n')
    $script:state = Snapshot
    Refused '-git, no git on the PATH' '-git: no git.exe' ($gitBase + @('-fixture', 'repo')) $tg @{ PATH = "$env:SystemRoot\System32" }
    # A config the harness must refuse, added to a repository and taken away again.
    function With-Config([string]$name, [string]$why, [string]$extra, [hashtable]$vars = @{}) {
        $cfg = "$gf\repo\.git\config"
        $was = [IO.File]::ReadAllText($cfg)
        [IO.File]::WriteAllText($cfg, $was + $extra)
        $script:state = Snapshot
        Refused $name $why ($gitBase + @('-fixture', 'repo')) $tg $vars
        [IO.File]::WriteAllText($cfg, $was)
        $script:state = Snapshot
    }
    foreach ($k in @(
            @('core', 'hooksPath', "$evil\hooks"), @('core', 'worktree', $victim), @('core', 'sshCommand', 'x'), @('core', 'fsmonitor', 'x'), @('core', 'pager', 'x'),
            @('core', 'editor', 'x'), @('core', 'askPass', 'x'), @('core', 'attributesFile', 'x'), @('core', 'excludesFile', 'x'), @('core', 'gitProxy', 'x'),
            @('include', 'path', "$evil\gitconfig"), @('includeIf "gitdir:x"', 'path', "$evil\gitconfig"), @('url "https://example.invalid/"', 'insteadOf', '../'),
            @('url "https://example.invalid/"', 'pushInsteadOf', '../'), @('remote', 'pushDefault', 'x'), @('remote "origin"', 'pushurl', 'https://example.invalid/y.git'),
            @('remote "origin"', 'receivepack', 'x'), @('remote "origin"', 'uploadpack', 'x'), @('remote "origin"', 'proxy', 'x'), @('branch "master"', 'pushRemote', 'x'),
            @('credential', 'helper', 'x'), @('gpg', 'program', 'x'), @('commit', 'gpgSign', 'true'), @('diff "x"', 'command', 'x'), @('diff', 'external', 'x'),
            @('filter "x"', 'clean', 'x'), @('merge "x"', 'driver', 'x'), @('alias', 'x', '!x'), @('extensions', 'worktreeConfig', 'true'), @('protocol', 'allow', 'always'),
            @('protocol "ext"', 'allow', 'always'), @('http', 'proxy', 'x'), @('pager', 'log', 'x'), @('safe', 'directory', '*'), @('submodule "x"', 'url', 'x'))) {
        With-Config "config [$($k[0]) $($k[1])]" 'does not let through' "[$($k[0])]`n`t$($k[1]) = $($k[2].Replace('\', '/'))`n"
    }
    With-Config 'config, malformed' 'cannot read its config' "[core`n"
    With-Config 'config, still refused under a hostile environment' 'does not let through' "[core]`n`thooksPath = x`n" @{ GIT_CONFIG_COUNT = '1'; GIT_CONFIG_KEY_0 = 'a.b'; GIT_CONFIG_VALUE_0 = 'c'; GIT_DIR = "$victim\.git" }
    foreach ($u in @('https://example.invalid/y.git', 'http://example.invalid/y.git', 'ssh://git@example.invalid/y.git', 'git@example.invalid:x/y.git', 'example.invalid:y.git', 'git://example.invalid/y.git', 'ext::sh -c x', 'file:///C:/x.git', '--upload-pack=x', '-x', 'a&b', 'C:x.git', '')) {
        With-Config "remote url [$u]" 'only a local path' "[remote `"origin`"]`n`turl = $u`n"
    }
    foreach ($u in @($outsideBare.Replace('\', '/'), $outsideBare, '..\..\..\outside-real.git', '../../../outside-real.git', '\\server\share\x.git', '//server/share/x.git', '\x.git', 'C:/x.git')) {
        With-Config "remote url [$u]" 'outside' "[remote `"origin`"]`n`turl = $($u.Replace('\', '\\'))`n"
    }
    With-Config 'a branch''s remote, a URL' 'names no remote' "[branch `"master`"]`n`tremote = https://example.invalid/y.git`n`tmerge = refs/heads/master`n"
    With-Config 'a branch''s remote, a path' 'names no remote' "[branch `"master`"]`n`tremote = $($outsideBare.Replace('\', '/'))`n`tmerge = refs/heads/master`n"
    # Remotes among the fixtures that still may not be used.
    New-Repo "$gf\notbare"
    [void](MakeDir "$gf\notarepo")
    New-Repo "$gf\hooked.git" '--bare'; [void](Put "$gf\hooked.git\hooks\post-receive" "#!/bin/sh`n")
    New-Repo "$gf\alternate.git" '--bare'; [void](Put "$gf\alternate.git\objects\info\alternates" "$victim\.git\objects`n")
    New-Repo "$gf\badkey.git" '--bare'; [IO.File]::AppendAllText("$gf\badkey.git\config", "[core]`n`thooksPath = x`n")
    New-Repo "$gf\chained.git" '--bare'; [IO.File]::AppendAllText("$gf\chained.git\config", "[remote `"up`"]`n`turl = https://example.invalid/y.git`n")
    New-Repo "$gf\flagged.git" '--bare'; [IO.File]::WriteAllText("$gf\flagged.git\config", "[core]`n`trepositoryformatversion = 0`n`tbare = false`n")
    New-Repo "$gf\linkinside.git" '--bare'; Junction "$gf\linkinside.git\leak" "$root\outside"
    Junction "$gf\junction.git" $outsideBare
    foreach ($c in @(@('..\nosuch.git', 'no such folder'), @('..\notbare', 'a repository inside it'), @('..\notarepo', 'cannot read its config'), @('..\hooked.git', 'an active hook'), @('..\alternate.git', 'can lead outside it'), @('..\badkey.git', 'does not let through'), @('..\chained.git', 'does not let through'), @('..\flagged.git', 'not a bare repository'), @('..\linkinside.git', 'a link inside it'), @('..\junction.git', 'a link'))) {
        With-Config "remote [$($c[0])]" $c[1] "[remote `"origin`"]`n`turl = $($c[0].Replace('\', '\\'))`n"
    }
    # What a repository's files may redirect.
    $n = 0
    function New-Case([scriptblock]$make, [string]$name, [string]$why) {
        $script:n++
        $f = "c$script:n"
        New-Repo "$gf\$f"
        & $make "$gf\$f"
        $script:state = Snapshot
        Refused $name $why ($gitBase + @('-fixture', $f)) $tg
    }
    New-Case { param($d) [void](Put "$d\.git\hooks\pre-commit" "#!/bin/sh`n") } 'an active hook' 'an active hook'
    New-Case { param($d) [void](Put "$d\.git\objects\info\alternates" "$victim\.git\objects`n") } 'alternates' 'can lead outside it'
    New-Case { param($d) [void](Put "$d\.git\objects\info\http-alternates" "https://example.invalid/`n") } 'http-alternates' 'can lead outside it'
    New-Case { param($d) [void](Put "$d\.git\commondir" "$victim\.git`n") } 'a commondir' 'can lead outside it'
    New-Case { param($d) [void](Put "$d\.git\gitdir" "$victim\.git`n") } 'a gitdir' 'can lead outside it'
    New-Case { param($d) [void](Put "$d\.git\config.worktree" "[core]`n`tworktree = x`n") } 'a config.worktree' 'can lead outside it'
    New-Case { param($d) [void](MakeDir "$d\.git\modules\x") } 'modules' 'can lead outside it'
    New-Case { param($d) Junction "$d\.git\hooks" "$root\evil" } 'the hooks folder is a junction' 'a link inside it'
    New-Case { param($d) Junction "$d\.git\objects\leak" "$victim\.git\objects" } 'a junction among the objects' 'a link inside it'
    New-Case { param($d) Junction "$d\leak" $victim } 'a junction to a real repository in the tree' 'a link inside it'
    New-Case { param($d) [void](MakeDir "$d\sub\.git") } 'another repository inside' 'a repository inside it'
    New-Case { param($d) [void](Put "$d\sub\.git" "gitdir: $victim\.git`n") } 'a gitfile inside' 'a repository inside it'
    New-Case { param($d) [void](Put "$d\.git\worktrees\w\gitdir" "$root\outside\w\.git`n"); [void](Put "$d\.git\worktrees\w\commondir" "../..`n") } 'a linked worktree outside' 'outside'
    New-Case { param($d) [void](MakeDir "$gf\wt1"); [void](Put "$gf\wt1\.git" "gitdir: $victim\.git`n"); [void](Put "$d\.git\worktrees\w\gitdir" "$gf\wt1\.git`n"); [void](Put "$d\.git\worktrees\w\commondir" "../..`n") } 'a linked worktree leading elsewhere' 'leads elsewhere'
    New-Case { param($d) [void](MakeDir "$gf\wt2"); [void](Put "$gf\wt2\.git" "gitdir: $d\.git\worktrees\w`n"); [void](Put "$d\.git\worktrees\w\gitdir" "$gf\wt2\.git`n"); [void](Put "$d\.git\worktrees\w\commondir" "$victim\.git`n") } 'a linked worktree with another commondir' 'leads elsewhere'
    [void](MakeDir "$gf\plain"); $script:state = Snapshot
    Refused '-git, a fixture with no repository' 'holds no repository' ($gitBase + @('-fixture', 'plain')) $tg
    if ($canSymlink) {
        New-Case { param($d) New-Item -ItemType SymbolicLink -Path "$d\.git\hooks" -Target "$root\evil" | Out-Null } 'the hooks folder is a symbolic link' 'a link inside it'
        New-Case { param($d) New-Item -ItemType SymbolicLink -Path "$d\leak.txt" -Target "$root\outside\keep.txt" | Out-Null } 'a file symbolic link in the tree' 'a link inside it'
    } else {
        NotRun 'symbolic links inside a repository' 'this account cannot create symbolic links; the check reads the reparse-point attribute junctions also have'
    }
    NotRun 'a git older than 2.31' 'only this machine''s git is at hand; the check reads git --version'

    '=== -git: what is accepted, and its launch ==='
    # A fixture as made by hand: git's sample hooks, a user, a sibling bare remote by a
    # relative path, a tracking branch, a linked worktree beside it.
    $good = "$gf\good"; $goodBare = "$gf\good-remote.git"
    [void](MakeDir $good); [void](Must-Git "init -q `"$good`"")
    Seed-Repo $good
    [void](MakeDir $goodBare); [void](Must-Git "init -q --bare `"$goodBare`"")
    [void](Must-Git "-C `"$good`" remote add origin ..\good-remote.git")
    [void](Must-Git "-C `"$good`" push -q -u origin HEAD")
    [void](Must-Git "-C `"$good`" worktree add -q -b side `"$gf\good-side`"")
    New-Repo "$gf\plainrepo"
    $abs = "$gf\abs"; New-Repo $abs; New-Repo "$gf\abs-remote.git" '--bare'
    [IO.File]::AppendAllText("$abs\.git\config", "[remote `"origin`"]`n`turl = $("$gf\abs-remote.git".Replace('\', '/'))`n`tfetch = +refs/heads/*:refs/remotes/origin/*`n")
    $script:state = Snapshot
    $runI = "$tg\acc-wt\gitrun-codex-max"
    Accepted 'a repository with a bare remote and a linked worktree' $runI ($gitBase[0..5] + @('-git', '-outName', 'gitrun', '-fixture', 'good')) $tg $root @("$good\mock.jsonl")
    Accepted 'a remote by an absolute path' "$tg\acc-wt\gitabs-codex-max" ($gitBase[0..5] + @('-git', '-outName', 'gitabs', '-fixture', 'abs')) $tg $root @("$abs\mock.jsonl")
    Accepted 'accepted under a hostile environment' "$tg\acc-wt\githostile-codex-max" ($gitBase[0..5] + @('-git', '-outName', 'githostile', '-fixture', 'plainrepo')) $tg $root @("$gf\plainrepo\mock.jsonl") @{ GIT_DIR = "$victim\.git"; GIT_WORK_TREE = $victim; GIT_CONFIG_COUNT = 'x' }
    $folder = [IO.Path]::GetDirectoryName(($gitCmd.Source -replace '\\(ucrt64|mingw64|mingw32|clangarm64)\\bin\\git\.exe\z', '\cmd\git.exe'))
    $wantGit =@('@echo off', $clearGit, "set `"LOCALAPPDATA=$runI\lad`"", "set `"APPDATA=$runI\ad`"", "set `"USERPROFILE=$runI\home`"", "set `"HOME=$runI\home`"", "set `"XDG_CONFIG_HOME=$runI\home\.config`"", "set `"PATH=C:\Windows\System32;$folder`"", "set `"GIT_CEILING_DIRECTORIES=$gf`"", 'set "GIT_CONFIG_NOSYSTEM=1"', 'set "GIT_ALLOW_PROTOCOL=file"', 'set "GIT_CONFIG_COUNT=2"', 'set "GIT_CONFIG_KEY_0=core.hooksPath"', "set `"GIT_CONFIG_VALUE_0=$runI\nohooks`"", 'set "GIT_CONFIG_KEY_1=core.fsmonitor"', 'set "GIT_CONFIG_VALUE_1=false"', 'set NO_COLOR=', "`"$acc`" --mock `"$good\mock.jsonl`"") -join "`n"
    $launchGit = ([IO.File]::ReadAllText("$runI\launch.cmd") -replace "`r`n", "`n").TrimEnd()
    Report ($launchGit -ceq $wantGit) 'its launch.cmd, exactly' $launchGit
    Report (@([IO.Directory]::GetFileSystemEntries("$runI\nohooks")).Count -eq 0) 'its hooks folder is empty' ''

    '=== -git: its effects, git under the launch''s settings, a hostile environment around it ==='
    # Each probe is launch.cmd but its last line, the build, then git commands; cmd.exe
    # is given every setting that could redirect git. No build runs here, and no window.
    [void](Put "$evil\hooks\pre-commit" "#!/bin/sh`necho ran > '$($evil.Replace('\', '/'))/ran-evil.txt'`n")
    [void](Put "$evil\gitconfig" "[core]`n`thooksPath = $($evil.Replace('\', '/'))/hooks`n[credential]`n`thelper = evil`n[url `"https://127.0.0.1:9/`"]`n`tinsteadOf = ../`n")
    [void](Put "$evil\xdg\git\config" "[core]`n`thooksPath = $($evil.Replace('\', '/'))/hooks`n[credential]`n`thelper = evil`n")
    $hostile = @{
        GIT_DIR = "$victim\.git"; GIT_WORK_TREE = $victim; GIT_INDEX_FILE = "$victim\.git\index"; GIT_COMMON_DIR = "$victim\.git"
        GIT_OBJECT_DIRECTORY = "$victim\.git\objects"; GIT_ALTERNATE_OBJECT_DIRECTORIES = "$victim\.git\objects"
        GIT_CONFIG_GLOBAL = "$evil\gitconfig"; GIT_CONFIG_SYSTEM = "$evil\gitconfig"; GIT_CONFIG = "$evil\gitconfig"; GIT_CONFIG_NOSYSTEM = '0'
        GIT_CONFIG_COUNT = '1'; GIT_CONFIG_KEY_0 = 'core.hooksPath'; GIT_CONFIG_VALUE_0 = "$evil\hooks"; GIT_CONFIG_PARAMETERS = "'core.hookspath=$evil\hooks'"
        GIT_ALLOW_PROTOCOL = 'https:http:ssh:git:ext:file'; GIT_SSH_COMMAND = "$evil\ssh.cmd"; GIT_ASKPASS = "$evil\askpass.cmd"; GIT_EXEC_PATH = "$evil\exec"
        GIT_TEMPLATE_DIR = $evil; GIT_NAMESPACE = 'evil'; XDG_CONFIG_HOME = "$evil\xdg"; HOME = $evil
    }
    function Under-Launch([string[]]$commands, [hashtable]$vars = $hostile) {
        $lines = @(Get-Content -LiteralPath "$runI\launch.cmd")
        Set-Content -LiteralPath "$runI\probe.cmd" -Encoding ascii -Value ($lines[0..($lines.Count - 2)] + @('set "LC_ALL=C"') + $commands)
        return Run-Process $env:ComSpec "/c `"$runI\probe.cmd`"" $temp $root $vars
    }
    $victimHead = Must-Git "-C `"$victim`" rev-parse HEAD"
    $victimState = (@(Walk $victim) | Sort-Object) -join "`n"
    $bareState = (@(Walk $outsideBare) | Sort-Object) -join "`n"
    $control = Run-Process $gitCmd.Source "-C `"$good`" rev-parse --show-toplevel" $temp $root @{ GIT_DIR = "$victim\.git"; GIT_WORK_TREE = $victim }
    Report ($control.Out.Trim() -ieq $victim.Replace('\', '/')) 'control: with no launch, inherited settings redirect git to another repository' $control.Out.Trim()
    $r = Under-Launch @("git -C `"$good`" rev-parse --show-toplevel")
    Report ($r.Exit -eq 0 -and $r.Out.Trim() -ieq $good.Replace('\', '/')) 'under the launch, git stays on the fixture' "$($r.Out.Trim()) $($r.Err)"
    $r = Under-Launch @("git -C `"$good`" config --show-origin --list")
    Report ($r.Exit -eq 0 -and $r.Out -notmatch 'evil' -and $r.Out -notmatch 'etc/gitconfig' -and $r.Out -notmatch 'ProgramData') 'no inherited, system or XDG setting reaches it' $r.Out
    $r = Under-Launch @("git -C `"$good`" config --get-all credential.helper")
    Report ($r.Exit -ne 0 -and -not $r.Out.Trim()) 'no credential helper is left for a push' $r.Out
    $r = Under-Launch @("git -C `"$good`" rev-parse --git-path hooks")
    Report ($r.Out.Trim().Replace('/', '\') -ieq "$runI\nohooks") 'its hooks folder is the run''s empty one' $r.Out

    # Hooks: a control proves such a hook runs when nothing stops it.
    New-Repo "$gf\hookcontrol"; Seed-Repo "$gf\hookcontrol"
    [void](Put "$gf\hookcontrol\.git\hooks\pre-commit" "#!/bin/sh`necho ran > '$($evil.Replace('\', '/'))/ran-control.txt'`n")
    [void](Must-Git "-C `"$gf\hookcontrol`" commit -q --allow-empty -m control")
    Report ([IO.File]::Exists("$evil\ran-control.txt")) 'control: with no launch, a repository''s hook runs' ''
    [void](Put "$good\.git\hooks\pre-commit" "#!/bin/sh`necho ran > '$($evil.Replace('\', '/'))/ran-repo.txt'`n")
    [void](Put "$goodBare\hooks\post-receive" "#!/bin/sh`necho ran > '$($evil.Replace('\', '/'))/ran-remote.txt'`n")
    $before = Must-Git "-C `"$good`" rev-parse HEAD"
    $r = Under-Launch @("git -C `"$good`" -c user.useConfigOnly=true commit -q --allow-empty -m probe")
    $after = Must-Git "-C `"$good`" rev-parse HEAD"
    Report ($r.Exit -eq 0 -and $after -ne $before) 'a commit on the fixture works' "$($r.Err)"
    Report (-not [IO.File]::Exists("$evil\ran-repo.txt")) 'a hook added to the repository after the checks does not run' ''
    Report (-not [IO.File]::Exists("$evil\ran-evil.txt")) 'nor does a hook an inherited setting names' ''
    $r = Under-Launch @("git -C `"$good`" push --porcelain origin HEAD:refs/heads/probe")
    Report ($r.Exit -eq 0 -and (Must-Git "-C `"$goodBare`" rev-parse refs/heads/probe") -eq $after) 'a push to the local bare remote works' "exit $($r.Exit): $($r.Out) $($r.Err)"
    Report (-not [IO.File]::Exists("$evil\ran-remote.txt")) 'a hook added to that remote after the checks does not run' ''
    [void](Must-Git "-C `"$good`" push -q origin HEAD:refs/heads/probecontrol")
    Report ([IO.File]::Exists("$evil\ran-remote.txt")) 'control: with no launch, that remote''s hook runs' ''

    # External remotes: refused before any connection, whatever is inherited.
    foreach ($u in @('https://127.0.0.1:9/x.git', 'http://127.0.0.1:9/x.git', 'ssh://127.0.0.1:9/x.git', 'git://127.0.0.1:9/x.git', 'ext::cmd.exe /c exit 1')) {
        $r = Under-Launch @("git -C `"$good`" push `"$u`" HEAD")
        Report ($r.Exit -ne 0 -and $r.Err -match 'not allowed') "a push to [$u] is refused" "$($r.Err)"
    }
    $r = Under-Launch @("git -C `"$good`" fetch `"https://127.0.0.1:9/x.git`"")
    Report ($r.Exit -ne 0 -and $r.Err -match 'not allowed') 'so is a fetch' "$($r.Err)"
    $r = Under-Launch @("git clone -q `"https://127.0.0.1:9/x.git`" `"$runI\cloned`"")
    Report ($r.Exit -ne 0 -and $r.Err -match 'not allowed') 'so is a clone' "$($r.Err)"
    $noProxy = $cleanGit + @{ HTTPS_PROXY = ''; HTTP_PROXY = ''; ALL_PROXY = ''; NO_PROXY = '*'; LC_ALL = 'C'; GIT_TERMINAL_PROMPT = '0' }
    $control = Run-Process $gitCmd.Source "-C `"$good`" -c http.proxy= push `"https://127.0.0.1:9/x.git`" HEAD" $temp $root $noProxy
    Report ($control.Exit -ne 0 -and $control.Err -notmatch 'not allowed') 'control: with no launch, the same push gets as far as 127.0.0.1 port 9' "$($control.Err)"

    # The search for a repository, and what must not have changed.
    New-Repo $tg
    $control = Git-Clean "-C `"$gf\plain`" rev-parse --is-inside-work-tree"
    Report ($control.Exit -eq 0) 'control: with no ceiling, git finds the repository above the fixtures' "exit $($control.Exit)"
    $r = Under-Launch @("git -C `"$gf\plain`" rev-parse --is-inside-work-tree")
    Report ($r.Exit -ne 0) 'under the launch, it does not' "exit $($r.Exit): $($r.Out)"
    Report ((Must-Git "-C `"$victim`" rev-parse HEAD") -eq $victimHead -and ((@(Walk $victim) | Sort-Object) -join "`n") -ceq $victimState) 'the repository outside is unchanged, byte for byte' ''
    Report (((@(Walk $outsideBare) | Sort-Object) -join "`n") -ceq $bareState) 'the bare repository outside too' ''
    Report (@([IO.Directory]::GetFileSystemEntries("$runI\nohooks")).Count -eq 0) 'the run''s hooks folder is still empty' ''
    $script:state = Snapshot
} else {
    NotRun '-git, all of it' 'no git.exe on the PATH'
}

'=== the harness''s text ==='
$text = [IO.File]::ReadAllText($harness)
$allLines = $text -split "`n"
$code = (@($allLines | Where-Object { $_ -notmatch '^\s*#' }) -join "`n")
foreach ($w in @('rmdir', 'Remove-Item', 'Move-Item', 'Rename-Item', 'Clear-Content', '.Delete(', 'Invoke-Expression', 'iex ', '-Recurse', '-Force', 'Clipboard', 'SendKeys', 'WriteAllText', '& powershell', '& $')) {
    Report (-not $code.Contains($w)) "no [$w] in its code" ''
}
# The old cmd /c rmdir is gone; cmd.exe /c still runs the generated launch.cmd, once.
Report (([regex]::Matches($code, 'cmd\.exe')).Count -eq 1 -and $code.Contains("'cmd.exe', '/c', `"```"`$launch```"`"")) 'cmd.exe is used once: /c and the quoted launch.cmd' ''
Report (([regex]::Matches($code, 'Start-Process')).Count -eq 1 -and $code.Contains('Start-Process wt.exe')) 'it starts one window, wt.exe' ''
Report (([regex]::Matches($code, '\[Diagnostics\.Process\]::Start')).Count -eq 1 -and ([regex]::Matches($code, 'Read-Git [''"]')).Count -eq 2 -and $code.Contains("Read-Git '--version'") -and $code.Contains('Read-Git "config --file')) 'it starts git itself only for its version and one config file' ''
Report ($text -cnotmatch '[^\x00-\x7F]' -and -not $text.Contains("`r")) 'ASCII and LF' ''
$table = $code.Substring($code.IndexOf('$stepArgs = @{'))
$table = $table.Substring(0, $table.IndexOf("`n}"))
$tableSteps = @([regex]::Matches($table, '\b([a-z]+) = ''') | ForEach-Object { $_.Groups[1].Value } | Sort-Object)
$body = $code.Substring($code.IndexOf('switch ($p[0])'))
$bodySteps = @([regex]::Matches($body, "(?m)^\s+'([a-z]+)' \{") | ForEach-Object { $_.Groups[1].Value } | Sort-Object)
Report (($tableSteps -join ' ') -ceq ($bodySteps -join ' ')) "the checked steps are the steps it runs ($($bodySteps.Count))" "$($tableSteps -join ' ') / $($bodySteps -join ' ')"
$head = (@($allLines | Where-Object { $_ -match '^\s*#' }) -join "`n")
$unnamed = @($bodySteps | Where-Object { $head -notmatch "(?m)^#\s+(.*[ ,])?$_\b" })
Report ($unnamed.Count -eq 0) 'its head names every step' ($unnamed -join ' ')

''
"$($script:passed) passed, $($script:failed) failed, $($script:notRun) not run"
"folder: %TEMP%\$(Split-Path -Leaf $root)"
if ($script:failed -ne 0) { exit 1 }
