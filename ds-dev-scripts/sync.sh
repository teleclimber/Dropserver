cd "$(dirname "$0")"/../ || exit

# rsync -r install/ developer@10.211.55.9:~/ds-files/install/
# rsync -r bin/ developer@10.211.55.9:~/ds-files/bin/

rsync -rt --itemize-changes --exclude=.git/ --exclude=.DS_Store --exclude=bin/ --exclude=frontend/node_modules/ . developer@10.211.55.9:~/go/src/github.com/teleclimber/DropServer/

#--modify-window=5