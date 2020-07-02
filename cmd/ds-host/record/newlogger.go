package record

import (
	"fmt"
	"log"
	"runtime"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// just experimenting with interfaces and what not here.
// I was thinking of passing loggers, but it might be easier to create them...?

// What other variables will we really want to pass on?
// - function name / code location

// defuse runtime's Callers etc... to get a stack trace in debug mode

// InitDsLogger sets flags on the default logger
func InitDsLogger() {
	// for now set flags and what not.
	//log.Ldate|log.Ltime|log.Lshortfile
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
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
func NewDsLogger() *DsLogger {
	return &DsLogger{}
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
	if ok {
		str += fmt.Sprintf("%v:%v ", file, line)
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
