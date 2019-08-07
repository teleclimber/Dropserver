package sandbox

import (
	"fmt"
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
	nextID    int //?
	poolMux   sync.Mutex
	requests  map[string]request             // unused?
	sandboxes map[domain.AppspaceID]*Sandbox // all sandboxes are always committed
	Config    *domain.RuntimeConfig
	Logger    domain.LogCLientI
	// metrics, ...
}

type request struct {
	appSpace        string
	app             string
	sandboxChannels []chan domain.SandboxI
}

// Init [used to] zaps existing sandboxes and creates fresh ones
// Now it just cleans up possible earlier sandboxes
// It does not create new ones until a request comes in.
// Hmm, it's possible host crashes while sandboxes stay up.
// How to deal?
// -> at least start with clean set up:
// - no deno processes running
// - no unix domain sockets
// - probably need a host session ID, and only talk to those who have that session.
func (sM *Manager) Init() {
	// ??
}

// StopAll takes all known sandboxes and stops them
func (sM *Manager) StopAll() {
	var stopWg sync.WaitGroup
	for _, c := range sM.sandboxes {
		// If we get to this point assume the connection from the host http proxy has been stopped
		// so it should be safe to shut things down
		// ..barring anything "waiting for"...
		go func(sb *Sandbox) {
			stopWg.Add(1)
			sb.Stop()
			stopWg.Done()
		}(c)
	}

	stopWg.Wait()
}

// startSandbox launches a new Node/deno instance for a specific sandbox
// not sure if it should return a channel or just a started sb.
// Problem is if it takes too long, would like to independently send timout as response to request.
func (sM *Manager) startSandbox(appspace *domain.Appspace, ch chan domain.SandboxI) {
	sandboxID := sM.nextID
	sM.nextID++ // TODO: this could fail if creating mutliple sandboxes at once. Use a service to lock!
	// .. or trust that it only gets called with poolMux locked by caller.

	fmt.Println("Creating new Sandbox", appspace.AppspaceID)

	newSandbox := Sandbox{ // <-- this really needs a maker fn of some sort??
		SandboxID: sandboxID,
		Status:    statusStarting,
		appspace:  appspace,
		statusSub: make(map[statusInt][]chan statusInt),
		LogClient: sM.Logger.NewSandboxLogClient(sandboxID)}

	sM.sandboxes[appspace.AppspaceID] = &newSandbox

	sM.recordSandboxStatusMetric()

	go func() {
		newSandbox.start()
		newSandbox.waitFor(statusReady)
		// sandbox may not be ready if it failed to start.
		// check status? Or maybe status ought to be checked by proxy for each request anyways?
		ch <- &newSandbox
	}()
}

// GetForAppSpace records the need for a sandbox and returns a channel
// OK, this might work
func (sM *Manager) GetForAppSpace(appspace *domain.Appspace) chan domain.SandboxI {
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
			c.waitFor(statusReady)
			ch <- c
			sM.recordSandboxStatusMetric() // really?
		}()
	} else {
		// Here we may want to start a sandbox.
		// But we may not have enough RAM available to do it efficiently?
		// For now just start one up. We'll fine-tune later.
		// OK, but still need to queue up requests? .. or not.
		// -> this could be the queueing mechanism.

		sM.startSandbox(appspace, ch) // pass the channel?
		// this ought to return quickly, like as soon as the sandbox data is established.
		// .. so as to not tie up poolMux

		go sM.killPool()
	}

	return ch
}

// Look for sandboxes to shut down to make room for others before you run out of resources.
func (sM *Manager) killPool() {
	sM.poolMux.Lock()
	defer sM.poolMux.Unlock()

	// Need to use num sb from config I think.
	numC := sM.Config.Sandbox.Num
	numKill := len(sM.sandboxes) - numC

	if numKill > 0 {
		var sortedSandboxes []*Sandbox

		for _, sb := range sM.sandboxes {
			if sb.Status == statusReady && !sb.appSpaceSession.tiedUp {
				duration := time.Since(sb.appSpaceSession.lastActive)
				sb.killScore = duration.Seconds() //kill least recently active
				sortedSandboxes = append(sortedSandboxes, sb)
			}
		}

		sort.Slice(sortedSandboxes, func(i, j int) bool {
			return sortedSandboxes[i].killScore > sortedSandboxes[j].killScore
		})

		for i := 0; i < numKill && i < len(sortedSandboxes); i++ {
			sandbox := sortedSandboxes[i]
			sandbox.Status = statusKilling // TODO: this should not be set here
			//sandbox.appspaceID = nil
			go sandbox.Stop() // function signature requires waitgroup, but we don't want to prevent this function from returning!
		}
	}

	go sM.recordSandboxStatusMetric()
}

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
