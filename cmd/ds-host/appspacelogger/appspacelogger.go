package appspacelogger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

// going to need a appspace log manager probably
// and appspace logger
// Or this will do with a openLoggers field and mayabe an EjectLogger(appsapceID)

const appspaceLoggerSource = "AppspaceLogger"

type logger struct {
	fd *os.File
	// aybe also a buffer for incoming log messages?
}

// AppspaceLogger appends log data from various sources into a single log
type AppspaceLogger struct {
	AppspaceLogEvents interface {
		Send(domain.AppspaceLogEvent)
	}
	AppspaceModel interface {
		GetFromID(domain.AppspaceID) (*domain.Appspace, error)
	}

	Config *domain.RuntimeConfig

	loggersMux sync.Mutex
	loggers    map[domain.AppspaceID]*logger
}

// Init makes maps
func (l *AppspaceLogger) Init() {
	l.loggers = make(map[domain.AppspaceID]*logger)
}

// Log writes to the log file for the appspace
func (l *AppspaceLogger) Log(appspaceID domain.AppspaceID, source string, message string) {
	l.writeLog(domain.AppspaceLogEvent{
		Time:       time.Now(),
		AppspaceID: appspaceID,
		Source:     source,
		Data:       "",
		Message:    message})
}

func (l *AppspaceLogger) sanitizeMessage(msg string) string {
	ret := fmt.Sprintf("%q", msg)
	return strings.Replace(ret, "{", "?", -1)
}

func (l *AppspaceLogger) writeLog(event domain.AppspaceLogEvent) {
	appspaceLogger, err := l.getLogger(event.AppspaceID)
	if err != nil {
		// probably send to a general log? or its' already been logged?
		return
	}

	event.Message = l.sanitizeMessage(event.Message)

	// do we need to escap message for \n?
	str := fmt.Sprintf("%s %s %s %s\n", event.Time.Format(time.RFC3339), event.Source, event.Message, event.Data)

	written := false
	if appspaceLogger.fd != nil {
		_, err := appspaceLogger.fd.WriteString(str)
		if err == nil {
			written = true
		}
	}

	if !written {
		// TODO: better handling?
		fmt.Println("Failed to write to appspace log: " + str)
	}

	if l.AppspaceLogEvents != nil {
		go l.AppspaceLogEvents.Send(event)
	}
}

func (l *AppspaceLogger) getLogger(appspaceID domain.AppspaceID) (*logger, error) {
	l.loggersMux.Lock()
	defer l.loggersMux.Unlock()

	appspaceLogger, ok := l.loggers[appspaceID]
	if !ok {
		appspace, err := l.AppspaceModel.GetFromID(appspaceID)
		if err != nil {
			l.getHostLogger("AppspaceModel.GetFromID").Error(err)
			return nil, err
		}

		logPath := filepath.Join(l.Config.Exec.AppspacesPath, appspace.LocationKey, "data", "logs", "log.txt")

		f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0664)
		if err != nil {
			l.getHostLogger("OpenFile").Error(err)
			return nil, err
		}

		appspaceLogger = &logger{
			fd: f}
		l.loggers[appspaceID] = appspaceLogger
	}

	return appspaceLogger, nil
}

// EjectLogger closees teh log file and removes the logger from the map
// This sadly does not prevent a new Log call from reopening the file, which could cause problems
func (l *AppspaceLogger) EjectLogger(appspaceID domain.AppspaceID) {
	// maybe
	// close file

	l.writeLog(domain.AppspaceLogEvent{
		AppspaceID: appspaceID,
		Source:     appspaceLoggerSource,
		Message:    "Ejecting log file"})

	logger, err := l.getLogger(appspaceID)
	if err != nil {
		// problem already logged to host log, just return
		return
	}

	// here we could remove the logger from the map so it can't get used again
	// but is that the approach to "ejecting"?
	// Are we removing from the map, or just closing the fd so we can free up the appspace files?
	l.loggersMux.Lock()
	delete(l.loggers, appspaceID)
	l.loggersMux.Unlock()

	// here we may need to put a lock, and then close the FD
	err = logger.fd.Close()
	if err != nil {
		l.getHostLogger("logger.fd.Close()").Error(err)
		return
	}
}

func (l *AppspaceLogger) getHostLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("AppspaceLogger")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
