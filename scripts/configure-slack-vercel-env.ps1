[CmdletBinding()]
param(
    [switch]$Deploy,
    [switch]$PreflightOnly
)

$ErrorActionPreference = "Stop"
$repositoryRoot = Split-Path $PSScriptRoot

function Resolve-NodeExecutable {
    $command = Get-Command node.exe -ErrorAction SilentlyContinue
    if ($command) {
        return $command.Source
    }

    $candidates = @(
        (Join-Path $env:ProgramFiles "nodejs\node.exe"),
        (Join-Path $env:LOCALAPPDATA "Programs\nodejs\node.exe")
    )
    $zedNodeRoot = Join-Path $env:LOCALAPPDATA "Zed\node"
    if (Test-Path -LiteralPath $zedNodeRoot) {
        $candidates += Get-ChildItem -LiteralPath $zedNodeRoot -Directory |
            Sort-Object LastWriteTime -Descending |
            ForEach-Object { Join-Path $_.FullName "node.exe" }
    }
    foreach ($candidate in $candidates) {
        if (Test-Path -LiteralPath $candidate) {
            return $candidate
        }
    }
    throw "Node.js was not found. Run scripts/install-integration-clis.ps1 first."
}

function Add-NodeToProcessPath {
    $nodeExecutable = Resolve-NodeExecutable
    $nodeDirectory = Split-Path $nodeExecutable
    $pathEntries = @($env:PATH -split ";" | Where-Object { $_ })
    $isAlreadyAvailable = [bool]($pathEntries | Where-Object {
        [System.StringComparer]::OrdinalIgnoreCase.Equals($_, $nodeDirectory)
    })
    if (-not $isAlreadyAvailable) {
        $env:PATH = $nodeDirectory + ";" + $env:PATH
    }
}

function Resolve-VercelCommand {
    $command = Get-Command vercel.cmd -ErrorAction SilentlyContinue
    if ($command) {
        return $command.Source
    }

    $nodeExecutable = Resolve-NodeExecutable
    $npmCommand = Join-Path (Split-Path $nodeExecutable) "npm.cmd"
    $npmPrefix = (& $npmCommand prefix --global).Trim()
    $candidate = Join-Path $npmPrefix "vercel.cmd"
    if (Test-Path -LiteralPath $candidate) {
        return $candidate
    }

    throw "Vercel CLI was not found. Run scripts/install-integration-clis.ps1 first."
}

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

function Set-ProductionVariable {
    param(
        [Parameter(Mandatory)][string]$Name,
        [Parameter(Mandatory)][string]$Value,
        [Parameter(Mandatory)][bool]$IsSensitive
    )

    $sensitivityFlag = if ($IsSensitive) { "--sensitive" } else { "--no-sensitive" }
    $previousErrorActionPreference = $ErrorActionPreference
    try {
        $ErrorActionPreference = "Continue"
        $Value | & $script:VercelCommand env add $Name production --force --yes $sensitivityFlag --no-color
        $exitCode = $LASTEXITCODE
    }
    finally {
        $ErrorActionPreference = $previousErrorActionPreference
    }
    if ($exitCode -ne 0) {
        throw "Failed to configure the Production variable: $Name"
    }
}

Add-NodeToProcessPath
$script:VercelCommand = Resolve-VercelCommand
$projectConfigPath = Join-Path $repositoryRoot ".vercel\project.json"
if (-not (Test-Path -LiteralPath $projectConfigPath)) {
    throw "The Vercel project is not linked. Run vercel link from the repository root."
}

$projectConfig = Get-Content -Raw -Encoding UTF8 $projectConfigPath | ConvertFrom-Json
if ($projectConfig.projectName -ne "spotdiggz") {
    throw "The linked Vercel project is not spotdiggz."
}

$previousErrorActionPreference = $ErrorActionPreference
try {
    $ErrorActionPreference = "Continue"
    $whoAmIOutput = & $script:VercelCommand whoami --no-color 2>&1 | Out-String
    $whoAmIExitCode = $LASTEXITCODE
}
finally {
    $ErrorActionPreference = $previousErrorActionPreference
}
if ($whoAmIExitCode -ne 0 -or [string]::IsNullOrWhiteSpace($whoAmIOutput)) {
    throw "Vercel CLI is not authenticated. Complete vercel login and retry."
}
if ($PreflightOnly) {
    Write-Output "vercel-production-preflight=PASS project=spotdiggz"
    return
}

$botToken = Read-PrivateValue "Slack Bot Token"
$signingSecret = Read-PrivateValue "Slack Signing Secret"
$teamId = Read-PrivateValue "Slack Workspace ID"
$ownerUserId = Read-PrivateValue "Slack Owner User ID"

if ($botToken -notmatch "^xoxb-[A-Za-z0-9-]{20,}$") {
    throw "Slack Bot Token has an invalid format."
}
if ($signingSecret -notmatch "^[A-Fa-f0-9]{32}$") {
    throw "Slack Signing Secret has an invalid format."
}
if ($teamId -notmatch "^T[A-Z0-9]{8,}$") {
    throw "Slack Workspace ID has an invalid format."
}
if ($ownerUserId -notmatch "^U[A-Z0-9]{8,}$") {
    throw "Slack Owner User ID has an invalid format."
}

Push-Location $repositoryRoot
try {
    $authHeaders = @{ Authorization = "Bearer $botToken" }
    $authResponse = Invoke-RestMethod "https://slack.com/api/auth.test" -Method Post -Headers $authHeaders
    if (-not $authResponse.ok -or $authResponse.team_id -ne $teamId) {
        throw "Slack Bot Token does not belong to the configured workspace."
    }
    Set-ProductionVariable "SLACK_BOT_TOKEN" $botToken $true
    Set-ProductionVariable "SLACK_SIGNING_SECRET" $signingSecret $true
    Set-ProductionVariable "SLACK_TEAM_ID" $teamId $true
    Set-ProductionVariable "SLACK_OWNER_USER_ID" $ownerUserId $true
    Write-Output "Configured Vercel Production variables for the Slack integration."

    if ($Deploy) {
        $previousErrorActionPreference = $ErrorActionPreference
        try {
            $ErrorActionPreference = "Continue"
            & $script:VercelCommand --prod --yes --no-color
            $deployExitCode = $LASTEXITCODE
        }
        finally {
            $ErrorActionPreference = $previousErrorActionPreference
        }
        if ($deployExitCode -ne 0) {
            throw "Vercel Production deployment failed."
        }

        $healthResponse = Invoke-WebRequest "https://spotdiggz.vercel.app/healthz" -UseBasicParsing
        $readyResponse = Invoke-WebRequest "https://spotdiggz.vercel.app/readyz" -UseBasicParsing
        if ($healthResponse.StatusCode -ne 200 -or $readyResponse.StatusCode -ne 200) {
            throw "Production health or readiness verification failed."
        }
        Write-Output "Verified Production deployment health and readiness."
    }
}
finally {
    $botToken = $null
    $signingSecret = $null
    $teamId = $null
    $ownerUserId = $null
    Pop-Location
}
