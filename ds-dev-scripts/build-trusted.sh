cd "$(dirname "$0")"/../ || exit

CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o dist/bin/ds-trusted cmd/ds-trusted/ds-trusted.go