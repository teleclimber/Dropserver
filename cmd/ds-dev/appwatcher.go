package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

//ignorePaths is a set of paths relative to the root of the app that should be ignored
var ignorePaths = []string{
	".git",
}

// DevAppWatcher reloads the app files as needed and sends events
type DevAppWatcher struct {
	AppGetter interface {
		Reprocess(userID domain.UserID, appID domain.AppID, locationKey string) (domain.AppGetKey, error)
		SubscribeKey(key domain.AppGetKey) (domain.AppGetEvent, <-chan domain.AppGetEvent)
		GetResults(key domain.AppGetKey) (domain.AppGetMeta, bool)
		DeleteKeyData(key domain.AppGetKey)
	} `checkinject:"required"`
	DevAppModel         *DevAppModel      `checkinject:"required"`
	DevAppspaceModel    *DevAppspaceModel `checkinject:"required"`
	DevAppProcessEvents interface {
		Send(AppProcessEvent)
	} `checkinject:"required"`
	AppVersionEvents interface {
		Send(string)
	} `checkinject:"required"`

	watcher     *fsnotify.Watcher
	ignorePaths []string

	runMux  sync.Mutex
	running bool

	dirtyMux sync.Mutex
	dirty    bool
	timer    *time.Timer
}

// Start loads the appfiles and launches file watching
func (w *DevAppWatcher) Start(appPath string) {
	go w.reprocessAppFiles()

	w.ignorePaths = make([]string, len(ignorePaths))
	for i, p := range ignorePaths {
		w.ignorePaths[i] = filepath.Join(appPath, p)
	}

	go w.watch(appPath)
}
func (w *DevAppWatcher) reprocessAppFiles() { // This should probably be handled by app getter?
	ok := w.setRunning()
	if !ok {
		w.resetTimer()
		return // can't run now, already running.
	}

	w.setClean()

	w.AppVersionEvents.Send("loading")

	appGetKey, err := w.AppGetter.Reprocess(ownerID, appID, "")
	if err != nil {
		panic(err)
	}

	lastEvent, appGetCh := w.AppGetter.SubscribeKey(appGetKey)
	if lastEvent.Done || appGetCh == nil {
		w.reloadMetadata(appGetKey)
		return
	}

	// subscribe and wait
	reloading := false
	for e := range appGetCh {
		if e.Done {
			// if processing is done, get results to get the errors.
			results, ok := w.AppGetter.GetResults(appGetKey)
			if ok {
				w.DevAppProcessEvents.Send(AppProcessEvent{
					Processing: false,
					Step:       e.Step,
					Errors:     results.Errors})
			}
		} else {
			w.DevAppProcessEvents.Send(AppProcessEvent{
				Processing: true,
				Step:       e.Step,
				Errors:     []string{}})
		}
		if !reloading && e.Done {
			reloading = true
			go w.reloadMetadata(appGetKey)
		}
	}
}

func (w *DevAppWatcher) reloadMetadata(appGetKey domain.AppGetKey) {
	defer w.unsetRunning()

	results, ok := w.AppGetter.GetResults(appGetKey)
	if !ok {
		// not sure what to do there...
		panic("no results from appGetter")
	}

	w.AppGetter.DeleteKeyData(appGetKey)

	if len(results.Errors) > 0 {
		w.AppVersionEvents.Send("error")
		return
	}

	w.DevAppModel.App = domain.App{
		OwnerID: ownerID,
		AppID:   appID,
		Created: time.Now(),
		Name:    results.VersionMetadata.AppName}

	w.DevAppModel.Ver = domain.AppVersion{
		AppID:       appID,
		AppName:     results.VersionMetadata.AppName,
		Version:     results.VersionMetadata.AppVersion,
		Schema:      results.Schema,
		Created:     time.Now(),
		LocationKey: ""}

	// Need to update appspace so that the app version is reflected
	w.DevAppspaceModel.Appspace.AppVersion = results.VersionMetadata.AppVersion

	w.AppVersionEvents.Send("ready")
}

func (w *DevAppWatcher) filesChanged() {
	w.setDirty()
}

func (w *DevAppWatcher) watch(appPath string) {
	fmt.Println("Watching: " + appPath)

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
				if event.Name == "" || event.Name == "." {
					// For some reason sometimes fsnotify sometimes has empty or "." event names
					// Usually when seen when rebuilding a frontend's dist dir (massive deletions and writes.)
					// Just ignore (no-op) and pretend it never happened.
				} else if event.Op&fsnotify.Create == fsnotify.Create {
					fileInfo, err := os.Stat(event.Name)
					if err != nil {
						panic(err) //deal with this if we hit it
					}
					if fileInfo.IsDir() {
						w.watchDir(event.Name)
					}
					w.filesChanged()
				} else if event.Op&fsnotify.Write == fsnotify.Write {
					w.filesChanged()
				} else if event.Op&fsnotify.Remove == fsnotify.Remove {
					// fsnotify removes watches automatically on Mac and Linux, apparently.
					// https://github.com/fsnotify/fsnotify/issues/238
					// On Windows it does not yet, apparently? Consequences?
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

func (w *DevAppWatcher) setDirty() {
	w.dirtyMux.Lock()
	w.dirty = true
	if w.timer == nil {
		w.timer = time.AfterFunc(100*time.Millisecond, w.reprocessAppFiles)
	} else {
		w.timer.Reset(100 * time.Millisecond)
	}
	w.dirtyMux.Unlock()
}

func (w *DevAppWatcher) resetTimer() {
	w.dirtyMux.Lock()
	if w.timer == nil {
		w.timer = time.AfterFunc(100*time.Millisecond, w.reprocessAppFiles)
	} else {
		w.timer.Reset(100 * time.Millisecond)
	}
	w.dirtyMux.Unlock()
}

func (w *DevAppWatcher) setClean() {
	w.dirtyMux.Lock()
	w.dirty = false
	w.timer = nil
	w.dirtyMux.Unlock()
}

func (w *DevAppWatcher) isDirty() bool {
	w.dirtyMux.Lock()
	defer w.dirtyMux.Unlock()
	return w.dirty
}

func (w *DevAppWatcher) setRunning() bool {
	w.runMux.Lock()
	defer w.runMux.Unlock()
	if w.running {
		return false
	}
	w.running = true
	return true
}

func (w *DevAppWatcher) isRunning() bool {
	w.runMux.Lock()
	defer w.runMux.Unlock()
	return w.running
}

func (w *DevAppWatcher) unsetRunning() {
	w.runMux.Lock()
	defer w.runMux.Unlock()
	if !w.running {
		panic("unsetting running while not running. Seems wrong.")
	}
	w.running = false
}
