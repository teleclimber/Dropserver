import { reactive } from 'vue';

import {ax, get, post, del} from '../controllers/userapi';

import { AppVersion } from './app_versions';

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

// type AppspaceMeta struct {
// 	AppspaceID int            `json:"appspace_id"`
// 	AppID      int            `json:"app_id"`
// 	AppVersion domain.Version `json:"app_version"`
// 	DomainName string         `json:"domain"`
// 	Created    time.Time      `json:"created_dt"`
// 	Paused     bool           `json:"paused"`
// 	Upgrade    *VersionMeta   `json:"upgrade,omitempty"`
// }

export class Appspace {
	loaded = false;

	id = 0;
	app_id = 0;
	app_version = '';
	domain_name = "";
	no_tls = false;
	port_string = "";
	dropid = "";
	created_dt = new Date();
	paused = false;
	upgrade :AppVersion|undefined;

	async fetch(id: number) {
		const resp_data = await get('/appspace/'+id);
		this.setFromRaw(resp_data);
	}
	async refresh() {
		await this.fetch(this.id);
	}
	setFromRaw(raw :any) {
		this.id = Number(raw.appspace_id);
		this.domain_name = raw.domain_name+'';
		this.no_tls = !!raw.no_tls;
		this.port_string = raw.port_string+'';
		this.dropid = raw.dropid+'';
		this.created_dt = new Date(raw.created_dt);
		this.paused = !!raw.paused;
		this.app_id = Number(raw.app_id);
		this.app_version = raw.app_version+'';

		if( raw.upgrade ) {
			this.upgrade = new AppVersion;
			this.upgrade.setFromRaw(raw.upgrade)
		}

		this.loaded = true;
	}
	
	// actions:
	async setPause(pause :boolean) {
		const data = await post('/appspace/'+this.id+'/pause', {pause});
		this.paused = pause;
	}
	async del() {
		await del('/appspace/'+this.id)
	}
}

export class Appspaces {
	loaded = false;

	as : Map<number,Appspace> = new Map();

	async fetchForOwner() {
		const resp_data = await get('/appspace');
		resp_data.forEach( (raw:any) => {
			const appspace = new Appspace;
			appspace.setFromRaw(raw);
			this.as.set(appspace.id, appspace);
		});
		this.loaded = true;
	}
	async fetchForApp(app_id: number) {
		const resp_data = await get('/appspace?app='+app_id);
		resp_data.forEach( (raw:any) => {
			const appspace = new Appspace;
			appspace.setFromRaw(raw);
			this.as.set(appspace.id, appspace);
		});
		this.loaded = true;
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

export type NewAppspaceData = {
	app_id:number,
	app_version:string,
	domain_name: string,
	subdomain: string,
	dropid: string,
}
export async function createAppspace(data:NewAppspaceData) :Promise<number> {
	const resp_data = await post('/appspace', data);
	return Number(resp_data.appspace_id);
}

// Appspace Restore routes:
export type AppspaceRestoreData = {
	loaded: boolean,
	err: {missing_files:string[], zip_files:string[]} | null,
	token: string,
	schema: number,
	//ds_api :number,
	// other stuff later
}
export async function uploadRestoreZip(appspace_id:number, zipFile :File) :Promise<AppspaceRestoreData> {	//return token
	const formData = new FormData();
	formData.append("zip", zipFile);
	const resp = await ax.post('/api/appspace/'+appspace_id+'/restore/upload', formData, {
		headers: {
			'Content-Type': 'multipart/form-data'
		}
	});

	const ret :AppspaceRestoreData = {
		loaded: true,
		err: resp.data.err || null,
		schema: Number(resp.data.schema),
		token:resp.data.token +''
	}
	return ret;
}

export async function selectRestoreBackup(appspace_id:number, backup_file: string) :Promise<AppspaceRestoreData> {
	const resp = await ax.post('/api/appspace/'+appspace_id+'/restore/', {backup_file});
	const ret :AppspaceRestoreData = {
		loaded: true,
		err: resp.data.err || null,
		schema: Number(resp.data.schema),
		token:resp.data.token +''
	}
	return ret;
}

export async function commitRestore(appspace_id:number, token: string) {
	await ax.post('/api/appspace/'+appspace_id+'/restore/'+token);
}