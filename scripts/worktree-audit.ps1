<#
.SYNOPSIS
    Read-only audit of this repository's worktrees against their PR states.

.DESCRIPTION
    Answers one question: which worktrees are leftovers, and what should be
    done with each of them?

    This script never removes anything. It classifies every registered
    worktree (and any orphaned `kg-*` directory next to the repo) into:

      auto-clean     PR MERGED, worktree clean, local HEAD == PR headRefOid
                     (no-PR branches are auto-clean only when they are a
                     proven ancestor of origin/main)
      needs-confirm  CLOSED but unmerged, dirty, HEAD mismatch, no PR and
                     stale, detached, locked, or unreadable
      active         PR OPEN, or no PR and recent commits
      prunable       registered but the directory is gone
      orphan         `kg-*` directory with no usable .git, not registered

    Exit codes:
      0  audit executed and produced a valid report (findings or not)
      1  environment/execution failure (git, gh, auth, proxy, fetch)
      2  invalid output (failed internal consistency checks)

    Exit code 0 says "the audit ran". It does NOT say "there are no
    leftovers"; leftovers are report content, not a failure.

.PARAMETER Json
    Emit a machine-readable JSON object on stdout. No colored text.

.PARAMETER Fetch
    Run `git fetch origin --prune` before auditing. Needed when the local
    `origin/main` may be behind (for example right after a remote merge).

.PARAMETER StaleDays
    A worktree with no PR and a last commit older than this many days is
    reported as needs-confirm instead of active. Default 7.

.PARAMETER PrLimit
    Maximum number of PRs fetched from `gh pr list`. Default 1000. Raise it
    as the repository accumulates PRs; a branch whose PR falls outside the
    window is reported as no-PR.

.PARAMETER SkipSize
    Skip recursive size calculation. The daily report is much faster; sizes
    are then reported as null.

.PARAMETER Proxy
    Proxy URL for gh/git network calls, e.g. http://127.0.0.1:7890. When
    omitted, existing HTTPS_PROXY/HTTP_PROXY environment variables are used.

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File scripts/worktree-audit.ps1

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File scripts/worktree-audit.ps1 -Json -Fetch -Proxy http://127.0.0.1:7890
#>
[CmdletBinding()]
param(
    [switch]$Json,
    [switch]$Fetch,
    [int]$StaleDays = 7,
    [int]$PrLimit = 1000,
    [switch]$SkipSize,
    [string]$Proxy
)

$ErrorActionPreference = 'Stop'

$utf8 = New-Object System.Text.UTF8Encoding($false)
[Console]::OutputEncoding = $utf8
$OutputEncoding = $utf8

. (Join-Path $PSScriptRoot 'worktree-lib.ps1')

function Throw-Audit {
    param([int]$Code, [string]$Stage, [string]$Message)
    $ex = New-Object System.Exception($Message)
    $ex.Data['exitCode'] = $Code
    $ex.Data['stage'] = $Stage
    throw $ex
}

function Assert-JsonArray {
    param($Value, [string]$Path)
    if ($null -eq $Value -or $Value -isnot [System.Array]) {
        Throw-Audit 2 'json-contract' "JSON contract violation: $Path must be an array."
    }
}

function Test-JsonArrayContract {
    param([string]$Json)
    $parsed = $Json | ConvertFrom-Json

    Assert-JsonArray $parsed.worktrees 'worktrees'
    Assert-JsonArray $parsed.orphans 'orphans'
    Assert-JsonArray $parsed.strayBranches 'strayBranches'

    $index = 0
    foreach ($wt in $parsed.worktrees) {
        Assert-JsonArray $wt.suggestedActions "worktrees[$index].suggestedActions"
        $index++
    }
    $index = 0
    foreach ($orphan in $parsed.orphans) {
        Assert-JsonArray $orphan.suggestedActions "orphans[$index].suggestedActions"
        $index++
    }
    $index = 0
    foreach ($stray in $parsed.strayBranches) {
        Assert-JsonArray $stray.suggestedActions "strayBranches[$index].suggestedActions"
        $index++
    }

    # PS 5.1 regression probe: a single-element or empty array must stay an
    # array after ConvertTo-Json. The top-level piped form @('x') | ConvertTo-Json
    # collapses to "x"; -InputObject is used everywhere for this reason.
    $probe = [ordered]@{ single = @('one'); empty = @(); pair = @('a', 'b') }
    $probeJson = ConvertTo-Json -InputObject $probe -Depth 5 -Compress
    $probeParsed = $probeJson | ConvertFrom-Json
    Assert-JsonArray $probeParsed.single 'self-test.single'
    Assert-JsonArray $probeParsed.empty 'self-test.empty'
    Assert-JsonArray $probeParsed.pair 'self-test.pair'
}

