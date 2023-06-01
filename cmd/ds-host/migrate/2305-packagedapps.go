package migrate

import (
	"encoding/json"
	"errors"
)

type appVerMan2305 struct {
	AppID   int    `db:"app_id" json:"-"`
	Name    string `db:"name" json:"name"`
	Version string `db:"version" json:"version"`
	Schema  int    `db:"schema" json:"schema"`
}

func packagedAppsUp(args *stepArgs) error {
	args.dbExec(`ALTER TABLE app_versions ADD COLUMN manifest JSON`)

	appVers := []appVerMan2305{}
	err := args.db.Handle.Select(&appVers, `SELECT apps.app_id, name, version, schema FROM app_versions LEFT JOIN apps USING(app_id)`)
	if err != nil {
		return err
	}
	for _, appVer := range appVers {
		manifestBytes, err := json.Marshal(appVer)
		if err != nil {
			return err
		}
		r := args.dbExec(`UPDATE app_versions SET manifest = json(?) WHERE app_id = ? AND version = ? `,
			manifestBytes, appVer.AppID, appVer.Version)
		n, err := r.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			return errors.New("no rows affected when updating app_versions")
		}
	}

	args.dbExec(`ALTER TABLE app_versions DROP COLUMN schema`)
	args.dbExec(`ALTER TABLE app_versions DROP COLUMN api`)
	args.dbExec(`ALTER TABLE apps DROP COLUMN name`)

	return args.dbErr
}

func packagedAppsDown(args *stepArgs) error {
	// - add columns to app_versions and apps
	// - for each row of app_versions, write appropriate values in columns
	// - figure out an app version that is nominal, and write app name in apps col.
	// - delete columns.
	h := args.db.Handle

	args.dbExec(`ALTER TABLE apps ADD COLUMN name TEXT`)
	args.dbExec(`ALTER TABLE app_versions ADD COLUMN schema INTEGER`)
	args.dbExec(`ALTER TABLE app_versions ADD COLUMN api INTEGER`)

	// for each app, get all versions + names, sort versions, take latest, and use name to fill name col in app table
	appIDs := []int{}
	err := h.Select(&appIDs, `SELECT app_id FROM apps`)
	if err != nil {
		return err
	}
	for _, appID := range appIDs {
		// could simplify by
		var name string
		err = h.Get(&name, `SELECT json_extract(manifest, '$.name') AS name 
			FROM app_versions WHERE app_id = ? ORDER BY created DESC LIMIT 1`, appID)
		if err != nil {
			return err
		}
		_, err := h.Exec(`UPDATE apps SET name = ? WHERE app_id = ?`, name, appID)
		if err != nil {
			return err
		}
	}

	appVers := []appVerMan2305{}
	err = h.Select(&appVers, `SELECT apps.app_id, version, 
		json_extract(manifest, '$.schema') AS schema
		FROM app_versions LEFT JOIN apps USING(app_id)`)
	if err != nil {
		return err
	}
	for _, appVer := range appVers {
		_, err = h.Exec(`UPDATE app_versions SET schema = ?, api = 0 WHERE app_id = ? AND version = ?`,
			appVer.Schema, appVer.AppID, appVer.Version)
		if err != nil {
			return err
		}
	}

	args.dbExec(`ALTER TABLE app_versions DROP COLUMN manifest`)

	return args.dbErr
}
