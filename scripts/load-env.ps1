$envPath = join-path "$PSScriptRoot" -ChildPath ".." -AdditionalChildPath "app", "cmd", ".env" -Resolve

function Set-Environment {
    if (-not(test-path $envPath)) {
        write-host ".env file not found. Exiting"
        exit
    }

    get-content $envPath | foreach-object {
        if ($_ -and $_ -notmatch '^\s*#') {
            $name, $value = $_ -split '=', 2
            $name = $name.Trim()
            $value = $value.Trim().Trim('"', "'")
            [Environment]::SetEnvironmentVariable($name, $value, 'Process')
        }
    }

    write-host "successfully loaded environment variables from .env" -ForegroundColor Green
}