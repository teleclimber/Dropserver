package appspacelogger

import (
	"path/filepath"
	"sync"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

// AppspaceLogger opens and manages appspace loggers
type AppspaceLogger struct {
	AppspaceModel interface {
		GetFromID(domain.AppspaceID) (*domain.Appspace, error)
	} `checkinject:"required"`
	AppspaceStatus interface {
		IsLockedClosed(domain.AppspaceID) bool
	} `checkinject:"required"`

	Config *domain.RuntimeConfig

	loggersMux sync.Mutex
	loggers    map[domain.AppspaceID]*Logger
}

// Init makes maps
func (l *AppspaceLogger) Init() {
	l.loggers = make(map[domain.AppspaceID]*Logger)
}

// Log writes to the log file for the appspace
func (l *AppspaceLogger) Log(appspaceID domain.AppspaceID, source string, message string) {
	logger := l.getLogger(appspaceID, true)
	if logger == nil {
		return
	}
	logger.Log(source, message)
}

// Get a reference to logger for app at locationKey
func (l *AppspaceLogger) Get(appspaceID domain.AppspaceID) domain.LoggerI {
	return l.getLogger(appspaceID, false)
}

// Open the log file and return the logger at locationKey
// I wonder if this one is actually necessary?
func (l *AppspaceLogger) Open(appspaceID domain.AppspaceID) domain.LoggerI {
	return l.getLogger(appspaceID, true)
}

// Close the log file for logger at locationKey
func (l *AppspaceLogger) Close(appspaceID domain.AppspaceID) {
	logger := l.getLogger(appspaceID, false)
	if logger == nil {
		return
	}
	err := logger.close()
	if err != nil {
		l.getHostLogger("Close logger.close").AppspaceID(appspaceID).Error(err)
	}
}

// Forget about this locationKey
// This closes the log file and drops all subscriptions
func (l *AppspaceLogger) Forget(appspaceID domain.AppspaceID) {
	logger := l.getLogger(appspaceID, false)
	if logger == nil {
		return
	}
	err := logger.close()
	if err != nil {
		l.getHostLogger("Forget logger.close").AppspaceID(appspaceID).Error(err)
	}

	logger.dropSubscriptions()

	l.loggersMux.Lock()
	defer l.loggersMux.Unlock()
	delete(l.loggers, appspaceID)
}

func (l *AppspaceLogger) getLogger(appspaceID domain.AppspaceID, open bool) *Logger {
	l.loggersMux.Lock()
	defer l.loggersMux.Unlock()

	logger, ok := l.loggers[appspaceID]
	if !ok {
		appspace, err := l.AppspaceModel.GetFromID(appspaceID)
		if err != nil {
			l.getHostLogger("getLogger AppspaceModel.GetFromID").AppspaceID(appspaceID).Error(err)
			return nil
		}

		logger = &Logger{
			logPath: filepath.Join(l.Config.Exec.AppspacesPath, appspace.LocationKey, "data", "logs", "log.txt")}

		l.loggers[appspaceID] = logger
	}

	if logger.fd == nil && open {
		if l.AppspaceStatus.IsLockedClosed(appspaceID) {
			l.getHostLogger("getLogger AppspaceStatus.IsLockedClosed").AppspaceID(appspaceID).Log("appspace is locked closed")
			return nil
		}

		err := logger.open()
		if err != nil {
			l.getHostLogger("getLogger logger.open").AppspaceID(appspaceID).Error(err)
			return nil
		}
	}

	return logger
}

func (l *AppspaceLogger) getHostLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("AppspaceLogger")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
