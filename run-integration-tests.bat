:: we can't use makefile for windows because it depends on CI makefile which depends on shell

:: build docker image
cd cmd/srcd-server
docker build -t srcd/cli-daemon -f Dockerfile ../..
cd ../..

:: compile engine-cli
if not exist build mkdir build
if not exist build/engine_windows_amd64 mkdir build/engine_windows_amd64
go build -o build/engine_windows_amd64/srcd.exe ./cmd/srcd

:: run tests
set SRCD_BIN=../../../build/engine_windows_amd64/srcd.exe
go test -parallel 1 -count 1 -tags=integration github.com/src-d/engine/cmd/srcd/cmd/ -v
