cd "$(dirname "$0")"/../ || exit

if [ "$1" == "debug" ]; then
	echo Compiling for debug...
	go build -gcflags="all=-N -l" -o dist/bin/ds-host ./cmd/ds-host
else
	go build -ldflags="-X main.cmd_version=`git describe --tags --dirty`" -o dist/bin/ds-host ./cmd/ds-host
fi
