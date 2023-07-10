package migrate

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

var preSteps []MigrationStep = []MigrationStep{{
	name: "1905-fresh-install",
	up:   freshInstallUp,
	down: freshInstallDown,
}, {
	name: "2203-sandboxusage",
	up:   sandboxUsageUp,
	down: sandboxUsageDown,
},
}

type TestAppVer2305 struct {
	AppID      int    `db:"app_id"`
	Version    string `db:"version"`
	Manifest   string `db:"manifest"`
	ManVersion string `db:"man_version"`
	Schema     int    `db:"schema"`
	Entrypoint string `db:"entrypoint"`
}

func TestPackagedAppsUp(t *testing.T) {
	handle, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		panic("Failed to open in-memory DB " + err.Error())
	}

	handle.SetMaxOpenConns(1)

	db := &domain.DB{
		Handle: handle}

	args := &stepArgs{
		db: db}

	for _, s := range preSteps {
		err := s.up(args)
		if err != nil {
			t.Fatal("Step returned an error", s, err)
		}
	}
	if args.dbErr != nil {
		t.Fatal(args.dbErr)
	}

	av, err := addAppVersion2305(handle)
	if err != nil {
		t.Fatal(err)
	}

	err = packagedAppsUp(args)
	if err != nil {
		t.Fatal(err)
	}

	var out TestAppVer2305
	db.Handle.QueryRowx(`SELECT app_id, version, manifest, 
		json_extract(manifest, '$.version') AS man_version,
		json_extract(manifest, '$.schema') AS schema,
		json_extract(manifest, '$.entrypoint') AS entrypoint
		FROM app_versions WHERE app_id = ? AND version = ?`,
		av.AppID, av.Version).StructScan(&out)

	t.Log(out)

	if out.Version != av.Version {
		t.Errorf("version: expected %s, got %s", av.Version, out.Version)
	}
	if out.Schema != av.Schema {
		t.Errorf("schema: expected %v, got %v", av.Schema, out.Schema)
	}
	if out.Entrypoint != "app.ts" {
		t.Error("expected entrypoint to be app.ts")
	}
}

func TestPackagedAppsDown(t *testing.T) {
	handle, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		panic("Failed to open in-memory DB " + err.Error())
	}

	handle.SetMaxOpenConns(1)

	db := &domain.DB{
		Handle: handle}

	args := &stepArgs{
		db: db}

	for _, s := range preSteps {
		err := s.up(args)
		if err != nil {
			t.Error("Step returned an error", s, err)
		}
	}

	av, err := addAppVersion2305(handle)
	if err != nil {
		t.Fatal(err)
	}

	err = packagedAppsUp(args)
	if err != nil {
		t.Fatal(err)
	}

	err = packagedAppsDown(args)
	if err != nil {
		t.Fatal(err)
	}

	var name string
	err = handle.Get(&name, `SELECT name FROM apps WHERE app_id = ?`, av.AppID)
	if err != nil {
		t.Error(err)
	}
	if name != av.Name {
		t.Errorf("Expected name %s, got %s", av.Name, name)
	}

	var schema int
	err = handle.Get(&schema, `SELECT schema FROM app_versions WHERE app_id = ? AND version = ?`, av.AppID, av.Version)
	if err != nil {
		t.Error(err)
	}
	if schema != av.Schema {
		t.Errorf("Expected schema %v, got %v", av.Schema, schema)
	}
}

func TestUpDownUpDown2305(t *testing.T) {
	handle, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		panic("Failed to open in-memory DB " + err.Error())
	}

	handle.SetMaxOpenConns(1)

	db := &domain.DB{
		Handle: handle}

	args := &stepArgs{
		db: db}

	for _, s := range preSteps {
		err := s.up(args)
		if err != nil {
			t.Error("Step returned an error", s, err)
		}
	}

	_, err = addAppVersion2305(handle)
	if err != nil {
		t.Fatal(err)
	}

	err = packagedAppsUp(args)
	if err != nil {
		t.Fatal(err)
	}
	err = packagedAppsDown(args)
	if err != nil {
		t.Fatal(err)
	}
	err = packagedAppsUp(args)
	if err != nil {
		t.Fatal(err)
	}
	err = packagedAppsDown(args)
	if err != nil {
		t.Fatal(err)
	}
}

type AV struct {
	AppID   int
	Version string
	Name    string
	Schema  int
}

func addAppVersion2305(h *sqlx.DB) (AV, error) {
	ret := AV{
		Version: "0.5.7",
		Name:    "Test App 123",
		Schema:  7,
	}
	r, err := h.Exec(`INSERT INTO apps ("owner_id", "name", "created") VALUES (?, ?, datetime("now"))`, 1, ret.Name)
	if err != nil {
		return ret, err
	}
	appID, err := r.LastInsertId()
	if err != nil {
		return ret, err
	}
	ret.AppID = int(appID)

	_, err = h.Exec(`INSERT INTO app_versions
		("app_id", "version", "schema", "api", "location_key", created) VALUES (?, ?, ?, ?, ?, datetime("now"))`,
		appID, ret.Version, ret.Schema, 0, "abc123")
	if err != nil {
		return ret, err
	}
	return ret, nil
}
