. "$PSScriptRoot\load-env.ps1"
Set-Environment
$migrationsDir = Join-Path "$PSScriptRoot" -ChildPath ".." -AdditionalChildPath "infra", "scripts", "migrations" -Resolve
if (-not(test-path $migrationsDir )) {
    Write-Host "Migrations directory not found at $migrationsDir. Exiting"
    exit
}

Write-Host "Running migrations"
$dbUrl = $env:DB_URL
Write-Host $dburl
Write-Host $migrationsDir
Write-Host $envPath
migrate.exe -database "$dbUrl" -path "$migrationsDir" up;