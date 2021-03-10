package testmocks

import "github.com/teleclimber/DropServer/cmd/ds-host/domain"

//go:generate mockgen -destination=db_mocks.go -package=testmocks github.com/teleclimber/DropServer/cmd/ds-host/testmocks DBManager

// DBManager is Migration interface
type DBManager interface {
	Open() (*domain.DB, error)
	GetHandle() *domain.DB
	GetSchema() string
	SetSchema(string) error
}
