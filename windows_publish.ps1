param (
  [int]$major = 1,
  [int]$minor = 0,
  [int]$patch = $(throw "-patch is required"),
  [string]$basePath = "infrastructure_agent",
  [string]$arch = "amd64",
  [string]$name = (get-item $PSScriptRoot).parent.Name,
  [switch]$testing = $false
)

if ($arch -ne "386" -And $arch -ne "amd64") {
  throw "-arch can only be 386 or amd64."
}

if ($arch -eq "386" -And $testing -eq $true) {
  throw "386 arch and testing flag are not supported."
}

if (-not (Test-Path env:GOPATH)) {
  throw "GOPATH not defined."
}

if (-not (Test-Path env:AWSBucketName) -Or -not (Test-Path env:AWSAccessKey) -Or -not (Test-Path env:AWSSecretKey)) {
  throw "AWS variables not defined."
}

$pkg = Join-Path -Path $env:GOPATH -ChildPath "src\github.com\newrelic\$name\pkg\windows\$name-$arch-installer\bin\Release\$name-$arch.$major.$minor.$patch.msi"
if (-not (Test-Path $pkg)) {
  throw "Integration package not found: $pkg"
}

if ($arch -eq "386") {
  $integrationVersion = "$basePath/windows/386/$name-386.$major.$minor.$patch.msi"
  $integration = "$basePath/windows/386/$name-386.msi"
} else {
  if ($testing -eq $true) {
    $basePath += "/test"
  }
  $integrationVersion = "$basePath/windows/$name.$major.$minor.$patch.msi"
  $integration = "$basePath/windows/$name.msi"
}

Write-S3Object -BucketName $env:AWSBucketName -File $pkg -Key $integrationVersion -CannedACLName Private -AccessKey $env:AWSAccessKey -SecretKey $env:AWSSecretKey
# in test we only publish packages with versioning
if ($testing -eq $false) {
  Write-S3Object -BucketName $env:AWSBucketName -File $pkg -Key $integration -CannedACLName Private -AccessKey $env:AWSAccessKey -SecretKey $env:AWSSecretKey
}