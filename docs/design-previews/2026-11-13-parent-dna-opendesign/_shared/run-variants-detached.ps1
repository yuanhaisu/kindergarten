$ErrorActionPreference = "Continue"
Set-Location "D:\yuanhaisu\personal_projects\kindergarten\docs\design-previews\2026-11-13-parent-dna-opendesign"

$log = Join-Path (Get-Location) "_shared\logs-variants\detached-current.log"
$pidFile = Join-Path (Get-Location) "_shared\logs-variants\detached-current.pid"
$PID | Out-File -FilePath $pidFile -Encoding ascii
"start $(Get-Date -Format o) pid=$PID" | Out-File -FilePath $log -Encoding utf8

node _shared/run-variants.mjs --all *>> $log
"variants done $(Get-Date -Format o)" | Out-File -FilePath $log -Append -Encoding utf8

node _shared/check-variants.mjs *>> $log
"check done $(Get-Date -Format o)" | Out-File -FilePath $log -Append -Encoding utf8

node _shared/build-variants-gallery.mjs *>> $log
"gallery done $(Get-Date -Format o)" | Out-File -FilePath $log -Append -Encoding utf8

"all done $(Get-Date -Format o)" | Out-File -FilePath $log -Append -Encoding utf8
