cd "$(dirname "$0")"/../ || exit

if [ "$1" == "debug" ]; then
	echo Compiling app dev command for debug...
	go build -gcflags="all=-N -l" -o dist/bin/ds-dev ./cmd/ds-dev
else
	go build -ldflags="-X main.cmd_version=`git describe --tags --dirty`" -o dist/bin/ds-dev ./cmd/ds-dev
fi