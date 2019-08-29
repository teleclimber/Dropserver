package usermodel

import (
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/migrate"
	"github.com/teleclimber/DropServer/internal/dserror"
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
	_, dsErr := userModel.GetFromID(10)
	if dsErr == nil || dsErr.Code() != dserror.NoRowsInResultSet {
		t.Error(dsErr)
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

	users := []struct{
		email string
		pw string
		admin bool
	} {
		{"abc@def", "gobblegobble", false},
		{"admin@bar", "bibblebibble", true},
		{"baz@bar", "fifflefiffle", false},
	}

	for _, u := range users {
		dbu, dsErr := userModel.Create(u.email, u.pw)
		if dsErr != nil {
			t.Fatal(dsErr)
		}

		if u.admin {
			dsErr = userModel.MakeAdmin(dbu.UserID)
			if dsErr != nil {
				t.Fatal(dsErr)
			}
		}
	}

	all, dsErr := userModel.GetAll()
	if dsErr != nil {
		t.Fatal(dsErr)
	}
	if len(all) != 3 {
		t.Error("should have 3 users")
	}

	admins, dsErr := userModel.GetAllAdmins()
	if dsErr != nil {
		t.Error(dsErr)
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

	user, dsErr := userModel.Create("bob@foo.com", "secretsauce")
	if dsErr != nil {
		t.Error(dsErr)
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

	_, dsErr := userModel.Create("bOb@foO.com", "secretsauce")
	if dsErr != nil {
		t.Error(dsErr)
	}

	_, dsErr = userModel.Create("Bob@Foo.com", "moresauce")
	if dsErr == nil {
		t.Error("should have errored")
	} else if dsErr.Code() != dserror.EmailExists {
		t.Error("wrong error", dsErr)
	}
}

func TestGetFromEmail(t *testing.T) {
	userModel := initBobModel()
	defer userModel.DB.Handle.Close()

	user, dsErr := userModel.GetFromEmail("bOb@foO.cOm")
	if dsErr != nil {
		t.Error(dsErr)
	}

	if user.Email != "bob@foo.com" {
		t.Error("input name does not match output name", user)
	}
}

func TestGetFromEmailNoRows(t *testing.T) {
	userModel := initBobModel()
	defer userModel.DB.Handle.Close()

	// There should be an error, but no panics
	_, dsErr := userModel.GetFromEmail("alice@bar.cOm")
	if dsErr == nil {
		t.Error("should have errored")
	} else if dsErr.Code() != dserror.NoRowsInResultSet {
		t.Error("wrong error", dsErr)
	}
}

func TestPassword(t *testing.T) {
	userModel := initBobModel()
	defer userModel.DB.Handle.Close()

	cases := []struct {
		email     string
		pw        string
		found     bool
		errorCode domain.ErrorCode
	}{
		{"bOb@foO.cOm", "secretsauce", true, dserror.InternalError}, // no error actually, but can't nil
		{"bOb@foO.cOm", "secretSauce", false, dserror.AuthenticationIncorrect},
		{"bOb@bar.cOm", "secretsauce", false, dserror.NoRowsInResultSet},
	}

	for _, c := range cases {
		user, dsErr := userModel.GetFromEmailPassword(c.email, c.pw)
		if c.found {
			if dsErr != nil {
				t.Error(dsErr)
			}
			if user.Email != "bob@foo.com" {
				t.Error("unexpected data mismatch", user)
			}
		} else {
			if dsErr == nil {
				t.Error("should have errored with no rows", c, user)
			} else if dsErr.Code() != c.errorCode {
				t.Error("wrong error", dsErr, c)
			}
		}
	}

}

func TestUpdatePassword(t *testing.T) {
	userModel := initBobModel()
	defer userModel.DB.Handle.Close()

	bob, dsErr := userModel.GetFromEmail("bob@foo.com")
	if dsErr != nil {
		t.Error(dsErr)
	}

	dsErr = userModel.UpdatePassword(bob.UserID, "secretspice")
	if dsErr != nil {
		t.Error(dsErr)
	}

	bob2, dsErr := userModel.GetFromEmailPassword("bob@foo.com", "secretspice")
	if dsErr != nil {
		t.Error(dsErr)
	}

	if bob2 == nil {
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

	_, dsErr := userModel.Create("BoB@Foo.Com", "secretsauce")
	if dsErr != nil {
		panic(dsErr)
	}

	return userModel
}

///////// admin
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

	dsErr := userModel.MakeAdmin(uID)
	if dsErr != nil {
		t.Error(dsErr)
	}

	is := userModel.IsAdmin(uID)
	if !is {
		t.Error("user should be admin")
	}

	// then do it again to see if it handles dupes elegantly
	dsErr = userModel.MakeAdmin(uID)
	if dsErr != nil {
		t.Error(dsErr)
	}

	// then delete the admin
	dsErr = userModel.DeleteAdmin(uID)
	if dsErr != nil {
		t.Error(dsErr)
	}

	is = userModel.IsAdmin(uID)
	if is {
		t.Error("user should NOT be admin anymore")
	}
}


