$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot
go run -tags resources gen_resources.go icon.go
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
if (Get-Process momontry -ErrorAction SilentlyContinue) {
    Stop-Process -Name momontry -Force
    Start-Sleep -Milliseconds 400
}
go build -ldflags "-H windowsgui -s -w" -o momontry.exe .
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
Write-Host "built $PSScriptRoot\momontry.exe"
