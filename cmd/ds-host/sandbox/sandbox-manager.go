package sandbox

import (
	"os"
	"sort"
	"sync"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// This is going to be different and quite simplified
// each sandbox is always tied to an appspace id
// So just have a map appspaceID[*sandbox]
// -> some question on how to deal with sandbox shutting down while a request for it arrives
// There is no recycling. You start the sb for an appspace, and then kill it completely.

// Manager manages sandboxes
type Manager struct {
	AppspaceLogger interface {
		Log(domain.AppspaceID, string, string)
	}
	nextID    int
	poolMux   sync.Mutex
	sandboxes map[domain.AppspaceID]domain.SandboxI // all sandboxes are always committed

	Services interface {
		Get(appspace *domain.Appspace, api domain.APIVersion) domain.ReverseServiceI
	}

	Config *domain.RuntimeConfig
}

type request struct {
	appSpace        string
	app             string
	sandboxChannels []chan domain.SandboxI
}

// Init creates maps
func (sM *Manager) Init() {
	sM.sandboxes = make(map[domain.AppspaceID]domain.SandboxI)

	err := os.MkdirAll(sM.Config.Sandbox.SocketsDir, 0700)
	if err != nil {
		panic(err)
	}
}

// StopAll takes all known sandboxes and stops them
func (sM *Manager) StopAll() {
	var stopWg sync.WaitGroup
	for _, c := range sM.sandboxes {
		// If we get to this point assume the connection from the host http proxy has been stopped
		// so it should be safe to shut things down
		// ..barring anything "waiting for"...
		go func(sb domain.SandboxI) {
			stopWg.Add(1)
			sb.Graceful()
			stopWg.Done()
		}(c)
	}

	stopWg.Wait()
}

// startSandbox launches a new Node/deno instance for a specific sandbox
// not sure if it should return a channel or just a started sb.
// Problem is if it takes too long, would like to independently send timout as response to request.
func (sM *Manager) startSandbox(appVersion *domain.AppVersion, appspace *domain.Appspace, ch chan domain.SandboxI) {
	sandboxID := sM.nextID
	sM.nextID++ // TODO: this could fail if creating mutliple sandboxes at once. Use a mutex to lock!
	// .. or trust that it only gets called with poolMux locked by caller.

	newSandbox := NewSandbox(sandboxID, appVersion, appspace, sM.Services.Get(appspace, appVersion.APIVersion), sM.Config)
	newSandbox.AppspaceLogger = sM.AppspaceLogger
	sM.sandboxes[appspace.AppspaceID] = newSandbox

	sM.recordSandboxStatusMetric()

	go func() {
		err := newSandbox.Start()
		if err != nil {
			close(ch)
			newSandbox.Kill()
			return
		}
		newSandbox.WaitFor(domain.SandboxReady)
		// sandbox may not be ready if it failed to start.
		// check status? Or maybe status ought to be checked by proxy for each request anyways?
		ch <- newSandbox
	}()
}

// GetForAppSpace records the need for a sandbox and returns a channel
// OK, this might work
func (sM *Manager) GetForAppSpace(appVersion *domain.AppVersion, appspace *domain.Appspace) chan domain.SandboxI {
	ch := make(chan domain.SandboxI)

	// get appVersion from model
	// Same for app if needed.

	sM.poolMux.Lock()
	defer sM.poolMux.Unlock()

	c, ok := sM.sandboxes[appspace.AppspaceID]
	if ok {
		//OK, but is it ready yet?
		// it may have *just* been started, so it'll get there but have to wait
		go func() {
			c.WaitFor(domain.SandboxReady)
			ch <- c
			sM.recordSandboxStatusMetric() // really?
		}()
	} else {
		// Here we may want to start a sandbox.
		// But we may not have enough RAM available to do it efficiently?
		// For now just start one up. We'll fine-tune later.
		// OK, but still need to queue up requests? .. or not.
		// -> this could be the queueing mechanism.

		sM.startSandbox(appVersion, appspace, ch)
		// this ought to return quickly, like as soon as the sandbox data is established.
		// .. so as to not tie up poolMux

		go sM.killPool()
	}

	return ch
}

// StopAppspace is used to stop an appspace sandbox from running if there is one
// it returns if/when no sanboxes are running for that appspace
func (sM *Manager) StopAppspace(appspaceID domain.AppspaceID) {
	sM.poolMux.Lock()
	s, ok := sM.sandboxes[appspaceID]
	if !ok {
		sM.poolMux.Unlock()
		return
	}

	delete(sM.sandboxes, appspaceID)
	sM.poolMux.Unlock()

	s.Graceful()

	s.WaitFor(domain.SandboxDead)
}

// TODO: lots of likely problems with sandbox manager due to lack of tests?

type killable struct {
	appspaceID domain.AppspaceID
	score      float64
}

// Look for sandboxes to shut down to make room for others before you run out of resources.
func (sM *Manager) killPool() {
	sM.poolMux.Lock()
	defer sM.poolMux.Unlock()

	// Need to use num sb from config I think.
	numC := sM.Config.Sandbox.Num
	numKill := len(sM.sandboxes) - numC

	if numKill > 0 {
		var sortedKillable []killable

		for appspaceID, sb := range sM.sandboxes {
			if sb.Status() == domain.SandboxReady && !sb.TiedUp() {
				sortedKillable = append(sortedKillable, killable{
					appspaceID,
					time.Since(sb.LastActive()).Seconds()})
			}
		}

		sort.Slice(sortedKillable, func(i, j int) bool {
			return sortedKillable[i].score > sortedKillable[j].score
		})

		for i := 0; i < numKill && i < len(sortedKillable); i++ {
			appspaceID := sortedKillable[i].appspaceID
			sandbox := sM.sandboxes[appspaceID]
			delete(sM.sandboxes, appspaceID)
			sandbox.SetStatus(domain.SandboxKilling) // have to set it here to prevent other requests being dispatched to it before it actually starts shutting down.
			go sandbox.Graceful()
		}
	}

	go sM.recordSandboxStatusMetric()
}

//
// func (sM *Manager) SendReverse(appspaceID domain.AppspaceID, msg domain.ReverseMessage) {
// 	sM.GetForAppSpace()	// argh, expects a appVersion.
//  // This is a bigger deal and not terribly efficient to *reply*.
//  // But will be OK for a call initiated from host (cron)
//  // ..where you expect to create a sandbox if there isn't one.
// }

func (sM *Manager) recordSandboxStatusMetric() {
	// var s = &record.SandboxStatuses{ //TODO nope do not use imported record. inject instead
	// 	Starting:   0,
	// 	Ready:      0,
	// 	Committing: 0,
	// 	Committed:  0,
	// 	Recycling:  0}
	// for _, c := range sM.sandboxes {
	// 	switch c.Status {
	// 	case "starting":
	// 		s.Starting++
	// 	case "ready":
	// 		s.Ready++
	// 	case "committing":
	// 		s.Committing++
	// 	case "committed":
	// 		s.Committed++
	// 	case "recycling":
	// 		s.Recycling++
	// 	}
	// }
	// record.SandboxStatusCounts(s) //TODO nope
}

// PrintSandboxes outputs containersa and status
func (sM *Manager) PrintSandboxes() {
	// var readys []string
	// for rc := sM.readySandboxes.Front(); rc != nil; rc = rc.Next() {
	// 	readys = append(readys, rc.Value.(*Sandbox).Name)
	// }

	// fmt.Println("Ready Sandboxes", readys)

	// fmt.Println("Committed sandboxes", sM.committedSandboxes)
	// for _, c := range sM.sandboxes {
	// 	tiedUp := "not-tied"
	// 	if c.appSpaceSession.tiedUp {
	// 		tiedUp = "tied-up"
	// 	}
	// 	fmt.Println(c.Name, c.Status, c.appSpaceID, tiedUp, c.recycleScore)
	// }
}
