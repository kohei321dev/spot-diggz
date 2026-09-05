[CmdletBinding()]
param(
    [ValidatePattern("^A[A-Z0-9]{8,}$")]
    [string]$AppId,

    [ValidatePattern("^T[A-Z0-9]{8,}$")]
    [string]$TeamId
)

$ErrorActionPreference = "Stop"
$repositoryRoot = Split-Path $PSScriptRoot
$slackExecutable = Join-Path $env:LOCALAPPDATA "slack-cli\bin\slack.exe"

function Read-PrivateValue {
    param([Parameter(Mandatory)][string]$Prompt)

    $secureValue = Read-Host $Prompt -AsSecureString
    $valuePointer = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($secureValue)
    try {
        return [Runtime.InteropServices.Marshal]::PtrToStringBSTR($valuePointer)
    }
    finally {
        [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($valuePointer)
    }
}

if (-not (Test-Path -LiteralPath $slackExecutable)) {
    throw "Slack CLI was not found. Run scripts/install-integration-clis.ps1 first."
}

$authOutput = & $slackExecutable auth list --no-color 2>&1 | Out-String
if ($LASTEXITCODE -ne 0 -or $authOutput -match "not logged in") {
    throw "Slack CLI is not authenticated. Complete slack login and retry."
}

if ([string]::IsNullOrWhiteSpace($TeamId)) {
    $TeamId = Read-PrivateValue "Slack Workspace ID"
}
if ($TeamId -notmatch "^T[A-Z0-9]{8,}$") {
    throw "Slack Workspace ID has an invalid format."
}

Push-Location $repositoryRoot
try {
    & $slackExecutable manifest validate --team $TeamId --no-color
    if ($LASTEXITCODE -ne 0) {
        throw "Slack App Manifest validation failed."
    }

    if ([string]::IsNullOrWhiteSpace($AppId)) {
        & $slackExecutable app link --team $TeamId --environment deployed --force --no-color
        $appSelector = "deployed"
    }
    else {
        & $slackExecutable app link --team $TeamId --app $AppId --environment deployed --force --no-color
        $appSelector = $AppId
    }
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to link the existing Slack app."
    }

    & $slackExecutable app install --team $TeamId --app $appSelector --environment deployed --force --no-color
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to apply the manifest or install the Slack app."
    }

    Write-Output "Validated and applied the Slack App Manifest."
}
finally {
    $AppId = $null
    $TeamId = $null
    Pop-Location
}
