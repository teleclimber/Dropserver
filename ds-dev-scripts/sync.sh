cd "$(dirname "$0")"/../ || exit

rsync -rt --itemize-changes --exclude=.git/ --exclude=.DS_Store --exclude=/dist/ --exclude=frontend/node_modules/ . developer@10.211.55.9:~/dropserver-code/

#--modify-window=5