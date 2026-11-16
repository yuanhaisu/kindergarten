<#
.SYNOPSIS
    Create an isolated sibling git worktree for one development session.

.DESCRIPTION
    Implements the parallel-development discipline documented in
    docs/agents/parallel-development.md:

        one session = one branch = one worktree

    The main worktree stays on `main` and clean; each slice gets its own
    sibling directory `..\kg-<Name>` on its own branch.

    Safety checks before creating anything:
      - current directory (or this script's repo) is a git repository
      - target path does not already exist
      - local branch does not already exist
      - remote branch does not already exist
      - base ref resolves to a commit

    Supports -WhatIf: prints the intended actions without fetching,
    creating a worktree, or creating a branch.

.PARAMETER Name
    Short kebab-case slice name, e.g. pickup-polish. Target directory is
    `..\kg-<Name>`.

.PARAMETER Branch
    Branch to create. Defaults to `feat/<Name>`.

.PARAMETER Base
    Commit-ish to branch from. Defaults to `origin/main`.

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File scripts/worktree-new.ps1 -Name pickup-polish

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File scripts/worktree-new.ps1 -Name parallel-development -Branch docs/parallel-development -WhatIf
#>
[CmdletBinding(SupportsShouldProcess = $true)]
param(
    [Parameter(Mandatory = $true, Position = 0)]
    [ValidatePattern('^[a-z0-9][a-z0-9-]*$')]
    [string]$Name,

    [string]$Branch,

    [string]$Base = 'origin/main'
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

if (-not $Branch) {
    $Branch = "feat/$Name"
}

# --- Validate before touching anything -------------------------------------
if (Test-Path -LiteralPath $target) {
    Fail "Target path already exists: $target. Pick another -Name or clean it first."
}

& git -C $repoRootPath check-ref-format --branch $Branch | Out-Null
if ($LASTEXITCODE -ne 0) {
    Fail "Invalid branch name: $Branch"
}

& git -C $repoRootPath show-ref --verify --quiet "refs/heads/$Branch"
if ($LASTEXITCODE -eq 0) {
    Fail "Local branch already exists: $Branch"
}

foreach ($existing in (Get-WorktreePaths)) {
    if ((Normalize-Path $existing) -eq (Normalize-Path $target)) {
        Fail "Path is already registered as a worktree: $target"
    }
}

# --- Fetch, then re-check against the fresh refs ---------------------------
if ($PSCmdlet.ShouldProcess('origin', 'Fetch and prune remote refs')) {
    Invoke-Git @('fetch', 'origin', '--prune')
    Write-Host "[OK] Fetched origin (pruned)." -ForegroundColor Green

    & git -C $repoRootPath ls-remote --exit-code --heads origin $Branch | Out-Null
    if ($LASTEXITCODE -eq 0) {
        Fail "Remote branch already exists: origin/$Branch"
    }
}

& git -C $repoRootPath rev-parse --verify --quiet "$Base^{commit}" | Out-Null
if ($LASTEXITCODE -ne 0) {
    Fail "Base ref not found: $Base. Run 'git fetch origin --prune' first or pass -Base."
}

# --- Create ----------------------------------------------------------------
if ($PSCmdlet.ShouldProcess($target, "Create worktree on new branch '$Branch' from '$Base'")) {
    Invoke-Git @('worktree', 'add', $target, '-b', $Branch, $Base) | Out-Null
    Write-Host "[OK] Worktree created: $target" -ForegroundColor Green
    Write-Host "[OK] Branch created:    $Branch (from $Base)" -ForegroundColor Green
}

# --- Next steps ------------------------------------------------------------
Write-Host ""
if ($WhatIfPreference) {
    Write-Host "[WhatIf] Nothing was fetched, created, or modified." -ForegroundColor Yellow
    Write-Host "[WhatIf] Would create: $target on branch '$Branch' from '$Base'." -ForegroundColor Yellow
}
else {
    Write-Host "Next steps:" -ForegroundColor Cyan
    Write-Host "  cd `"$target`""
    Write-Host "  npm ci"
    Write-Host "  # edit and verify, then stage exact paths only:"
    Write-Host "  git add -- <exact paths>"
    Write-Host "  git commit -m `"...`""
    Write-Host "  git push -u origin $Branch"
    Write-Host "  gh pr create --base main --head $Branch --title `"...`""
    Write-Host ""
    Write-Host "After the PR is merged (the merge bot runs this):" -ForegroundColor Cyan
    Write-Host "  powershell -ExecutionPolicy Bypass -File scripts/worktree-clean.ps1 -Pr <PR number> -DeleteRemote"
}
