param(
    [Parameter(Mandatory = $true)][string]$CreatorExe,
    [Parameter(Mandatory = $true)][string]$OutputRoot
)
$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path $PSScriptRoot -Parent
$project = Join-Path $repoRoot 'clients/fair-doudizhu-cocos'
$evidence = Join-Path $repoRoot '.runtime/goal0045-cocos'
New-Item -ItemType Directory -Force -Path $evidence | Out-Null
New-Item -ItemType Directory -Force -Path $OutputRoot | Out-Null
$scene = (Get-Content (Join-Path $project 'assets/scenes/Entry.scene.meta') -Raw | ConvertFrom-Json).uuid
$options = "platform=web-desktop;debug=true;outputName=web-desktop;buildPath=$OutputRoot;startScene=$scene"
$report = @{ creator = '3.8.8-win-121518'; platform = 'web-desktop'; scene = $scene; result = 'failed' }
try {
    if (-not (Test-Path $CreatorExe)) { throw 'Creator executable is missing' }
    $arguments = @('--no-sandbox', '--disable-gpu', '--project', "`"$project`"", '--build', "`"$options`"")
    $process = Start-Process -FilePath $CreatorExe -ArgumentList $arguments -WorkingDirectory $project -PassThru `
        -RedirectStandardOutput (Join-Path $evidence 'creator.stdout.log') `
        -RedirectStandardError (Join-Path $evidence 'creator.stderr.log')
    if (-not $process.WaitForExit(600000)) {
        & taskkill /PID $process.Id /T /F | Out-Null
        throw 'Creator exceeded the 10-minute build deadline; inspect the preserved logs'
    }
    $report.exitCode = $process.ExitCode
    # Creator documents 36 as success; treating ordinary process exit 0 as build success is incorrect.
    if ($process.ExitCode -ne 36) { throw "Creator build exited $($process.ExitCode), expected 36" }
    $index = Join-Path $OutputRoot 'web-desktop/index.html'
    if (-not (Test-Path $index)) { throw 'Creator did not produce the Web entry artifact' }
    Compress-Archive -Path (Join-Path $OutputRoot 'web-desktop/*') -DestinationPath (Join-Path $evidence 'web-desktop.zip') -Force
    $report.result = 'passed'
} catch {
    $report.error = $_.Exception.Message
    throw
} finally {
    $report.completedAt = (Get-Date).ToUniversalTime().ToString('o')
    $report | ConvertTo-Json -Depth 5 | Set-Content (Join-Path $evidence 'creator-build.json') -Encoding utf8
}
