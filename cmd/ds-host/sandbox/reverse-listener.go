package sandbox

import (
	"path"
	"sync"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
	"github.com/teleclimber/DropServer/internal/twine"
)

// This is really appspaceAPIServer or something like that
// it should be in its own module? (maybe not?)
// There is one server for all sandboxes. -> nope
// -> Somehow have to identify sandboxes even though you can't trust them.
// -> only reliable solution is to use unix sockets, one per sandbox?
// -> meaning one per live sandbox.
// Use this for http over unix domain socket:
// https://gist.github.com/teknoraver/5ffacb8757330715bcbcc90e6d46ac74

// Right now we have a really clumsy messageing system that essentially has no future
// I just need to send a port over and I'm not sure I can do it easily.
// Ultimately, if this is the system that accepts appsapceAPI requests,
// ..it will need to be far more capable, and probably HTTP based?
// -> or not HTTP. But maybe json-in-netstrings.
// .. or even more arbitrary: <cmd><size><arbitrary data string (json, raw log data, etc...)>
// ..where <cmd> is known finite size and always the same size (like 1 char)

//Let's think about how reverse listener passes stautus updates to sandbox?
// There will only be one recipient for status changes, right?
// Also, there aren't that many messages for status:
// - hi (with port)
// - bye (shutdown initiated)
// - something about the connection dropping unexpectedly
//   ..but is that even meaningful with HTTP? it doesn't expect the connection to "be there"

// This could potentially be completely separated out from sandbox package
// Particularly if sandbox struct itself is passed to reverse server.
// ..since that way you can always call back to sandbox to pass status/errors?
// -> but it's kind of better to have sb status close to sb, no?
// ..and pass everything else out to a more standard looking server.

type revStartStatus int

const (
	revFailStart revStartStatus = iota // TODO not be needed, just close it
	revReady
)

const (
	metaDbService int = 10
	routesService int = 11
	dbService     int = 12
)

type reverseListener struct {
	Config            *domain.RuntimeConfig
	AppspaceDBManager domain.AppspaceDBManager // This should probably be a ReverseServer (one per appspace)
	sessionID         int
	services          *domain.ReverseServices
	appspace          *domain.Appspace
	socketPath        string
	twine             *twine.Twine
	errorChan         chan domain.Error
	startChan         chan revStartStatus

	closedMux sync.Mutex
	closed    bool
}

// ^^ it should have a way of connecting to DBs of thei specific appspace
// It probably needs to know the appspace id
// ..and be given an object that allows it to pass all DB requests off?

//func initializeSockets() ...?

func newReverseListener(config *domain.RuntimeConfig, sessionID int, appspace *domain.Appspace, services *domain.ReverseServices) (*reverseListener, domain.Error) {
	rl := reverseListener{
		Config:    config,
		sessionID: sessionID,
		services:  services,
		appspace:  appspace,
		errorChan: make(chan domain.Error),
		startChan: make(chan revStartStatus),

		closed: false}

	rl.socketPath = path.Join(getSocketsDir(config.Sandbox.SocketsDir, appspace.AppspaceID), "rev.sock")

	twine, err := twine.NewServer(rl.socketPath)
	if err != nil {
		return nil, dserror.FromStandard(err)
	}
	rl.twine = twine

	go rl.monitorReady()
	go rl.monitorErrors()
	go rl.monitorMessages()

	return &rl, nil
}

func (rl *reverseListener) monitorReady() {
	_, ok := <-rl.twine.ReadyChan
	if !ok {
		rl.startChan <- revFailStart
	} else {
		rl.startChan <- revReady
	}
}
func (rl *reverseListener) monitorErrors() {
	for err := range rl.twine.ErrorChan {
		rl.errorChan <- dserror.FromStandard(err)
	}
}
func (rl *reverseListener) monitorMessages() {

	for message := range rl.twine.MessageChan {
		// switch over service
		switch message.ServiceID() {
		case metaDbService:
		case routesService:
			rl.services.Routes.Command(rl.appspace, message)
		case dbService:
		}
	}
}

// func (rl *reverseListener) send(msg string) { // return err?
// 	_, err := (*rl.conn).Write([]byte(msg))
// 	if err != nil {
// 		fmt.Println(err)
// 		os.Exit(1)
// 	}
// }
// func (rl *reverseListener) waitFor(msg string) {
// 	done := make(chan bool)
// 	rl.msgSub[msg] = done
// 	<-done
// 	delete(rl.msgSub, msg)
// }
// func (rl reverseListener) close() {
// 	//conn.end() or some such
// 	// err := os.Remove(rl.sockPath)
// 	// if err != nil {
// 	// 	fmt.Println(err)
// 	// 	//os.Exit(1)	// don't exit. if file didn't exist it errs.
// 	// }
// }

func (rl *reverseListener) close() {
	rl.closedMux.Lock()
	defer rl.closedMux.Unlock()

	if rl.closed {
		return
	}

	// if rl.conn != nil {
	// 	rc := *rl.conn
	// 	rc.Close() //might return an error
	// }

	close(rl.errorChan)
	close(rl.startChan)
	//close()

	rl.closed = true
}
