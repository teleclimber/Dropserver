package sandbox

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"golang.org/x/sys/unix"
)

var errDone = errors.New("done watching this bwrap json status file")

type BwrapJsonStatus struct {
	f        *os.File
	filename string

	inotifyFd        int
	inotifyWatchdesc int

	pidMux sync.Mutex
	pidCh  chan struct{}
	pidSet bool
	pid    int

	exitCode int
}

func NewBwrapJsonStatus(basePath string) (BwrapStatusJsonI, error) { // prob need to pass runtime.Config
	f, err := os.CreateTemp(basePath, "bwrap-json-status-")
	if err != nil {
		return nil, err
	}

	b := &BwrapJsonStatus{
		f:        f,
		filename: f.Name(),
		exitCode: -999,
	}

	go func() {
		err := b.follow()
		if err != nil && err != errDone {
			b.getLogger("b.follow()").Error(err)
		}
	}()

	return b, nil
}

func (b *BwrapJsonStatus) GetFile() *os.File {
	return b.f
}

// Stop waiting for changes and
// clean up everything
func (b *BwrapJsonStatus) Stop() {
	b.pidMux.Lock()
	if b.pidCh != nil {
		close(b.pidCh)
	}
	b.pidMux.Unlock()

	success, err := unix.InotifyRmWatch(b.inotifyFd, uint32(b.inotifyWatchdesc))
	if err != nil {
		b.getLogger("Stop() unix.InotifyRmWatch()").Error(err)
	}
	if success == -1 {
		b.getLogger("Stop() unix.InotifyRmWatch()").Log("success is -1")
	}
	unix.Close(b.inotifyFd)
	b.f.Close()
	os.RemoveAll(b.filename)
}

func (b *BwrapJsonStatus) follow() error {
	fd, err := unix.InotifyInit() // Note that we could create just one inotify instance and reuse it for all sandboxes
	if err != nil {
		return err
	}
	b.inotifyFd = fd
	b.inotifyWatchdesc, err = unix.InotifyAddWatch(fd, b.filename, unix.IN_MODIFY)
	if err != nil && err != io.EOF {
		return err
	}
	for {
		err = b.readData()
		if err != nil {
			return err
		}
		err = waitForChange(fd)
		if err != nil {
			return err
		}
	}
}

// Need yo be able to quit it!  Maybe use Context here?
func waitForChange(fd int) error {
	for {
		var buf [unix.SizeofInotifyEvent]byte
		_, err := unix.Read(fd, buf[:])
		if err != nil {
			return err
		}
		r := bytes.NewReader(buf[:])
		var ev = unix.InotifyEvent{}
		_ = binary.Read(r, binary.LittleEndian, &ev)
		if ev.Mask&unix.IN_MODIFY == unix.IN_MODIFY {
			return nil
		}
	}
}

func (b *BwrapJsonStatus) readData() error {
	contents, err := os.ReadFile(b.filename)
	if err != nil {
		return err
	}

	var objmap map[string]json.RawMessage
	pieces := strings.Split(string(contents), "\n")
	for _, p := range pieces {
		if strings.TrimSpace(p) == "" {
			continue
		}
		err = json.Unmarshal([]byte(p), &objmap) // we can keep using objmap b as it should just build up all the data in there
		if err != nil {
			// Ignore errors. It seems incomplete json strings get flushed to disk.
		}
	}

	err = b.readPid(objmap)
	if err != nil {
		return err
	}

	err = b.readExitCode(objmap)
	if err != nil {
		return err
	}

	return nil
}

func (b *BwrapJsonStatus) readPid(objmap map[string]json.RawMessage) error {
	b.pidMux.Lock()
	defer b.pidMux.Unlock()
	if b.pidSet {
		return nil
	}
	if raw, ok := objmap["child-pid"]; ok {
		var pid int
		err := json.Unmarshal(raw, &pid)
		if err != nil {
			return err
		}

		b.pid = pid
		b.pidSet = true
		if b.pidCh != nil {
			b.pidCh <- struct{}{}
		}
	}
	return nil
}

func (b *BwrapJsonStatus) WaitPid() (int, bool) {
	pid, ok := b.initWaitPid()
	if ok {
		fmt.Println("got pid right wawy")
		return pid, true
	}

	tRef := time.Now()

	<-b.pidCh

	fmt.Printf("Wait for pid: %s \n", time.Since(tRef))

	if b.pid == 0 {
		return 0, false
	}

	return b.pid, true
}

func (b *BwrapJsonStatus) initWaitPid() (int, bool) {
	b.pidMux.Lock()
	defer b.pidMux.Unlock()

	if b.pidSet {
		return b.pid, true
	}

	if b.pidCh != nil {
		panic("already waiting on pid")
	}

	b.pidCh = make(chan struct{})
	return 0, false
}

func (b *BwrapJsonStatus) readExitCode(objmap map[string]json.RawMessage) error {
	if b.exitCode != -999 {
		return nil
	}
	if raw, ok := objmap["exit-code"]; ok {
		var code int
		err := json.Unmarshal(raw, &code)
		if err != nil {
			return err
		}
		b.exitCode = code
		b.Stop()
		return errDone
	}
	return nil
}

func (m *BwrapJsonStatus) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("BwrapJsonStatus")
	if note != "" {
		l.AddNote(note)
	}
	return l
}
