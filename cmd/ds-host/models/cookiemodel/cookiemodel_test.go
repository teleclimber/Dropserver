package cookiemodel

import (
	"testing"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/migrate"
)

func TestPrepareStatements(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	cookieModel := &CookieModel{
		DB: db}

	cookieModel.PrepareStatements()
}

func TestGetNoRows(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	cookieModel := &CookieModel{
		DB: db}

	cookieModel.PrepareStatements()

	// There should be an error, but no panics
	c, dsErr := cookieModel.Get("foo")
	if dsErr != nil {
		t.Error(dsErr)
	}
	if c != nil {
		t.Error("cookie should have been nil")
	}
}

func TestCRUD(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	cookieModel := &CookieModel{
		DB: db}

	cookieModel.PrepareStatements()

	userID := domain.UserID(100)

	expires := time.Now().Add(120 * time.Second)

	c := domain.Cookie{
		UserID:      userID,
		UserAccount: true,
		Expires:     expires}

	cookieID, err := cookieModel.Create(c)
	if err != nil {
		t.Error(err)
	}
	if cookieID == "" {
		t.Error("cookieID should not be empty")
	}

	cookie, err := cookieModel.Get(cookieID)
	if err != nil {
		t.Error(err)
	}
	if cookie.UserID != userID {
		t.Error("mismatched returned value: user_id", cookie)
	}
	if cookie.UserAccount != true {
		t.Error("mismatched data: user_account", cookie)
	}

	// can't compare expires times directly because something in the way timezones are represented changes.
	dt := cookie.Expires.Sub(c.Expires)
	if dt.Seconds() > 0 {
		t.Error("mismatched data: expires", cookie, expires, dt)
	}

}

func TestUpdateExpires(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{
		Handle: h}

	cookieModel := &CookieModel{
		DB: db}

	cookieModel.PrepareStatements()

	c := domain.Cookie{
		UserID:      domain.UserID(100),
		UserAccount: true,
		Expires:     time.Now()}

	cookieID, err := cookieModel.Create(c)
	if err != nil {
		t.Error(err)
	}

	expires := time.Date(2019, time.Month(5), 29, 6, 2, 0, 0, time.UTC)
	dsErr := cookieModel.UpdateExpires(cookieID, expires)
	if dsErr != nil {
		t.Error(dsErr)
	}

	c2, dsErr := cookieModel.Get(cookieID)
	if dsErr != nil {
		t.Error(dsErr)
	}

	if c2.Expires != expires {
		t.Error("mismatched expires", c2.Expires, expires)
	}
}

// more things to test:
// - appspace_id
// - bad input of appspace_id + user_account?

