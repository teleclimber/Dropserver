package migrate

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

func TestIndexOf(t *testing.T) {
	orderedSteps := []string{"a", "b", "c"}
	m := Migrator{
		OrderedSteps: orderedSteps}

	cases := []struct {
		input string
		index int
		ok    bool
	}{
		{"a", 0, true},
		{"c", 2, true},
		{"z", -1, false},
	}

	for _, c := range cases {
		index, ok := m.indexOf(c.input)
		if index != c.index {
			t.Error("mismatched index for "+c.input, c)
		}
		if ok != c.ok {
			t.Error("mismatched ok for "+c.input, c)
		}
	}
}

func TestDoStepUp(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	called := false

	bStep := migrationStep{
		up: func(a *stepArgs) error {
			called = true
			return nil
		}}

	orderedSteps := []string{"a", "b"}
	stringSteps := map[string]migrationStep{
		"b": bStep,
	}

	dbm := testmocks.NewMockDBManager(mockCtrl)
	dbm.EXPECT().GetHandle().Return(&domain.DB{})
	dbm.EXPECT().SetSchema("b")

	m := Migrator{
		OrderedSteps: orderedSteps,
		StringSteps:  stringSteps,
		DBManager:    dbm}

	err := m.doStep(1, true)
	if err != nil {
		t.Error("should not have gotten error", err)
	}
	if !called {
		t.Error("migration function not called")
	}
}

func TestDoStepDown(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	called := false

	bStep := migrationStep{
		down: func(a *stepArgs) error {
			called = true
			return nil
		}}

	orderedSteps := []string{"a", "b"}
	stringSteps := map[string]migrationStep{
		"b": bStep,
	}

	dbm := testmocks.NewMockDBManager(mockCtrl)
	dbm.EXPECT().GetHandle().Return(&domain.DB{})
	dbm.EXPECT().SetSchema("a")

	m := Migrator{
		OrderedSteps: orderedSteps,
		StringSteps:  stringSteps,
		DBManager:    dbm}

	err := m.doStep(1, false)
	if err != nil {
		t.Error("should not have gotten error", err)
	}
	if !called {
		t.Error("migration function not called")
	}
}

// Test that an aerror returned by the step is passed to the caller.
func TestDoStepError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	bStep := migrationStep{
		up: func(a *stepArgs) error {
			return errors.New("Migration not possible")
		}}

	orderedSteps := []string{"a", "b"}
	stringSteps := map[string]migrationStep{
		"b": bStep,
	}

	dbm := testmocks.NewMockDBManager(mockCtrl)
	dbm.EXPECT().GetHandle().Return(&domain.DB{})

	m := Migrator{
		OrderedSteps: orderedSteps,
		StringSteps:  stringSteps,
		DBManager:    dbm}

	err := m.doStep(1, true)
	if err == nil {
		t.Error("should have gotten error")
	}
}

// test Migrate, which determines migration steps to run
// situations:
// - fromIndex -1 to index >=0 (fresh install to current?)
// - fromIndex n to index >= n (arbitrary up)
// - from/to equal
// - arbitrary down migration

// test migrate up from no schema "" to "a"
func TestMigrateFresh(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	called := false

	aStep := migrationStep{
		up: func(a *stepArgs) error {
			called = true
			return nil
		}}

	orderedSteps := []string{"a"}
	stringSteps := map[string]migrationStep{
		"a": aStep,
	}

	dbm := testmocks.NewMockDBManager(mockCtrl)
	dbm.EXPECT().GetSchema().Return("")
	dbm.EXPECT().GetHandle().Return(&domain.DB{})
	dbm.EXPECT().SetSchema("a")

	m := Migrator{
		OrderedSteps: orderedSteps,
		StringSteps:  stringSteps,
		DBManager:    dbm}

	err := m.Migrate("")
	if err != nil {
		t.Error("should not have gotten error", err)
	}
	if !called {
		t.Error("migration function not called")
	}
}

// test migrate down from b to a
func TestMigrateDown(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	called := false

	bStep := migrationStep{
		down: func(a *stepArgs) error {
			called = true
			return nil
		}}

	orderedSteps := []string{"a", "b"}
	stringSteps := map[string]migrationStep{
		"b": bStep,
	}

	dbm := testmocks.NewMockDBManager(mockCtrl)
	dbm.EXPECT().GetSchema().Return("b")
	dbm.EXPECT().GetHandle().Return(&domain.DB{})
	dbm.EXPECT().SetSchema("a")

	m := Migrator{
		OrderedSteps: orderedSteps,
		StringSteps:  stringSteps,
		DBManager:    dbm}

	err := m.Migrate("a")
	if err != nil {
		t.Error("should not have gotten error", err)
	}
	if !called {
		t.Error("migration function not called")
	}
}
