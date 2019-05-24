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


