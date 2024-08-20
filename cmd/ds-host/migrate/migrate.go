package migrate

import (
	"errors"
	"fmt"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// handles migrations of dropserver data.
// Including initial installation

// We realize that migrations can involve the DB
// as well as any other data.

var ErrNoMigrationNeeded = errors.New("no migration needed")

// Migrator manages the migration process
type Migrator struct {
	Steps     []MigrationStep       `checkinject:"required"`
	Config    *domain.RuntimeConfig `checkinject:"required"`
	DBManager interface {
		GetHandle() *domain.DB
		GetSchema() string
		SetSchema(string) error
	} `checkinject:"required"`
}

// LastStepName returns the last (current) schema name
func (m *Migrator) LastStepName() string {
	return m.Steps[len(m.Steps)-1].name
}

func (m *Migrator) AppspaceMigrationRequired(to string) (bool, error) {
	fromIndex, toIndex, err := m.getIndices(to)
	if err != nil {
		return false, err
	}
	if fromIndex == -1 { // if there is no schema -> it's a new data dir -> no appspaces -> don't migrate.
		return false, nil
	}
	return m.Steps[fromIndex].appspaceMetaDBSchema != m.Steps[toIndex].appspaceMetaDBSchema, nil
}

// Migrate transforms the DB and anything else to match schema at "to"
// if to is "" it will migrate to the last step.
func (m *Migrator) Migrate(to string) error {
	fromIndex, toIndex, err := m.getIndices(to)
	if err != nil {
		return err
	}
	if fromIndex == toIndex {
		return ErrNoMigrationNeeded
	}
	if toIndex > fromIndex {
		for i := fromIndex + 1; i <= toIndex; i++ {
			err := m.doStep(i, true)
			if err != nil {
				return err
			}
		}
	} else {
		for i := fromIndex; i > toIndex; i-- {
			err := m.doStep(i, false)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// getIndices return the indices of the to and from migration steps
// based on current state of DB and the "to" string
func (m *Migrator) getIndices(to string) (fromIndex int, toIndex int, err error) {
	var ok bool

	from := m.DBManager.GetSchema()
	fromIndex = -1
	if from != "" {
		fromIndex, ok = m.indexOf(from)
		if !ok {
			err = errors.New("Migration string not found. Migration string: " + from)
			return
		}
	}

	if to == "" {
		// if to is not specified go to the last step
		to = m.LastStepName()
	}
	toIndex, ok = m.indexOf(to)
	if !ok {
		err = errors.New("migration string not found. Migration string: " + to)
	}

	return
}

func (m *Migrator) doStep(index int, up bool) error {
	mStep := m.Steps[index]

	args := &stepArgs{
		db: m.DBManager.GetHandle()}

	var err error
	if up {
		err = mStep.up(args)
	} else {
		err = mStep.down(args)
	}

	if err != nil {
		// do some cleaning up?
		return err
	}

	if up {
		fmt.Println("Completed migration step: up", mStep.name)
	} else {
		mStep = m.Steps[index-1]
		fmt.Println("Completed migration step: down", mStep.name)
	}

	err = m.DBManager.SetSchema(mStep.name)
	if err != nil {
		return err
	}

	return nil
}

func (m *Migrator) indexOf(strStep string) (index int, ok bool) {
	for i, val := range m.Steps {
		if strStep == val.name {
			return i, true
		}
	}
	return -1, false
}
