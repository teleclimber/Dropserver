package appspacelogger

import (
	"bytes"
	"errors"
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

type logger struct {
	fd *os.File

	logMux sync.Mutex
}

// AppspaceLogger appends log data from various sources into a single log
type AppspaceLogger struct {
	AppspaceModel interface {
		GetFromID(domain.AppspaceID) (*domain.Appspace, error)
	} `checkinject:"required"`
	AppspaceStatus interface {
		IsLockedClosed(domain.AppspaceID) bool
	} `checkinject:"required"`

	Config *domain.RuntimeConfig

	loggersMux sync.Mutex
	loggers    map[domain.AppspaceID]*logger

	statusSubMux sync.Mutex
	statusSubs   map[domain.AppspaceID][]chan bool

	entriesSubMux sync.Mutex
	entriesSubs   map[domain.AppspaceID][]chan string
}

// Init makes maps
func (l *AppspaceLogger) Init() {
	l.loggers = make(map[domain.AppspaceID]*logger)
	l.statusSubs = make(map[domain.AppspaceID][]chan bool)
	l.entriesSubs = make(map[domain.AppspaceID][]chan string)
}

// Log writes to the log file for the appspace
func (l *AppspaceLogger) Log(appspaceID domain.AppspaceID, source string, message string) {
	str := fmt.Sprintf("%s %s %s\n", time.Now().Format(time.RFC3339), source, sanitizeMessage(message))
	l.writeLog(appspaceID, str)
}

func sanitizeMessage(m string) string {
	m = strings.ReplaceAll(m, "\r", "")
	m = strings.ReplaceAll(m, "\n", "\\n")
	return m
}

func (l *AppspaceLogger) writeLog(appspaceID domain.AppspaceID, entry string) {
	appspaceLogger, err := l.getLogger(appspaceID)
	if err != nil {
		// already logged to host logger, just return
		return
	}

	if appspaceLogger.fd == nil {
		// this shouldn't happen
		l.getHostLogger("writeLog").AppspaceID(appspaceID).Error(errors.New("appspaceLogger.fd is nil"))
		return
	}

	appspaceLogger.logMux.Lock()
	defer appspaceLogger.logMux.Unlock()

	_, err = appspaceLogger.fd.WriteString(entry)
	if err != nil {
		l.getHostLogger("writeLog, appspaceLogger.fd.WriteString(str)").AppspaceID(appspaceID).Error(err)
		return
	}

	l.sendEntryEvent(appspaceID, entry)
}

func (l *AppspaceLogger) getLogger(appspaceID domain.AppspaceID) (*logger, error) {
	l.loggersMux.Lock()
	defer l.loggersMux.Unlock()

	appspaceLogger, ok := l.loggers[appspaceID]
	if !ok {
		appspace, err := l.AppspaceModel.GetFromID(appspaceID)
		if err != nil {
			l.getHostLogger("getLogger AppspaceModel.GetFromID").AppspaceID(appspaceID).Error(err)
			return nil, err
		}

		logPath := filepath.Join(l.Config.Exec.AppspacesPath, appspace.LocationKey, "data", "logs", "log.txt")

		if l.AppspaceStatus.IsLockedClosed(appspaceID) {
			l.getHostLogger("getLogger AppspaceStatus.IsLockedClosed").AppspaceID(appspaceID).Log("appspace is locked closed")
			return nil, domain.ErrAppspaceLockedClosed
		}

		f, err := os.OpenFile(logPath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0664)
		if err != nil {
			l.getHostLogger("getLogger OpenFile").AppspaceID(appspaceID).Error(err)
			return nil, err
		}

		appspaceLogger = &logger{
			fd: f}
		l.loggers[appspaceID] = appspaceLogger

		// Send log status open to any subscribers:
		l.sendStatusEvent(appspaceID, true)
	}

	return appspaceLogger, nil
}

// OpenLogger opens the log
// This is only necessary to signal that the log is open and ready to be read from
func (l *AppspaceLogger) OpenLogger(appspaceID domain.AppspaceID) error {
	_, err := l.getLogger(appspaceID)
	return err
}

// EjectLogger closes the log file and removes the logger from the map
func (l *AppspaceLogger) EjectLogger(appspaceID domain.AppspaceID) {
	l.Log(appspaceID, "ds-host", "Ejecting log file")

	logger, err := l.getLogger(appspaceID)
	if err != nil {
		// problem already logged to host log, just return
		return
	}

	l.loggersMux.Lock()
	delete(l.loggers, appspaceID)
	l.loggersMux.Unlock()

	// here we may need to put a lock, and then close the FD
	err = logger.fd.Close()
	if err != nil {
		l.getHostLogger("logger.fd.Close()").AppspaceID(appspaceID).Error(err)
		return
	}

	l.sendStatusEvent(appspaceID, false)
}

// GetLastBytes returns the log lines found in the last n bytes of the file.
func (l *AppspaceLogger) GetLastBytes(appspaceID domain.AppspaceID, n int64) (domain.AppspaceLogChunk, error) {
	appspaceLogger, err := l.getLogger(appspaceID)
	if err != nil {
		// already logged to host logger, just return
		return domain.AppspaceLogChunk{}, err
	}

	chunk, err := getChunk(appspaceLogger.fd, -n, 0)
	if err != nil {
		l.getHostLogger("GetLastBytes getChunk()").AppspaceID(appspaceID).Error(err)
		return domain.AppspaceLogChunk{}, err
	}

	return chunk, nil
}

func (l *AppspaceLogger) SubscribeEntries(appspaceID domain.AppspaceID, n int64) (domain.AppspaceLogChunk, <-chan string, error) {
	// here a lock seems appropirate.
	// May need to split the GetLAstBytes fn. into high level / low level.

	appspaceLogger, err := l.getLogger(appspaceID)
	if err != nil {
		// already logged to host logger, just return
		return domain.AppspaceLogChunk{}, nil, err
	}

	appspaceLogger.logMux.Lock()
	defer appspaceLogger.logMux.Unlock()

	chunk, err := getChunk(appspaceLogger.fd, -n, 0)
	if err != nil {
		l.getHostLogger("SubscribeEntries getChunk()").AppspaceID(appspaceID).Error(err)
		return domain.AppspaceLogChunk{}, nil, err
	}

	// Now need to do subscription which is a whole thing in itself.
	ch := l.subscribeEntries(appspaceID)

	return chunk, ch, nil
}

// getChunk returns a part of the log, trimmed on new lines
// If "from" is negative the beginning of the chunk is relative to the end of the file
// A "to" of 0 is interpreted as end of file
func getChunk(fd *os.File, from, to int64) (domain.AppspaceLogChunk, error) {
	if to < 0 {
		return domain.AppspaceLogChunk{}, fmt.Errorf("AppspaceLogger getChunk: negative from or to: %v, %v", from, to)
	}

	fStat, err := fd.Stat()
	if err != nil {
		return domain.AppspaceLogChunk{}, err
	}
	fSize := fStat.Size()
	if fSize == 0 {
		return domain.AppspaceLogChunk{}, nil
	}

	start := from
	end := to
	if start < 0 {
		start = fSize + start
	}
	if start < 0 {
		start = 0
	}
	if start >= fSize {
		return domain.AppspaceLogChunk{}, fmt.Errorf("AppspaceLogger getChunk: from: %v greater than size of log: %v", from, fSize)
	}
	// go back one byte to capture the previous line's \n
	if start != 0 {
		start = start - 1
	}
	if end == 0 {
		end = fSize // +1?
	}
	n := end - start
	if n <= 0 {
		return domain.AppspaceLogChunk{}, fmt.Errorf("AppspaceLogger getChunk: from: %v, to: %v resulted in invalid range for log of size %v", from, to, fSize)
	}

	buf := make([]byte, n)
	rSize, err := fd.ReadAt(buf, start)
	if err != nil {
		return domain.AppspaceLogChunk{}, err
	}
	if rSize != int(n) {
		return domain.AppspaceLogChunk{}, fmt.Errorf("AppspaceLogger getChunk: expect to read %v bytes, got %v", n, rSize)
	}

	// if starting mid-file, trim the first (preusmably incomplete) log line from result
	trimStart := 0
	if start != 0 {
		trimStart = bytes.Index(buf, []byte("\n")) + 1
	}

	// find the last occurence of \n, and trim accordingly.
	trimEnd := bytes.LastIndex(buf, []byte("\n"))
	if trimEnd != int(n) {
		trimEnd = trimEnd + 1
	}

	// if we don't have anything to return, send a zero-value chunk.
	// however this could misinterpreted by by consumer?
	// Maybe consumer should check that from and to are different. Or that string is non-empty Seems reasonable.
	if trimStart == trimEnd {
		return domain.AppspaceLogChunk{}, nil
	}

	chunk := domain.AppspaceLogChunk{
		From:    start + int64(trimStart),
		To:      start + int64(trimEnd),
		Content: bytes.NewBuffer(buf[trimStart:trimEnd]).String()}

	return chunk, nil
}

func (l *AppspaceLogger) SubscribeStatus(appspaceID domain.AppspaceID) (bool, <-chan bool) {
	l.loggersMux.Lock()
	defer l.loggersMux.Unlock()
	l.statusSubMux.Lock()
	defer l.statusSubMux.Unlock()

	subs, ok := l.statusSubs[appspaceID]
	if !ok {
		subs = make([]chan bool, 0, 10)
		l.statusSubs[appspaceID] = subs
	}
	ch := make(chan bool)
	l.statusSubs[appspaceID] = append(subs, ch)

	_, ok = l.loggers[appspaceID]
	return ok, ch
}
func (l *AppspaceLogger) UnsubscribeStatus(appspaceID domain.AppspaceID, ch <-chan bool) {
	l.statusSubMux.Lock()
	defer l.statusSubMux.Unlock()
	subs, ok := l.statusSubs[appspaceID]
	if ok {
		for i, c := range subs {
			if c == ch {
				subs[i] = subs[len(subs)-1]
				l.statusSubs[appspaceID] = subs[:len(subs)-1]
				close(c)
				return
			}
		}
	}
	l.getHostLogger("UnsubscribeStatus()").AppspaceID(appspaceID).Log("Failed to find a subscriber to unsubscribe")
}
func (l *AppspaceLogger) sendStatusEvent(appspaceID domain.AppspaceID, status bool) {
	l.statusSubMux.Lock()
	defer l.statusSubMux.Unlock()
	subs, ok := l.statusSubs[appspaceID]
	if !ok {
		return
	}
	for _, c := range subs {
		c <- status
	}
}

func (l *AppspaceLogger) subscribeEntries(appspaceID domain.AppspaceID) <-chan string {
	l.entriesSubMux.Lock()
	defer l.entriesSubMux.Unlock()
	subs, ok := l.entriesSubs[appspaceID]
	if !ok {
		subs = make([]chan string, 0, 10)
		l.entriesSubs[appspaceID] = subs
	}
	ch := make(chan string)
	l.entriesSubs[appspaceID] = append(subs, ch)
	return ch
}
func (l *AppspaceLogger) UnsubscribeEntries(appspaceID domain.AppspaceID, ch <-chan string) {
	l.entriesSubMux.Lock()
	defer l.entriesSubMux.Unlock()
	subs, ok := l.entriesSubs[appspaceID]
	if ok {
		for i, c := range subs {
			if c == ch {
				subs[i] = subs[len(subs)-1]
				l.entriesSubs[appspaceID] = subs[:len(subs)-1]
				close(c)
				return
			}
		}
	}
	l.getHostLogger("UnsubscribeEntries()").AppspaceID(appspaceID).Log("Failed to find a subscriber to unsubscribe")
}
func (l *AppspaceLogger) sendEntryEvent(appspaceID domain.AppspaceID, entry string) {
	l.entriesSubMux.Lock()
	defer l.entriesSubMux.Unlock()
	subs, ok := l.entriesSubs[appspaceID]
	if !ok {
		return
	}
	for _, c := range subs {
		c <- entry
	}
}

func (l *AppspaceLogger) getHostLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("AppspaceLogger")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
