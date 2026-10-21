# acc.ps1, run by acc.cmd at the repository's root (plan 23, brief
# amendment 21): starts the Agent Command Center from this repository with
# go run, its project field prefilled with the folder acc runs in. Arguments
# go to the program, for example: acc --provider codex. --mock runs alone, so
# then no --project is added.
#
# In Windows Terminal the dashboard opens in a tab of its own, in the same
# window, with the profile "Agent Command Center" (font size 9, the owner's
# size); the tab closes when the dashboard exits cleanly. The profile is a
# Terminal fragment, the extension Windows Terminal documents for apps,
# written below when missing or different; Terminal's own settings.json is
# never edited. Outside Windows Terminal the dashboard runs in place.
#
# No param block: the program's arguments (--provider, --mock...) stay in
# $args unbound. ASCII only: Windows PowerShell 5.1 reads a BOM-less script
# as ANSI.

$repo = Split-Path -Parent $PSScriptRoot
$here = (Get-Location).ProviderPath
$profileName = 'Agent Command Center'
$profileGuid = '{27195de0-7daf-4340-8ef6-2c640982bd39}'

if ($env:WT_SESSION -and -not $env:ACC_IN_TAB -and $env:WT_PROFILE_ID -ne $profileGuid) {
    $fragmentDir = Join-Path $env:LOCALAPPDATA 'Microsoft\Windows Terminal\Fragments\AgentCommandCenter'
    $fragmentFile = Join-Path $fragmentDir 'agent-command-center.json'
    # Its commandline is one no other command starts with: Terminal gives a
    # tab opened with a commandline and no profile the profile whose
    # commandline begins it, so "powershell.exe -NoLogo" would have made any
    # such PowerShell tab small too (found by plan 23's tests).
    $fragment = @"
{
  "profiles": [
    {
      "guid": "$profileGuid",
      "name": "$profileName",
      "commandline": "powershell.exe -NoLogo -NoExit -Command \"Write-Host 'Agent Command Center: type acc in a project folder to start the dashboard.'\"",
      "font": { "size": 9 },
      "closeOnExit": "graceful",
      "tabTitle": "$profileName"
    }
  ]
}
"@
    $current = if (Test-Path -LiteralPath $fragmentFile) { [System.IO.File]::ReadAllText($fragmentFile) } else { $null }
    if ($current -ne $fragment) {
        New-Item -ItemType Directory -Force -Path $fragmentDir | Out-Null
        [System.IO.File]::WriteAllText($fragmentFile, $fragment, (New-Object System.Text.UTF8Encoding($false)))
        # A running Terminal reads fragments when it reloads its settings,
        # which it does when its settings.json changes: setting the file's
        # time is enough, its bytes are left as they are (Terminal 1.23).
        $now = Get-Date
        foreach ($settings in @(
                (Join-Path $env:LOCALAPPDATA 'Packages\Microsoft.WindowsTerminal_8wekyb3d8bbwe\LocalState\settings.json'),
                (Join-Path $env:LOCALAPPDATA 'Packages\Microsoft.WindowsTerminalPreview_8wekyb3d8bbwe\LocalState\settings.json'),
                (Join-Path $env:LOCALAPPDATA 'Microsoft\Windows Terminal\settings.json'))) {
            if (Test-Path -LiteralPath $settings) { (Get-Item -LiteralPath $settings).LastWriteTime = $now }
        }
        Start-Sleep -Seconds 2
    }
    # The tab runs this script again, marked as in its tab. The command is
    # base64, so no quoting goes through wt, which splits its own arguments
    # on semicolons.
    $quoted = ($args | ForEach-Object { "'" + ("$_" -replace "'", "''") + "'" }) -join ' '
    $command = "`$env:ACC_IN_TAB = '1'; "
    if ($env:ANTHROPIC_MODEL) { $command += "`$env:ANTHROPIC_MODEL = '" + ($env:ANTHROPIC_MODEL -replace "'", "''") + "'; " }
    $command += "& '" + ($PSCommandPath -replace "'", "''") + "' $quoted; exit `$LASTEXITCODE"
    $encoded = [Convert]::ToBase64String([System.Text.Encoding]::Unicode.GetBytes($command))
    wt.exe -w 0 new-tab --profile $profileName --startingDirectory $here powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -EncodedCommand $encoded
    exit $LASTEXITCODE
}

$mock = @($args | Where-Object { "$_" -match '^--?mock(=|$)' }).Count -gt 0
$code = 1
Push-Location -LiteralPath $repo
try {
    if ($mock) { go run ./cmd/agent-command-center @args }
    else { go run ./cmd/agent-command-center --project $here @args }
    if ($LASTEXITCODE -ne $null) { $code = $LASTEXITCODE }
} finally { Pop-Location }
exit $code
