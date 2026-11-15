<#
.SYNOPSIS
    Remove a slice worktree and delete its branch after the PR is merged.

.DESCRIPTION
    Companion to scripts/worktree-new.ps1. Cleanup is a destructive operation,
    so it refuses unless everything checks out:

      1. `..\kg-<Name>` exists and is a registered git worktree
      2. the worktree has no uncommitted or untracked changes
      3. the checked-out branch is not main/master and is not detached
      4. the branch is an ancestor of -Base (default origin/main), i.e. merged

    Then runs `git worktree remove`, `git branch -d`, and optionally
    `git push origin --delete` with -DeleteRemote.

    Supports -WhatIf: prints the intended actions without fetching,
    removing, or deleting anything.

.PARAMETER Name
    Slice name passed to worktree-new.ps1, e.g. pickup-polish.
    Target directory is `..\kg-<Name>`.

.PARAMETER Base
    Ref the branch must be merged into. Defaults to `origin/main`.

.PARAMETER DeleteRemote
    Also delete the branch on origin after removing it locally.

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File scripts/worktree-clean.ps1 -Name pickup-polish

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File scripts/worktree-clean.ps1 -Name pickup-polish -DeleteRemote -WhatIf
#>
[CmdletBinding(SupportsShouldProcess = $true)]
param(
    [Parameter(Mandatory = $true, Position = 0)]
    [ValidatePattern('^[a-z0-9][a-z0-9-]*$')]
    [string]$Name,

    [string]$Base = 'origin/main',

    [switch]$DeleteRemote
)

$ErrorActionPreference = 'Stop'

function Fail {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
    exit 1
}

function Invoke-Git {
    param([string[]]$GitArgs)
    # Do NOT redirect stderr. In Windows PowerShell 5.1, `git ... 2>&1` turns
    # git's stderr (progress, hints) into NativeCommandError records that
    # terminate the script under $ErrorActionPreference = 'Stop'. Let stderr
    # print through and rely on $LASTEXITCODE instead.
    $output = & git -C $repoRootPath @GitArgs
    if ($LASTEXITCODE -ne 0) {
        Fail ("git {0} failed (exit {1}). See git output above." -f ($GitArgs -join ' '), $LASTEXITCODE)
    }
    return $output
}

function Get-WorktreePaths {
    $paths = @()
    foreach ($line in (& git -C $repoRootPath worktree list --porcelain)) {
        if ($line -match '^worktree (.+)$') {
            $paths += $matches[1].Trim()
        }
    }
    return $paths
}

function Normalize-Path {
    param([string]$Path)
    return ([System.IO.Path]::GetFullPath($Path) -replace '\\', '/').TrimEnd('/').ToLowerInvariant()
}

