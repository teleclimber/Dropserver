package testmocks

import "github.com/teleclimber/DropServer/cmd/ds-host/domain"

//go:generate mockgen -destination=appspacelogger_mocks.go -package=testmocks github.com/teleclimber/DropServer/cmd/ds-host/testmocks LoggerI,AppLogger,AppspaceLogger

type LoggerI interface {
	Log(source, message string)
	SubscribeStatus() (bool, <-chan bool)
	UnsubscribeStatus(ch <-chan bool)
	GetLastBytes(n int64) (domain.LogChunk, error)
	SubscribeEntries(n int64) (domain.LogChunk, <-chan string, error)
	UnsubscribeEntries(ch <-chan string)
}

type AppLogger interface {
	Log(locationKey string, source, message string)
	Get(locationKey string) domain.LoggerI
	Open(locationKey string) domain.LoggerI
	Close(locationKey string)
	Forget(locationKey string)
}

type AppspaceLogger interface {
	Log(appspaceID domain.AppspaceID, source, message string)
	Get(appspaceID domain.AppspaceID) domain.LoggerI
	Open(appspaceID domain.AppspaceID) domain.LoggerI
	Close(appspaceID domain.AppspaceID)
	Forget(appspaceID domain.AppspaceID)
}
