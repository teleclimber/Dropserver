cd "$(dirname "$0")"/../ || exit

go build -o dist/bin/ds-mount-appspace cmd/ds-mount-appspace/ds-mount-appspace.go