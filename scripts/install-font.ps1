# install-font.ps1, run by acc.ps1 under Windows Terminal (plan 56, brief
# amendment 57): installs Cascadia Mono NF, bundled in this repository,
# for the current user when Windows has no font of that name, as Windows'
# own Install does: the files checked against the bundle's SHA256SUMS,
# copied to the user's font folder (a file already there with the same
# bytes reused), registered under HKCU, their creation time made the
# registration's (plan 55's rule takes it for the install's), loaded into
# the session and announced. It prints one line when it installs or fails
# and then returns $true; otherwise it prints nothing and returns $false.
# A failure never stops acc. The folders and keys are parameters, so that
# tests can use temporary ones; -NoSession leaves the session's fonts
# alone. ASCII only: Windows PowerShell 5.1 reads a BOM-less script as
# ANSI.
param(
    [string]$Bundle = (Join-Path (Split-Path -Parent $PSScriptRoot) 'third_party\fonts\cascadia-code-2407.24'),
    [string]$FontsDir = (Join-Path $env:LOCALAPPDATA 'Microsoft\Windows\Fonts'),
    [string]$Key = 'HKCU:\Software\Microsoft\Windows NT\CurrentVersion\Fonts',
    [string[]]$Registered = @('HKCU:\Software\Microsoft\Windows NT\CurrentVersion\Fonts', 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Fonts'),
    [switch]$NoSession
)

$face = 'Cascadia Mono NF'

# A font of that name whose file is there, for the user or the machine,
# put there by acc, by hand or by another installer: nothing to do.
foreach ($k in $Registered) {
    $item = Get-Item -LiteralPath $k -ErrorAction SilentlyContinue
    if (-not $item) { continue }
    foreach ($n in $item.GetValueNames()) {
        if (-not $n.StartsWith("$face ")) { continue }
        $file = [string]$item.GetValue($n)
        if ($file -and -not [System.IO.Path]::IsPathRooted($file)) { $file = Join-Path "$env:WINDIR\Fonts" $file }
        if ($file -and (Test-Path -LiteralPath $file)) { return $false }
    }
}

$files = [ordered]@{
    'CascadiaMonoNF.ttf'       = "$face (TrueType)"
    'CascadiaMonoNFItalic.ttf' = "$face Italic (TrueType)"
}
try {
    $ErrorActionPreference = 'Stop'
    $sums = @{}
    foreach ($line in [System.IO.File]::ReadAllLines((Join-Path $Bundle 'SHA256SUMS'))) {
        if ($line -match '^([0-9a-f]{64})  (\S+)$') { $sums[$Matches[2]] = $Matches[1] }
    }
    $installed = @()
    foreach ($name in $files.Keys) {
        $src = Join-Path $Bundle $name
        $hash = (Get-FileHash -LiteralPath $src -Algorithm SHA256).Hash
        if (-not $sums.ContainsKey($name) -or $hash -ne $sums[$name]) { throw "$src is not the bundled file (SHA-256 $hash)" }
        $dst = Join-Path $FontsDir $name
        if (Test-Path -LiteralPath $dst) {
            if ((Get-FileHash -LiteralPath $dst -Algorithm SHA256).Hash -ne $hash) { throw "$dst is another file of that name, left as it is" }
        } else {
            New-Item -ItemType Directory -Force -Path $FontsDir | Out-Null
            Copy-Item -LiteralPath $src -Destination $dst
        }
        $installed += $dst
    }
    if (-not (Test-Path -LiteralPath $Key)) { New-Item -Path $Key -Force | Out-Null }
    $i = 0
    foreach ($name in $files.Keys) {
        New-ItemProperty -LiteralPath $Key -Name $files[$name] -Value $installed[$i] -PropertyType String -Force | Out-Null
        $i++
    }
    if (-not ('AccFontInstall' -as [type])) {
        Add-Type -TypeDefinition @"
using System;
using System.Runtime.InteropServices;
using Microsoft.Win32.SafeHandles;
public static class AccFontInstall {
    [DllImport("kernel32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
    static extern SafeFileHandle CreateFileW(string name, uint access, uint share, IntPtr security, uint disposition, uint flags, IntPtr template);
    [DllImport("kernel32.dll", SetLastError = true)]
    static extern bool SetFileTime(SafeFileHandle file, ref long created, IntPtr accessed, IntPtr written);
    [DllImport("gdi32.dll", CharSet = CharSet.Unicode)]
    public static extern int AddFontResourceW(string file);
    [DllImport("user32.dll")]
    public static extern IntPtr SendMessageTimeoutW(IntPtr window, uint message, IntPtr wparam, IntPtr lparam, uint flags, uint timeout, out IntPtr result);
    // Created sets a file's creation time through a handle opened for its
    // attributes only (FILE_WRITE_ATTRIBUTES, every sharing mode), which a
    // font in use does not refuse.
    public static void Created(string file, DateTime at) {
        using (SafeFileHandle h = CreateFileW(file, 0x100, 7, IntPtr.Zero, 3, 0, IntPtr.Zero)) {
            if (h.IsInvalid) throw new System.ComponentModel.Win32Exception(Marshal.GetLastWin32Error());
            long t = at.ToFileTimeUtc();
            if (!SetFileTime(h, ref t, IntPtr.Zero, IntPtr.Zero)) throw new System.ComponentModel.Win32Exception(Marshal.GetLastWin32Error());
        }
    }
}
"@
    }
    # The registration's time, which a file reused or made again under its
    # name within 15 s (Windows' file name tunnelling) would not have.
    $now = Get-Date
    foreach ($f in $installed) { [AccFontInstall]::Created($f, $now) }
    if (-not $NoSession) {
        foreach ($f in $installed) { [void][AccFontInstall]::AddFontResourceW($f) }
        $result = [IntPtr]::Zero
        [void][AccFontInstall]::SendMessageTimeoutW([IntPtr]0xffff, 0x001D, [IntPtr]::Zero, [IntPtr]::Zero, 0x0002, 1000, [ref]$result)
    }
    Write-Host "Agent Command Center: $face installed for this account; the Explorer's icons appear once Windows Terminal starts again."
    return $true
} catch {
    Write-Host "Agent Command Center: $face not installed: $($_.Exception.Message)"
    return $true
}