function Write-AuditText {
    param($Result)

    Write-Host ""
    Write-Host ("worktree 审计 — {0}" -f (Get-Date).ToString('yyyy-MM-dd HH:mm')) -ForegroundColor Cyan
    Write-Host ("仓库：{0}" -f $Result.repo)
    $fetchedText = '否'
    if ($Result.summary.fetched) { $fetchedText = '是' }
    Write-Host ("fetch：{0} | 扫描 PR：{1}" -f $fetchedText, $Result.summary.prsScanned)
    Write-Host ("总 worktree：{0}（不含主工作树）" -f $Result.summary.totalWorktrees)
    Write-Host ("可立即清理：{0}（有 PR {1} / 无 PR {2}） | 需确认：{3} | 活跃：{4} | 失效登记：{5} | 孤儿目录：{6} | 无 worktree 的本地分支：{7}" -f `
        $Result.summary.autoClean, $Result.summary.autoCleanPr, $Result.summary.autoCleanManual, `
        $Result.summary.needsConfirm, $Result.summary.active, `
        $Result.summary.prunable, $Result.summary.orphans, $Result.summary.strayBranches)
    Write-Host ""

    $sections = @(
        @{
            Title = '可立即清理（有 PR，bot 可直接执行）'
            Color = 'Green'
            Filter = { $_.category -eq 'auto-clean' -and $_.cleanupMode -eq 'pr' }
        },
        @{
            Title = '可立即清理（无 PR，需按命令手动执行）'
            Color = 'Green'
            Filter = { $_.category -eq 'auto-clean' -and $_.cleanupMode -eq 'manual' }
        },
        @{ Title = '需确认'; Color = 'Yellow'; Filter = { $_.category -eq 'needs-confirm' } },
        @{ Title = '失效登记（目录已不存在）'; Color = 'DarkYellow'; Filter = { $_.category -eq 'prunable' } },
        @{ Title = '活跃（无需动作）'; Color = 'Gray'; Filter = { $_.category -eq 'active' } }
    )

    foreach ($section in $sections) {
        $items = @($Result.worktrees | Where-Object $section.Filter)
        Write-Host ("== {0} ({1}) ==" -f $section.Title, $items.Count) -ForegroundColor $section.Color
        if ($items.Count -eq 0) {
            Write-Host "（无）"
            Write-Host ""
            continue
        }
        $i = 0
        foreach ($item in $items) {
            $i++
            $prText = 'no-pr'
            if ($item.pr) { $prText = ('PR #{0} {1}' -f $item.pr.number, $item.pr.state) }
            $sizeText = 'size=skipped'
            if ($null -ne $item.sizeMB) { $sizeText = ('size={0} MB' -f $item.sizeMB) }
            Write-Host ("{0}. {1} | branch={2} | {3} | dirty={4} | {5} | last={6}" -f `
                $i, (Split-Path -Leaf $item.path), $item.branch, $prText, $item.dirtyCount, $sizeText, $item.lastCommitAt)
            if ($item.reason) { Write-Host ("   reason: {0}" -f $item.reason) }
            foreach ($action in @($item.suggestedActions)) {
                Write-Host ("   action: {0}" -f $action)
            }
        }
        Write-Host ""
    }

    Write-Host ("== 孤儿目录 ({0}) ==" -f $Result.orphans.Count) -ForegroundColor Magenta
    if ($Result.orphans.Count -eq 0) {
        Write-Host "（无）"
    }
    else {
        $i = 0
        foreach ($orphan in $Result.orphans) {
            $i++
            $sizeText = 'size=skipped'
            if ($null -ne $orphan.sizeMB) { $sizeText = ('size={0} MB' -f $orphan.sizeMB) }
            Write-Host ("{0}. {1} | {2} | lastWrite={3}" -f $i, $orphan.path, $sizeText, $orphan.lastWrite)
            if ($orphan.sampleEntries.Count -gt 0) {
                Write-Host ("   sample: {0}" -f ($orphan.sampleEntries -join ', '))
            }
            foreach ($action in @($orphan.suggestedActions)) {
                Write-Host ("   action: {0}" -f $action)
            }
        }
    }
    Write-Host ""

    Write-Host ("== 无 worktree 的本地分支 ({0}) ==" -f $Result.strayBranches.Count) -ForegroundColor DarkCyan
    if ($Result.strayBranches.Count -eq 0) {
        Write-Host "（无）"
    }
    else {
        $i = 0
        foreach ($stray in $Result.strayBranches) {
            $i++
            $prText = 'no-pr'
            if ($stray.prNumber) { $prText = ('PR #{0} {1}' -f $stray.prNumber, $stray.prState) }
            Write-Host ("{0}. {1} | {2} | last={3}" -f $i, $stray.branch, $prText, $stray.lastCommitAt)
            foreach ($action in @($stray.suggestedActions)) {
                Write-Host ("   action: {0}" -f $action)
            }
        }
    }
    Write-Host ""
    Write-Host "本报告只读。清理必须由用户确认后执行。" -ForegroundColor Cyan
}

