<#
Regenerates the MSIX logo assets from desktop/build/appicon.png (1024x1024).
Run it from anywhere; the PNGs are committed, so this is only needed when the
icon changes.
#>
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Drawing

$here = Split-Path -Parent $MyInvocation.MyCommand.Path
$source = Join-Path $here '..\..\appicon.png'
$out = Join-Path $here 'Assets'

# name -> size. The manifest references StoreLogo, Square44x44Logo and
# Square150x150Logo; the target sizes keep the taskbar and Start icons sharp.
$sizes = [ordered]@{
	'StoreLogo.png'                         = 50
	'Square44x44Logo.png'                   = 44
	'Square44x44Logo.targetsize-24.png'     = 24
	'Square44x44Logo.targetsize-256.png'    = 256
	'Square150x150Logo.png'                 = 150
}

$img = [System.Drawing.Image]::FromFile((Resolve-Path $source))
try {
	foreach ($name in $sizes.Keys) {
		$size = $sizes[$name]
		$bmp = New-Object System.Drawing.Bitmap $size, $size, ([System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
		$g = [System.Drawing.Graphics]::FromImage($bmp)
		try {
			$g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
			$g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::HighQuality
			$g.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
			$g.Clear([System.Drawing.Color]::Transparent)
			$g.DrawImage($img, 0, 0, $size, $size)
			$bmp.Save((Join-Path $out $name), [System.Drawing.Imaging.ImageFormat]::Png)
		} finally {
			$g.Dispose()
			$bmp.Dispose()
		}
	}
} finally {
	$img.Dispose()
}
