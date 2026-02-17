package usermodel

import (
	"database/sql"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/migrate"
	"github.com/teleclimber/DropServer/cmd/ds-host/testmocks"
)

func TestPrepareStatements(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	mockEvents := testmocks.NewMockInstanceUserAuthsChangeEvents(mockCtrl)

	userModel := &UserModel{
		DB:                            db,
		InstanceUserAuthsChangeEvents: mockEvents}

	userModel.PrepareStatements()
}

func TestGetFromIDError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	mockEvents := testmocks.NewMockInstanceUserAuthsChangeEvents(mockCtrl)

	userModel := &UserModel{
		DB:                            db,
		InstanceUserAuthsChangeEvents: mockEvents}

	userModel.PrepareStatements()

	// There should be an error, but no panics
	_, err := userModel.GetFromID(10)
	if err == nil || err != sql.ErrNoRows {
		t.Error(err)
	}
}

// test get all.
func TestGetAll(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	mockEvents := testmocks.NewMockInstanceUserAuthsChangeEvents(mockCtrl)
	mockEvents.EXPECT().Send(gomock.Any()).Times(3)

	userModel := &UserModel{
		DB:                            db,
		InstanceUserAuthsChangeEvents: mockEvents}

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
		dbu, err := userModel.CreateWithEmail(u.email, u.pw)
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

func TestCreateWithEmail(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	mockEvents := testmocks.NewMockInstanceUserAuthsChangeEvents(mockCtrl)
	mockEvents.EXPECT().Send(gomock.Any())

	userModel := &UserModel{
		DB:                            db,
		InstanceUserAuthsChangeEvents: mockEvents}

	userModel.PrepareStatements()

	user, err := userModel.CreateWithEmail("bob@foo.com", "secretsauce")
	if err != nil {
		t.Error(err)
	}

	if user.Email != "bob@foo.com" {
		t.Error("input name does not match output name", user)
	}
	if !user.HasPassword {
		t.Error("expected HasPassword to be true")
	}
	if user.TSNetIdentifier != "" {
		t.Errorf("expected empty tsnet identifier, got %s", user.TSNetIdentifier)
	}
}

func TestCreateEmailDupe(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	mockEvents := testmocks.NewMockInstanceUserAuthsChangeEvents(mockCtrl)
	mockEvents.EXPECT().Send(gomock.Any())

	userModel := &UserModel{
		DB:                            db,
		InstanceUserAuthsChangeEvents: mockEvents}

	userModel.PrepareStatements()

	_, err := userModel.CreateWithEmail("bOb@foO.com", "secretsauce")
	if err != nil {
		t.Error(err)
	}

	_, err = userModel.CreateWithEmail("Bob@Foo.com", "moresauce")
	if err == nil {
		t.Error("should have errored")
	} else if err != domain.ErrIdentifierExists {
		t.Error("wrong error", err)
	}
}

func TestCreateWithTSNet(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	mockEvents := testmocks.NewMockInstanceUserAuthsChangeEvents(mockCtrl)
	mockEvents.EXPECT().Send(gomock.Any())

	userModel := &UserModel{
		DB:                            db,
		InstanceUserAuthsChangeEvents: mockEvents}

	userModel.PrepareStatements()

	user, err := userModel.CreateWithTSNet("bob@foo.com", "Bob@Foo")
	if err != nil {
		t.Error(err)
	}

	if user.TSNetIdentifier != "bob@foo.com" {
		t.Error("input name does not match output name", user)
	}
	if user.HasPassword {
		t.Error("expected has password to be false")
	}
}

func TestCreateTSnNetDupe(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	mockEvents := testmocks.NewMockInstanceUserAuthsChangeEvents(mockCtrl)
	mockEvents.EXPECT().Send(gomock.Any())

	userModel := &UserModel{
		DB:                            db,
		InstanceUserAuthsChangeEvents: mockEvents}

	userModel.PrepareStatements()

	_, err := userModel.CreateWithTSNet("bob@foo.com", "Bob@Foo")
	if err != nil {
		t.Error(err)
	}

	_, err = userModel.CreateWithTSNet("bob@foo.com", "Bob@Foo")
	if err == nil {
		t.Error("should have errored")
	} else if err != domain.ErrIdentifierExists {
		t.Error("wrong error", err)
	}
}

func TestGetFromEmail(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	userModel := initBobEmailModel(mockCtrl)
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
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	userModel := initBobEmailModel(mockCtrl)
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
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	userModel := initBobEmailModel(mockCtrl)
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
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	userModel := initBobEmailModel(mockCtrl)
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
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	userModel := initBobEmailModel(mockCtrl)
	defer userModel.DB.Handle.Close()

	_, err := userModel.CreateWithEmail("alice@wonder.land", "whiterabbit")
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
	if err != domain.ErrIdentifierExists {
		t.Error(err)
	}
}

func TestUpdatePassword(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	userModel := initBobEmailModel(mockCtrl)
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

func initBobEmailModel(mockCtrl *gomock.Controller) *UserModel {
	h := migrate.MakeSqliteDummyDB()

	db := &domain.DB{
		Handle: h}

	mockEvents := testmocks.NewMockInstanceUserAuthsChangeEvents(mockCtrl)
	mockEvents.EXPECT().Send(gomock.Any()).AnyTimes()

	userModel := &UserModel{
		DB:                            db,
		InstanceUserAuthsChangeEvents: mockEvents}

	userModel.PrepareStatements()

	_, err := userModel.CreateWithEmail("BoB@Foo.Com", "secretsauce")
	if err != nil {
		panic(err)
	}

	return userModel
}

func TestUpdateDisplayName(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	userModel := initBobEmailModel(mockCtrl)
	defer userModel.DB.Handle.Close()

	bob, err := userModel.GetFromEmail("bob@foo.com")
	if err != nil {
		t.Fatal(err)
	}

	if bob.DisplayName != "" {
		t.Error("expected empty display name for new user")
	}

	err = userModel.UpdateDisplayName(bob.UserID, "Bob Smith")
	if err != nil {
		t.Fatal(err)
	}

	bob2, err := userModel.GetFromID(bob.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if bob2.DisplayName != "Bob Smith" {
		t.Errorf("expected display name 'Bob Smith', got '%s'", bob2.DisplayName)
	}

	// Update again to verify overwrite
	err = userModel.UpdateDisplayName(bob.UserID, "Robert")
	if err != nil {
		t.Fatal(err)
	}

	bob3, err := userModel.GetFromID(bob.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if bob3.DisplayName != "Robert" {
		t.Errorf("expected display name 'Robert', got '%s'", bob3.DisplayName)
	}
}

func TestUpdateDisplayImage(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	userModel := initBobEmailModel(mockCtrl)
	defer userModel.DB.Handle.Close()

	bob, err := userModel.GetFromEmail("bob@foo.com")
	if err != nil {
		t.Fatal(err)
	}

	if bob.DisplayImage != "" {
		t.Error("expected empty display image for new user")
	}

	err = userModel.UpdateDisplayImage(bob.UserID, "avatar.png")
	if err != nil {
		t.Fatal(err)
	}

	bob2, err := userModel.GetFromID(bob.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if bob2.DisplayImage != "avatar.png" {
		t.Errorf("expected display image 'avatar.png', got '%s'", bob2.DisplayImage)
	}

	// Clear image
	err = userModel.UpdateDisplayImage(bob.UserID, "")
	if err != nil {
		t.Fatal(err)
	}

	bob3, err := userModel.GetFromID(bob.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if bob3.DisplayImage != "" {
		t.Errorf("expected empty display image, got '%s'", bob3.DisplayImage)
	}
}

func TestUpdateDeleteTSNet(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	userModel := initBobEmailModel(mockCtrl)

	user, err := userModel.GetFromEmail("bOb@foO.cOm")
	if err != nil {
		t.Error(err)
	}

	err = userModel.UpdateTSNet(user.UserID, "1@headscale.my.site", "Bob@my.site")
	if err != nil {
		t.Error(err)
	}
	user, err = userModel.GetFromID(user.UserID)
	if err != nil {
		t.Error(err)
	}
	if user.TSNetIdentifier != "1@headscale.my.site" || user.TSNetExtraName != "Bob@my.site" {
		t.Error("wrong tsnet values", user)
	}

	err = userModel.DeleteTSNet(user.UserID)
	if err != nil {
		t.Error(err)
	}
	user, err = userModel.GetFromID(user.UserID)
	if err != nil {
		t.Error(err)
	}
	if user.TSNetIdentifier != "" || user.TSNetExtraName != "" {
		t.Error("wrong tsnet values", user)
	}
}

func TestGetFromTSNet(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	mockEvents := testmocks.NewMockInstanceUserAuthsChangeEvents(mockCtrl)
	mockEvents.EXPECT().Send(gomock.Any())

	userModel := &UserModel{
		DB:                            db,
		InstanceUserAuthsChangeEvents: mockEvents}

	userModel.PrepareStatements()

	_, err := userModel.CreateWithTSNet("1@headscale.my.site", "Bob@Foo")
	if err != nil {
		t.Error(err)
	}

	user, err := userModel.GetFromTSNet("1@headscale.my.site")
	if err != nil {
		t.Error(err)
	}
	if user.TSNetExtraName != "Bob@Foo" {
		t.Error("got wrong data")
	}
}

// /////// admin
func TestIsAdmin(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	h := migrate.MakeSqliteDummyDB()

	db := &domain.DB{
		Handle: h}

	mockEvents := testmocks.NewMockInstanceUserAuthsChangeEvents(mockCtrl)

	userModel := &UserModel{
		DB:                            db,
		InstanceUserAuthsChangeEvents: mockEvents}

	userModel.PrepareStatements()

	is := userModel.IsAdmin(domain.UserID(999))
	if is {
		t.Error("Should not be an admin")
	}
}

func TestManageAdmin(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	h := migrate.MakeSqliteDummyDB()

	db := &domain.DB{
		Handle: h}

	mockEvents := testmocks.NewMockInstanceUserAuthsChangeEvents(mockCtrl)

	userModel := &UserModel{
		DB:                            db,
		InstanceUserAuthsChangeEvents: mockEvents}

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
