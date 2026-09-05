[CmdletBinding()]
param()

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

$nodeExecutable = Resolve-NodeExecutable
$nodeDirectory = Split-Path $nodeExecutable
$pathEntries = @($env:PATH -split ";" | Where-Object { $_ })
if (-not [bool]($pathEntries | Where-Object {
    [System.StringComparer]::OrdinalIgnoreCase.Equals($_, $nodeDirectory)
})) {
    $env:PATH = $nodeDirectory + ";" + $env:PATH
}

function Assert-PathExists {
    param([Parameter(Mandatory)][string]$LiteralPath)

    if (-not (Test-Path -LiteralPath $LiteralPath)) {
        throw "Required path was not found: $LiteralPath"
    }
}

function Resolve-VercelCommand {
    $command = Get-Command vercel.cmd -ErrorAction SilentlyContinue
    if ($command) {
        return $command.Source
    }

    $npmCommand = Join-Path (Split-Path $script:NodeExecutable) "npm.cmd"
    Assert-PathExists $npmCommand
    $npmPrefix = (& $npmCommand prefix --global).Trim()
    $candidate = Join-Path $npmPrefix "vercel.cmd"
    if (Test-Path -LiteralPath $candidate) {
        return $candidate
    }
    throw "Vercel CLI was not found. Run scripts/install-integration-clis.ps1 first."
}

function Test-UserPathEntry {
    param([Parameter(Mandatory)][string]$ExpectedEntry)

    $userPath = [System.Environment]::GetEnvironmentVariable("Path", "User")
    $entries = @($userPath -split ";" | Where-Object { $_ })
    return [bool]($entries | Where-Object {
        [System.StringComparer]::OrdinalIgnoreCase.Equals($_, $ExpectedEntry)
    })
}

$slackExecutable = Join-Path $env:LOCALAPPDATA "slack-cli\bin\slack.exe"
Assert-PathExists $slackExecutable

$script:NodeExecutable = $nodeExecutable
$vercelCommand = Resolve-VercelCommand
Assert-PathExists $vercelCommand

if (-not (Test-UserPathEntry (Split-Path $slackExecutable))) {
    throw "Slack CLI directory is missing from the user PATH."
}
$requiredRepositoryPaths = @(
    ".slack\config.json",
    ".slack\hooks.json",
    "slack-manifest.json",
    "scripts\slack-manifest-hook.mjs",
    "scripts\configure-slack-app.ps1",
    "scripts\configure-slack-vercel-env.ps1",
    "scripts\configure-github-vercel-env.ps1"
)
foreach ($relativePath in $requiredRepositoryPaths) {
    Assert-PathExists (Join-Path $repositoryRoot $relativePath)
}

$manifestPath = Join-Path $repositoryRoot "slack-manifest.json"
$manifest = Get-Content -Raw -Encoding UTF8 $manifestPath | ConvertFrom-Json
$slashCommands = @($manifest.features.slash_commands)
$botScopes = @($manifest.oauth_config.scopes.bot)
if ($manifest.settings.socket_mode_enabled -ne $false) {
    throw "Slack Manifest must disable Socket Mode."
}
if ($slashCommands.Count -ne 1 -or $slashCommands[0].command -ne "/spotdiggz") {
    throw "Slack Manifest must define only the /spotdiggz command."
}
if ($slashCommands[0].url -ne "https://spotdiggz.vercel.app/integrations/slack/commands") {
    throw "Slack Manifest contains an unexpected command URL."
}
if ($botScopes.Count -ne 2 -or $botScopes[0] -ne "commands" -or $botScopes[1] -ne "chat:write") {
    throw "Slack Manifest must request only the commands and chat:write bot scopes."
}
if ($manifest.settings.interactivity.is_enabled -ne $true) {
    throw "Slack Manifest must enable interactivity."
}
if ($manifest.settings.interactivity.request_url -ne "https://spotdiggz.vercel.app/integrations/slack/commands") {
    throw "Slack Manifest contains an unexpected interactivity URL."
}

$hookScript = Join-Path $repositoryRoot "scripts\slack-manifest-hook.mjs"
& $nodeExecutable $hookScript | Out-Null
if ($LASTEXITCODE -ne 0) {
    throw "Slack Manifest hook execution failed."
}

$vercelProjectPath = Join-Path $repositoryRoot ".vercel\project.json"
Assert-PathExists $vercelProjectPath
$vercelProject = Get-Content -Raw -Encoding UTF8 $vercelProjectPath | ConvertFrom-Json
if ($vercelProject.projectName -ne "spotdiggz") {
    throw "The linked Vercel project is not spotdiggz."
}

& $slackExecutable version
if ($LASTEXITCODE -ne 0) {
    throw "Slack CLI version check failed."
}
& $vercelCommand --version
if ($LASTEXITCODE -ne 0) {
    throw "Vercel CLI version check failed."
}

Write-Output "local-integration-prerequisites=PASS"
