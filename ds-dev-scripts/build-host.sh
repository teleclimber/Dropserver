cd "$(dirname "$0")"/../ || exit

GOOS=linux go build -o bin/ds-host cmd/ds-host/ds-host.go