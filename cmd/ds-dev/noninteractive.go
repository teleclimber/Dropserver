package main

import (
	"fmt"
	"os"
)

type NonInteractive struct {
	AppWatcher *DevAppWatcher
	DevAppProcessEvents interface {
		Subscribe() (AppProcessEvent, <-chan AppProcessEvent)
		Unsubscribe(<-chan AppProcessEvent)
	} `checkinject:"required"`
	AppVersionEvents interface {
		Subscribe(chan<- string)
		Unsubscribe(chan<- string)
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

func (n *NonInteractive) LoadAppData() AppProcessEvent {
	_, procCh := n.DevAppProcessEvents.Subscribe()
	defer n.DevAppProcessEvents.Unsubscribe(procCh)

	verCh := make(chan string)
	n.AppVersionEvents.Subscribe(verCh)
	defer n.AppVersionEvents.Unsubscribe(verCh)

	go n.AppWatcher.ReprocessAppFiles()

	var lastProc AppProcessEvent
	for {
		select {
		case ev := <-procCh:
			if ev.Processing {
				fmt.Println(ev.Step)
			} else {
				lastProc = ev
			}
		case state := <-verCh:
			if state == "ready" || state == "error" {
				return lastProc
			}
		}
	}
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
