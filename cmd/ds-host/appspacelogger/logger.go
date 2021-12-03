package appspacelogger

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

func sanitizeMessage(m string) string {
	m = strings.ReplaceAll(m, "\r", "")
	m = strings.ReplaceAll(m, "\n", "\\n")
	return m
}

// Logger system sufficiently genric to support app and appspace logs
type Logger struct {
	fd *os.File

	logPath string

	logMux sync.Mutex

	statusSubMux sync.Mutex
	statusSubs   []chan bool

	entriesSubMux sync.Mutex
	entriesSubs   []chan string
}

func (l *Logger) open() error {
	f, err := os.OpenFile(l.logPath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0664)
	if err != nil {
		return err
	}
	l.fd = f
	l.sendStatusEvent(true)
	return nil
}
func (l *Logger) close() error {
	l.logMux.Lock()
	defer l.logMux.Unlock()
	if l.fd == nil {
		return nil
	}
	err := l.fd.Close() // do we need to lock
	if err != nil {
		return err
	}
	l.fd = nil
	return nil
}

func (l *Logger) Log(source, message string) {
	str := fmt.Sprintf("%s %s %s", time.Now().Format(time.RFC3339), source, sanitizeMessage(message))
	l.writeEntry(str)
	// maybe check error and write to host log.
}
func (l *Logger) writeEntry(entry string) error {
	l.logMux.Lock()
	defer l.logMux.Unlock()

	if l.fd == nil {
		return errors.New("unable to write to log: log file is not open")
	}

	_, err := l.fd.WriteString(entry + "\n")
	if err != nil {
		return err
	}

	l.sendEntryEvent(entry)

	return nil
}

// GetLastBytes returns the log lines found in the last n bytes of the file.
func (l *Logger) GetLastBytes(n int64) (domain.LogChunk, error) {
	if l.fd == nil {
		return domain.LogChunk{}, errors.New("trying to get Last Bytes of closed log")
	}

	chunk, err := getChunk(l.fd, -n, 0)
	if err != nil {
		l.getHostLogger("GetLastBytes getChunk()").Error(err)
		return domain.LogChunk{}, err
	}

	return chunk, nil
}

func (l *Logger) SubscribeEntries(n int64) (domain.LogChunk, <-chan string, error) {
	// here a lock seems appropirate.
	// May need to split the GetLAstBytes fn. into high level / low level.

	l.logMux.Lock()
	defer l.logMux.Unlock()

	chunk, err := getChunk(l.fd, -n, 0)
	if err != nil {
		l.getHostLogger("SubscribeEntries getChunk()").Error(err)
		return domain.LogChunk{}, nil, err
	}

	// Now need to do subscription which is a whole thing in itself.
	ch := l.subscribeEntries()

	return chunk, ch, nil
}

// getChunk returns a part of the log, trimmed on new lines
// If "from" is negative the beginning of the chunk is relative to the end of the file
// A "to" of 0 is interpreted as end of file
func getChunk(fd *os.File, from, to int64) (domain.LogChunk, error) {
	if to < 0 {
		return domain.LogChunk{}, fmt.Errorf("AppspaceLogger getChunk: negative from or to: %v, %v", from, to)
	}

	fStat, err := fd.Stat()
	if err != nil {
		return domain.LogChunk{}, err
	}
	fSize := fStat.Size()
	if fSize == 0 {
		return domain.LogChunk{}, nil
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
		return domain.LogChunk{}, fmt.Errorf("AppspaceLogger getChunk: from: %v greater than size of log: %v", from, fSize)
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
		return domain.LogChunk{}, fmt.Errorf("AppspaceLogger getChunk: from: %v, to: %v resulted in invalid range for log of size %v", from, to, fSize)
	}

	buf := make([]byte, n)
	rSize, err := fd.ReadAt(buf, start)
	if err != nil {
		return domain.LogChunk{}, err
	}
	if rSize != int(n) {
		return domain.LogChunk{}, fmt.Errorf("AppspaceLogger getChunk: expect to read %v bytes, got %v", n, rSize)
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
		return domain.LogChunk{}, nil
	}

	chunk := domain.LogChunk{
		From:    start + int64(trimStart),
		To:      start + int64(trimEnd),
		Content: bytes.NewBuffer(buf[trimStart:trimEnd]).String()}

	return chunk, nil
}

func (l *Logger) SubscribeStatus() (bool, <-chan bool) {
	l.statusSubMux.Lock()
	defer l.statusSubMux.Unlock()
	ch := make(chan bool)
	l.statusSubs = append(l.statusSubs, ch)
	return l.fd != nil, ch
}
func (l *Logger) UnsubscribeStatus(ch <-chan bool) {
	l.statusSubMux.Lock()
	defer l.statusSubMux.Unlock()
	subs := l.statusSubs
	for i, c := range subs {
		if c == ch {
			subs[i] = subs[len(subs)-1]
			l.statusSubs = subs[:len(subs)-1]
			close(c)
			return
		}
	}
	l.getHostLogger("UnsubscribeStatus()").Log("Failed to find a subscriber to unsubscribe")
}
func (l *Logger) sendStatusEvent(status bool) {
	l.statusSubMux.Lock()
	defer l.statusSubMux.Unlock()
	for _, c := range l.statusSubs {
		c <- status
	}
}

func (l *Logger) subscribeEntries() <-chan string {
	l.entriesSubMux.Lock()
	defer l.entriesSubMux.Unlock()
	ch := make(chan string)
	l.entriesSubs = append(l.entriesSubs, ch)
	return ch
}
func (l *Logger) UnsubscribeEntries(ch <-chan string) {
	l.entriesSubMux.Lock()
	defer l.entriesSubMux.Unlock()
	subs := l.entriesSubs
	for i, c := range subs {
		if c == ch {
			subs[i] = subs[len(subs)-1]
			l.entriesSubs = subs[:len(subs)-1]
			close(c)
			return
		}
	}
	l.getHostLogger("UnsubscribeEntries()").Log("Failed to find a subscriber to unsubscribe")
}
func (l *Logger) sendEntryEvent(entry string) {
	l.entriesSubMux.Lock()
	defer l.entriesSubMux.Unlock()
	for _, c := range l.entriesSubs {
		c <- entry
	}
}

func (l *Logger) dropSubscriptions() {
	l.entriesSubMux.Lock()
	for _, c := range l.entriesSubs {
		close(c)
	}
	l.entriesSubs = make([]chan string, 0)
	l.entriesSubMux.Unlock()

	l.statusSubMux.Lock()
	for _, c := range l.statusSubs {
		close(c)
	}
	l.statusSubs = make([]chan bool, 0)
	l.statusSubMux.Unlock()
}

func (l *Logger) getHostLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("app[space]Logger").AddNote(l.logPath)
	if note != "" {
		r.AddNote(note)
	}
	return r
}
