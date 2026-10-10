[CmdletBinding()]
param(
    [string]$GoExecutable = '',
    [switch]$Restart
)

$ErrorActionPreference = 'Stop'
$backendRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$debugBinary = Join-Path $backendRoot 'bin\meowhome-debug.exe'

if (-not (Test-Path -LiteralPath (Join-Path $backendRoot '.env') -PathType Leaf)) {
    throw 'Missing backend/.env. Configure the local backend before starting it.'
}

if (-not $GoExecutable) {
    $userGo = Join-Path $env:USERPROFILE 'go\bin\go.exe'
    if (Test-Path -LiteralPath $userGo -PathType Leaf) {
        $GoExecutable = $userGo
    } else {
        $GoExecutable = (Get-Command go.exe -ErrorAction Stop).Source
    }
}
if (-not (Test-Path -LiteralPath $GoExecutable -PathType Leaf)) {
    throw 'Go executable not found. Pass -GoExecutable with the installed go.exe path.'
}

# Only restart the debug executable built inside this backend directory.
# Other projects, go.exe processes and unrelated services are not stopped.
$debugProcesses = @(Get-Process -Name 'meowhome-debug' -ErrorAction SilentlyContinue |
    Where-Object { $_.Path -eq $debugBinary })
if ($debugProcesses.Count -gt 0) {
    if (-not $Restart) {
        Write-Host 'The project debug backend is running. Use -Restart to run it in this terminal.'
        return
    }
    foreach ($debugProcess in $debugProcesses) {
        Stop-Process -Id $debugProcess.Id -ErrorAction Stop
        $debugProcess.WaitForExit()
    }
}

$previousGoPath = $env:GOPATH
Push-Location -LiteralPath $backendRoot
try {
    if (-not $env:GOPATH) {
        $env:GOPATH = Join-Path $env:USERPROFILE 'go-work'
    }
    Write-Host 'Starting MeowHome. Keep this terminal open during phone debugging.'
    Write-Host 'Close the server with Ctrl+C. Configuration is read from backend/.env.'
    & $GoExecutable run ./cmd/server
    if ($LASTEXITCODE -ne 0) {
        throw "Backend exited with code $LASTEXITCODE. Check the server output above."
    }
} finally {
    $env:GOPATH = $previousGoPath
    Pop-Location
}
