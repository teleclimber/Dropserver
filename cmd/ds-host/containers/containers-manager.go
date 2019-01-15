package containers

import (
	"fmt"
)

// Manager manages containers
type Manager struct {
	containers []*Container
}

// StartContainer actually just reccycles an existing container for now
func (cM *Manager) StartContainer() {
	newContainer := Container{
		Name:       "c7",
		Status:     "starting",
		Address:    "http://10.140.177.203:3030",
		appSpaceID: "",
		statusSub:  make(map[string][]chan bool)}

	cM.containers = append(cM.containers, &newContainer)
	// ^^ you want it in there early so that you don't start another one?

	newContainer.recycleListener = newRecycleListener("c7", newContainer.onRecyclerMsg)

	newContainer.recycle()
}

// GetForAppSpace tries to find an available container for an app-space
func (cM *Manager) GetForAppSpace(app string, appSpace string) (retContainer *Container, ok bool) {
	// first look to see if there is a container that is already commited
	for _, c := range cM.containers {
		if c.appSpaceID == appSpace {
			retContainer = c
			ok = true //note that it might not be ready!
			c.waitFor("committed")
			break
		}
	}

	if !ok {
		// now see if there is a container we can commit
		for _, c := range cM.containers {
			if c.Status == "ready" && c.appSpaceID == "" {
				c.commit(app, appSpace)
				retContainer = c
				ok = true
				break
			}
		}
	}

	if !ok {
		// now see if there is a container we can commit
		for _, c := range cM.containers {
			if c.Status == "starting" || c.Status == "recycling" {
				// can we have a c.reserve?
				fmt.Println("reserving container that is starting or recycling")
				c.appSpaceID = appSpace
				c.waitFor("ready")
				c.commit(app, appSpace)
				retContainer = c
				ok = true
				break
			}
		}
	}

	// next we can also try to recycle a container or start a new one
	if !ok {
		// now see if there is a container we can recycle
		var candidate *Container
		for _, c := range cM.containers {
			if c.Status == "committed" && !c.isTiedUp() {
				if candidate == nil {
					candidate = c
				} else if candidate.appSpaceSession.lastActive.After(c.appSpaceSession.lastActive) {
					candidate = c
				}
				// loop over tasks and see the latest finish time
				// recycle the container with the longest idle state
				// or other option is to keep a running tally for each container?

				fmt.Println("reserving container that is starting or recycling")
				c.appSpaceID = appSpace
				c.waitFor("ready")
				c.commit(app, appSpace)
				retContainer = c
				ok = true
				break
			}
		}

		if candidate != nil {
			// go ahead and recycle this one
			candidate.recycle()
			candidate.commit(app, appSpace)
			retContainer = candidate
			ok = true
		}
	}

	return
}
