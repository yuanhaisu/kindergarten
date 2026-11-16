<#
.SYNOPSIS
    Remove a slice worktree after its PR reached a terminal state.

.DESCRIPTION
    Companion to scripts/worktree-new.ps1. Cleanup is destructive, so this
    script refuses unless every check passes:

      1. exactly one of -Pr, -Branch, or -Name identifies the slice
      2. the slice has a PR whose state is MERGED (or CLOSED with
         -AllowClosed after the owner confirmed abandonment)
      3. the worktree (if any) is registered, is not the main worktree,
         is not detached, and is not locked
      4. `git status --porcelain` is empty (including untracked files)
      5. the local HEAD equals the PR's headRefOid. A mismatch means there
         may be commits that never reached the PR; refuse instead of guessing.

    Then runs `git worktree remove`, `git branch -D`, and optionally
    `git push origin --delete` with -DeleteRemote. It never uses
    `git worktree remove --force`; `git branch -D` is a force delete, but it
    is allowed only after the PR state and headRefOid checks above pass.

    The branch-ancestor check used by the previous version is gone: this
    repository merges some PRs with squash/rebase, so a merged branch is
    often NOT an ancestor of origin/main. PR state is the authority.

    Exit codes: 0 = cleaned or already clean, 1 = refused/failed.
    With -Json, stdout is one JSON object; human logs are suppressed.

.PARAMETER Name
    Slice name used by worktree-new.ps1, e.g. pickup-polish. Target is
    `..\kg-<Name>`. Kept for backwards compatibility.

.PARAMETER Pr
    PR number, e.g. 33. Preferred when a bot cleans up right after merge.

.PARAMETER Branch
    Head branch name, e.g. feat/pickup-polish. Useful for manual cleanup.

.PARAMETER AllowClosed
    Allow cleanup when the PR is CLOSED without being merged. Only pass
    this after the owner explicitly confirmed the work is abandoned.

.PARAMETER DeleteRemote
    Also delete the head branch on origin. Missing remote branches are
    treated as success.

.PARAMETER Json
    Emit one JSON result object on stdout.

.PARAMETER Proxy
    Proxy URL for GitHub calls, e.g. http://127.0.0.1:7890. When omitted,
    existing HTTPS_PROXY/HTTP_PROXY environment variables are used.

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File scripts/worktree-clean.ps1 -Pr 33 -DeleteRemote

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File scripts/worktree-clean.ps1 -Pr 33 -WhatIf
#>
[CmdletBinding(SupportsShouldProcess = $true)]
param(
    [Parameter(Position = 0)]
    [ValidatePattern('^[a-z0-9][a-z0-9-]*$')]
    [string]$Name,

    [int]$Pr = 0,

    [string]$Branch,

    [switch]$AllowClosed,

    [switch]$DeleteRemote,

    [switch]$Json,

    [string]$Proxy
)

$ErrorActionPreference = 'Stop'

$utf8 = New-Object System.Text.UTF8Encoding($false)
[Console]::OutputEncoding = $utf8
$OutputEncoding = $utf8

. (Join-Path $PSScriptRoot 'worktree-lib.ps1')

function Throw-Clean {
    param([string]$Stage, [string]$Message)
    $ex = New-Object System.Exception($Message)
    $ex.Data['stage'] = $Stage
    throw $ex
}

function Write-Log {
    param([string]$Message, [string]$Color = 'Gray')
    if (-not $Json) {
        Write-Host $Message -ForegroundColor $Color
    }
}

function Invoke-GitAllowFail {
    param([string[]]$GitArgs, [string]$Path)
    if (-not $Path) { $Path = $repoRootPath }
    $previous = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try {
        $output = @(& git -C $Path @GitArgs 2>&1)
        $code = $LASTEXITCODE
    }
    finally {
        $ErrorActionPreference = $previous
    }
    $text = (@($output) | ForEach-Object { $_.ToString() }) -join "`n"
    return [pscustomobject]@{
        ExitCode = $code
        Output = $text
    }
}

function Invoke-Git {
    param([string[]]$GitArgs)
    $gitResult = Invoke-GitAllowFail -GitArgs $GitArgs
    if ($gitResult.ExitCode -ne 0) {
        Throw-Clean 'git' ("git {0} failed (exit {1}): {2}" -f ($GitArgs -join ' '), $gitResult.ExitCode, $gitResult.Output.Trim())
    }
    return $gitResult.Output
}

$repoRootPath = $null
$prInfo = $null
$result = [ordered]@{
    ok = $false
    whatIf = [bool]$WhatIfPreference
    pr = $null
    branch = $null
    worktree = $null
    removed = $false
    branchDeleted = $false
    remoteDeleted = $false
    actions = @()
}

