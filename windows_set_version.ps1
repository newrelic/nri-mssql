param (
	 [int]$major = $(throw "-major is required"),
	 [int]$minor = $(throw "-minor is required"),
	 [int]$patch = $(throw "-patch is required"),
	 [int]$build = 0
)

if (-not (Test-Path env:GOPATH)) {
	Write-Error "GOPATH not defined."
}
$projectRootPath = Join-Path -Path $env:GOPATH -ChildPath "src\github.com\newrelic\nri-mssql"

$versionInfoPath = Join-Path -Path $projectRootPath -ChildPath "pkg\windows\versioninfo.json"
if ((Test-Path "$versionInfoPath.template" -PathType Leaf) -eq $False) {
	Write-Error "$versionInfoPath.template not found."
}
Copy-Item -Path "$versionInfoPath.template" -Destination $versionInfoPath -Force

$versionInfo = Get-Content -Path $versionInfoPath -Encoding UTF8
$versionInfo = $versionInfo -replace "{MajorVersion}", $major
$versionInfo = $versionInfo -replace "{MinorVersion}", $minor
$versionInfo = $versionInfo -replace "{PatchVersion}", $patch
$versionInfo = $versionInfo -replace "{BuildVersion}", $build
Set-Content -Path $versionInfoPath -Value $versionInfo

#$wix386Path = Join-Path -Path projectRootPath -ChildPath "pkg\windows\nri-mssql-386-installer\Product.wxs"
$wixAmd64Path = Join-Path -Path $projectRootPath -ChildPath "pkg\windows\nri-mssql-amd64-installer\Product.wxs"

Function ProcessProductFile($productPath) {
	if ((Test-Path "$productPath.template" -PathType Leaf) -eq $False) {
		Write-Error "$productPath.template not found."
	}
	Copy-Item -Path "$productPath.template" -Destination $productPath -Force

	$product = Get-Content -Path $productPath -Encoding UTF8
	$product = $product -replace "{IntegrationVersion}", "$major.$minor.$patch"
	Set-Content -Value $product -Path $productPath
}

#ProcessProductFile($wix386Path)
ProcessProductFile($wixAmd64Path)
