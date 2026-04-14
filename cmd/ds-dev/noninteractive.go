package main

import (
	"fmt"
	"os"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

type NonInteractive struct {
	AppWatcher          *DevAppWatcher
	DevAppModel         *DevAppModel
	DevAppProcessEvents interface {
		Subscribe() (AppProcessEvent, <-chan AppProcessEvent)
		Unsubscribe(<-chan AppProcessEvent)
	} `checkinject:"required"`
	AppVersionEvents interface {
		Subscribe(chan<- string)
		Unsubscribe(chan<- string)
	} `checkinject:"required"`
	AppLogger interface {
		Open(string) domain.LoggerI
		Close(string)
	} `checkinject:"required"`
	AppspaceInfoModel interface {
		GetSchema(domain.AppspaceID) (int, error)
	} `checkinject:"required"`
	AppspaceLogger interface {
		Close(domain.AppspaceID)
		Open(domain.AppspaceID) domain.LoggerI
	} `checkinject:"required"`
	AppspaceStatus interface {
		WaitTempPaused(domain.AppspaceID, string) chan struct{}
	} `checkinject:"required"`
	DevSandboxManager interface {
		StopAppspace(domain.AppspaceID)
	} `checkinject:"required"`
	AppspaceMetaDB interface {
		CloseConn(domain.AppspaceID) error
	} `checkinject:"required"`
	MigrationJobModel interface {
		CreateFromSchema(migrateTo int) error
	} `checkinject:"required"`
	MigrationJobEvents interface {
		Subscribe() <-chan domain.MigrationJob
		Unsubscribe(<-chan domain.MigrationJob)
	} `checkinject:"required"`
}

func (n *NonInteractive) LoadApp() {
	appLog := n.AppLogger.Open("")
	stopAppLog := n.relayLog(appLog)

	results := n.LoadAppData()

	stopAppLog()
	n.AppLogger.Close("")

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

func (n *NonInteractive) Migrate() {
	n.LoadApp()

	asLog := n.AppspaceLogger.Open(appspaceID)
	stopAsLog := n.relayLog(asLog)

	migrationCh := n.MigrationJobEvents.Subscribe()
	defer n.MigrationJobEvents.Unsubscribe(migrationCh)

	err := n.MigrationJobModel.CreateFromSchema(n.DevAppModel.Ver.Schema) // migrate the appspace to the app's schema.
	if err != nil && err != errNoMigrationNeeded {
		panic(err)
	}
	if err == errNoMigrationNeeded {
		fmt.Printf("No migration needed (schema %v)\n", n.DevAppModel.Ver.Schema)
	} else {
		appspaceSchema, err := n.AppspaceInfoModel.GetSchema(appspaceID)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Migrating appspace from schema %v to schema %v\n", appspaceSchema, n.DevAppModel.Ver.Schema)

		for job := range migrationCh {
			if job.AppspaceID != appspaceID {
				continue
			}
			if job.Finished.Valid {
				if job.Error.Valid {
					fmt.Println("Migration failed: " + job.Error.String)
					os.Exit(1)
				}
				fmt.Println("Migration complete")
				break
			}
		}
	}

	stopAsLog()
}

// relayLog subscribes to a logger's entries and prints them to stdout.
// It returns a function that stops the relay and unsubscribes.
func (n *NonInteractive) relayLog(logger domain.LoggerI) func() {
	_, ch, err := logger.SubscribeEntries(0)
	if err != nil {
		return func() {}
	}
	done := make(chan struct{})
	go func() {
		for {
			select {
			case entry, ok := <-ch:
				if !ok {
					return
				}
				fmt.Println(entry)
			case <-done:
				return
			}
		}
	}()
	return func() {
		close(done)
		logger.UnsubscribeEntries(ch)
	}
}

func (n *NonInteractive) LoadAppData() AppProcessEvent {
	_, procCh := n.DevAppProcessEvents.Subscribe()
	defer n.DevAppProcessEvents.Unsubscribe(procCh)

	verCh := make(chan string)
	n.AppVersionEvents.Subscribe(verCh)
	defer n.AppVersionEvents.Unsubscribe(verCh)

	go n.AppWatcher.ReprocessAppFiles()

	logger := record.NewDsLogger().AddNote("LoadAppData")

	var lastProc AppProcessEvent
	for {
		select {
		case ev := <-procCh:
			if ev.Processing {
				logger.Log(ev.Step)
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
