cd "$(dirname "$0")"/../ || exit

rsync -rt --itemize-changes --exclude=.git/ --exclude=.DS_Store --exclude=/dist/ --exclude=node_modules --delete . developer@192.168.1.53:~/dropserver-code/

#--modify-window=5