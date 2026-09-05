[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$repositoryPrefix = $repositoryRoot.TrimEnd([char[]]@('\', '/')) + [System.IO.Path]::DirectorySeparatorChar
$docsRoot = Join-Path $repositoryRoot 'docs'
$errors = New-Object System.Collections.Generic.List[string]

function Get-RepositoryRelativePath {
    param([Parameter(Mandatory = $true)][string]$LiteralPath)

    return $LiteralPath.Substring($repositoryRoot.Length).TrimStart([char[]]@('\', '/')).Replace('\', '/')
}

$requiredFiles = @(
    'README.md',
    'docs/README.md',
    'docs/product.md',
    'docs/requirements.md',
    'docs/architecture.md',
    'docs/security.md',
    'docs/specifications/README.md',
    'docs/specifications/facility-data.md',
    'docs/specifications/web-ui.md',
    'docs/specifications/chat-integrations.md',
    'docs/specifications/facility-catalog.openapi.yaml',
    'docs/decisions/README.md',
    'docs/decisions/TEMPLATE.md',
    'docs/process/development.md',
    'docs/process/documentation.md',
    'docs/process/release.md',
    'docs/operations/README.md',
    'docs/guides/README.md',
    'docs/research/README.md',
    'docs/research/ui/README.md',
    'docs/archive/README.md'
)

foreach ($relativePath in $requiredFiles) {
    $absolutePath = Join-Path $repositoryRoot $relativePath
    if (-not (Test-Path -LiteralPath $absolutePath -PathType Leaf)) {
        $errors.Add("Required file is missing: $relativePath")
        continue
    }

    if ((Get-Item -LiteralPath $absolutePath).Length -eq 0) {
        $errors.Add("Required file is empty: $relativePath")
    }
}

$legacyCompatibilityFiles = @{
    'docs/product_baseline.md' = 'docs/product.md'
    'docs/architecture/quality-attributes.md' = 'docs/architecture.md'
    'docs/security/security-baseline.md' = 'docs/security.md'
    'docs/how-to-use.md' = 'docs/guides/how-to-use.md'
    'docs/adr/0001-repository-strategy.md' = 'docs/decisions/0001-repository-strategy.md'
    'docs/adr/0002-facility-data-source-and-freshness.md' = 'docs/decisions/0002-facility-data-source-and-freshness.md'
    'docs/adr/0003-recommendation-engine-before-ai.md' = 'docs/decisions/0003-recommendation-engine-before-ai.md'
    'docs/adr/0004-localization-strategy.md' = 'docs/decisions/0004-localization-strategy.md'
}

foreach ($legacyPath in $legacyCompatibilityFiles.Keys) {
    $absolutePath = Join-Path $repositoryRoot $legacyPath
    if (-not (Test-Path -LiteralPath $absolutePath -PathType Leaf)) {
        $errors.Add("Compatibility stub is missing: $legacyPath")
        continue
    }

    $content = Get-Content -LiteralPath $absolutePath -Raw -Encoding utf8
    if ($content -notmatch '(?m)^- Status: Compatibility stub\s*$') {
        $errors.Add("Compatibility stub status is missing: $legacyPath")
    }

    $canonicalPath = $legacyCompatibilityFiles[$legacyPath]
    $canonicalName = Split-Path -Leaf $canonicalPath
    if ($content -notmatch [regex]::Escape($canonicalName)) {
        $errors.Add("Compatibility stub does not reference its canonical source '$canonicalPath': $legacyPath")
    }
}

if (Test-Path -LiteralPath $docsRoot -PathType Container) {
    $markdownFiles = Get-ChildItem -LiteralPath $docsRoot -Recurse -File -Filter '*.md'
    foreach ($file in $markdownFiles) {
        $relativePath = Get-RepositoryRelativePath -LiteralPath $file.FullName
        if ($file.Name -in @('README.md', 'TEMPLATE.md') -or $legacyCompatibilityFiles.ContainsKey($relativePath)) {
            continue
        }

        if ($file.Name -notmatch '^[a-z0-9]+(?:-[a-z0-9]+)*\.md$') {
            $errors.Add("Markdown filename must use lowercase kebab-case: $relativePath")
        }
    }
}

$decisionRoot = Join-Path $docsRoot 'decisions'
$decisionIndexPath = Join-Path $decisionRoot 'README.md'
$decisionIndex = ''
if (Test-Path -LiteralPath $decisionIndexPath -PathType Leaf) {
    $decisionIndex = Get-Content -LiteralPath $decisionIndexPath -Raw -Encoding utf8
}

$decisionFiles = @()
if (Test-Path -LiteralPath $decisionRoot -PathType Container) {
    $decisionFiles = @(
        Get-ChildItem -LiteralPath $decisionRoot -File -Filter '*.md' |
            Where-Object { $_.Name -notin @('README.md', 'TEMPLATE.md') }
    )
}

$seenIds = @{}
$requiredDecisionFields = @(
    'Status',
    'Date',
    'Type',
    'Related Issues',
    'Related Pull Requests',
    'Affected Docs',
    'Supersedes',
    'Superseded By'
)
$allowedStatuses = @('proposed', 'accepted', 'superseded', 'deprecated', 'rejected')

foreach ($file in $decisionFiles) {
    if ($file.Name -notmatch '^(?<id>\d{4})-[a-z0-9]+(?:-[a-z0-9]+)*\.md$') {
        $errors.Add("Decision filename is invalid: docs/decisions/$($file.Name)")
        continue
    }

    $id = $Matches.id
    if ($seenIds.ContainsKey($id)) {
        $errors.Add("Decision ID is duplicated: $id")
    } else {
        $seenIds[$id] = $file.Name
    }

    $content = Get-Content -LiteralPath $file.FullName -Raw -Encoding utf8
    if ($content -notmatch "(?m)^# (?:DR-|ADR[- ]?)${id}(?::|\s).+") {
        $errors.Add("Decision title must use DR-${id}, ADR-${id}, or ADR ${id}: docs/decisions/$($file.Name)")
    }

    foreach ($field in $requiredDecisionFields) {
        $fieldMatches = [regex]::Matches($content, "(?m)^- $([regex]::Escape($field)):")
        if ($fieldMatches.Count -eq 0) {
            $errors.Add("Decision field '$field' is missing: docs/decisions/$($file.Name)")
        } elseif ($fieldMatches.Count -gt 1) {
            $errors.Add("Decision field '$field' is duplicated: docs/decisions/$($file.Name)")
        }
    }

    $statusMatch = [regex]::Match($content, '(?im)^- Status:\s*(?<status>[^\r\n]+)')
    if ($statusMatch.Success) {
        $status = $statusMatch.Groups['status'].Value.Trim().ToLowerInvariant()
        if ($status -notin $allowedStatuses) {
            $errors.Add("Decision status '$status' is invalid: docs/decisions/$($file.Name)")
        }
    }

    if ($decisionIndex -notmatch [regex]::Escape($file.Name)) {
        $errors.Add("Decision is not listed in docs/decisions/README.md: $($file.Name)")
    }
}

$linkFiles = @(
    @(Get-Item -LiteralPath (Join-Path $repositoryRoot 'README.md')) +
    @(Get-ChildItem -LiteralPath $docsRoot -Recurse -File |
        Where-Object { $_.Extension -in @('.md', '.html') })
)
$markdownLinkPattern = [regex]'!?\[[^\]]*\]\((?<target><[^>]+>|[^)\s]+)(?:\s+"[^"]*")?\)'
$htmlLinkPattern = [regex]'(?i)(?:href|src)\s*=\s*["''](?<target>[^"'']+)["'']'

foreach ($file in $linkFiles) {
    $content = Get-Content -LiteralPath $file.FullName -Raw -Encoding utf8
    $matches = @($markdownLinkPattern.Matches($content)) + @($htmlLinkPattern.Matches($content))

    foreach ($match in $matches) {
        $target = $match.Groups['target'].Value.Trim().Trim('<', '>')
        if ([string]::IsNullOrWhiteSpace($target) -or $target.StartsWith('#')) {
            continue
        }

        if ($target -match '^[a-z][a-z0-9+.-]*:' -or $target.StartsWith('//') -or $target.StartsWith('/')) {
            continue
        }

        $pathOnly = ($target -split '[?#]', 2)[0]
        if ([string]::IsNullOrWhiteSpace($pathOnly)) {
            continue
        }

        try {
            $pathOnly = [Uri]::UnescapeDataString($pathOnly)
        } catch {
            $relativePath = Get-RepositoryRelativePath -LiteralPath $file.FullName
            $errors.Add("Internal link cannot be decoded in ${relativePath}: $target")
            continue
        }

        $resolvedPath = [System.IO.Path]::GetFullPath((Join-Path $file.DirectoryName $pathOnly))
        $isRepositoryRoot = $resolvedPath.Equals($repositoryRoot, [System.StringComparison]::OrdinalIgnoreCase)
        $isInsideRepository = $resolvedPath.StartsWith($repositoryPrefix, [System.StringComparison]::OrdinalIgnoreCase)
        if (-not $isRepositoryRoot -and -not $isInsideRepository) {
            $relativePath = Get-RepositoryRelativePath -LiteralPath $file.FullName
            $errors.Add("Internal link escapes repository in ${relativePath}: $target")
            continue
        }

        if (-not (Test-Path -LiteralPath $resolvedPath)) {
            $relativePath = Get-RepositoryRelativePath -LiteralPath $file.FullName
            $errors.Add("Broken internal link in ${relativePath}: $target")
        }
    }
}

if ($errors.Count -gt 0) {
    foreach ($message in $errors) {
        Write-Host "ERROR: $message"
    }
    exit 1
}

Write-Host "Documentation validation passed. Checked $($requiredFiles.Count) required files, $($legacyCompatibilityFiles.Count) compatibility stubs, $($decisionFiles.Count) decision records, and $($linkFiles.Count) linked documents."
