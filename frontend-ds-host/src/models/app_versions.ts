import {Resource, DocumentBuilder} from '../utils/jsonapi_utils';

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

export class AppVersion {
	loaded = false;
	app_id = 0;
	version ='';
	name= '';
	api = 0;
	schema = 0;
	created_dt = new Date;

	//app: App,
	appspaces: Appspace[] = [];

	// async fetch(id: number) {
	// 	const resp_data = await get('/appspaces/'+id+'?include=app_version');
	// 	this.setFromResource(new Resource(resp_data.data));
	// }
	setFromResource(r :Resource) {
		// app version id is composite of app id and version
		[this.app_id, this.version] = parseId(r.idString());
		this.name = r.attrString('name');
		this.api = r.attrNumber('api');
		this.schema = r.attrNumber('schema');
		this.created_dt = r.attrDate('created_dt');
		this.loaded = true;
	}
	
}

// export function ReactiveAppVersion

// parseId takes a app version composite id 3-0.1.2 and splits it into app_id and version string
export function parseId(id_str: string) :[number, string] {
	const pieces = id_str.split('-', 2);
	const app_id = Number(pieces[0]);
	return [app_id, pieces[1]];
}