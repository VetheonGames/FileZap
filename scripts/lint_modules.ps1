# Script to run golangci-lint on all modules
$ErrorActionPreference = "Stop"
$modules = @(".", "Client", "Divider", "Network Core", "Reconstructor")
$root = Get-Location
$config = (Resolve-Path ".golangci.yml").Path
$failed = 0

Write-Host "`nRunning linter checks..."

foreach ($m in $modules) {
    Write-Host "`nModule: $m"
    
    $path = Join-Path $root $m
    if (Test-Path $path) {
        Set-Location $path
        
        $result = & golangci-lint run --config="$config" --timeout=5m 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Host "PASS" -ForegroundColor Green
        } else {
            Write-Host "FAIL" -ForegroundColor Red
            Write-Output $result
            $failed++
        }
        
        Set-Location $root
    } else {
        Write-Host "ERROR: Path not found" -ForegroundColor Red
        $failed++
    }
}

if ($failed -eq 0) {
    Write-Host "`nAll modules passed" -ForegroundColor Green
} else {
    Write-Host "`n$failed module(s) failed" -ForegroundColor Red
    Write-Host "To fix:"
    Write-Host "1. Run go fmt ./..."
    Write-Host "2. Check .golangci.yml"
    Write-Host "3. Fix error checks"
    Write-Host "4. Update APIs"
    Write-Host "5. Clean unused code"
}

exit $failed
