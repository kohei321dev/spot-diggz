[CmdletBinding(SupportsShouldProcess = $true, ConfirmImpact = 'Medium')]
param(
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^[0-9]+$')]
    [string]$ApplicationId,

    [Parameter(Mandatory = $true)]
    [ValidatePattern('^[0-9]+$')]
    [string]$GuildId
)

$endpoint = "https://discord.com/api/v10/applications/$ApplicationId/guilds/$GuildId/commands"
if (-not $PSCmdlet.ShouldProcess($endpoint, 'Register or update the /spotdiggz guild command')) {
    return
}

$secureToken = Read-Host 'Discord setup Bot Token' -AsSecureString
$credential = [System.Management.Automation.PSCredential]::new('bot', $secureToken)
$setupBotToken = $credential.GetNetworkCredential().Password

try {
    $payload = @{
        name        = 'spotdiggz'
        type        = 1
        description = 'Find recommended skate spots'
    } | ConvertTo-Json -Compress

    $command = Invoke-RestMethod `
        -Method Post `
        -Uri $endpoint `
        -Headers @{ Authorization = "Bot $setupBotToken" } `
        -ContentType 'application/json' `
        -Body $payload

    Write-Output "Registered Discord command: $($command.name)"
}
finally {
    $setupBotToken = $null
    $credential = $null
    $secureToken = $null
}
