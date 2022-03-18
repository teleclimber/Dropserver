package sandboxruns

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/migrate"
	"github.com/teleclimber/DropServer/internal/nulltypes"
)

func TestPrepareStatements(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	m := &SandboxRunsModel{
		DB: db}

	m.PrepareStatements()
}

func TestUpdateNoRow(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	m := &SandboxRunsModel{
		DB: db}

	m.PrepareStatements()

	err := m.update("foo", nil, 0, 0)
	if err == nil || err.Error() != "sandbox id not in database" {
		t.Errorf("Expected error: sandbox id not in database, got %v", err)
	}
}

func TestCreateApp(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	m := &SandboxRunsModel{
		DB: db}

	m.PrepareStatements()

	ids := domain.SandboxRunIDs{
		SandboxID:  "sandbox-1",
		OwnerID:    domain.UserID(123),
		AppID:      domain.AppID(456),
		Version:    domain.Version("0.5.0"),
		AppspaceID: domain.NewNullAppspaceID(),
		Operation:  "test-op",
		CGroup:     "test-cgroup"}
	start := time.Now()

	err := m.Create(ids, start)
	if err != nil {
		t.Error(err)
	}

	runs, err := m.GetApp(ids.OwnerID, ids.AppID)
	if err != nil {
		t.Error(err)
	}
	if len(runs) != 1 {
		t.Fatal("expected one run")
	}
	c := domain.SandboxRun{
		ids,
		start,
		nulltypes.NewTime(time.Now(), false),
		0,
		0}
	if !cmp.Equal(c, runs[0]) {
		t.Log(cmp.Diff(c, runs[0]))
		t.Error("found differences in expected output")
	}
}

func TestCreateEndAppspace(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	m := &SandboxRunsModel{
		DB: db}

	m.PrepareStatements()

	appspaceID := domain.AppspaceID(789)
	ids := domain.SandboxRunIDs{
		SandboxID:  "sandbox-1",
		OwnerID:    domain.UserID(123),
		AppID:      domain.AppID(456),
		Version:    domain.Version("0.5.0"),
		AppspaceID: domain.NewNullAppspaceID(appspaceID),
		Operation:  "test-op",
		CGroup:     "test-cgroup"}
	start := time.Now()
	end := time.Now().Add(time.Minute)

	err := m.Create(ids, start)
	if err != nil {
		t.Error(err)
	}

	err = m.End(ids.SandboxID, end, 7.7, 128)
	if err != nil {
		t.Error(err)
	}

	runs, err := m.GetAppspace(ids.OwnerID, appspaceID)
	if err != nil {
		t.Error(err)
	}
	if len(runs) != 1 {
		t.Fatal("expected one run")
	}
	c := domain.SandboxRun{
		ids,
		start,
		nulltypes.NewTime(end, true),
		7.7,
		128}
	if !cmp.Equal(c, runs[0]) {
		t.Log(cmp.Diff(c, runs[0]))
		t.Error("found differences in expected output")
	}
}
