cd "$(dirname "$0")"/../ || exit

GOOS=linux CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o bin/ds-trusted cmd/ds-trusted/ds-trusted.go