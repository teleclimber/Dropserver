cd "$(dirname "$0")"/../ || exit

# this moves assets from their source or build output folders to dist.
# This will be fully obsolete when all resources are embedded into ds-host executable.

mkdir -p dist/resources/

cp -r resources/* dist/resources/

