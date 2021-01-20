import { ref, reactive } from 'vue';
import type {Ref} from 'vue';

import {get, patch} from '../controllers/userapi';
import {Document, Resource, DocumentBuilder} from '../utils/jsonapi_utils';
import {parseId, AppVersion} from './app_versions';

// these are owner's appspaces, not remotes.

// hierarchical data:
// - appspaces (listing)
// - appspace (node of above list)
// - appspace status (derived live data on readiness of appspace)
// - appspace upgrade available 

// relations:
// - appspace -> appversion
// - appspace =>* contacts

// For relations (contacts here), we need:
// - related contact ids
// - data for these ids.
//   ..here should be a preview

// from Go:
// type Appspace struct {
// 	ID         string    `json:"id" api:"appspaces"`
// 	Subdomain  string    `json:"subdomain" api:"attr"`
// 	Created    time.Time `json:"created_dt" api:"attr"`
// 	Paused     bool      `json:"paused" api:"attr"`
// 	AppVersion string    `json:"app_version" api:"rel,app_versions,appspaces"`
// 	//Owner string `json:"owner" api:"rel,..`
// }

export class Appspace {
	loaded = false;
	id = 0;
	subdomain = "";
	created_dt = new Date();
	paused = false;

	app_id = 0;
	version = '';

	app_version = new AppVersion;// should this just be an id, and we can ftch it from appversions collection when needed?

	async fetch(id: number) {
		const resp_data = await get('/appspaces/'+id+'?include=app_version');
		const doc = new Document(resp_data);
		// check for errors in doc.
		const res = doc.getResource();
		this.setFromResource(res);

		const app_version_rel = res.relOne('app_version');
		const inc_res = doc.getIncluded('app_versions', app_version_rel.idString());
		this.app_version.setFromResource(inc_res);
	}
	setFromResource(r :Resource) {
		this.id = r.idNumber();
		this.subdomain = r.attrString('subdomain');
		this.created_dt = r.attrDate('created_dt');
		this.paused = r.attrBool('paused');

		const app_version_rel = r.relOne('app_version');
		[this.app_id, this.version] = parseId(app_version_rel.idString());

		console.log('app version', app_version_rel, this);

		this.loaded = true;
	}
	
	// actions:
	async setPause(pause :boolean) {
		const id_str = this.id+''; 
		const doc = new DocumentBuilder('appspaces', id_str);
		doc.setAttr('paused', pause);
		const data = patch('/appspaces/'+id_str, doc.getJSON());
		this.paused = pause;
	}
}

export function ReactiveAppspace() {
	return reactive(new Appspace);
}

export class Appspaces {
	as : Map<number,Appspace> = new Map();

	async fetchForOwner() {
		const resp_data = await get('/appspaces?include=app_version&filter=owner');
		const doc = new Document(resp_data);
		doc.getCollection().forEach(res => {
			const appspace = new Appspace;
			appspace.setFromResource(res);
			this.as.set(appspace.id, appspace);

			const app_version_rel = res.relOne('app_version');
			const inc_res = doc.getIncluded('app_versions', app_version_rel.idString());
			appspace.app_version.setFromResource(inc_res);
		});
	}

	get asArray() : Appspace[] {
		// maybe this should return an empty array if all_loaded === false
		// Otherwise, some views might load some appspaces, then the appspace view will render a partial list.
		return Array.from(this.as.values());
	}
}

export function ReactiveAppspaces() {
	return reactive(new Appspaces);
}