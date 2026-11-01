param([string]$file, [string]$regions)
# image-colours.ps1: an image's colours by pixel region. For each region
# "name:x0,y0,x1,y1" (inclusive), the most frequent colour (the ground)
# and the glyphs' core: the brightest tenth of the pixels at least 120
# levels (summed) from the ground, averaged; and the darkest tenth of those,
# for lines darker than the ground. ASCII only.
Add-Type -AssemblyName System.Drawing
$bmp = [System.Drawing.Bitmap]::FromFile($file)
foreach ($r in ($regions -split ';' | Where-Object { $_ -ne '' })) {
    $name, $box = $r -split ':', 2
    $c = $box -split ',' | ForEach-Object { [int]$_ }
    $count = @{}
    $all = New-Object System.Collections.Generic.List[object]
    for ($y = $c[1]; $y -le $c[3]; $y++) {
        for ($x = $c[0]; $x -le $c[2]; $x++) {
            $p = $bmp.GetPixel($x, $y)
            $k = '{0:X2}{1:X2}{2:X2}' -f $p.R, $p.G, $p.B
            $count[$k] = 1 + [int]$count[$k]
            $all.Add(@([int]$p.R, [int]$p.G, [int]$p.B))
        }
    }
    $top3 = ($count.GetEnumerator() | Sort-Object Value -Descending | Select-Object -First 3 | ForEach-Object { "#{0} {1}" -f $_.Key, $_.Value }) -join ', '
    $ground = ($count.GetEnumerator() | Sort-Object Value -Descending | Select-Object -First 1).Key
    $gr = [Convert]::ToInt32($ground.Substring(0, 2), 16); $gg = [Convert]::ToInt32($ground.Substring(2, 2), 16); $gb = [Convert]::ToInt32($ground.Substring(4, 2), 16)
    $fg = @($all | Where-Object { [Math]::Abs($_[0] - $gr) + [Math]::Abs($_[1] - $gg) + [Math]::Abs($_[2] - $gb) -ge 120 } | Sort-Object { $_[0] + $_[1] + $_[2] })
    $core = 'none'
    if ($fg.Count -gt 0) {
        $top = $fg[([int]($fg.Count * 0.9))..($fg.Count - 1)]
        $avg = @(0, 1, 2 | ForEach-Object { $i = $_; [int](($top | ForEach-Object { $_[$i] } | Measure-Object -Average).Average) })
        $core = '#{0:X2}{1:X2}{2:X2}' -f $avg[0], $avg[1], $avg[2]
    }
    "{0,-22} {1}  glyphs {2} ({3} px)" -f $name, $top3, $core, $fg.Count
}
$bmp.Dispose()
