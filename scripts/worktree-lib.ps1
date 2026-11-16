# Shared helpers for worktree-audit.ps1 and worktree-clean.ps1.
# PowerShell 5.1 compatible: no ternary, no null-coalescing, no && chains.

function Set-WorktreeProxy {
    param([string]$Proxy)
    # Windows environment variables are case-insensitive; HTTPS_PROXY covers
    # both gh and git, so there is no need for lowercase duplicates.
    $state = @{
        HTTPS_PROXY = $env:HTTPS_PROXY
        HTTP_PROXY  = $env:HTTP_PROXY
    }
    if ($Proxy) {
        $env:HTTPS_PROXY = $Proxy
        $env:HTTP_PROXY = $Proxy
    }
    return $state
}

function Restore-WorktreeProxy {
    param($State)
    if (-not $State) { return }
    $env:HTTPS_PROXY = $State.HTTPS_PROXY
    $env:HTTP_PROXY = $State.HTTP_PROXY
}

function Get-NormalizedPath {
    param([string]$Path)
    return ([System.IO.Path]::GetFullPath($Path) -replace '\\', '/').TrimEnd('/').ToLowerInvariant()
}

function Get-WorktreeRepoRoot {
    $candidate = Split-Path -Parent $PSScriptRoot
    $root = & git -C $candidate rev-parse --show-toplevel
    if ($LASTEXITCODE -ne 0 -or -not $root) {
        $root = & git rev-parse --show-toplevel
    }
    if ($LASTEXITCODE -ne 0 -or -not $root) {
        throw "Not inside a git repository. Run this from the repository or keep it under scripts/."
    }
    return ([System.IO.Path]::GetFullPath(([string]($root | Select-Object -First 1)).Trim())).TrimEnd('\')
}

function Get-WorktreeRecords {
    param([string]$RepoRoot)
    $lines = & git -C $RepoRoot worktree list --porcelain
    if ($LASTEXITCODE -ne 0) { throw "git worktree list failed." }

    $records = @()
    $current = $null
    foreach ($line in @($lines)) {
        if ($line -match '^worktree (.+)$') {
            if ($null -ne $current) {
                $records += [pscustomobject]$current
            }
            $current = @{
                Path = $matches[1].Trim()
                Head = ''
                Branch = ''
                Detached = $false
                Bare = $false
                Locked = $false
                Prunable = $false
                PrunableReason = ''
            }
        }
        elseif ($line -match '^HEAD (.+)$') { $current.Head = $matches[1].Trim() }
        elseif ($line -match '^branch refs/heads/(.+)$') { $current.Branch = $matches[1].Trim() }
        elseif ($line -eq 'detached') { $current.Detached = $true }
        elseif ($line -eq 'bare') { $current.Bare = $true }
        elseif ($line -match '^prunable(?: (.+))?$') {
            $current.Prunable = $true
            if ($matches[1]) { $current.PrunableReason = $matches[1].Trim() }
        }
        elseif ($line -eq 'locked' -or $line -match '^locked ') { $current.Locked = $true }
    }
    if ($null -ne $current) {
        $records += [pscustomobject]$current
    }
    return @($records)
}

function Resolve-GhPath {
    $cmd = Get-Command gh -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }

    $candidates = @(
        (Join-Path $env:USERPROFILE 'bin\gh.exe'),
        (Join-Path $env:USERPROFILE 'bin\gh.cmd'),
        'C:\Program Files\GitHub CLI\gh.exe',
        'C:\Program Files (x86)\GitHub CLI\gh.exe'
    )
    foreach ($candidate in $candidates) {
        if ($candidate -and (Test-Path -LiteralPath $candidate)) { return $candidate }
    }
    return $null
}

function Invoke-Gh {
    param(
        [Parameter(Mandatory = $true)][string]$GhPath,
        [Parameter(Mandatory = $true)][string[]]$GhArgs
    )
    # Native stderr becomes terminating under $ErrorActionPreference = 'Stop'
    # in Windows PowerShell 5.1, so relax it for the call and check the exit code.
    $previous = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try {
        $output = @(& $GhPath @GhArgs 2>&1)
        $code = $LASTEXITCODE
    }
    finally {
        $ErrorActionPreference = $previous
    }

    $text = (@($output) | ForEach-Object { $_.ToString() }) -join "`n"
    if ($code -ne 0) {
        throw ("gh {0} failed (exit {1}): {2}" -f ($GhArgs -join ' '), $code, $text.Trim())
    }
    return $text
}

function Get-PrIndex {
    param(
        [Parameter(Mandatory = $true)][string]$GhPath,
        [int]$Limit = 300
    )
    $text = Invoke-Gh -GhPath $GhPath -GhArgs @(
        'pr', 'list', '--state', 'all', '--limit', [string]$Limit,
        '--json', 'number,state,headRefName,mergedAt,closedAt,headRefOid,title,url'
    )
    $text = $text.Trim()
    if (-not $text) { throw "gh pr list returned empty output." }

    try {
        $parsed = $text | ConvertFrom-Json
    }
    catch {
        throw ("gh pr list returned invalid JSON: {0}" -f $_.Exception.Message)
    }

    # Windows PowerShell 5.1 returns a JSON array as one object; @() would
    # wrap it as a single nested element. foreach enumerates it correctly.
    $items = @()
    foreach ($entry in $parsed) { $items += $entry }

    # One branch can have several PRs (reopened); the newest PR wins.
    $index = @{}
    foreach ($pr in ($items | Sort-Object -Property number)) {
        $branch = [string]$pr.headRefName
        if (-not $branch) { continue }
        $index[$branch] = $pr
    }
    return [pscustomobject]@{
        Index = $index
        Count = $items.Count
        Items = $items
    }
}

function Get-LastCommitIso {
    param([string]$Path)
    $value = & git -C $Path log -1 --format=%cI
    if ($LASTEXITCODE -ne 0 -or -not $value) { return $null }
    return ([string]($value | Select-Object -First 1)).Trim()
}

function ConvertTo-AgeDays {
    param([string]$Iso)
    if (-not $Iso) { return $null }
    try {
        $when = [datetime]::Parse($Iso, [System.Globalization.CultureInfo]::InvariantCulture, [System.Globalization.DateTimeStyles]::RoundtripKind)
        $delta = (Get-Date).ToUniversalTime() - $when.ToUniversalTime()
        return [math]::Round($delta.TotalDays, 1)
    }
    catch {
        return $null
    }
}

function Get-DirSizeMB {
    param([string]$Path)
    try {
        $sum = (Get-ChildItem -LiteralPath $Path -Recurse -Force -File -ErrorAction SilentlyContinue |
            Measure-Object -Property Length -Sum).Sum
        if (-not $sum) { return 0 }
        return [math]::Round($sum / 1MB, 1)
    }
    catch {
        return $null
    }
}
