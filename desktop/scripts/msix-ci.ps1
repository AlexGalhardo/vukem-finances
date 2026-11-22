<#
CI checks of the MSIX on a hosted Windows runner (elevated). Needs no secrets.
Each step is run by the Desktop workflow on its own:

  Sign     creates a throwaway self-signed certificate (private key only in the
           current user's store, never written to disk), signs the package with
           it and trusts its public part so the package can be installed
  Smoke    installs the package, runs the headless smoke test on the installed
           executable, then uninstalls
  Cert     runs the Windows App Certification Kit when it is installed
  Cleanup  removes the throwaway certificate from the stores

The certificate is only good for this job: it must never be committed or
published, and nothing here writes it to the artifact folder.
#>
param(
	[Parameter(Mandatory)][ValidateSet('Sign', 'Smoke', 'Cert', 'Cleanup')][string]$Step,
	[string]$Package = '',
	[string]$Publisher = 'CN=VukemFinancesCI',
	[string]$IdentityName = 'VukemFinances.CI',
	[string]$ReportDir = '',
	[switch]$SkipTrust
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$friendlyName = 'vukem-finances CI (throwaway)'
$tmp = if ($env:RUNNER_TEMP) { $env:RUNNER_TEMP } else { [System.IO.Path]::GetTempPath() }
$cerPath = Join-Path $tmp 'vukem-ci.cer'

function Test-Admin {
	$id = [Security.Principal.WindowsIdentity]::GetCurrent()
	return ([Security.Principal.WindowsPrincipal]::new($id)).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Find-SdkTool([string]$name) {
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

function Get-CiCerts {
	return @(Get-ChildItem Cert:\CurrentUser\My | Where-Object { $_.FriendlyName -eq $friendlyName })
}

switch ($Step) {
	'Sign' {
		$cert = New-SelfSignedCertificate -Type Custom -Subject $Publisher `
			-KeyUsage DigitalSignature -FriendlyName $friendlyName `
			-CertStoreLocation 'Cert:\CurrentUser\My' -NotAfter (Get-Date).AddDays(1) `
			-TextExtension @('2.5.29.37={text}1.3.6.1.5.5.7.3.3', '2.5.29.19={text}')
		Export-Certificate -Cert $cert -FilePath $cerPath | Out-Null

		$signtool = Find-SdkTool 'signtool.exe'
		& $signtool sign /fd SHA256 /sha1 $cert.Thumbprint /s My $Package
		if ($LASTEXITCODE -ne 0) { throw "signtool failed with exit code $LASTEXITCODE" }

		if ($SkipTrust) {
			Write-Host 'sign: trust skipped (-SkipTrust), the package will not install'
		} else {
			if (-not (Test-Admin)) { throw 'trusting the certificate needs an elevated shell' }
			Import-Certificate -FilePath $cerPath -CertStoreLocation 'Cert:\LocalMachine\TrustedPeople' | Out-Null
		}
		Write-Host "sign: signed $Package with a throwaway certificate ($($cert.Thumbprint))"
	}

	'Smoke' {
		Add-AppxPackage -Path $Package
		try {
			$pkg = Get-AppxPackage -Name $IdentityName
			if (-not $pkg) { throw "package $IdentityName is not installed" }
			Write-Host "smoke: installed $($pkg.PackageFullName) at $($pkg.InstallLocation)"

			$exe = Join-Path $pkg.InstallLocation 'vukem-finances.exe'
			if (-not (Test-Path -LiteralPath $exe)) { throw "the installed package has no $exe" }

			$smoke = (Join-Path $PSScriptRoot 'smoke.sh').Replace('\', '/')
			& bash $smoke ($exe.Replace('\', '/'))
			if ($LASTEXITCODE -ne 0) { throw "the smoke test failed with exit code $LASTEXITCODE" }
		} finally {
			Get-AppxPackage -Name $IdentityName | Remove-AppxPackage
		}
		if (Get-AppxPackage -Name $IdentityName) { throw 'the package is still installed after the uninstall' }
		Write-Host 'smoke: uninstalled'
	}

	'Cert' {
		$appcert = Join-Path ${env:ProgramFiles(x86)} 'Windows Kits\10\App Certification Kit\appcert.exe'
		if (-not (Test-Path -LiteralPath $appcert)) {
			Write-Host '::warning::appcert.exe is not installed on this runner, the certification kit was skipped'
			return
		}
		if (-not $ReportDir) { $ReportDir = $tmp }
		# appcert refuses a relative report path.
		$ReportDir = (New-Item -ItemType Directory -Force -Path $ReportDir).FullName
		$report = Join-Path $ReportDir 'appcert-report.xml'
		& $appcert reset
		& $appcert test -appxpackagepath $Package -reportoutputpath $report
		$code = $LASTEXITCODE
		if (Test-Path -LiteralPath $report) {
			$overall = Select-String -Path $report -Pattern 'OVERALL_RESULT="(\w+)"' | Select-Object -First 1
			Write-Host "cert: report at $report ($($overall.Matches[0].Value))"
		}
		if ($code -ne 0) { throw "appcert.exe exited with code $code" }
	}

	'Cleanup' {
		foreach ($cert in Get-CiCerts) {
			Remove-Item -LiteralPath "Cert:\CurrentUser\My\$($cert.Thumbprint)" -Force
			$trusted = "Cert:\LocalMachine\TrustedPeople\$($cert.Thumbprint)"
			if (Test-Path -LiteralPath $trusted) { Remove-Item -LiteralPath $trusted -Force }
		}
		Remove-Item -LiteralPath $cerPath -Force -ErrorAction SilentlyContinue
		Write-Host 'cleanup: throwaway certificate removed'
	}
}
