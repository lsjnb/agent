set GOOS=linux
set GOOS=windows
set GOARCH=amd64
go build -ldflags "-s -w -X main.version=1.0.0 -X main.arch=amd64" -o nezha-agent.exe ./cmd/agent


$env:GOOS="linux"
$env:GOARCH="amd64"
go build -ldflags "-s -w -X main.version=1.0.0 -X main.arch=amd64" -o nezha-agent.exe ./cmd/agent
