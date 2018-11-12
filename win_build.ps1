<#
    .SYNOPSIS
        This script verifies, tests, builds and packages the New Relic Infrastructure Agent
#>
param (
    # Target architecture: amd64 (default) or 386
    [ValidateSet("amd64", "386")]
    [string]$arch="amd64",
    # Build number (to be attached to the agent version)
    [string]$buildNum="dev",
    # Creates a signed installer
    [switch]$installer=$false,
    # Skip tests
    [switch]$skipTests=$false
)

echo "--- Configuring version for artifacts"

# If the build number is numeric
if ([System.Int32]::TryParse($buildNum, [ref]0)) {
    .\windows_set_version.ps1 -patch $buildNum
} else {
    echo " - setting 1.0.0 as development version"
    .\windows_set_version.ps1 -patch 0
}

echo "--- Checking dependencies"

echo "Checking Go..."
go version
if (-not $?)
{
    echo "Can't find Go"
    exit -1
}

echo "Checking MSBuild.exe..."
$msBuild = (Get-ItemProperty hklm:\software\Microsoft\MSBuild\ToolsVersions\4.0).MSBuildToolsPath
if ($msBuild.Length -eq 0) {
    echo "Can't find MSBuild tool. .NET Framework 4.0.x must be installed"
    exit -1
}
echo $msBuild

$env:GOOS="windows"
$env:GOARCH=$arch

echo "--- Collecting files"

$goFiles = go list ./...

echo "--- Format check"

$wrongFormat = go fmt $goFiles

if ($wrongFormat -and ($wrongFormat.Length -gt 0))
{
    echo "ERROR: Wrong format for files:"
    echo $wrongFormat
    exit -1
}

if (-Not $skipTests) {
    echo "--- Running tests"

    go test $goFiles
    if (-not $?)
    {
        echo "Failed running tests"
        exit -1
    }    
}

echo "--- Running Build"

go build -v $goFiles
if (-not $?)
{
    echo "Failed building files"
    exit -1
}

echo "--- Collecting Go main files"

$packages = go list -f "{{.ImportPath}} {{.Name}}" ./...  | ConvertFrom-String -PropertyNames Path, Name
$goMains = $packages | ? { $_.Name -eq 'main' } | % { $_.Path }

Foreach ($pkg in $goMains)
{
    echo "generating $pkg"
    go generate $pkg
}

echo "--- Running Full Build"

Foreach ($pkg in $goMains)
{
    $fileName = ([io.fileinfo]$pkg).BaseName
    echo "creating $fileName"
    go build -ldflags "-X main.buildVersion=1.0.$buildNum" -o ".\target\bin\windows_$arch\$fileName.exe" $pkg
}

If (-Not $installer) {
    exit 0
}

echo "--- Building Installer"

Push-Location -Path "pkg\windows\newrelic-infra-$arch-installer\newrelic-infra"

. $msBuild/MSBuild.exe newrelic-infra-installer.wixproj

if (-not $?)
{
    echo "Failed building installer"
    Pop-Location
    exit -1
}

echo "Making versioned installed copy"

cd bin\Release

cp newrelic-infra-$arch.msi newrelic-infra-$arch.1.0.$buildNum.msi
cp newrelic-infra-$arch.msi newrelic-infra.msi

Pop-Location
