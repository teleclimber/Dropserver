package testmocks

import "github.com/teleclimber/DropServer/cmd/ds-host/domain"

//go:generate mockgen -destination=appspacelogger_mocks.go -package=testmocks github.com/teleclimber/DropServer/cmd/ds-host/testmocks LoggerI,AppspaceLogger

type LoggerI interface {
	Log(source, message string)
	SubscribeStatus() (bool, <-chan bool)
	UnsubscribeStatus(ch <-chan bool)
	GetLastBytes(n int64) (domain.LogChunk, error)
	SubscribeEntries(n int64) (domain.LogChunk, <-chan string, error)
	UnsubscribeEntries(ch <-chan string)
}

// Add AppLogger

type AppspaceLogger interface {
	Log(appspaceID domain.AppspaceID, source, message string)
	Get(appspaceID domain.AppspaceID) domain.LoggerI
	Open(appspaceID domain.AppspaceID) domain.LoggerI
	Close(appspaceID domain.AppspaceID)
	Forget(appspaceID domain.AppspaceID)
}
