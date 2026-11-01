param([string]$file, [string]$regions)
# glyph-colour.ps1: the foreground colour of thin glyphs in a reference image. For each
# region "name:x0,y0,x1,y1", the pixels at least 40 levels brighter than the
# region's darkest tenth; their brightest tenth averaged ("core"), and their
# count. ASCII only.
Add-Type -AssemblyName System.Drawing
$bmp = [System.Drawing.Bitmap]::FromFile($file)
foreach ($r in ($regions -split ';' | Where-Object { $_ -ne '' })) {
    $name, $box = $r -split ':', 2
    $c = $box -split ',' | ForEach-Object { [int]$_ }
    $px = New-Object System.Collections.Generic.List[object]
    for ($y = $c[1]; $y -le $c[3]; $y++) {
        for ($x = $c[0]; $x -le $c[2]; $x++) {
            $p = $bmp.GetPixel($x, $y)
            $px.Add(@($p.R, $p.G, $p.B, ($p.R + $p.G + $p.B)))
        }
    }
    $sorted = $px | Sort-Object { $_[3] }
    $n = $sorted.Count
    $dark = ($sorted[0..([Math]::Max([int]($n / 10) - 1, 0))] | ForEach-Object { $_[3] } | Measure-Object -Average).Average
    $fgPx = @($sorted | Where-Object { $_[3] -ge $dark + 120 })
    if ($fgPx.Count -eq 0) { "{0,-22} no foreground" -f $name; continue }
    $top = $fgPx[([int]($fgPx.Count * 0.9))..($fgPx.Count - 1)]
    $avg = @(0, 1, 2 | ForEach-Object { $i = $_; [int](($top | ForEach-Object { $_[$i] } | Measure-Object -Average).Average) })
    $mid = $fgPx[[int]($fgPx.Count / 2)]
    "{0,-22} core #{1:X2}{2:X2}{3:X2}  median #{4:X2}{5:X2}{6:X2}  fg pixels {7}" -f $name, $avg[0], $avg[1], $avg[2], $mid[0], $mid[1], $mid[2], $fgPx.Count
}
$bmp.Dispose()
