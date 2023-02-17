package usermodel

import (
	"database/sql"
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/migrate"
)

func TestPrepareStatements(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	userModel := &UserModel{
		DB: db}

	userModel.PrepareStatements()
}

func TestGetFromIDError(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	userModel := &UserModel{
		DB: db}

	userModel.PrepareStatements()

	// There should be an error, but no panics
	_, err := userModel.GetFromID(10)
	if err == nil || err != sql.ErrNoRows {
		t.Error(err)
	}
}

// test get all.
func TestGetAll(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	userModel := &UserModel{
		DB: db}

	userModel.PrepareStatements()

	users := []struct {
		email string
		pw    string
		admin bool
	}{
		{"abc@def", "gobblegobble", false},
		{"admin@bar", "bibblebibble", true},
		{"baz@bar", "fifflefiffle", false},
	}

	for _, u := range users {
		dbu, err := userModel.Create(u.email, u.pw)
		if err != nil {
			t.Fatal(err)
		}

		if u.admin {
			err = userModel.MakeAdmin(dbu.UserID)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	all, err := userModel.GetAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Error("should have 3 users")
	}

	admins, err := userModel.GetAllAdmins()
	if err != nil {
		t.Error(err)
	}

	if len(admins) != 1 {
		t.Error("should only be one admin")
	}

	adminID := admins[0]

	for _, a := range all {
		if a.Email == "admin@bar" && a.UserID != adminID {
			t.Error("expected adminID to conincide with admin@bar")
		}
	}
}

func TestCreate(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	userModel := &UserModel{
		DB: db}

	userModel.PrepareStatements()

	user, err := userModel.Create("bob@foo.com", "secretsauce")
	if err != nil {
		t.Error(err)
	}

	if user.Email != "bob@foo.com" {
		t.Error("input name does not match output name", user)
	}
}

func TestCreateDupe(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	userModel := &UserModel{
		DB: db}

	userModel.PrepareStatements()

	_, err := userModel.Create("bOb@foO.com", "secretsauce")
	if err != nil {
		t.Error(err)
	}

	_, err = userModel.Create("Bob@Foo.com", "moresauce")
	if err == nil {
		t.Error("should have errored")
	} else if err != domain.ErrEmailExists {
		t.Error("wrong error", err)
	}
}

func TestGetFromEmail(t *testing.T) {
	userModel := initBobModel()
	defer userModel.DB.Handle.Close()

	user, err := userModel.GetFromEmail("bOb@foO.cOm")
	if err != nil {
		t.Error(err)
	}

	if user.Email != "bob@foo.com" {
		t.Error("input name does not match output name", user)
	}
}

func TestGetFromEmailNoRows(t *testing.T) {
	userModel := initBobModel()
	defer userModel.DB.Handle.Close()

	// There should be an error, but no panics
	_, err := userModel.GetFromEmail("alice@bar.cOm")
	if err == nil {
		t.Error("should have errored")
	} else if err != sql.ErrNoRows {
		t.Error("wrong error", err)
	}
}

func TestPassword(t *testing.T) {
	userModel := initBobModel()
	defer userModel.DB.Handle.Close()

	cases := []struct {
		email string
		pw    string
		found bool
		err   error
	}{
		{"bOb@foO.cOm", "secretsauce", true, nil},
		{"bOb@foO.cOm", "secretSauce", false, domain.ErrBadAuth},
		{"bOb@bar.cOm", "secretsauce", false, sql.ErrNoRows},
	}

	for _, c := range cases {
		user, err := userModel.GetFromEmailPassword(c.email, c.pw)
		if c.found {
			if err != nil {
				t.Error(err)
			}
			if user.Email != "bob@foo.com" {
				t.Error("unexpected data mismatch", user)
			}
		} else {
			if err == nil {
				t.Error("should have errored with no rows", c, user)
			} else if err != c.err {
				t.Error("wrong error", err, c)
			}
		}
	}

}

func TestUpdateEmail(t *testing.T) {
	userModel := initBobModel()
	defer userModel.DB.Handle.Close()

	bob, err := userModel.GetFromEmail("bob@foo.com")
	if err != nil {
		t.Error(err)
	}

	err = userModel.UpdateEmail(bob.UserID, "bob@bar.com")
	if err != nil {
		t.Error(err)
	}

	bob2, err := userModel.GetFromEmail("bob@bar.com")
	if err != nil {
		t.Error(err)
	}

	if bob.UserID != bob2.UserID {
		t.Error("got wrong user id")
	}
}

func TestUpdateEmailDupe(t *testing.T) {
	userModel := initBobModel()
	defer userModel.DB.Handle.Close()

	_, err := userModel.Create("alice@wonder.land", "whiterabbit")
	if err != nil {
		t.Error(err)
	}

	bob, err := userModel.GetFromEmail("bob@foo.com")
	if err != nil {
		t.Error(err)
	}

	err = userModel.UpdateEmail(bob.UserID, "alice@wonder.land")
	if err == nil {
		t.Error("expected error because of dupe email")
	}
	if err != domain.ErrEmailExists {
		t.Error(err)
	}
}

func TestUpdatePassword(t *testing.T) {
	userModel := initBobModel()
	defer userModel.DB.Handle.Close()

	bob, err := userModel.GetFromEmail("bob@foo.com")
	if err != nil {
		t.Error(err)
	}

	err = userModel.UpdatePassword(bob.UserID, "secretspice")
	if err != nil {
		t.Error(err)
	}

	bob2, err := userModel.GetFromEmailPassword("bob@foo.com", "secretspice")
	if err != nil {
		t.Error(err)
	}

	if bob2.UserID != bob.UserID {
		t.Error("Should have returned bob again")
	}
}

func initBobModel() *UserModel {
	h := migrate.MakeSqliteDummyDB()

	db := &domain.DB{
		Handle: h}

	userModel := &UserModel{
		DB: db}

	userModel.PrepareStatements()

	_, err := userModel.Create("BoB@Foo.Com", "secretsauce")
	if err != nil {
		panic(err)
	}

	return userModel
}

// /////// admin
func TestIsAdmin(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()

	db := &domain.DB{
		Handle: h}

	userModel := &UserModel{
		DB: db}

	userModel.PrepareStatements()

	is := userModel.IsAdmin(domain.UserID(999))
	if is {
		t.Error("Should not be an admin")
	}
}

func TestManageAdmin(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()

	db := &domain.DB{
		Handle: h}

	userModel := &UserModel{
		DB: db}

	userModel.PrepareStatements()

	uID := domain.UserID(77)

	err := userModel.MakeAdmin(uID)
	if err != nil {
		t.Error(err)
	}

	is := userModel.IsAdmin(uID)
	if !is {
		t.Error("user should be admin")
	}

	// then do it again to see if it handles dupes elegantly
	err = userModel.MakeAdmin(uID)
	if err != nil {
		t.Error(err)
	}

	// then delete the admin
	err = userModel.DeleteAdmin(uID)
	if err != nil {
		t.Error(err)
	}

	is = userModel.IsAdmin(uID)
	if is {
		t.Error("user should NOT be admin anymore")
	}
}
