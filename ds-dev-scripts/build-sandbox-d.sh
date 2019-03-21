cd "$(dirname "$0")"/../ || exit

# GOOS=linux go build -o bin/ds-recycler-d cmd/ds-recycler-d/ds-recycler-d.go

CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o bin/ds-sandbox-d cmd/ds-sandbox-d/ds-sandbox-d.go