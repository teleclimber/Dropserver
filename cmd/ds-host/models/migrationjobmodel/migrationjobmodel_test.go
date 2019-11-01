package migrationjobmodel

import (
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/migrate"
	"github.com/teleclimber/DropServer/internal/dserror"
	"github.com/teleclimber/DropServer/internal/nulltypes"
)

func TestPrepareStatements(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	model := &MigrationJobModel{
		DB: db}

	model.PrepareStatements()
}

func TestGetPendingEmpty(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	model := &MigrationJobModel{
		DB: db}

	model.PrepareStatements()

	// There should be an error, but no panics
	_, dsErr := model.GetPending()
	if dsErr != nil {
		t.Error(dsErr)
	}
}

func TestCreate(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	model := &MigrationJobModel{
		DB: db}

	model.PrepareStatements()

	uID := domain.UserID(7)
	asID := domain.AppspaceID(9)

	job, err := model.Create(uID, asID, "0.0.1", true)
	if err != nil {
		t.Fatal(err)
	}
	if job.ToVersion != "0.0.1" {
		t.Fatal("job doesn't match")
	}
	if job.Created.IsZero() {
		t.Fatal("created should not be zero value")
	}
	if job.Started.Valid {
		t.Fatal("started should be null")
	}

	// now check that we replace OK:
	job2, err := model.Create(uID, asID, "0.0.9", true)
	if err != nil {
		t.Error(err)
	}
	if job2.ToVersion != "0.0.9" {
		t.Error("job doesn't match")
	}

	// now try to get the first one, it should have been replaced by job2
	_, err = model.GetJob(job.JobID)
	if err == nil {
		t.Error("should have errored. Job should not be present")
	}
	if err != nil && err.Code() != dserror.NoRowsInResultSet {
		t.Error(err)
	}
}

func TestSetStarted(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	model := &MigrationJobModel{
		DB: db}

	model.PrepareStatements()

	uID := domain.UserID(7)
	asID := domain.AppspaceID(9)

	job, err := model.Create(uID, asID, "0.0.1", true)
	if err != nil {
		t.Fatal(err)
	}
	if job.Started.Valid {
		t.Error("job started should not be valid")
	}

	ok, err := model.SetStarted(job.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected ok")
	}

	job, err = model.GetJob(job.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if !job.Started.Valid {
		t.Error("expected started to be Valid")
	}

	ok, err = model.SetStarted(999)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected not OK because it's a made up job ID")
	}
}

func TestGetPending(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	model := &MigrationJobModel{
		DB: db}

	model.PrepareStatements()

	uID := domain.UserID(7)
	asID1 := domain.AppspaceID(11)
	asID2 := domain.AppspaceID(12)

	job1, err := model.Create(uID, asID1, "0.0.1", true)
	if err != nil {
		t.Error(err)
	}

	job2, err := model.Create(uID, asID2, "0.0.9", false)
	if err != nil {
		t.Error(err)
	}

	jobs, err := model.GetPending()
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 2 {
		t.Error("should have gotten 2 jobs")
	}
	if jobs[0].JobID != job1.JobID {
		t.Error("got wrong job")
	}

	ok, err := model.SetStarted(jobs[0].JobID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("should be OK")
	}

	jobs, err = model.GetPending()
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 {
		t.Error("should have gotten 1 jobs")
	}
	if jobs[0].JobID != job2.JobID {
		t.Error("got wrong job")
	}

	// TODO: more rows in DB and change created dates
}

func TestSetFinished(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	model := &MigrationJobModel{
		DB: db}

	model.PrepareStatements()

	uID := domain.UserID(7)
	asID := domain.AppspaceID(9)

	// create one job
	job, err := model.Create(uID, asID, "0.0.1", true)
	if err != nil {
		t.Fatal(err)
	}

	var errStr nulltypes.NullString

	// try to set it to finished prematurely
	err = model.SetFinished(job.JobID, errStr)
	if err == nil {
		t.Fatal("expected an error")
	}
	if err.Code() != dserror.NoRowsAffected {
		t.Error("expected No Rows affected error")
	}

	// now actually set job to started
	ok, err := model.SetStarted(job.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("should have been ok")
	}

	// ..so that now SetFinished should work

	err = model.SetFinished(job.JobID, errStr)
	if err != nil {
		t.Fatal(err)
	}

	job, err = model.GetJob(job.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if !job.Finished.Valid {
		t.Error("expect job finished to be valid")
	}
	if job.Error.Valid {
		t.Error("expected Error to be null")
	}
}

func TestSetFinishedError(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	model := &MigrationJobModel{
		DB: db}

	model.PrepareStatements()

	uID := domain.UserID(7)
	asID := domain.AppspaceID(9)

	// create one job
	job, err := model.Create(uID, asID, "0.0.1", true)
	if err != nil {
		t.Fatal(err)
	}

	// start the job
	ok, err := model.SetStarted(job.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("should have been ok")
	}

	// then set to finished with an error string
	errStr := nulltypes.NewString("some error", true)
	err = model.SetFinished(job.JobID, errStr)
	if err != nil {
		t.Fatal(err)
	}

	job, err = model.GetJob(job.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if !job.Error.Valid {
		t.Error("expected Error to be valid")
	}
	if job.Error.String != "some error" {
		t.Error("expect correct error string")
	}
}
