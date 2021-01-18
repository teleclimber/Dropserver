import type {Appspace} from './appspaces';
// From go:
// type AppVersion struct {
// 	ID         string    `json:"id" api:"app_versions"`
// 	Name       string    `json:"name"  api:"attr"`
// 	Version    string    `json:"version"  api:"attr"`
// 	APIVersion int       `json:"api" api:"attr"`
// 	Schema     int       `json:"schema" api:"attr"`
// 	Created    time.Time `json:"created_dt" api:"attr"`
// 	App        string    `json:"app" api:"rel,apps,versions"`
// 	Appspaces  []string  `json:"appspaces" api:"rel,appspaces,app_version"`
// }

export type AppVersion = {
	
	version: string,
	name: string,
	api: number,
	schema: number,
	created_dt: Date,

	//app: App,
	appspaces: Appspace[],
}