$exitCode = 0
$failure = $null
$proxyState = Set-WorktreeProxy -Proxy $Proxy

try {
    $modes = 0
    if ($Name) { $modes++ }
    if ($Pr -gt 0) { $modes++ }
    if ($Branch) { $modes++ }
    if ($modes -ne 1) {
        Throw-Clean 'args' 'Specify exactly one of -Name, -Pr, or -Branch.'
    }

    $repoRootPath = Get-WorktreeRepoRoot
    $records = @(Get-WorktreeRecords -RepoRoot $repoRootPath)
    if ($records.Count -eq 0) {
        Throw-Clean 'worktree' 'git worktree list returned no worktrees.'
    }
    $mainPathKey = Get-NormalizedPath $records[0].Path

    $targetPath = $null
    if ($Name) {
        $targetPath = Join-Path (Split-Path -Parent $repoRootPath) ('kg-' + $Name)
        if (-not (Test-Path -LiteralPath $targetPath)) {
            Throw-Clean 'worktree' "Worktree path does not exist: $targetPath"
        }
        $registered = $false
        foreach ($record in $records) {
            if ((Get-NormalizedPath $record.Path) -eq (Get-NormalizedPath $targetPath)) {
                $registered = $true
                break
            }
        }
        if (-not $registered) {
            Throw-Clean 'worktree' "'$targetPath' exists but is not a registered git worktree."
        }
        $branchValue = & git -C $targetPath rev-parse --abbrev-ref HEAD
        if ($LASTEXITCODE -ne 0 -or -not $branchValue) {
            Throw-Clean 'worktree' "Cannot determine the branch checked out in $targetPath"
        }
        $Branch = ([string]($branchValue | Select-Object -First 1)).Trim()
    }

    $ghPath = Resolve-GhPath
    if (-not $ghPath) {
        Throw-Clean 'gh' 'gh CLI not found. Install GitHub CLI or add it to PATH.'
    }

    # Resolve the PR from -Pr or from the head branch.
    if ($Pr -gt 0) {
        $text = Invoke-Gh -GhPath $ghPath -GhArgs @(
            'pr', 'view', [string]$Pr, '--json', 'number,state,headRefName,headRefOid,title,url'
        )
        try {
            $prInfo = $text | ConvertFrom-Json
        }
        catch {
            Throw-Clean 'gh-output' ("gh pr view {0} returned invalid JSON: {1}" -f $Pr, $_.Exception.Message)
        }
        $Branch = [string]$prInfo.headRefName
    }
    else {
        if (-not $Branch) {
            Throw-Clean 'args' 'Could not determine the branch. Pass -Branch or -Pr.'
        }
        $text = Invoke-Gh -GhPath $ghPath -GhArgs @(
            'pr', 'list', '--head', $Branch, '--state', 'all', '--limit', '20',
            '--json', 'number,state,headRefName,headRefOid,title,url'
        )
        try {
            $parsed = $text | ConvertFrom-Json
        }
        catch {
            Throw-Clean 'gh-output' ("gh pr list --head {0} returned invalid JSON: {1}" -f $Branch, $_.Exception.Message)
        }
        # PS 5.1 returns a JSON array as one object; enumerate explicitly.
        $items = @()
        foreach ($entry in $parsed) { $items += $entry }
        if ($items.Count -eq 0) {
            Throw-Clean 'pr' "No PR found for branch '$Branch'. Refusing to clean without a terminal PR state."
        }
        $prInfo = $items | Sort-Object -Property number | Select-Object -Last 1
    }

    if (-not $prInfo) {
        Throw-Clean 'pr' 'No PR information resolved.'
    }

    # Record PR context before validation so refusals carry it into -Json.
    $result.pr = [ordered]@{
        number = $prInfo.number
        state = $prInfo.state
        headRefName = [string]$prInfo.headRefName
        headRefOid = $prInfo.headRefOid
        title = $prInfo.title
        url = $prInfo.url
    }
    $result.branch = [string]$prInfo.headRefName

    if ($prInfo.state -eq 'OPEN') {
        Throw-Clean 'pr' "PR #$($prInfo.number) is still OPEN. Wait for MERGED/CLOSED."
    }
    if ($prInfo.state -eq 'CLOSED' -and -not $AllowClosed) {
        Throw-Clean 'pr' "PR #$($prInfo.number) was CLOSED without merge. Pass -AllowClosed only after the owner confirms abandonment."
    }
    if ($prInfo.state -ne 'MERGED' -and $prInfo.state -ne 'CLOSED') {
        Throw-Clean 'pr' "Unexpected PR state: $($prInfo.state)"
    }

    $Branch = [string]$prInfo.headRefName
    if (-not $Branch) {
        Throw-Clean 'pr' 'PR has no headRefName.'
    }
    if ($Branch -eq 'main' -or $Branch -eq 'master') {
        Throw-Clean 'safety' "Refusing to clean branch '$Branch'."
    }

    # Locate the worktree by branch (or the explicit -Name path).
    $worktree = $null
    foreach ($record in $records) {
        if ($record.Branch -eq $Branch) { $worktree = $record; break }
    }
    if ($targetPath) {
        if ($null -eq $worktree) {
            Throw-Clean 'worktree' "No registered worktree is checked out on '$Branch'."
        }
        if ((Get-NormalizedPath $worktree.Path) -ne (Get-NormalizedPath $targetPath)) {
            Throw-Clean 'worktree' "Branch '$Branch' is checked out at '$($worktree.Path)', not '$targetPath'."
        }
    }

    $branchExists = $false
    $showRef = Invoke-GitAllowFail @('show-ref', '--verify', '--quiet', "refs/heads/$Branch")
    if ($showRef.ExitCode -eq 0) { $branchExists = $true }

    if ($null -eq $worktree) {
        # The worktree is already gone. Delete the stray local branch only if
        # it still points at exactly what the PR merged.
        if (-not $branchExists) {
            Write-Log "[OK] Nothing to clean: no worktree or local branch for '$Branch'." 'Green'
            $result.ok = $true
            $result.actions = @('already-clean')
            $exitCode = 0
        }
        else {
            $headText = Invoke-GitAllowFail @('rev-parse', $Branch)
            $localHead = $headText.Output.Trim()
            if ($localHead -ne [string]$prInfo.headRefOid) {
                Throw-Clean 'safety' "Local branch '$Branch' is at $localHead but the PR merged $($prInfo.headRefOid); refusing to delete."
            }
            $proceed = $true
            if ($Json) {
                if ($WhatIfPreference) { $proceed = $false }
            }
            else {
                $proceed = $PSCmdlet.ShouldProcess($Branch, "Delete local branch '$Branch'")
            }
            if ($proceed) {
                # -D is gated by the PR state + headRefOid checks above.
                Invoke-Git @('branch', '-D', $Branch) | Out-Null
                Write-Log "[OK] Local branch deleted: $Branch" 'Green'
                $result.branchDeleted = $true
                $result.actions += 'delete-local-branch'
            }
            else {
                $result.actions += 'would-delete-local-branch'
            }
            if ($DeleteRemote) {
                if (-not $proceed) {
                    $result.actions += 'would-delete-remote-branch'
                }
                else {
                    $remoteResult = Invoke-GitAllowFail @('push', 'origin', '--delete', $Branch)
                    if ($remoteResult.ExitCode -ne 0 -and $remoteResult.Output -notmatch 'remote ref does not exist|unable to delete') {
                        Throw-Clean 'remote' ("git push origin --delete {0} failed: {1}" -f $Branch, $remoteResult.Output.Trim())
                    }
                    Write-Log "[OK] Remote branch deleted or already absent: origin/$Branch" 'Green'
                    $result.remoteDeleted = $true
                    $result.actions += 'delete-remote-branch'
                }
            }
            $result.ok = $true
            $exitCode = 0
        }
    }
    else {
        $targetPath = $worktree.Path
        $result.worktree = $targetPath

        if ((Get-NormalizedPath $targetPath) -eq $mainPathKey) {
            Throw-Clean 'safety' 'Refusing to clean the main worktree.'
        }
        if ((Get-NormalizedPath $targetPath) -eq (Get-NormalizedPath $repoRootPath)) {
            Throw-Clean 'safety' 'Cannot remove the worktree this script is running from. Run it from the main worktree.'
        }
        if ($worktree.Locked) {
            Throw-Clean 'safety' "Worktree is locked: $targetPath. Run 'git worktree unlock' first."
        }
        if ($worktree.Detached) {
            Throw-Clean 'safety' 'Worktree is in detached HEAD state; refusing to guess which branch to delete.'
        }

        $status = Invoke-GitAllowFail -Path $targetPath -GitArgs @('status', '--porcelain')
        if ($status.ExitCode -ne 0) {
            Throw-Clean 'safety' "git status failed in $targetPath; refusing to modify the worktree."
        }
        $dirty = @($status.Output -split "`n" | Where-Object { $_ -and $_.Trim().Length -gt 0 })
        if ($dirty.Count -gt 0) {
            $preview = ($dirty | Select-Object -First 10) -join "`n"
            Throw-Clean 'safety' ("Worktree has {0} uncommitted/untracked entries. Commit or discard them first.`n{1}" -f $dirty.Count, $preview)
        }

        $headText = Invoke-GitAllowFail -Path $targetPath -GitArgs @('rev-parse', 'HEAD')
        $localHead = $headText.Output.Trim()
        if ($localHead -ne [string]$prInfo.headRefOid) {
            Throw-Clean 'safety' ("Local HEAD {0} does not match PR headRefOid {1}. There may be commits that never reached the PR; refusing to delete." -f $localHead, $prInfo.headRefOid)
        }

        $proceed = $true
        if ($Json) {
            if ($WhatIfPreference) { $proceed = $false }
        }
        else {
            $proceed = $PSCmdlet.ShouldProcess($targetPath, "Remove worktree and delete local branch '$Branch'")
        }
        if ($proceed) {
            Invoke-Git @('worktree', 'remove', $targetPath) | Out-Null
            Write-Log "[OK] Worktree removed: $targetPath" 'Green'
            $result.removed = $true
            $result.actions += 'remove-worktree'

            if ($branchExists) {
                # -D is gated by the PR state + headRefOid checks above.
                Invoke-Git @('branch', '-D', $Branch) | Out-Null
                Write-Log "[OK] Local branch deleted: $Branch" 'Green'
                $result.branchDeleted = $true
                $result.actions += 'delete-local-branch'
            }

            if ($DeleteRemote) {
                $remoteResult = Invoke-GitAllowFail @('push', 'origin', '--delete', $Branch)
                if ($remoteResult.ExitCode -ne 0 -and $remoteResult.Output -notmatch 'remote ref does not exist|unable to delete') {
                    Throw-Clean 'remote' ("git push origin --delete {0} failed: {1}" -f $Branch, $remoteResult.Output.Trim())
                }
                Write-Log "[OK] Remote branch deleted or already absent: origin/$Branch" 'Green'
                $result.remoteDeleted = $true
                $result.actions += 'delete-remote-branch'
            }
        }
        else {
            Write-Log "[WhatIf] Would remove worktree: $targetPath" 'Yellow'
            Write-Log "[WhatIf] Would delete local branch: $Branch" 'Yellow'
            if ($DeleteRemote) {
                Write-Log "[WhatIf] Would delete remote branch: origin/$Branch" 'Yellow'
            }
            $result.actions = @('remove-worktree', 'delete-local-branch')
            if ($DeleteRemote) { $result.actions += 'delete-remote-branch' }
            $result.ok = $true
            $exitCode = 0
        }

        $result.ok = $true
        $exitCode = 0
    }
}
catch {
    $stage = 'unexpected'
    if ($null -ne $_.Exception.Data -and $_.Exception.Data.Contains('stage')) {
        $stage = [string]$_.Exception.Data['stage']
    }
    $failure = [ordered]@{
        ok = $false
        exitCode = 1
        stage = $stage
        error = $_.Exception.Message
        pr = $result.pr
        branch = $result.branch
        worktree = $result.worktree
        actions = @()
    }
    $exitCode = 1
}
finally {
    Restore-WorktreeProxy -State $proxyState
}

if ($null -ne $failure) {
    if ($Json) {
        Write-Output (ConvertTo-Json -InputObject $failure -Depth 6)
    }
    else {
        Write-Host ("[ERROR] stage={0}: {1}" -f $failure.stage, $failure.error) -ForegroundColor Red
    }
    [Console]::Error.WriteLine(("[worktree-clean] stage={0}: {1}" -f $failure.stage, $failure.error))
    exit $exitCode
}

if ($Json) {
    $resultJson = ConvertTo-Json -InputObject $result -Depth 6
    $parsedResult = $resultJson | ConvertFrom-Json
    if ($null -eq $parsedResult.actions -or $parsedResult.actions -isnot [System.Array]) {
        [Console]::Error.WriteLine('[worktree-clean] JSON contract violation: actions must be an array.')
        exit 1
    }
    Write-Output $resultJson
}
elseif ($WhatIfPreference) {
    Write-Host ""
    Write-Host "[WhatIf] Nothing was removed or deleted." -ForegroundColor Yellow
}
elseif ($result.ok) {
    Write-Host ""
    Write-Host "[DONE] Worktree lifecycle cleanup finished." -ForegroundColor Green
}
exit $exitCode
