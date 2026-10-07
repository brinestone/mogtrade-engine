$migrationsDir = Join-Path "$PSScriptRoot" -ChildPath ".." -AdditionalChildPath "infra", "scripts", "migrations" -Resolve
$envPath = Join-Path "$PSScriptRoot" -ChildPath ".." -AdditionalChildPath "app", "cmd", ".env" -Resolve
if (-not(test-path $migrationsDir )) {
    Write-Host "Migrations directory not found at $migrationsDir. Exiting"
    exit
}
if (test-path $envPath) {
    get-content $envPath | ForEach-Object {
        if ($_ -and $_ -notmatch '^\s*#') {
            $name, $value = $_ -split '=', 2
            $name = $name.Trim()
            $value = $value.Trim().Trim('"', "'")
            [Environment]::SetEnvironmentVariable($name, $value, 'Process')
        }
    } 
    Write-Host "Successfully loaded environment variables from .env" -ForegroundColor Green
}
else {
    Write-Warning ".env file not found. Aborting"
    exit
}

Write-Host "Running migrations"
$dbUrl = $env:DB_URL
Write-Host $dburl
Write-Host $migrationsDir
Write-Host $envPath
migrate.exe -database "$dbUrl" -path "$migrationsDir" up;