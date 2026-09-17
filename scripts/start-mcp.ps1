$buildPath = 'tmp'
if (!(Test-Path $buildPath)) {
mkdir -p $buildPath
}

go build -o "$buildPath\mcp.exe" .\build_specs.go
"$buildPath\mcp.exe"