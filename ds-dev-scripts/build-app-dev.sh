cd "$(dirname "$0")"/../ || exit

# run pkger to bundle static asset files
pkger

if [ "$1" == "debug" ]; then
	echo Compiling app dev command for debug...
	go build -gcflags="all=-N -l" -o dist/bin/ds-dev cmd/ds-dev/ds-dev.go
else
	go build -o dist/bin/ds-dev ./cmd/ds-dev
fi