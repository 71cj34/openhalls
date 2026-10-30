$env:GOOS = "linux"; $env:GOARCH = "amd64"
go build -C ./src/hallview -o ../../hallview
Remove-Item Env:\GOOS, Env:\GOARCH