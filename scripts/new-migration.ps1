param(
    [Parameter(Mandatory=$true)]
    [string]$MigrationName
)
. "$PSScriptRoot\load-env.ps1"
Set-Environment
$migrationsDir = join-path "$PSScriptRoot" -ChildPath ".." -AdditionalChildPath "infra", "scripts", "migrations" -Resolve

if (-not(test-path $migrationsDir)) {
    write-host "Migrations directory not found at $migrationsDir. Exiting"
    exit
}

migrate create -ext sql "$migrationsDir" -seq "$MigrationName"