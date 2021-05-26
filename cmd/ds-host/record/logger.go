package record

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

var multiWriter io.Writer
var logFile *os.File

// InitDsLogger sets flags on the default logger
func InitDsLogger() {
	log.SetFlags(log.Ldate | log.Ltime)
}

// SetLogOutput sets the path of the log file
func SetLogOutput(logPath string) error {
	if logPath == "" {
		return nil
	}
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}

	multiWriter = io.MultiWriter(os.Stderr, logFile)

	log.SetOutput(multiWriter)

	return nil
}

// CloseLogOutput closes the log file, if any
func CloseLogOutput() error {
	if logFile == nil {
		return nil
	}

	log.SetOutput(os.Stderr)
	err := logFile.Close()
	if err != nil {
		return err
	}
	logFile = nil
	return nil
}

// Debug just logs the message if in debug mode
func Debug(message string) {
	if os.Getenv("DEBUG") != "" {
		log.Print("DEBUG: " + message)
	}
}

// Log just logs the message
func Log(message string) {
	log.Print(message)
}

// DsLogger wraps the standard log or anoything else we use late
type DsLogger struct {
	//... stash variables like appspace id
	appID         domain.AppID
	hasAppID      bool
	appVersion    domain.Version
	appspaceID    domain.AppspaceID
	hasAppspaceID bool
	userID        domain.UserID
	hasUserID     bool
	note          string // note is extra info that is not a predetermined variable, but not the actual log message
}

// NewDsLogger returns a new struct. But is it necessary?
func NewDsLogger(notes ...string) *DsLogger {
	l := DsLogger{}
	for _, n := range notes {
		if n != "" {
			l.AddNote(n)
		}
	}
	return &l
}

// Clone returns a new DsLogger with all values copied.
func (l *DsLogger) Clone() *DsLogger {
	c := *l // This little trick only works as long as none of the fields are reference types.
	return &c
}

// AppID is set on all uses of the returned logger
// If app id is already set, the old app id is added to note
func (l *DsLogger) AppID(appID domain.AppID) *DsLogger {
	if l.hasAppID && l.appID != appID {
		l.AddNote(fmt.Sprintf("ex-app-id:%v", l.appID))
	}
	l.hasAppID = true
	l.appID = appID

	return l
}

// AppVersion adds app version to logger
// If already set, old version is added to note.
func (l *DsLogger) AppVersion(version domain.Version) *DsLogger {
	if l.appVersion != domain.Version("") {
		l.AddNote(fmt.Sprintf("ex-app-version:%v", l.appVersion))
	}
	l.appVersion = version

	return l
}

// AppspaceID is set on all uses of the returned logger
func (l *DsLogger) AppspaceID(appspaceID domain.AppspaceID) *DsLogger {
	if l.hasAppspaceID && l.appspaceID != appspaceID {
		l.AddNote(fmt.Sprintf("ex-appspace-id:%v", l.appspaceID))
	}
	l.hasAppspaceID = true
	l.appspaceID = appspaceID

	return l
}

// UserID is set on all uses of the returned logger
// If user id is already set, the old user id is added to note
func (l *DsLogger) UserID(userID domain.UserID) *DsLogger {
	if l.hasUserID && l.userID != userID {
		l.AddNote(fmt.Sprintf("ex-user-id:%v", l.userID))
	}
	l.hasUserID = true
	l.userID = userID

	return l
}

// AddNote appends a note to the not string
func (l *DsLogger) AddNote(note string) *DsLogger {
	if l.note != "" {
		l.note += ", "
	}
	l.note += note

	return l
}

// Debug writes the message to the log if debug mode is on
func (l *DsLogger) Debug(message string) {
	if os.Getenv("DEBUG") != "" {
		log.Print("DEBUG: " + l.contextStr() + message)
	}
}

// Log writes the message to the log
func (l *DsLogger) Log(message string) {
	log.Print(l.contextStr() + message)
}

// Error writes the error to the log
func (l *DsLogger) Error(err error) {
	log.Print(l.contextStr() + "Error: " + err.Error())
}

func (l *DsLogger) contextStr() string {
	str := ""
	_, file, line, ok := runtime.Caller(2)
	f := filepath.Base(file)
	p := filepath.Base(filepath.Dir(file))
	if ok {
		str += fmt.Sprintf("%v/%v:%v ", p, f, line)
	}
	if l.hasAppID {
		str += fmt.Sprintf("a:%v ", l.appID)
	}
	if l.appVersion != domain.Version("") {
		str += fmt.Sprintf("v:%v ", l.appVersion)
	}
	if l.hasAppspaceID {
		str += fmt.Sprintf("as:%v ", l.appspaceID)
	}
	if l.hasUserID {
		str += fmt.Sprintf("u:%v ", l.userID)
	}
	if l.note != "" {
		str += "(" + l.note + ") "
	}
	return str
}
