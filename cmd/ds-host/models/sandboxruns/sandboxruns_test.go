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

	err := m.update(123, nil, 0, 0, 0)
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
		Start:       start,
		End:         nulltypes.NewTime(time.Now(), false),
		CpuUsec:     0,
		MemoryBytes: 0,
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

	err = m.End(ids.SandboxID, end, 222, 777, 128)
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
		Start:       start,
		End:         nulltypes.NewTime(end, true),
		TiedUpMs:    222,
		CpuUsec:     777,
		MemoryBytes: 128,
	}
	c := domain.SandboxRun{ids, data}
	if !cmp.Equal(c, runs[0]) {
		t.Log(cmp.Diff(c, runs[0]))
		t.Error("found differences in expected output")
	}
}

func TestAppspaceSum(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	m := &SandboxRunsModel{
		DB: db}

	m.PrepareStatements()

	appspaceID1 := domain.AppspaceID(789)
	id1 := domain.SandboxRunIDs{
		Instance:   "ds-test",
		LocalID:    456,
		OwnerID:    domain.UserID(123),
		AppID:      domain.AppID(456),
		Version:    domain.Version("0.5.0"),
		AppspaceID: domain.NewNullAppspaceID(appspaceID1),
		Operation:  "test-op",
		CGroup:     "test-cgroup"}

	start1 := time.Date(2022, time.March, 18, 17, 0, 0, 0, time.UTC)
	createRun(m, t, id1, start1, 222, 1111, 128)

	start2 := time.Date(2022, time.April, 12, 12, 41, 0, 0, time.UTC)
	createRun(m, t, id1, start2, 222, 1111, 128)

	id2 := id1
	id2.AppspaceID = domain.NewNullAppspaceID(domain.AppspaceID(999))
	createRun(m, t, id2, start1, 999, 9999, 99)

	// empty set:
	sums, err := m.AppsaceSums(id1.OwnerID, appspaceID1, firstOf(time.February), firstOf(time.March))
	if err != nil {
		t.Error(err)
	}
	expected := domain.SandboxRunSums{
		TiedUpMs:     0,
		CpuUsec:      0,
		MemoryByteMs: 0,
	}
	if !cmp.Equal(sums, expected) {
		t.Log(cmp.Diff(sums, expected))
		t.Error("found differences in expected output")
	}

	// just one run, excludes run by other appspace
	sums, err = m.AppsaceSums(id1.OwnerID, appspaceID1, firstOf(time.March), firstOf(time.April))
	if err != nil {
		t.Error(err)
	}
	expected = domain.SandboxRunSums{
		TiedUpMs:     222,
		CpuUsec:      1111,
		MemoryByteMs: 222 * 128,
	}
	if !cmp.Equal(sums, expected) {
		t.Log(cmp.Diff(sums, expected))
		t.Error("found differences in expected output")
	}

	// Actually sum runs correctly:
	sums, err = m.AppsaceSums(id1.OwnerID, appspaceID1, firstOf(time.March), firstOf(time.May))
	if err != nil {
		t.Error(err)
	}
	expected = domain.SandboxRunSums{
		TiedUpMs:     444,
		CpuUsec:      2222,
		MemoryByteMs: 444 * 128,
	}
	if !cmp.Equal(sums, expected) {
		t.Log(cmp.Diff(sums, expected))
		t.Error("found differences in expected output")
	}
}

func createRun(m *SandboxRunsModel, t *testing.T, ids domain.SandboxRunIDs, start time.Time, tiedUpMs int, cpuUsec int, memBytes int) {
	id, err := m.Create(ids, start)
	if err != nil {
		t.Fatal(err)
	}
	err = m.End(id, start, tiedUpMs, cpuUsec, memBytes)
	if err != nil {
		t.Error(err)
	}
}

func firstOf(m time.Month) time.Time {
	return time.Date(2022, m, 1, 0, 0, 0, 0, time.UTC)
}
