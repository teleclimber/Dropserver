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

	err := m.update(123, nil, 0, 0)
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
		Instance:   "ds-test",
		LocalID:    456,
		OwnerID:    domain.UserID(123),
		AppID:      domain.AppID(456),
		Version:    domain.Version("0.5.0"),
		AppspaceID: domain.NewNullAppspaceID(),
		Operation:  "test-op",
		CGroup:     "test-cgroup"}
	start := time.Now()

	id, err := m.Create(ids, start)
	if err != nil {
		t.Error(err)
	}
	ids.SandboxID = id

	runs, err := m.GetApp(ids.OwnerID, ids.AppID)
	if err != nil {
		t.Error(err)
	}
	if len(runs) != 1 {
		t.Fatal("expected one run")
	}
	data := domain.SandboxRunData{
		Start:   start,
		End:     nulltypes.NewTime(time.Now(), false),
		CpuTime: 0,
		Memory:  0,
	}
	c := domain.SandboxRun{ids, data}
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
		Instance:   "ds-test",
		LocalID:    456,
		OwnerID:    domain.UserID(123),
		AppID:      domain.AppID(456),
		Version:    domain.Version("0.5.0"),
		AppspaceID: domain.NewNullAppspaceID(appspaceID),
		Operation:  "test-op",
		CGroup:     "test-cgroup"}
	start := time.Now()
	end := time.Now().Add(time.Minute)

	id, err := m.Create(ids, start)
	if err != nil {
		t.Error(err)
	}
	ids.SandboxID = id

	err = m.End(ids.SandboxID, end, 777, 128)
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
	data := domain.SandboxRunData{
		Start:   start,
		End:     nulltypes.NewTime(end, true),
		CpuTime: 777,
		Memory:  128,
	}
	c := domain.SandboxRun{ids, data}
	if !cmp.Equal(c, runs[0]) {
		t.Log(cmp.Diff(c, runs[0]))
		t.Error("found differences in expected output")
	}
}
