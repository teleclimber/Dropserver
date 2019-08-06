package sandbox

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
	"github.com/teleclimber/DropServer/internal/shiftpath"
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

//Let's think about how reverse listener passes stautus updates to sandbox?
// There will only be one recipient for status changes, right?
// Also, there aren't that many messages for status:
// - hi (with port)
// - bye (shutdown initiated)
// - something about the connection dropping unexpectedly
//   ..but is that even meaningful with HTTP? it doesn't expect the connection to "be there"

type reverseListener struct {
	server     *http.Server
	socketPath string
	errorChan  chan domain.Error
	portChan   chan int
	portSent   bool
}

//func initializeSockets() ...?

func newReverseListener(config *domain.RuntimeConfig, ID int) (*reverseListener, domain.Error) {
	rl := reverseListener{
		socketPath: path.Join(config.Sandbox.SocketsDir, fmt.Sprintf("%d.sock", ID)),
		errorChan:  make(chan domain.Error),
		portChan:   make(chan int),
		portSent:   false}

	// I thgink we shold also create the directory just in case it's not there?
	// Or we need a general initialization function that sets the directory up and removes everything
	// ..so that we don't delay things here.

	if err := os.RemoveAll(rl.socketPath); err != nil {
		log.Print(err) //TODO: proper logger please
		// log error then return nil, err.
	}

	rl.server = &http.Server{
		Handler: &rl,
	}

	unixListener, err := net.Listen("unix", rl.socketPath)
	if err != nil {
		panic(err) // TODO: proper errors please
		// log error then return nil, err.
	}

	go rl.server.Serve(unixListener)

	return &rl, nil
}
func (rl *reverseListener) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	head, URLTail := shiftpath.ShiftPath(req.URL.Path)
	if head == "status" {
		if req.Method == http.MethodPost {
			if URLTail == "/hi" {
				rl.handleHi(w, req)
			} else {
				w.WriteHeader(404) // TODO: each error / 404 should be logged
				// log
			}
		} else {
			w.WriteHeader(404)
			// log
		}
	} else {
		w.WriteHeader(404)
		// log
	}
	// else if head is appspaceAPI then swith off to appspaceapi package (separate)
}
func (rl *reverseListener) handleHi(w http.ResponseWriter, req *http.Request) {
	// error handling here:
	// There shouldn't be errors?
	// So if there is one prob means someone is probing system?
	// Should sandbox continue if it triggers errors here?
	// Why not whats the harm?
	// just make sure these are logged on host.
	// Actually if handle Hi has a problem, we should probably shut it down?

	body, err := ioutil.ReadAll(io.LimitReader(req.Body, 1024))
	if err != nil {
		w.WriteHeader(422)
		// TODO log the error
		rl.errorChan <- dserror.New(dserror.SandboxReverseBadData, "Could not readall")
		return
	}
	if err := req.Body.Close(); err != nil {
		w.WriteHeader(422)
		// TODO: log
		rl.errorChan <- dserror.New(dserror.SandboxReverseBadData, "Failed to close body")
		return
	}

	var hiData struct {
		Port int `json:"port"`
	}
	if err := json.Unmarshal(body, &hiData); err != nil {
		w.WriteHeader(422)
		//TODO: log
		rl.errorChan <- dserror.New(dserror.SandboxReverseBadData, "Could not Unmarshall JSON")
		return
	}

	if hiData.Port < 1000 {
		w.WriteHeader(422)
		//panic("got port less than 1000 for sandbox")
		//TODO: log
		rl.errorChan <- dserror.New(dserror.SandboxReverseBadData, "Port less than 1000")
		return
	}

	if rl.portSent {
		w.WriteHeader(500)
		// TODO: log
		rl.errorChan <- dserror.New(dserror.SandboxReverseBadData, "Port already sent")
		return
	}

	w.WriteHeader(200)
	rl.portChan <- hiData.Port
	rl.portSent = true
}

func (rl reverseListener) close() {
	// TODO: need to shut down the server when that makes sense to do so
	rl.server.Close()
	//rl.server.Shutdown(ctx) //graceful. Use Close to kill conns. Not clear which to use

}
