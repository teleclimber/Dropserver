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

// Migrator manages the migration process
type Migrator struct {
	OrderedSteps []string
	StringSteps  map[string]migrationStep
	Config       *domain.RuntimeConfig `checkinject:"required"`
	DBManager    interface {
		GetHandle() *domain.DB
		GetSchema() string
		SetSchema(string) error
	} `checkinject:"required"`

	// import other things that migration steps need to touch
}

// LastStepName returns the last (current) schema name
func (m *Migrator) LastStepName() string {
	return m.OrderedSteps[len(m.OrderedSteps)-1]
}

// Migrate transforms the DB and anything else to match schema at "to"
// if to is "" it will migrate to the last step.
func (m *Migrator) Migrate(to string) error {
	// get current migration level
	// find from and to in orderedMigrations
	// -- nodejs version created backups. We should make that optional
	// launch migrations

	from := m.DBManager.GetSchema() // may need to return an error? or is blank string th eonly thing that matters?

	var fromIndex = -1
	if from != "" {
		var ok bool
		fromIndex, ok = m.indexOf(from)
		if !ok {
			return errors.New("Migration string not found. Migration string: " + from)
		}
	}

	if to == "" {
		// if to is not specified go to the last step
		to = m.LastStepName()
	}

	toIndex, ok := m.indexOf(to)
	if !ok {
		return errors.New("Migration string not found. Migration string: " + to)
	}

	if fromIndex == toIndex {
		return errors.New("Migration nonsensical: from and to are the same")
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

func (m *Migrator) doStep(index int, up bool) error {
	stepStr := m.OrderedSteps[index]
	mStep, ok := m.StringSteps[stepStr]
	if !ok {
		return fmt.Errorf("Could not find migration step at %d", index)
	}

	args := &stepArgs{
		db: m.DBManager.GetHandle()}

	var err error
	if up {
		err = mStep.up(args)
	} else {
		err = mStep.down(args)
	}

	// ^^ I would like to check for the existence of function before we call it
	// otherwise panics are hard to reason about.
	// Other option is to use an interface somehow

	if err != nil {
		// do some cleaning up?
		return err
	}

	if up {
		fmt.Println("Completed migration step: up", stepStr)
	} else {
		stepStr = m.OrderedSteps[index-1]
		fmt.Println("Completed migration step: down", stepStr)
	}

	err = m.DBManager.SetSchema(stepStr)
	if err != nil {
		return err
	}

	// ^^ should we really do this here?
	// it could be easier to have Migrate do it once, instead of at each step?
	// Also, migrate should set it to something recognizable as in-transit?

	return nil
}

func (m *Migrator) indexOf(strStep string) (index int, ok bool) {
	for i, val := range m.OrderedSteps {
		if strStep == val {
			return i, true
		}
	}
	return -1, false
}
