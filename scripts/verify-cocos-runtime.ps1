param(
    [Parameter(Mandatory = $true)][string]$CreatorExe,
    [Parameter(Mandatory = $true)][string]$OutputRoot,
    [ValidateSet('web-desktop', 'wechatgame', 'bytedance-mini-game')][string]$Platform = 'web-desktop'
)
$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path $PSScriptRoot -Parent
$project = Join-Path $repoRoot 'clients/fair-doudizhu-cocos'
$evidence = Join-Path $repoRoot '.runtime/goal0045-cocos'
New-Item -ItemType Directory -Force -Path $evidence | Out-Null
New-Item -ItemType Directory -Force -Path $OutputRoot | Out-Null
$scene = (Get-Content (Join-Path $project 'assets/scenes/Entry.scene.meta') -Raw | ConvertFrom-Json).uuid
$options = @{ platform = $Platform; debug = $true; outputName = $Platform; buildPath = $OutputRoot; startScene = $scene }
if ($Platform -ne 'web-desktop') {
    $options.packages = @{ $Platform = @{ orientation = 'landscape' } }
}
$config = Join-Path $evidence "creator-$Platform-options.json"
$options | ConvertTo-Json -Depth 5 | Set-Content $config -Encoding utf8
$report = @{ creator = '3.8.8-win-121518'; platform = $Platform; scene = $scene; result = 'failed' }
try {
    if (-not (Test-Path $CreatorExe)) { throw 'Creator executable is missing' }
    $arguments = @('--no-sandbox', '--disable-gpu', '--project', "`"$project`"", '--build', "`"configPath=$config`"")
    $process = Start-Process -FilePath $CreatorExe -ArgumentList $arguments -WorkingDirectory $project -PassThru `
        -RedirectStandardOutput (Join-Path $evidence "creator.$Platform.stdout.log") `
        -RedirectStandardError (Join-Path $evidence "creator.$Platform.stderr.log")
    if (-not $process.WaitForExit(600000)) {
        & taskkill /PID $process.Id /T /F | Out-Null
        throw 'Creator exceeded the 10-minute build deadline; inspect the preserved logs'
    }
    $report.exitCode = $process.ExitCode
    # Creator documents 36 as success; treating ordinary process exit 0 as build success is incorrect.
    if ($process.ExitCode -ne 36) { throw "Creator build exited $($process.ExitCode), expected 36" }
    $entry = if ($Platform -eq 'web-desktop') { 'index.html' } else { 'game.js' }
    if (-not (Test-Path (Join-Path $OutputRoot "$Platform/$entry"))) { throw "Creator did not produce $Platform/$entry" }
    if ($Platform -ne 'web-desktop') {
        $gameConfig = Get-Content (Join-Path $OutputRoot "$Platform/game.json") -Raw | ConvertFrom-Json
        if ($gameConfig.deviceOrientation -ne 'landscape') { throw "$Platform did not export the required landscape orientation" }
        $report.orientation = $gameConfig.deviceOrientation
    }
    Compress-Archive -Path (Join-Path $OutputRoot "$Platform/*") -DestinationPath (Join-Path $evidence "$Platform.zip") -Force
    $report.result = 'passed'
} catch {
    $report.error = $_.Exception.Message
    throw
} finally {
    $report.completedAt = (Get-Date).ToUniversalTime().ToString('o')
    $report | ConvertTo-Json -Depth 5 | Set-Content (Join-Path $evidence "creator-$Platform-build.json") -Encoding utf8
}