# --- Locate the repository -------------------------------------------------
$repoRoot = & git rev-parse --show-toplevel
if ($LASTEXITCODE -ne 0 -or -not $repoRoot) {
    $candidate = Split-Path -Parent $PSScriptRoot
    if (Test-Path -LiteralPath (Join-Path $candidate '.git')) {
        $repoRoot = & git -C $candidate rev-parse --show-toplevel
    }
}
if ($LASTEXITCODE -ne 0 -or -not $repoRoot) {
    Fail "Not inside a git repository. Run this from the repository or keep it under scripts/."
}
$repoRootPath = ([System.IO.Path]::GetFullPath(([string]($repoRoot | Select-Object -First 1)).Trim())).TrimEnd('\')

$parent = Split-Path -Parent $repoRootPath
$target = Join-Path $parent ("kg-" + $Name)

# --- Existence -------------------------------------------------------------
if (-not (Test-Path -LiteralPath $target)) {
    if ($WhatIfPreference) {
        Write-Host "[WhatIf] Worktree path does not exist: $target" -ForegroundColor Yellow
        Write-Host "[WhatIf] A real run would exit 1 (target missing). Intended sequence would be:" -ForegroundColor Yellow
        Write-Host "  git fetch origin --prune"
        Write-Host "  git worktree remove $target"
        Write-Host "  git branch -d feat/$Name"
        if ($DeleteRemote) {
            Write-Host "  git push origin --delete feat/$Name"
        }
        Write-Host "[WhatIf] Nothing was fetched, removed, or deleted." -ForegroundColor Yellow
        exit 0
    }
    Fail "Worktree path does not exist: $target"
}

$normalizedTarget = Normalize-Path $target
$registered = $false
foreach ($existing in (Get-WorktreePaths)) {
    if ((Normalize-Path $existing) -eq $normalizedTarget) {
        $registered = $true
    }
}
if (-not $registered) {
    Fail "'$target' exists but is not a registered git worktree. Refusing to touch it."
}

# --- Inspect the worktree --------------------------------------------------
$branchOutput = & git -C $target rev-parse --abbrev-ref HEAD
if ($LASTEXITCODE -ne 0 -or -not $branchOutput) {
    Fail "Cannot determine the branch checked out in $target"
}
$branch = ([string]($branchOutput | Select-Object -First 1)).Trim()
if ($branch -eq 'HEAD') {
    Fail "Worktree is in detached HEAD state; refusing to guess which branch to delete."
}
if ($branch -eq 'main' -or $branch -eq 'master') {
    Fail "Refusing to clean a worktree checked out on '$branch'."
}

$dirty = @(& git -C $target status --porcelain)
if ($LASTEXITCODE -ne 0) {
    Fail "git status failed in $target; refusing to modify the worktree."
}
$dirty = @($dirty | Where-Object { $_ -and $_.ToString().Trim().Length -gt 0 })
if ($dirty.Count -gt 0) {
    $preview = ($dirty | Select-Object -First 10) -join "`n"
    Fail "Worktree has $($dirty.Count) uncommitted/untracked entries. Commit or discard them first.`n$preview"
}
Write-Host "[OK] Worktree is clean: $target" -ForegroundColor Green
Write-Host "[OK] Branch: $branch" -ForegroundColor Green

# --- Refresh refs, then require the branch to be merged --------------------
if ($PSCmdlet.ShouldProcess('origin', 'Fetch and prune remote refs')) {
    Invoke-Git @('fetch', 'origin', '--prune')
    Write-Host "[OK] Fetched origin (pruned)." -ForegroundColor Green
}

& git -C $repoRootPath rev-parse --verify --quiet "$Base^{commit}" | Out-Null
if ($LASTEXITCODE -ne 0) {
    Fail "Base ref not found: $Base. Run 'git fetch origin --prune' first or pass -Base."
}

& git -C $repoRootPath merge-base --is-ancestor $branch $Base
if ($LASTEXITCODE -ne 0) {
    Fail "Branch '$branch' is not merged into '$Base'. Merge the PR first; refusing to delete unmerged work."
}
Write-Host "[OK] Branch '$branch' is merged into '$Base'." -ForegroundColor Green

# --- Remove ----------------------------------------------------------------
if ($PSCmdlet.ShouldProcess($target, "Remove worktree and delete local branch '$branch'")) {
    Invoke-Git @('worktree', 'remove', $target) | Out-Null
    Write-Host "[OK] Worktree removed: $target" -ForegroundColor Green

    Invoke-Git @('branch', '-d', $branch) | Out-Null
    Write-Host "[OK] Local branch deleted: $branch" -ForegroundColor Green
}

if ($DeleteRemote) {
    if ($PSCmdlet.ShouldProcess("origin/$branch", 'Delete remote branch')) {
        Invoke-Git @('push', 'origin', '--delete', $branch) | Out-Null
        Write-Host "[OK] Remote branch deleted: origin/$branch" -ForegroundColor Green
    }
}

Write-Host ""
if ($WhatIfPreference) {
    Write-Host "[WhatIf] Nothing was fetched, removed, or deleted." -ForegroundColor Yellow
}
else {
    Write-Host "[DONE] Slice '$Name' cleaned up." -ForegroundColor Green
}
