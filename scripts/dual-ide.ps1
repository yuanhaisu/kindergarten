# dual-ide.ps1 — 同时启动两个微信开发者工具实例，分别驱动老师端与家长端
#
# 背景：微信开发者工具的登录态是「单账号单客户端」。同一账号开两个实例会互踢。
#       用**两个不同微信号**登录两个实例（各自独立 user-data-dir），即可双端并行。
#
# 前置条件（一次性人工配置）：
#   1. 第二个微信号已加为老师端小程序的开发者（mp.weixin.qq.com → 成员管理）
#   2. 两个实例各自登录过对应微信号（首次运行会弹二维码）
#
# 端口分配：
#   实例 A（主账号 → 家长端）  CDP 9222  服务端口 43534  自动化 9420
#   实例 B（第二账号 → 老师端）CDP 9223  服务端口 44500  自动化 9421
#
# 用法：
#   powershell -ExecutionPolicy Bypass -File scripts/dual-ide.ps1          # 启动两端
#   powershell -ExecutionPolicy Bypass -File scripts/dual-ide.ps1 -Auto    # 启动并开自动化
#   powershell -ExecutionPolicy Bypass -File scripts/dual-ide.ps1 -Status  # 只看状态
#
# ⚠️ CLI 端口发现写死读 %LOCALAPPDATA%\微信开发者工具\User Data\<hash>\Default\.ide
#    本脚本通过改写该文件在两个实例间切换 CLI 目标，调用 cli.bat 前须先 setIdePort。

param(
    [switch]$Auto,
    [switch]$Status,
    [switch]$Kill
)

$ErrorActionPreference = "Stop"

$IDE      = "D:\Program Files (x86)\微信web开发者工具"
$EXE      = "$IDE\wechatdevtools.exe"
$PKG      = "$IDE\code\package.nw"
$CLI      = "$IDE\cli.bat"
$REPO     = "D:\yuanhaisu\personal_projects\kindergarten"
$TEACHER  = "$REPO\teacher\dist\dev\mp-weixin"
$PARENT   = "$REPO\parent\dist\dev\mp-weixin"

# 端口分配（与 CLAUDE.md 保持一致）
$PORTS = @{
    A = @{ Cdp = 9222; Service = 43534; Remote = 3799; Auto = 9420; Udd = "C:\Users\33139\AppData\Local\微信开发者工具\User Data\e4658fad35ce6c95a73c611dacbb7ff0"; Project = $PARENT;  Label = "家长端/主账号" }
    B = @{ Cdp = 9223; Service = 44500; Remote = 3899; Auto = 9421; Udd = "C:\Users\33139\AppData\Local\Temp\wxide-teacher-udd";                                  Project = $TEACHER; Label = "老师端/第二账号" }
}

# CLI 的 userDirPath 写死为 <UDD标准路径>/Default，不认 --user-data-dir
$CLI_IDE_FILE = "C:\Users\33139\AppData\Local\微信开发者工具\User Data\e4658fad35ce6c95a73c611dacbb7ff0\Default\.ide"

function Set-IdePort([int]$Port) {
    Set-Content -Path $CLI_IDE_FILE -Value "$Port" -NoNewline -Encoding ascii
}

function Get-Status {
    foreach ($k in @("A", "B")) {
        $p = $PORTS[$k]
        $cdp = netstat -ano | Select-String "LISTENING" | Select-String ":$($p.Cdp)\s"
        $svc = netstat -ano | Select-String "LISTENING" | Select-String ":$($p.Service)\s"
        $auto = netstat -ano | Select-String "LISTENING" | Select-String ":$($p.Auto)\s"
        $win = Get-Process -Name "wechatdevtools" -ErrorAction SilentlyContinue |
               Where-Object { $_.MainWindowTitle -match "幼儿园" } |
               Select-Object -ExpandProperty MainWindowTitle
        Write-Host ("[{0}] {1}" -f $k, $p.Label)
        Write-Host ("    CDP {0}: {1}" -f $p.Cdp,     $(if ($cdp) { "OK" } else { "DOWN" }))
        Write-Host ("    服务 {0}: {1}" -f $p.Service, $(if ($svc) { "OK" } else { "DOWN" }))
        Write-Host ("    自动化 {0}: {1}" -f $p.Auto, $(if ($auto) { "OK" } else { "DOWN" }))
    }
    Write-Host "窗口标题: $($win -join ' | ')"
}

function Start-Instance($k) {
    $p = $PORTS[$k]
    Write-Host "启动实例 $k ($($p.Label))  CDP=$($p.Cdp) 服务=$($p.Service)"
    $args = "`"$PKG`" `"$($p.Project)`" --cli --enable-service-port " +
            "--ide-http-port $($p.Service) --remote-port $($p.Remote) " +
            "--remote-debugging-port=$($p.Cdp) `"`" `"`" `"`" " +
            "`"--user-data-dir=$($p.Udd)`" `"--package-dir=$PKG`" " +
            "--app-session-id=dual$k-$(Get-Random)"
    $psi = New-Object System.Diagnostics.ProcessStartInfo
    $psi.FileName = $EXE
    $psi.Arguments = $args
    $psi.UseShellExecute = $false
    $proc = [System.Diagnostics.Process]::Start($psi)
    Write-Host "  PID=$($proc.Id)"
    return $proc
}

function Enable-Auto($k) {
    $p = $PORTS[$k]
    Write-Host "开自动化 实例 $k  auto-port=$($p.Auto)"
    Set-IdePort $p.Service
    & $CLI auto --port $p.Service --project $p.Project --auto-port $p.Auto --trust-project
}

if ($Kill) {
    Write-Host "关闭所有微信开发者工具进程..."
    Get-Process | Where-Object { $_.Name -match "wechatdevtools|微信开发者工具" } |
        ForEach-Object { Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue }
    # 残留的 cli login 会占 3799 端口并干扰下次启动，一并清掉
    Get-CimInstance Win32_Process | Where-Object {
        $_.Name -eq "node.exe" -and $_.CommandLine -like "*wechatwebdevtools*cli*"
    } | ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }
    Write-Host "已清理。"
    return
}

if ($Status) {
    Get-Status
    return
}

# 启动
Get-Process | Where-Object { $_.Name -match "wechatdevtools|微信开发者工具" } | ForEach-Object {
    Write-Host "警告：已有 DevTools 进程 $($_.Id)，先 -Kill 清理"
}
Start-Instance "B" | Out-Null
Start-Sleep -Seconds 10
Start-Instance "A" | Out-Null
Start-Sleep -Seconds 10

if ($Auto) {
    Enable-Auto "A"
    Enable-Auto "B"
}

Write-Host ""
Get-Status
Write-Host ""
Write-Host "MCP 调用时指定端口："
Write-Host "  老师端 wechat_automator(auto_port=9421)  wechat_navigate(cdp_port=9223)"
Write-Host "  家长端 wechat_automator(auto_port=9420)  wechat_navigate(cdp_port=9222)"
