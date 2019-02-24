cd "$(dirname "$0")"/../ || exit

rsync -r install/ developer@10.211.55.9:~/ds-files/install/
rsync -r bin/ developer@10.211.55.9:~/ds-files/bin/
