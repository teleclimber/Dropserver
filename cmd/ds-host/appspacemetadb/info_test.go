package appspacemetadb

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestInfoGetNoSchema(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	asID := domain.AppspaceID(7)

	db := getV0TestDBHandle()
	dbConn := domain.NewMockDbConn(mockCtrl)
	dbConn.EXPECT().GetHandle().Return(db)

	appspaceMetaDB := domain.NewMockAppspaceMetaDB(mockCtrl)
	appspaceMetaDB.EXPECT().GetConn(asID).Return(dbConn, nil)

	infoModel := &InfoModel{
		AppspaceMetaDB: appspaceMetaDB,
		appspaceID:     asID}

	s, err := infoModel.GetSchema()
	if err != nil {
		t.Error(err)
	}
	if s != 0 {
		t.Error("expected schema to be 0")
	}
}

func TestInfoSetSchema(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	asID := domain.AppspaceID(7)

	db := getV0TestDBHandle()
	dbConn := domain.NewMockDbConn(mockCtrl)
	dbConn.EXPECT().GetHandle().AnyTimes().Return(db)

	appspaceMetaDB := domain.NewMockAppspaceMetaDB(mockCtrl)
	appspaceMetaDB.EXPECT().GetConn(asID).AnyTimes().Return(dbConn, nil)

	infoModel := &InfoModel{
		AppspaceMetaDB: appspaceMetaDB,
		appspaceID:     asID}

	err := infoModel.SetSchema(2)
	if err != nil {
		t.Error(err)
	}

	s, err := infoModel.GetSchema()
	if err != nil {
		t.Error(err)
	}
	if s != 2 {
		t.Error("expected schema to be 2")
	}
}

func getV0TestDBHandle() *sqlx.DB {
	// Beware of in-memory DBs: they vanish as soon as the connection closes!
	// We may be able to start a sqlx transaction to avoid problems with that?
	// See: https://github.com/jmoiron/sqlx/issues/164
	handle, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		panic("Failed to open in-memory DB " + err.Error())
	}

	handle.SetMaxOpenConns(1)

	v0h := v0handle{
		handle: handle}

	v0h.migrateUpToV0()

	if v0h.err != nil {
		panic(v0h.err)
	}

	return handle
}
