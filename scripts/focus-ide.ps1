# focus-ide.ps1 — 把微信开发者工具窗口置前
#
# 背景：nwjs 版微信开发者工具的截图依赖合成层，窗口被覆盖时
#       Page.captureScreenshot 会挂死不返回（报「daemon 响应超时」）。
#       每次调用 wechat_screenshot 前先跑本脚本。
#
# 用法:  powershell -ExecutionPolicy Bypass -File scripts/focus-ide.ps1
# 双实例时用 -Title 过滤目标窗口（子串匹配 MainWindowTitle，匹配在 PowerShell
# 进程内进行、不受终端编码影响）：
#   powershell ... focus-ide.ps1 -Title 家长端
#   powershell ... focus-ide.ps1 -Title 老师端

param(
    # 1 = SW_RESTORE（从最小化恢复并置前）
    [int]$ShowCmd = 9,
    # 可选：按窗口标题子串筛选目标实例（双 IDE 并存时必须指定，
    # 否则 Select-Object -First 1 会取到错的窗口，目标仍被遮挡 → 截图挂死）
    [string]$Title = ""
)

Add-Type @"
using System;
using System.Runtime.InteropServices;
public class FocusIde {
    [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr hWnd);
    [DllImport("user32.dll")] public static extern bool ShowWindow(IntPtr hWnd, int nCmdShow);
    [DllImport("user32.dll")] public static extern bool IsIconic(IntPtr hWnd);
    [DllImport("user32.dll")] public static extern IntPtr GetForegroundWindow();
}
"@

$candidates = Get-Process -Name "wechatdevtools" -ErrorAction SilentlyContinue |
        Where-Object { $_.MainWindowHandle -ne 0 }
if ($Title -ne "") {
    $candidates = $candidates | Where-Object { $_.MainWindowTitle -like "*$Title*" }
}
$proc = $candidates | Select-Object -First 1

if (-not $proc) {
    Write-Output "FAIL: WeChat DevTools window not found (is the IDE running?)"
    exit 1
}

$hwnd = $proc.MainWindowHandle
[void][FocusIde]::ShowWindow($hwnd, $ShowCmd)
[void][FocusIde]::SetForegroundWindow($hwnd)

# 等待焦点真正生效，再返回给调用方
Start-Sleep -Milliseconds 600

$fg = [FocusIde]::GetForegroundWindow()
if ($fg -eq $hwnd) {
    Write-Output ("OK: focused -> " + $proc.MainWindowTitle)
    exit 0
} else {
    # 有时前台切换有延迟，再试一次
    [void][FocusIde]::SetForegroundWindow($hwnd)
    Start-Sleep -Milliseconds 600
    if ([FocusIde]::GetForegroundWindow() -eq $hwnd) {
        Write-Output ("OK: focused -> " + $proc.MainWindowTitle)
        exit 0
    }
    Write-Output ("WARN: window activated but not foreground (title=" + $proc.MainWindowTitle + ")")
    exit 2
}
