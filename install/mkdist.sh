cd "$(dirname "$0")"/../ || exit

# this moves assets from their source or build output folders to dist.

mkdir -p dist/resources/webpack-html/
mkdir -p dist/static/

# move ./install/ to /dist/install ?
# or at least some parts of install should make it in dist.
# do that later.

cp -r resources/* dist/resources/

cp -r public-static/* dist/static/

cp -r frontend/dist/resources/* dist/resources/webpack-html/
cp -r frontend/dist/static/* dist/static/