$exitCode = 0
$failure = $null
$result = $null
$resultJson = $null
$proxyState = Set-WorktreeProxy -Proxy $Proxy

try {
    try {
        $repoRoot = Get-WorktreeRepoRoot
    }
    catch {
        Throw-Audit 1 'git' $_.Exception.Message
    }

    if ($Fetch) {
        & git -C $repoRoot fetch origin --prune
        if ($LASTEXITCODE -ne 0) {
            Throw-Audit 1 'fetch' "git fetch origin --prune failed (exit $LASTEXITCODE)."
        }
    }

    try {
        $records = @(Get-WorktreeRecords -RepoRoot $repoRoot)
    }
    catch {
        Throw-Audit 1 'git' $_.Exception.Message
    }
    if ($records.Count -eq 0) {
        Throw-Audit 2 'parse' "git worktree list returned no worktrees."
    }

    $mainRecord = $records[0]
    $mainPathKey = Get-NormalizedPath $mainRecord.Path

    $ghPath = Resolve-GhPath
    if (-not $ghPath) {
        Throw-Audit 1 'gh' "gh CLI not found. Install GitHub CLI or add it to PATH."
    }

    try {
        $prData = Get-PrIndex -GhPath $ghPath -Limit $PrLimit
    }
    catch {
        Throw-Audit 1 'gh' $_.Exception.Message
    }
    if ($null -eq $prData) {
        Throw-Audit 2 'gh-output' "gh pr list produced no parseable result."
    }

    & git -C $repoRoot rev-parse --verify --quiet origin/main | Out-Null
    $hasOriginMain = ($LASTEXITCODE -eq 0)

    $worktrees = @()
    foreach ($record in $records) {
        $pathKey = Get-NormalizedPath $record.Path
        if ($pathKey -eq $mainPathKey) { continue }

        $item = [ordered]@{
            path = $record.Path
            branch = $record.Branch
            head = $record.Head
            detached = $record.Detached
            locked = $record.Locked
            dirty = $false
            dirtyCount = 0
            headMatchesPr = $null
            lastCommitAt = $null
            ageDays = $null
            sizeMB = $null
            category = ''
            cleanupMode = ''
            reason = ''
            pr = $null
            suggestedActions = @()
        }

        if ($record.Prunable -or -not (Test-Path -LiteralPath $record.Path)) {
            $item.category = 'prunable'
            $item.reason = 'registered worktree path no longer exists'
            $item.suggestedActions = @(
                "git worktree prune",
                "确认后：git branch -D $($record.Branch)"
            )
            $worktrees += [pscustomobject]$item
            continue
        }

        $lastCommitAt = Get-LastCommitIso -Path $record.Path
        $item.lastCommitAt = $lastCommitAt
        $item.ageDays = ConvertTo-AgeDays -Iso $lastCommitAt

        $statusLines = @(& git -C $record.Path status --porcelain)
        $statusOk = ($LASTEXITCODE -eq 0)
        $dirtyCount = 0
        if ($statusOk) {
            $dirtyCount = @($statusLines | Where-Object { $_ -and $_.ToString().Trim().Length -gt 0 }).Count
        }
        $item.dirtyCount = $dirtyCount
        $item.dirty = ($dirtyCount -gt 0)

        $prInfo = $null
        if ($record.Branch) {
            $prInfo = $prData.Index[$record.Branch]
        }
        if ($prInfo) {
            $item.pr = [ordered]@{
                number = $prInfo.number
                state = $prInfo.state
                title = $prInfo.title
                url = $prInfo.url
                headRefOid = $prInfo.headRefOid
            }
            if ($prInfo.headRefOid) {
                $item.headMatchesPr = ($record.Head -eq $prInfo.headRefOid)
            }
            else {
                $item.headMatchesPr = $false
            }
        }

        if ($record.Detached) {
            $item.category = 'needs-confirm'
            $item.reason = 'detached HEAD'
            $item.suggestedActions = @("确认无需保留后：git -C $($record.Path) log --oneline -5")
        }
        elseif ($record.Locked) {
            $item.category = 'needs-confirm'
            $item.reason = 'worktree is locked'
            $item.suggestedActions = @("git worktree unlock $($record.Path)  # 确认后再操作")
        }
        elseif (-not $statusOk) {
            $item.category = 'needs-confirm'
            $item.reason = 'git status failed; cannot verify cleanliness'
            $item.suggestedActions = @("git -C $($record.Path) status")
        }
        elseif ($null -ne $prInfo) {
            if ($prInfo.state -eq 'OPEN') {
                $item.category = 'active'
                $item.reason = 'PR is open'
            }
            elseif ($prInfo.state -eq 'MERGED') {
                if ($item.dirty) {
                    $item.category = 'needs-confirm'
                    $item.reason = 'PR merged but worktree is dirty'
                    $item.suggestedActions = @(
                        "git -C $($record.Path) status",
                        "提交或丢弃改动后再清理"
                    )
                }
                elseif (-not $item.headMatchesPr) {
                    $item.category = 'needs-confirm'
                    $item.reason = 'PR merged but local HEAD differs from PR headRefOid (possible post-merge commits)'
                    $item.suggestedActions = @(
                        "git -C $($record.Path) log --oneline $($prInfo.headRefOid)..HEAD",
                        "确认无遗漏后再清理"
                    )
                }
                else {
                    $item.category = 'auto-clean'
                    $item.cleanupMode = 'pr'
                    $item.reason = 'PR merged, worktree clean, HEAD matches PR'
                    $item.suggestedActions = @(
                        "powershell -ExecutionPolicy Bypass -File scripts/worktree-clean.ps1 -Pr $($prInfo.number) -DeleteRemote"
                    )
                }
            }
            elseif ($prInfo.state -eq 'CLOSED') {
                $item.category = 'needs-confirm'
                $item.reason = 'PR closed without merge'
                $item.suggestedActions = @(
                    "确认放弃后：powershell -ExecutionPolicy Bypass -File scripts/worktree-clean.ps1 -Pr $($prInfo.number) -AllowClosed -DeleteRemote"
                )
            }
            else {
                $item.category = 'needs-confirm'
                $item.reason = "unexpected PR state: $($prInfo.state)"
                $item.suggestedActions = @("gh pr view $($prInfo.number)")
            }
        }
        else {
            # An ancestor check is only meaningful when there is nothing local
            # to lose. A dirty worktree is never auto-clean.
            $isAncestor = $false
            if (-not $item.dirty -and $hasOriginMain -and $record.Branch) {
                & git -C $repoRoot merge-base --is-ancestor $record.Branch origin/main | Out-Null
                if ($LASTEXITCODE -eq 0) { $isAncestor = $true }
            }

            if ($item.dirty) {
                if ($null -ne $item.ageDays -and $item.ageDays -ge $StaleDays) {
                    $item.category = 'needs-confirm'
                    $item.reason = "no PR, uncommitted changes, last commit $($item.ageDays) days ago"
                    $item.suggestedActions = @("git -C $($record.Path) status")
                }
                else {
                    $item.category = 'active'
                    $item.reason = 'no PR; uncommitted work in progress'
                }
            }
            elseif ($isAncestor) {
                $item.category = 'auto-clean'
                $item.cleanupMode = 'manual'
                $item.reason = 'branch is fully contained in origin/main (no PR); worktree-clean.ps1 cannot clean it'
                $item.suggestedActions = @(
                    "git worktree remove $($record.Path)",
                    "git branch -D $($record.Branch)"
                )
            }
            elseif ($null -ne $item.ageDays -and $item.ageDays -ge $StaleDays) {
                $item.category = 'needs-confirm'
                $item.reason = "no PR and last commit is $($item.ageDays) days old"
                $item.suggestedActions = @(
                    "确认废弃后：git worktree remove $($record.Path)",
                    "确认废弃后：git branch -D $($record.Branch)"
                )
            }
            else {
                $item.category = 'active'
                $item.reason = 'no PR but recent commits'
            }
        }

        if (-not $SkipSize -and $item.category -ne 'active') {
            $item.sizeMB = Get-DirSizeMB -Path $record.Path
        }

        $worktrees += [pscustomobject]$item
    }

    # Orphaned `kg-*` directories: not registered, and no usable git dir.
    $parent = Split-Path -Parent $repoRoot
    $registered = @{}
    foreach ($record in $records) {
        $registered[(Get-NormalizedPath $record.Path)] = $true
    }

    $orphans = @()
    foreach ($dir in @(Get-ChildItem -LiteralPath $parent -Directory -Force -ErrorAction SilentlyContinue)) {
        if ($dir.Name -notlike 'kg-*') { continue }
        $key = Get-NormalizedPath $dir.FullName
        if ($registered.ContainsKey($key)) { continue }

        $usableGit = $false
        if (Test-Path -LiteralPath (Join-Path $dir.FullName '.git')) {
            & git -C $dir.FullName rev-parse --git-dir | Out-Null
            if ($LASTEXITCODE -eq 0) { $usableGit = $true }
        }
        if ($usableGit) { continue }

        $sizeMB = $null
        if (-not $SkipSize) {
            $sizeMB = Get-DirSizeMB -Path $dir.FullName
        }
        $sample = @(Get-ChildItem -LiteralPath $dir.FullName -Force -ErrorAction SilentlyContinue |
            Select-Object -First 5 -ExpandProperty Name)

        $orphans += [pscustomobject]@{
            path = $dir.FullName
            hasUsableGit = $false
            sizeMB = $sizeMB
            lastWrite = $dir.LastWriteTime.ToString('o')
            sampleEntries = $sample
            suggestedActions = @(
                "确认内容后删除：Remove-Item -LiteralPath `"$($dir.FullName)`" -Recurse"
            )
        }
    }

    # Local branches that no worktree has checked out.
    $checkedOut = @{}
    foreach ($record in $records) {
        if ($record.Branch) { $checkedOut[$record.Branch] = $true }
    }

    $strayBranches = @()
    $localBranches = @(& git -C $repoRoot for-each-ref --format='%(refname:short)' refs/heads)
    if ($LASTEXITCODE -ne 0) {
        Throw-Audit 1 'git' "git for-each-ref refs/heads failed."
    }
    foreach ($branch in $localBranches) {
        if (-not $branch) { continue }
        if ($branch -eq $mainRecord.Branch) { continue }
        if ($checkedOut.ContainsKey($branch)) { continue }

        $lastIso = $null
        $value = & git -C $repoRoot log -1 --format=%cI $branch
        if ($LASTEXITCODE -eq 0 -and $value) {
            $lastIso = ([string]($value | Select-Object -First 1)).Trim()
        }

        $prInfo = $prData.Index[$branch]
        $prNumber = $null
        $prState = $null
        if ($prInfo) {
            $prNumber = $prInfo.number
            $prState = $prInfo.state
        }

        $actions = @()
        if ($prState -eq 'MERGED') {
            $actions = @("确认后：git branch -D $branch")
        }
        elseif ($prState -eq 'CLOSED') {
            $actions = @("确认放弃后：git branch -D $branch")
        }
        elseif ($prState -eq 'OPEN') {
            $actions = @("PR 仍 OPEN，保留")
        }
        else {
            $age = ConvertTo-AgeDays -Iso $lastIso
            if ($null -ne $age -and $age -ge $StaleDays) {
                $actions = @("无 PR 且 $age 天未提交；确认废弃后：git branch -D $branch")
            }
            else {
                $actions = @("无 PR 但近期有提交，保留")
            }
        }

        $strayBranches += [pscustomobject]@{
            branch = $branch
            prNumber = $prNumber
            prState = $prState
            lastCommitAt = $lastIso
            suggestedActions = $actions
        }
    }

    $autoClean = @($worktrees | Where-Object { $_.category -eq 'auto-clean' }).Count
    $autoCleanPr = @($worktrees | Where-Object { $_.category -eq 'auto-clean' -and $_.cleanupMode -eq 'pr' }).Count
    $autoCleanManual = @($worktrees | Where-Object { $_.category -eq 'auto-clean' -and $_.cleanupMode -eq 'manual' }).Count
    $needsConfirm = @($worktrees | Where-Object { $_.category -eq 'needs-confirm' }).Count
    $active = @($worktrees | Where-Object { $_.category -eq 'active' }).Count
    $prunable = @($worktrees | Where-Object { $_.category -eq 'prunable' }).Count

    # Internal consistency: exit 2 means "do not trust this report".
    if (($autoClean + $needsConfirm + $active + $prunable) -ne $worktrees.Count) {
        Throw-Audit 2 'validate' "category counts do not add up to worktree count."
    }
    if (($autoCleanPr + $autoCleanManual) -ne $autoClean) {
        Throw-Audit 2 'validate' "auto-clean split does not add up to auto-clean count."
    }
    foreach ($item in $worktrees) {
        if (-not $item.path -or -not $item.category) {
            Throw-Audit 2 'validate' "a worktree entry is missing path or category."
        }
    }

    $result = [ordered]@{
        ok = $true
        generatedAt = (Get-Date).ToString('o')
        repo = $mainRecord.Path
        mainWorktree = $mainRecord.Path
        summary = [ordered]@{
            totalWorktrees = $worktrees.Count
            autoClean = $autoClean
            autoCleanPr = $autoCleanPr
            autoCleanManual = $autoCleanManual
            needsConfirm = $needsConfirm
            active = $active
            prunable = $prunable
            orphans = $orphans.Count
            strayBranches = $strayBranches.Count
            prsScanned = $prData.Count
            fetched = [bool]$Fetch
        }
        worktrees = @($worktrees)
        orphans = @($orphans)
        strayBranches = @($strayBranches)
    }

    $resultJson = ConvertTo-Json -InputObject $result -Depth 10
    Test-JsonArrayContract -Json $resultJson
}
catch {
    $code = 1
    $stage = 'unexpected'
    if ($null -ne $_.Exception.Data) {
        if ($_.Exception.Data.Contains('exitCode')) { $code = [int]$_.Exception.Data['exitCode'] }
        if ($_.Exception.Data.Contains('stage')) { $stage = [string]$_.Exception.Data['stage'] }
    }
    $failure = [ordered]@{
        ok = $false
        exitCode = $code
        stage = $stage
        error = $_.Exception.Message
        generatedAt = (Get-Date).ToString('o')
    }
    $exitCode = $code
}
finally {
    Restore-WorktreeProxy -State $proxyState
}

if ($null -ne $failure) {
    if ($Json) {
        Write-Output (ConvertTo-Json -InputObject $failure -Depth 6)
    }
    else {
        Write-Host ("[ERROR] stage={0} exit={1}: {2}" -f $failure.stage, $failure.exitCode, $failure.error) -ForegroundColor Red
    }
    [Console]::Error.WriteLine(("[worktree-audit] stage={0} exit={1}: {2}" -f $failure.stage, $failure.exitCode, $failure.error))
    exit $exitCode
}

if ($Json) {
    Write-Output $resultJson
}
else {
    Write-AuditText -Result $result
}
exit 0
