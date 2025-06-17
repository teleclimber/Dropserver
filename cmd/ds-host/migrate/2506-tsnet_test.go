package migrate

import (
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/nulltypes"
)

type userTSNet struct {
	UserID     uint32               `db:"user_id"`
	Email      nulltypes.NullString `db:"email"`
	Password   nulltypes.NullString `db:"password"`
	TSNetID    nulltypes.NullString `db:"tsnet_identifier"`
	TSNetExtra nulltypes.NullString `db:"tsnet_extra_name"`
}

func TestTSNetIntegrationUp(t *testing.T) {
	handle, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		panic("Failed to open in-memory DB " + err.Error())
	}

	handle.SetMaxOpenConns(1)

	db := &domain.DB{
		Handle: handle}

	args := &stepArgs{
		db: db}

	runMigrationUpTo(t, args, "2311-appurls")

	_, err = handle.Exec(`INSERT INTO users ("email", "password") VALUES (?, ?)`, "useremail", "userpass")
	if err != nil {
		t.Fatal(err)
	}

	err = tsnetIntegrationUp(args)
	if err != nil {
		t.Error(err)
	}

	var u userTSNet

	// first check that there is a user w that email?
	err = handle.Get(&u, `SELECT * FROM users`)
	if err != nil {
		t.Error(err)
	}
	if !u.Email.Valid || u.Email.String != "useremail" || !u.Password.Valid || u.Password.String != "userpass" {
		t.Errorf("Expected user to be present %v", u)
	}

	// add another user
	_, err = handle.Exec(`INSERT INTO users ("email", "password") VALUES (?, ?)`, "usertwo", "userpass2")
	if err != nil {
		t.Error(err)
	}

	// add another user with conflicting email
	_, err = handle.Exec(`INSERT INTO users ("email", "password") VALUES (?, ?)`, "useremail", "userpass3")
	if err == nil {
		t.Error("expected adding a duplicate email to cause an error")
	}

	// add user with only tsnet data.
	_, err = handle.Exec(`INSERT INTO users ("tsnet_identifier", "tsnet_extra_name") VALUES (?, ?)`, "tsnetuser", "tsnetextra")
	if err != nil {
		t.Error(err)
	}

	// adding same user again causes error
	_, err = handle.Exec(`INSERT INTO users ("tsnet_identifier", "tsnet_extra_name") VALUES (?, ?)`, "tsnetuser", "tsnetextra")
	if err == nil {
		t.Error("expected error on dupe tsnet user")
	}
}

func TestTSNetIntegrationDownNullEmailPass(t *testing.T) {
	handle, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		panic("Failed to open in-memory DB " + err.Error())
	}
	handle.SetMaxOpenConns(1)
	db := &domain.DB{
		Handle: handle}
	args := &stepArgs{
		db: db}
	runMigrationUpTo(t, args, "2506-tsnet")

	// add user with only tsnet data leaving username and password NULL, which should cause error in down migration.
	_, err = handle.Exec(`INSERT INTO users ("tsnet_identifier", "tsnet_extra_name") VALUES (?, ?)`, "tsnetuser", "tsnetextra")
	if err != nil {
		t.Error(err)
	}

	err = tsnetIntegrationDown(args)
	if err == nil {
		t.Fatal("expected error due to null email/password")
	}
	if !strings.Contains(err.Error(), "to downgrade from 2506-tsnet all users must have an email and password") {
		t.Error(err)
	}
}

func TestTSNetIntegrationDown(t *testing.T) {
	handle, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		panic("Failed to open in-memory DB " + err.Error())
	}
	handle.SetMaxOpenConns(1)
	db := &domain.DB{
		Handle: handle}
	args := &stepArgs{
		db: db}
	runMigrationUpTo(t, args, "2506-tsnet")

	_, err = handle.Exec(`INSERT INTO users ("email", "password") VALUES (?, ?)`, "useremail", "userpass")
	if err != nil {
		t.Fatal(err)
	}

	err = tsnetIntegrationDown(args)
	if err != nil {
		t.Error(err)
	}
}
