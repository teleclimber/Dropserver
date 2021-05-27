package cookiemodel

import (
	"database/sql"
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

	_, err := cookieModel.Get("foo")
	if err != sql.ErrNoRows {
		t.Error(err)
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
	dom := "as1.ds.dev"

	c := domain.Cookie{
		UserID:      userID,
		UserAccount: true,
		Expires:     expires,
		DomainName:  dom}

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
	if cookie.DomainName != dom {
		t.Error("mismatched cookie domain")
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
	err = cookieModel.UpdateExpires(cookieID, expires)
	if err != nil {
		t.Error(err)
	}

	c2, err := cookieModel.Get(cookieID)
	if err != nil {
		t.Error(err)
	}

	if c2.Expires != expires {
		t.Error("mismatched expires", c2.Expires, expires)
	}
}

func TestDelete(t *testing.T) {
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

	_, err = cookieModel.Get(cookieID)
	if err != nil {
		t.Error(err)
	}

	err = cookieModel.Delete(cookieID)
	if err != nil {
		t.Error(err)
	}

	_, err = cookieModel.Get(cookieID)
	if err != sql.ErrNoRows {
		t.Error("expecte err no rows")
	}
}

// more things to test:
// - appspace_id
// - bad input of appspace_id + user_account?
