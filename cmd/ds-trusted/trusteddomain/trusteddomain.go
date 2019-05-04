package trusteddomain


//go:generate mockgen -destination=mocks.go -package=trusteddomain github.com/teleclimber/DropServer/cmd/ds-trusted/trusteddomain AppFilesI
// ^^ remember to add new interfaces to list of interfaces to mock ^^


import (
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// AppFilesI is the interface for saving and reading files of an application
type AppFilesI interface {
	Save(*domain.TrustedSaveAppFiles) (string, domain.Error)
	ReadMeta(string) (*domain.AppFilesMetadata, domain.Error)
}


