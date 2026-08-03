[CmdletBinding()]
param(
    [string]$SlackCliVersion = "4.6.0",
    [string]$VercelCliVersion = "58.4.4"
)

$ErrorActionPreference = "Stop"

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
    throw "Node.js was not found. Install Node.js 22 or later and retry."
}

function Add-UserPathEntry {
    param([Parameter(Mandatory)][string]$PathEntry)

    $resolvedEntry = [System.IO.Path]::GetFullPath($PathEntry)
    $userPath = [System.Environment]::GetEnvironmentVariable("Path", "User")
    $entries = @($userPath -split ";" | Where-Object { $_ })
    if ($entries | Where-Object { [System.StringComparer]::OrdinalIgnoreCase.Equals($_, $resolvedEntry) }) {
        return
    }

    $updatedPath = (@($entries) + $resolvedEntry) -join ";"
    [System.Environment]::SetEnvironmentVariable("Path", $updatedPath, "User")
}

$slackExecutable = Join-Path $env:LOCALAPPDATA "slack-cli\bin\slack.exe"
if (-not (Test-Path -LiteralPath $slackExecutable)) {
    $installerPath = Join-Path ([System.IO.Path]::GetTempPath()) "spot-diggz-slack-cli-installer.ps1"
    try {
        Invoke-WebRequest "https://downloads.slack-edge.com/slack-cli/install-windows.ps1" -OutFile $installerPath
        & $installerPath -Version $SlackCliVersion -SkipGit $true
    }
    finally {
        Remove-Item -LiteralPath $installerPath -Force -ErrorAction SilentlyContinue
    }
}

$nodeExecutable = Resolve-NodeExecutable
$nodeDirectory = Split-Path $nodeExecutable
$pathEntries = @($env:PATH -split ";" | Where-Object { $_ })
if (-not [bool]($pathEntries | Where-Object {
    [System.StringComparer]::OrdinalIgnoreCase.Equals($_, $nodeDirectory)
})) {
    $env:PATH = $nodeDirectory + ";" + $env:PATH
}
$npmCommand = Join-Path $nodeDirectory "npm.cmd"
if (-not (Test-Path -LiteralPath $npmCommand)) {
    throw "npm.cmd was not found. Check the Node.js installation."
}

& $npmCommand install --global "vercel@$VercelCliVersion"
if ($LASTEXITCODE -ne 0) {
    throw "Failed to install Vercel CLI."
}

$npmPrefix = (& $npmCommand prefix --global).Trim()
$vercelCommand = Join-Path $npmPrefix "vercel.cmd"
if (-not (Test-Path -LiteralPath $vercelCommand)) {
    throw "vercel.cmd was not found in the npm global prefix."
}

Add-UserPathEntry (Split-Path $slackExecutable)
Add-UserPathEntry $npmPrefix

& $slackExecutable version
& $vercelCommand --version
