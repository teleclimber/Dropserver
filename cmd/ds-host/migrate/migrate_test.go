package migrate

import (
	"errors"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

func TestIndexOf(t *testing.T) {
	orderedSteps := []MigrationStep{{"a", nil, nil, 0}, {"b", nil, nil, 0}, {"c", nil, nil, 0}}
	m := Migrator{
		Steps: orderedSteps}

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

	aStep := MigrationStep{name: "a"}
	bStep := MigrationStep{
		name: "b",
		up: func(a *stepArgs) error {
			called = true
			return nil
		}}

	dbm := testmocks.NewMockDBManager(mockCtrl)
	dbm.EXPECT().GetHandle().Return(&domain.DB{})
	dbm.EXPECT().SetSchema("b")

	m := Migrator{
		Steps:     []MigrationStep{aStep, bStep},
		DBManager: dbm}

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

	aStep := MigrationStep{name: "a"}
	bStep := MigrationStep{
		name: "b",
		down: func(a *stepArgs) error {
			called = true
			return nil
		}}

	dbm := testmocks.NewMockDBManager(mockCtrl)
	dbm.EXPECT().GetHandle().Return(&domain.DB{})
	dbm.EXPECT().SetSchema("a")

	m := Migrator{
		Steps:     []MigrationStep{aStep, bStep},
		DBManager: dbm}

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

	aStep := MigrationStep{name: "a"}
	bStep := MigrationStep{
		name: "b",
		up: func(a *stepArgs) error {
			return errors.New("Migration not possible")
		}}

	dbm := testmocks.NewMockDBManager(mockCtrl)
	dbm.EXPECT().GetHandle().Return(&domain.DB{})

	m := Migrator{
		Steps:     []MigrationStep{aStep, bStep},
		DBManager: dbm}

	err := m.doStep(1, true)
	if err == nil {
		t.Error("should have gotten error")
	}
}

func TestGetIndices(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	dbm := testmocks.NewMockDBManager(mockCtrl)

	m := Migrator{
		Steps:     []MigrationStep{{"a", nil, nil, 0}, {"b", nil, nil, 0}, {"c", nil, nil, 0}},
		DBManager: dbm}

	cases := []struct {
		startSchema string
		toParam     string
		from        int
		to          int
		err         bool
	}{
		{"", "", -1, 2, false},
		{"a", "", 0, 2, false},
		{"b", "a", 1, 0, false},
		{"Z", "a", 0, 0, true},
		{"", "Z", 0, 0, true},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("start schema: %v var %v", c.startSchema, c.toParam), func(t *testing.T) {
			dbm.EXPECT().GetSchema().Return(c.startSchema)
			rFrom, rTo, rErr := m.getIndices(c.toParam)
			if c.err && rErr == nil {
				t.Error("expected error")
			} else if !c.err && rErr != nil {
				t.Errorf("got unexpected error: %v", rErr)
			}
			if rErr == nil && (rFrom != c.from || rTo != c.to) {
				t.Errorf("got unexpected results: %v %v, %v %v", rFrom, c.from, rTo, c.to)
			}
		})
	}
}

func TestAppspaceMigrationRequired(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	dbm := testmocks.NewMockDBManager(mockCtrl)

	m := Migrator{
		Steps:     []MigrationStep{{"a", nil, nil, 0}, {"b", nil, nil, 0}, {"c", nil, nil, 1}},
		DBManager: dbm}

	cases := []struct {
		startSchema string
		toParam     string
		result      bool
	}{
		{"", "", false},
		{"", "a", false},
		{"a", "", true},
		{"b", "a", false},
		{"c", "a", true},
		{"b", "a", false},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("start schema: %v var %v", c.startSchema, c.toParam), func(t *testing.T) {
			dbm.EXPECT().GetSchema().Return(c.startSchema)
			result, err := m.AppspaceMigrationRequired(c.toParam)
			if err != nil {
				t.Error(err)
			}
			if result != c.result {
				t.Errorf("got unexpected result: %v", result)
			}
		})
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

	aStep := MigrationStep{
		name: "a",
		up: func(a *stepArgs) error {
			called = true
			return nil
		}}

	dbm := testmocks.NewMockDBManager(mockCtrl)
	dbm.EXPECT().GetSchema().Return("")
	dbm.EXPECT().GetHandle().Return(&domain.DB{})
	dbm.EXPECT().SetSchema("a")

	m := Migrator{
		Steps:     []MigrationStep{aStep},
		DBManager: dbm}

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

	aStep := MigrationStep{name: "a"}
	bStep := MigrationStep{
		name: "b",
		down: func(a *stepArgs) error {
			called = true
			return nil
		}}

	dbm := testmocks.NewMockDBManager(mockCtrl)
	dbm.EXPECT().GetSchema().Return("b")
	dbm.EXPECT().GetHandle().Return(&domain.DB{})
	dbm.EXPECT().SetSchema("a")

	m := Migrator{
		Steps:     []MigrationStep{aStep, bStep},
		DBManager: dbm}

	err := m.Migrate("a")
	if err != nil {
		t.Error("should not have gotten error", err)
	}
	if !called {
		t.Error("migration function not called")
	}
}
