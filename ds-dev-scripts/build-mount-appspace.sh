cd "$(dirname "$0")"/../ || exit

GOOS=linux go build -o bin/ds-mount-appspace cmd/ds-mount-appspace/ds-mount-appspace.go