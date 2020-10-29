package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/models/appfilesmodel"
)

//ignorePaths is a set of paths relative to the root of the app that should be ignored
var ignorePaths = []string{
	".git",
}

// DevAppWatcher reloads the app files as needed and sends events
type DevAppWatcher struct {
	AppFilesModel    *appfilesmodel.AppFilesModel
	DevAppModel      *DevAppModel
	DevAppspaceModel *DevAppspaceModel
	AppVersionEvents interface {
		Send(domain.AppID)
	}

	watcher     *fsnotify.Watcher
	ignorePaths []string
}

// Start loads the appfiles and launches file watching
func (w *DevAppWatcher) Start(appPath string) {
	w.load()

	w.ignorePaths = make([]string, len(ignorePaths))
	for i, p := range ignorePaths {
		w.ignorePaths[i] = filepath.Join(appPath, p)
	}

	go w.watch(appPath)
}
func (w *DevAppWatcher) load() {
	appFilesMeta, dsErr := w.AppFilesModel.ReadMeta("")
	if dsErr != nil {
		panic(dsErr.ToStandard().Error())
	}

	w.DevAppModel.App = domain.App{
		OwnerID: ownerID,
		AppID:   appID,
		Created: time.Now(),
		Name:    appFilesMeta.AppName}

	w.DevAppModel.Ver = domain.AppVersion{
		AppID:       appID,
		AppName:     appFilesMeta.AppName,
		Version:     appFilesMeta.AppVersion,
		Schema:      appFilesMeta.SchemaVersion,
		Created:     time.Now(),
		LocationKey: ""}

	// Need to update appspace so that the app version is reflected
	w.DevAppspaceModel.Appspace.AppVersion = appFilesMeta.AppVersion

	w.AppVersionEvents.Send(appID)
}

func (w *DevAppWatcher) filesChanged() {
	// TODO really need to throttle this function call!
	w.load()
}

func (w *DevAppWatcher) watch(appPath string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	w.watcher = watcher

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					fileInfo, err := os.Stat(event.Name)
					if err != nil {
						panic(err) //deal with this if we hit it
					}
					if fileInfo.IsDir() {
						w.watchDir(event.Name)
					}
					w.filesChanged()
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					w.filesChanged()
				}
				if event.Op&fsnotify.Remove == fsnotify.Remove {
					// remove watchers?
					w.filesChanged()
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = w.watchDir(appPath)
	if err != nil {
		log.Fatal(err)
	}
}

func (w *DevAppWatcher) watchDir(dir string) error {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}
		if info.IsDir() && !w.ignorePath(path) {
			fmt.Println("adding: " + path)
			err := w.watcher.Add(path)
			if err != nil {
				return err
			}
		}
		return nil
	})

	return err
}

func (w *DevAppWatcher) ignorePath(p string) bool {
	for _, ignorePath := range w.ignorePaths {
		rel, err := filepath.Rel(ignorePath, p)
		if err != nil {
			panic(err) //if this happens we'll deal with it
		}
		if !strings.HasPrefix(rel, "..") {
			return true
		}
	}
	return false
}
