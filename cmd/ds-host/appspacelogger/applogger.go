package appspacelogger

import (
	"path/filepath"
	"sync"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

// AppLogger opens and manages logs for each app-version (identified by location key.)
// Used when running sandbox to get app routes, etc... for example.
type AppLogger struct {
	Location2Path interface {
		AppMeta(string) string
	} `checkinject:"required"`

	loggersMux sync.Mutex
	loggers    map[string]*Logger // by location key
}

func (l *AppLogger) Init() {
	l.loggers = make(map[string]*Logger)
}

// Log a message to app logger at location key
func (l *AppLogger) Log(locationKey string, source string, message string) {
	logger := l.getLogger(locationKey, true)
	if logger == nil {
		return
	}
	logger.Log(source, message)
}

// Get a reference to logger for app at locationKey
func (l *AppLogger) Get(locationKey string) domain.LoggerI {
	return l.getLogger(locationKey, false)
}

// Open the log file and return the logger at locationKey
// I wonder if this one is actually necessary?
func (l *AppLogger) Open(locationKey string) domain.LoggerI {
	return l.getLogger(locationKey, true)
}

// Close the log file for logger at locationKey
func (l *AppLogger) Close(locationKey string) {
	logger := l.getLogger(locationKey, false)
	if logger == nil {
		return
	}
	err := logger.close()
	if err != nil {
		l.getHostLogger("Eject logger.close").AddNote("locationKey: " + locationKey).Error(err)
	}
}

// Forget about this locationKey
// This closes the log file and drops all subscriptions
func (l *AppLogger) Forget(locationKey string) {
	logger := l.getLogger(locationKey, false)
	if logger == nil {
		return
	}
	err := logger.close()
	if err != nil {
		l.getHostLogger("Eject logger.close").AddNote("locationKey: " + locationKey).Error(err)
	}

	logger.dropSubscriptions()

	l.loggersMux.Lock()
	defer l.loggersMux.Unlock()
	delete(l.loggers, locationKey)
}

func (l *AppLogger) getLogger(locationKey string, open bool) (logger *Logger) { // maybe pass flag to only open if requested.
	l.loggersMux.Lock()
	defer l.loggersMux.Unlock()

	logger, ok := l.loggers[locationKey]
	if !ok {
		logger = &Logger{
			logPath: filepath.Join(l.Location2Path.AppMeta(locationKey), "log.txt")}
		// logger.Init()?

		l.loggers[locationKey] = logger
	}

	if logger.fd == nil && open {
		err := logger.open()
		if err != nil {
			l.getHostLogger("getLogger logger.open").AddNote("locationKey: " + locationKey).Error(err)
			return nil
		}
	}

	return logger
}

func (l *AppLogger) getHostLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("AppLogger")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
