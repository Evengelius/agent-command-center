param([string]$file, [string]$text, [string]$regions)
# screen-colours.ps1: colours read back from a Windows Terminal screenshot (Shot.Save,
# the whole window: tab bar and borders included). The text grid is
# calibrated from the image: zone A's left border and the sidebar's right
# border (the leftmost and rightmost long vertical frame lines) give the
# columns, zone C's top and bottom borders (the long horizontal lines on the
# rows the text dump shows them) the rows. For each region
# "name:c0,r0,c1,r1" (cells, inclusive), the ground, its most frequent
# colour, and the glyphs' core, the brightest tenth of the pixels at least
# 120 levels (summed) from the ground, averaged. ASCII only.
Add-Type -AssemblyName System.Drawing
$rows = [IO.File]::ReadAllText($text, [Text.Encoding]::UTF8).TrimStart([char]0xFEFF) -split "`r?`n"
if ($rows[-1] -eq '') { $rows = $rows[0..($rows.Count - 2)] }
$cols = ($rows | ForEach-Object { $_.Length } | Measure-Object -Maximum).Maximum
# The text rows drawn as lines: three quarters of their cells "-" (U+2500),
# in the lower half.
$lineRows = @()
for ($i = [int]($rows.Count / 2); $i -lt $rows.Count; $i++) {
    $n = ($rows[$i].ToCharArray() | Where-Object { $_ -eq [char]0x2500 }).Count
    if ($n -ge $cols * 0.75) { $lineRows += $i }
}
$bmp = [System.Drawing.Bitmap]::FromFile($file)
$W = $bmp.Width; $H = $bmp.Height
$rect = New-Object System.Drawing.Rectangle(0, 0, $W, $H)
$data = $bmp.LockBits($rect, [System.Drawing.Imaging.ImageLockMode]::ReadOnly, [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
$buf = New-Object byte[] ($data.Stride * $H)
[Runtime.InteropServices.Marshal]::Copy($data.Scan0, $buf, 0, $buf.Length)
$stride = $data.Stride
$bmp.UnlockBits($data)
function Px([int]$x, [int]$y) { $o = $y * $stride + $x * 4; return @($buf[$o + 2], $buf[$o + 1], $buf[$o]) }
function Lit([int]$x, [int]$y) { $p = Px $x $y; return ($p[0] + $p[1] + $p[2]) -gt 150 }
# Vertical lines: a column lit over most of the middle half of the height.
$ya = [int]($H * 0.3); $yb = [int]($H * 0.6)
$xs = @()
for ($x = 0; $x -lt $W; $x++) {
    $n = 0; for ($y = $ya; $y -lt $yb; $y += 2) { if (Lit $x $y) { $n++ } }
    if ($n -ge ($yb - $ya) / 2 * 0.9) { $xs += $x }
}
$xL = ($xs | Where-Object { $_ -le $xs[0] + 3 } | Measure-Object -Average).Average
$xR = ($xs | Where-Object { $_ -ge $xs[-1] - 3 } | Measure-Object -Average).Average
$cw = ($xR - $xL) / ($cols - 1)
$x0 = $xL - $cw / 2
# Horizontal lines: a row lit over most of the width; the first and last in
# the lower half are zone C's top and bottom borders.
$ys = @()
for ($y = [int]($H / 2); $y -lt $H; $y++) {
    $n = 0; for ($x = [int]$xL; $x -lt [int]$xR; $x += 3) { if (Lit $x $y) { $n++ } }
    if ($n -ge ($xR - $xL) / 3 * 0.9) { $ys += $y }
}
$groups = @(); $cur = @()
foreach ($y in $ys) { if ($cur.Count -gt 0 -and $y -gt $cur[-1] + 1) { $groups += ,$cur; $cur = @() }; $cur += $y }
if ($cur.Count -gt 0) { $groups += ,$cur }
# The window's own edge, under the text, is no text row.
if ($groups.Count -gt $lineRows.Count) { $groups = $groups[0..($lineRows.Count - 1)] }
$gy = ($groups | ForEach-Object { [int]($_ | Measure-Object -Average).Average }) -join ","
if ($groups.Count -ne $lineRows.Count) { throw ("{0} line rows in the text ({2}), {1} lines in the image at y {3}" -f $lineRows.Count, $groups.Count, ($lineRows -join ","), $gy) }
$rTop = $lineRows[0]; $rBot = $lineRows[-1]
$yT = ($groups[0] | Measure-Object -Average).Average
$yB = ($groups[-1] | Measure-Object -Average).Average
$ch = ($yB - $yT) / ($rBot - $rTop)
$y0 = $yT - $ch / 2 - $rTop * $ch
"grid {0}x{1}; cell {2:N2} x {3:N2} px; origin {4:N1},{5:N1} (line rows {6} and {7})" -f $cols, $rows.Count, $cw, $ch, $x0, $y0, $rTop, $rBot
foreach ($r in ($regions -split ';' | Where-Object { $_ -ne '' })) {
    $name, $box = $r -split ':', 2
    $c = $box -split ',' | ForEach-Object { [int]$_ }
    $px0 = [int]($x0 + $c[0] * $cw + 1); $px1 = [int]($x0 + ($c[2] + 1) * $cw - 2)
    $py0 = [int]($y0 + $c[1] * $ch + 1); $py1 = [int]($y0 + ($c[3] + 1) * $ch - 2)
    $count = @{}
    $all = New-Object System.Collections.Generic.List[object]
    for ($y = $py0; $y -le $py1; $y++) {
        for ($x = $px0; $x -le $px1; $x++) {
            $p = Px $x $y
            $k = '{0:X2}{1:X2}{2:X2}' -f $p[0], $p[1], $p[2]
            $count[$k] = 1 + [int]$count[$k]
            $all.Add($p)
        }
    }
    $ground = ($count.GetEnumerator() | Sort-Object Value -Descending | Select-Object -First 1).Key
    $gr = [Convert]::ToInt32($ground.Substring(0, 2), 16); $gg = [Convert]::ToInt32($ground.Substring(2, 2), 16); $gb = [Convert]::ToInt32($ground.Substring(4, 2), 16)
    $fg = @($all | Where-Object { [Math]::Abs($_[0] - $gr) + [Math]::Abs($_[1] - $gg) + [Math]::Abs($_[2] - $gb) -ge 120 } | Sort-Object { [int]$_[0] + $_[1] + $_[2] })
    $core = 'none'
    if ($fg.Count -gt 0) {
        $top = $fg[([int]($fg.Count * 0.9))..($fg.Count - 1)]
        $avg = @(0, 1, 2 | ForEach-Object { $i = $_; [int](($top | ForEach-Object { $_[$i] } | Measure-Object -Average).Average) })
        $core = '#{0:X2}{1:X2}{2:X2}' -f $avg[0], $avg[1], $avg[2]
    }
    $txt = ''
    for ($ry = $c[1]; $ry -le $c[3] -and $ry -lt $rows.Count; $ry++) { $line = $rows[$ry]; if ($line.Length -gt $c[0]) { $txt += $line.Substring($c[0], [Math]::Min($c[2] - $c[0] + 1, $line.Length - $c[0])) } }
    "{0,-24} ground #{1} ({2}/{3} px)  glyphs {4} ({5} px)  text '{6}'" -f $name, $ground, $count[$ground], $all.Count, $core, $fg.Count, $txt.Trim()
}
$bmp.Dispose()
