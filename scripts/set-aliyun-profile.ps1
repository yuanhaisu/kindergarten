param(
    [string]$Profile = "mcp-readonly",
    [string]$Region = "cn-guangzhou",
    [string]$ConfigPath = "$env:USERPROFILE\.aliyun\config.json",
    [string]$AccessKeyId,
    [string]$AccessKeySecret
)

$ErrorActionPreference = "Stop"

if (-not $AccessKeyId) {
    $AccessKeyId = (Read-Host "AccessKey ID").Trim()
}
if (-not $AccessKeySecret) {
    $secure = Read-Host -AsSecureString "AccessKey Secret"
    $bstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($secure)
    try {
        $AccessKeySecret = [Runtime.InteropServices.Marshal]::PtrToStringAuto($bstr)
    }
    finally {
        [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($bstr)
    }
}

if ([string]::IsNullOrWhiteSpace($AccessKeyId) -or [string]::IsNullOrWhiteSpace($AccessKeySecret)) {
    Write-Error "AccessKey ID and Secret must not be empty."
    exit 1
}

$dir = Split-Path -Parent $ConfigPath
if (-not (Test-Path $dir)) {
    New-Item -ItemType Directory -Force -Path $dir | Out-Null
}

$profiles = @()
$metaPath = ""
if (Test-Path $ConfigPath) {
    try {
        $existing = Get-Content -Raw -Path $ConfigPath | ConvertFrom-Json
        if ($existing.profiles) { $profiles = @($existing.profiles) }
        if ($existing.meta_path) { $metaPath = $existing.meta_path }
    }
    catch {
        Write-Warning "Existing config is not valid JSON; it will be replaced."
    }
}

$profiles = @($profiles | Where-Object { $_.name -ne $Profile })
$newProfile = [pscustomobject]@{
    name              = $Profile
    mode              = "AK"
    access_key_id     = $AccessKeyId
    access_key_secret = $AccessKeySecret
    region_id         = $Region
    output_format     = "json"
    language          = "en"
}
$profiles += $newProfile

$config = [pscustomobject]@{
    current   = $Profile
    profiles  = $profiles
    meta_path = $metaPath
}

$json = $config | ConvertTo-Json -Depth 10
[System.IO.File]::WriteAllText($ConfigPath, $json, (New-Object System.Text.UTF8Encoding($false)))

$masked = if ($AccessKeyId.Length -gt 8) {
    $AccessKeyId.Substring(0, 4) + "****" + $AccessKeyId.Substring($AccessKeyId.Length - 4)
}
else {
    "****"
}
Write-Host "Saved profile '$Profile' to $ConfigPath (AK: $masked, region: $Region)"

if (Get-Command icacls.exe -ErrorAction SilentlyContinue) {
    $grant = "$($env:USERNAME):F"
    & icacls.exe $ConfigPath /inheritance:r /grant:r $grant | Out-Null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "ACL tightened to $env:USERNAME"
    }
    else {
        Write-Warning "icacls exited with $LASTEXITCODE; check file permissions manually."
    }
}
