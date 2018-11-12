param (
	 [int]$major = 1,
	 [int]$minor = 0,
	 [int]$patch = $(throw "-patch is required"),
	 [int]$build = 0
)

if (-not (Test-Path env:GOPATH)) {
	Write-Error "GOPATH not defined."
}
$agentPath = Join-Path -Path $env:GOPATH -ChildPath "src\go.datanerd.us\p\meatballs\infra-agent"

$versionInfoPath = Join-Path -Path $agentPath -ChildPath "main\newrelic-infra\versioninfo.json"
if ((Test-Path "$versionInfoPath.template" -PathType Leaf) -eq $False) {
	Write-Error "$versionInfoPath.template not found."
}
Copy-Item -Path "$versionInfoPath.template" -Destination $versionInfoPath -Force

$versionInfo = Get-Content -Path $versionInfoPath -Encoding UTF8
$versionInfo = $versionInfo -replace "{AgentMajorVersion}", $major
$versionInfo = $versionInfo -replace "{AgentMinorVersion}", $minor
$versionInfo = $versionInfo -replace "{AgentPatchVersion}", $patch
$versionInfo = $versionInfo -replace "{AgentBuildVersion}", $build
Set-Content -Path $versionInfoPath -Value $versionInfo

$infra386Path = Join-Path -Path $agentPath -ChildPath "pkg\windows\newrelic-infra-386-installer\newrelic-infra\Product.wxs"
$infra386StarbucksPath = Join-Path -Path $agentPath -ChildPath "pkg\windows\newrelic-infra-386-installer-starbucks\newrelic-infra\Product.wxs"
$infraAmd64Path = Join-Path -Path $agentPath -ChildPath "pkg\windows\newrelic-infra-amd64-installer\newrelic-infra\Product.wxs"
$infraAmd64StarbucksPath = Join-Path -Path $agentPath -ChildPath "pkg\windows\newrelic-infra-amd64-installer-starbucks\newrelic-infra\Product.wxs"

Function ProcessProductFile($productPath) {
	if ((Test-Path "$productPath.template" -PathType Leaf) -eq $False) {
		Write-Error "$productPath.template not found."
	}
	Copy-Item -Path "$productPath.template" -Destination $productPath -Force

	$product = Get-Content -Path $productPath -Encoding UTF8
	$product = $product -replace "{AgentVersion}", "$major.$minor.$patch"
	Set-Content -Value $product -Path $productPath
}

ProcessProductFile($infra386Path)
ProcessProductFile($infra386StarbucksPath)
ProcessProductFile($infraAmd64Path)
ProcessProductFile($infraAmd64StarbucksPath)
