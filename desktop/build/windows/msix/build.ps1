<#
Packs the production executable into an MSIX with MakeAppx (Windows SDK).

The package is NOT signed here: the Store signs the package it publishes, and
CI signs a throwaway copy (desktop/scripts/msix-ci.ps1). The identity defaults
are fake, so a package built without them can never be submitted.

  -Exe                   the production vukem-finances.exe
  -Out                   the .msix file to write
  -Version               X.Y.Z (a pre-release suffix is dropped); the package
                         version is X.Y.Z.0 because the Store reserves the
                         last part
  -IdentityName          "Package/Identity/Name" of Partner Center
  -Publisher             "Package/Identity/Publisher" (CN=...) of Partner Center
  -PublisherDisplayName  "Package/Properties/PublisherDisplayName" of Partner Center
#>
param(
	[Parameter(Mandatory)][string]$Exe,
	[Parameter(Mandatory)][string]$Out,
	[string]$Version = '0.0.0-dev',
	[string]$IdentityName = 'VukemFinances.CI',
	[string]$Publisher = 'CN=VukemFinancesCI',
	[string]$PublisherDisplayName = 'vukem-finances CI (fake identity)',
	[string]$DisplayName = 'vukem-finances'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$here = Split-Path -Parent $MyInvocation.MyCommand.Path

if (-not (Test-Path -LiteralPath $Exe -PathType Leaf)) {
	throw "executable not found: $Exe (run the Windows build first)"
}

if ($Version -notmatch '^(\d+)\.(\d+)\.(\d+)(?:[-+].*)?$') {
	throw "version must start with X.Y.Z, got '$Version'"
}
foreach ($part in $Matches[1], $Matches[2], $Matches[3]) {
	if ([int]$part -gt 65535) { throw "version part out of range (0-65535): $part" }
}
$packageVersion = '{0}.{1}.{2}.0' -f [int]$Matches[1], [int]$Matches[2], [int]$Matches[3]

if ($IdentityName -notmatch '^[A-Za-z0-9][A-Za-z0-9.-]{2,49}$') {
	throw "IdentityName must be 3-50 letters, digits, dots or dashes, got '$IdentityName'"
}
if ($Publisher -notmatch '^CN=') {
	throw "Publisher must be a distinguished name starting with CN=, got '$Publisher'"
}

function Find-SdkTool([string]$name) {
	$onPath = Get-Command $name -ErrorAction SilentlyContinue
	if ($onPath) { return $onPath.Source }
	$kits = Join-Path ${env:ProgramFiles(x86)} 'Windows Kits\10\bin'
	$found = Get-ChildItem -Path $kits -Directory -ErrorAction SilentlyContinue |
		Where-Object { $_.Name -match '^\d+\.\d+\.\d+\.\d+$' } |
		Sort-Object { [version]$_.Name } -Descending |
		ForEach-Object { Join-Path $_.FullName "x64\$name" } |
		Where-Object { Test-Path -LiteralPath $_ } |
		Select-Object -First 1
	if (-not $found) { throw "$name not found: install the Windows SDK" }
	return $found
}

function Escape-Xml([string]$value) {
	return [System.Security.SecurityElement]::Escape($value)
}

$makeAppx = Find-SdkTool 'makeappx.exe'

$stage = Join-Path ([System.IO.Path]::GetTempPath()) ("vukem-msix-" + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $stage | Out-Null
try {
	Copy-Item -LiteralPath $Exe -Destination (Join-Path $stage 'vukem-finances.exe')
	Copy-Item -LiteralPath (Join-Path $here 'Assets') -Destination (Join-Path $stage 'Assets') -Recurse

	$manifest = [System.IO.File]::ReadAllText((Join-Path $here 'AppxManifest.template.xml'))
	$tokens = @{
		'@@IDENTITY_NAME@@'          = $IdentityName
		'@@PUBLISHER@@'              = $Publisher
		'@@VERSION@@'                = $packageVersion
		'@@DISPLAY_NAME@@'           = $DisplayName
		'@@PUBLISHER_DISPLAY_NAME@@' = $PublisherDisplayName
	}
	foreach ($token in $tokens.Keys) {
		$manifest = $manifest.Replace($token, (Escape-Xml $tokens[$token]))
	}
	if ($manifest -match '@@[A-Z_]+@@') { throw "unreplaced token in the manifest: $($Matches[0])" }
	[System.IO.File]::WriteAllText((Join-Path $stage 'AppxManifest.xml'), $manifest, (New-Object System.Text.UTF8Encoding($false)))

	$outFull = [System.IO.Path]::GetFullPath($Out)
	New-Item -ItemType Directory -Force -Path (Split-Path -Parent $outFull) | Out-Null

	& $makeAppx pack /o /d $stage /p $outFull
	if ($LASTEXITCODE -ne 0) { throw "makeappx failed with exit code $LASTEXITCODE" }
} finally {
	Remove-Item -LiteralPath $stage -Recurse -Force -ErrorAction SilentlyContinue
}

Write-Host "msix: $outFull (identity $IdentityName, version $packageVersion)"
