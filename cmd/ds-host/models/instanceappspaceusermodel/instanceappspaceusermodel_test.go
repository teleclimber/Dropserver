package instanceappspaceusermodel

import (
	"strings"
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/migrate"
)

func TestPrepareStatements(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	db := &domain.DB{Handle: h}

	model := &InstanceAppspaceModel{DB: db}
	model.PrepareStatements()
}

func TestCreateGet(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	model := &InstanceAppspaceModel{DB: &domain.DB{Handle: h}}
	model.PrepareStatements()

	userID := domain.UserID(7)
	appID := domain.AppspaceID(11)
	proxyID := domain.ProxyID("abc123")

	_, err := model.Get(userID, appID)
	if err == nil {
		t.Error("expected error for missing mapping")
	}
	if err != domain.ErrNoRowsInResultSet {
		t.Error(err)
	}

	err = model.Create(userID, appID, proxyID)
	if err != nil {
		t.Error(err)
	}

	_, err = model.Get(userID, appID)
	if err != nil {
		t.Error(err)
	}
}

func TestGetForUserAndAppspace(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	model := &InstanceAppspaceModel{DB: &domain.DB{Handle: h}}
	model.PrepareStatements()

	userID := domain.UserID(7)

	err := model.Create(userID, domain.AppspaceID(1), domain.ProxyID("uvw789"))
	if err != nil {
		t.Error(err)
	}
	err = model.Create(userID, domain.AppspaceID(2), domain.ProxyID("abc123"))
	if err != nil {
		t.Error(err)
	}
	err = model.Create(domain.UserID(13), domain.AppspaceID(1), domain.ProxyID("ijk456"))
	if err != nil {
		t.Error(err)
	}

	mappingsForUser, err := model.GetForUser(userID)
	if err != nil {
		t.Error(err)
	}
	if len(mappingsForUser) != 2 {
		t.Error("expected two mappings for user")
	}

	mappingsForApp, err := model.GetForAppspace(domain.AppspaceID(1))
	if err != nil {
		t.Error(err)
	}
	if len(mappingsForApp) != 2 {
		t.Error("expected two mappings for appspace")
	}
}

func TestDuplicateMapping(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	model := &InstanceAppspaceModel{DB: &domain.DB{Handle: h}}
	model.PrepareStatements()

	userID := domain.UserID(7)
	appID := domain.AppspaceID(1)

	err := model.Create(userID, appID, domain.ProxyID("abc123"))
	if err != nil {
		t.Error(err)
	}
	err = model.Create(userID, appID, domain.ProxyID("def567"))
	if err == nil {
		t.Error("expected error on duplicate mapping")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Error("expected error string to contain: already exists")
	}
}

func TestDelete(t *testing.T) {
	h := migrate.MakeSqliteDummyDB()
	defer h.Close()

	model := &InstanceAppspaceModel{DB: &domain.DB{Handle: h}}
	model.PrepareStatements()

	userID := domain.UserID(7)
	appID := domain.AppspaceID(11)

	err := model.Create(userID, appID, domain.ProxyID("abc123"))
	if err != nil {
		t.Error(err)
	}

	_, err = model.Get(userID, appID)
	if err != nil {
		t.Error(err)
	}

	err = model.Delete(userID, appID)
	if err != nil {
		t.Error(err)
	}

	_, err = model.Get(userID, appID)
	if err == nil {
		t.Error("expected error getting deleted mapping")
	}
	if err != domain.ErrNoRowsInResultSet {
		t.Error("expected sql.ErrNoRows after delete")
	}
}

// Test for SetUsersForAppspace: replace existing users for an appspace, ensure other appspaces unaffected, and clearing works.
// func TestSetUsersForAppspace(t *testing.T) {
// 	h := migrate.MakeSqliteDummyDB()
// 	defer h.Close()

// 	model := &InstanceAppspaceModel{DB: &domain.DB{Handle: h}}
// 	model.PrepareStatements()

// 	// seed initial mappings
// 	if err := model.Create(domain.UserID(1), domain.AppspaceID(5)); err != nil {
// 		t.Error(err)
// 	}
// 	if err := model.Create(domain.UserID(2), domain.AppspaceID(5)); err != nil {
// 		t.Error(err)
// 	}
// 	// a mapping for a different appspace should remain untouched
// 	if err := model.Create(domain.UserID(6), domain.AppspaceID(7)); err != nil {
// 		t.Error(err)
// 	}

// 	// replace users for appspace 5 with [3,4]
// 	if err := model.SetUsersForAppspace(domain.AppspaceID(5), []domain.UserID{3, 4}); err != nil {
// 		t.Error(err)
// 	}

// 	mappings, err := model.GetForAppspace(domain.AppspaceID(5))
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	if len(mappings) != 2 {
// 		t.Errorf("expected 2 mappings for appspace 5, got %d", len(mappings))
// 	}
// 	found3, found4 := false, false
// 	for _, m := range mappings {
// 		if m.UserID == domain.UserID(3) {
// 			found3 = true
// 		}
// 		if m.UserID == domain.UserID(4) {
// 			found4 = true
// 		}
// 	}
// 	if !found3 || !found4 {
// 		t.Error("expected mappings for users 3 and 4 after SetUsersForAppspace")
// 	}

// 	// ensure other appspace mapping remains
// 	other, err := model.GetForAppspace(domain.AppspaceID(7))
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	if len(other) != 1 {
// 		t.Error("expected mapping for other appspace to remain")
// 	}

// 	// clear users for appspace 5
// 	if err := model.SetUsersForAppspace(domain.AppspaceID(5), []domain.UserID{}); err != nil {
// 		t.Error(err)
// 	}
// 	mappings, err = model.GetForAppspace(domain.AppspaceID(5))
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	if len(mappings) != 0 {
// 		t.Error("expected zero mappings for appspace 5 after clearing")
// 	}
// }
