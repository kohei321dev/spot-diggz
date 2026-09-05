[CmdletBinding()]
param(
    [switch]$Deploy,
    [switch]$PreflightOnly,
    [string]$BaseURL = "https://spotdiggz.vercel.app",
    [string]$GitHubOwner = "kohei321dev"
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

function Invoke-VercelWhoAmI {
    $previousErrorActionPreference = $ErrorActionPreference
    try {
        $ErrorActionPreference = "Continue"
        $output = & $script:VercelCommand whoami --no-color 2>&1 | Out-String
        $exitCode = $LASTEXITCODE
    }
    finally {
        $ErrorActionPreference = $previousErrorActionPreference
    }
    if ($exitCode -ne 0 -or [string]::IsNullOrWhiteSpace($output)) {
        throw "Vercel CLI is not authenticated. Complete vercel login and retry."
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

function New-AuthenticationSecret {
    $bytes = New-Object byte[] 48
    $generator = [Security.Cryptography.RandomNumberGenerator]::Create()
    try {
        $generator.GetBytes($bytes)
        return [Convert]::ToBase64String($bytes)
    }
    finally {
        $generator.Dispose()
        [Array]::Clear($bytes, 0, $bytes.Length)
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

$parsedBaseURL = $null
if (-not [Uri]::TryCreate($BaseURL, [UriKind]::Absolute, [ref]$parsedBaseURL) -or
    $parsedBaseURL.Scheme -ne "https" -or
    $parsedBaseURL.AbsolutePath -ne "/" -or
    -not [string]::IsNullOrEmpty($parsedBaseURL.Query) -or
    -not [string]::IsNullOrEmpty($parsedBaseURL.Fragment) -or
    -not [string]::IsNullOrEmpty($parsedBaseURL.UserInfo)) {
    throw "BaseURL must be an HTTPS origin without credentials, path, query, or fragment."
}
$BaseURL = $BaseURL.TrimEnd("/")
if ($GitHubOwner -notmatch "^[A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?$") {
    throw "GitHubOwner has an invalid format."
}

Invoke-VercelWhoAmI
if ($PreflightOnly) {
    Write-Output "github-vercel-preflight=PASS project=spotdiggz callback=$BaseURL/auth/github/callback"
    return
}

$clientID = (Read-Host "GitHub OAuth Client ID").Trim()
$clientSecret = Read-PrivateValue "GitHub OAuth Client Secret"
$authenticationSecret = New-AuthenticationSecret
try {
    if ($clientID -notmatch "^[A-Za-z0-9._-]{10,128}$") {
        throw "GitHub OAuth Client ID has an invalid format."
    }
    if ([string]::IsNullOrWhiteSpace($clientSecret) -or $clientSecret.Length -lt 20 -or $clientSecret -match "\s") {
        throw "GitHub OAuth Client Secret has an invalid format."
    }

    Push-Location $repositoryRoot
    try {
        Set-ProductionVariable "APP_BASE_URL" $BaseURL $false
        Set-ProductionVariable "GITHUB_OWNER" $GitHubOwner $false
        Set-ProductionVariable "GITHUB_CLIENT_ID" $clientID $false
        Set-ProductionVariable "GITHUB_CLIENT_SECRET" $clientSecret $true
        Set-ProductionVariable "AUTH_SECRET" $authenticationSecret $true
        Write-Output "Configured GitHub owner authentication for Vercel Production. Existing owner sessions were invalidated."

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

            $healthResponse = Invoke-WebRequest "$BaseURL/healthz" -UseBasicParsing
            $readyResponse = Invoke-WebRequest "$BaseURL/readyz" -UseBasicParsing
            if ($healthResponse.StatusCode -ne 200 -or $readyResponse.StatusCode -ne 200) {
                throw "Production health or readiness verification failed."
            }
            Write-Output "Verified Production deployment health and readiness."
        }
    }
    finally {
        Pop-Location
    }
}
finally {
    $clientID = $null
    $clientSecret = $null
    $authenticationSecret = $null
}
