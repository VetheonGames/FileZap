# Get all go.mod files recursively
$goModFiles = Get-ChildItem -Path "." -Filter "go.mod" -Recurse

# For each go.mod file found
foreach ($file in $goModFiles) {
    Write-Host "Tidying module in: $($file.DirectoryName)"
    
    # Change to the directory containing go.mod
    Push-Location $file.DirectoryName
    
    # Run go mod tidy
    go mod tidy
    
    # Return to original directory
    Pop-Location
}

Write-Host "Finished tidying all modules"
