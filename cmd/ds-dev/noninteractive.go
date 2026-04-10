package main

import (
	"fmt"
	"os"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

type NonInteractive struct {
	AppGetter interface {
		Reprocess(userID domain.UserID, appID domain.AppID, locationKey string) (domain.AppGetKey, error)
		GetResults(key domain.AppGetKey) (domain.AppGetMeta, bool)
		DeleteKeyData(key domain.AppGetKey)
	}
	AppGetterEvents interface {
		SubscribeOwner(domain.UserID) <-chan domain.AppGetEvent
		Unsubscribe(<-chan domain.AppGetEvent)
	} `checkinject:"required"`
}

func (n *NonInteractive) LoadApp() {
	results := n.LoadAppData()
	if len(results.Errors) != 0 {
		for _, e := range results.Errors {
			fmt.Println(e)
		}
		fmt.Println("Loading app failed. Please fix the errors above and try again.")
		os.Exit(1)
	}

	if len(results.Warnings) != 0 {
		for k, w := range results.Warnings {
			fmt.Printf("Warning: %v: %s\n", k, w)
		}
	}
}

func (n *NonInteractive) LoadAppData() domain.AppGetMeta {
	appGetKey, err := n.AppGetter.Reprocess(ownerID, appID, "")
	if err != nil {
		panic(err)
	}

	appGetCh := n.AppGetterEvents.SubscribeOwner(ownerID)
	defer n.AppGetterEvents.Unsubscribe(appGetCh)

	rChan := make(chan domain.AppGetMeta, 1)
	done := false
	for e := range appGetCh {
		if e.Key != appGetKey {
			continue
		}
		if e.Done {
			if !done {
				fmt.Println("Done processing app")
				go func() { // have to do this to prevent deadlock
					r := n.getResults(appGetKey)
					rChan <- r
				}()
			}
			done = true
			go n.AppGetterEvents.Unsubscribe(appGetCh) // unsubscribe to stop for loop
		} else {
			fmt.Println(e.Step)
		}
	}
	return <-rChan
}

func (n *NonInteractive) getResults(appGetKey domain.AppGetKey) domain.AppGetMeta {
	results, ok := n.AppGetter.GetResults(appGetKey)
	if !ok {
		panic("no appGetKey. This is a bug in ds-dev.")
	}
	n.AppGetter.DeleteKeyData(appGetKey)
	return results
}

func checkOutputDir(outDir string) {
	info, err := os.Stat(outDir)
	if err == os.ErrNotExist {
		fmt.Println("Output dir does not exist: " + outDir)
		os.Exit(1)
	}
	if err != nil {
		fmt.Println("Error opening output dir: ", err)
		os.Exit(1)
	}
	if !info.IsDir() {
		fmt.Println("Output Directory is not a directory: " + outDir)
		os.Exit(1)
	}
}
