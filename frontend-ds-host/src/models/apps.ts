import {reactive} from 'vue';
import {get, post} from '../controllers/userapi';

import {AppVersion} from './app_versions';

// apps, is basically a collection of app versions.

// From go:
// type App struct {
// 	ID       string    `json:"id" api:"apps"`
// 	Name     string    `json:"name" api:"attr"`
// 	Created  time.Time `json:"created_dt" api:"attr"`
// 	Versions []string  `json:"versions" api:"rel,app_versions,app"`
// 	//Owner    string    `json:"owner" api:"rel,users"`
// }

export class App {
	loaded = false;
	app_id = 0;
	name= '';
	created_dt = new Date;

	versions : AppVersion[] = [];	// wondering if thous should be some sort of collection, so that it can be sorted, filtered, etc...

	async fetch(app_id: number) {
		const resp_data = await get('/application/'+app_id);
		this.setFromRaw(resp_data);
	}
	setFromRaw(raw :any) {
		this.app_id = Number(raw.app_id);
		this.name = raw.name + '';
		this.created_dt = new Date(raw.created_dt);

		if( Array.isArray(raw.versions) ) {
			this.versions = raw.versions.reverse().map((rawVer:any) => {
				const av = new AppVersion();
				av.setFromRaw(rawVer);
				return av;
			});
		}

		this.loaded = true;
	}
}

export class Apps {
	apps : Map<number,App> = new Map();

	async fetchForOwner() {
		const resp_data = await get('/application');
		resp_data.apps.forEach( (raw:any) => {
			const app = new App;
			app.setFromRaw(raw);
			this.apps.set(app.app_id, app);
		});
	}

	get asArray() : App[] {
		// maybe this should return an empty array if all_loaded === false
		// Otherwise, some views might load some appspaces, then the appspace view will render a partial list.
		return Array.from(this.apps.values());
	}
}

export function ReactiveApps() {
	return reactive(new Apps);
}


export type SelectedFile = {
	file: File,
	rel_path: string
}

// NewAppVersionResp is returned by the server when it reads the contents of a new app code
// (whether it's a new version or an all new app).
// It returns any errors / problems found in the files, and the app version data if passable.
export type UploadVersionResp = {
	key: string, // key is used to commit the uploaded files to their "destination" (new app, new app version)
	prev_version: string,
	next_version: string,
	errors?: string[],	// maybe array of strings?
	version_metadata?: AppVersion
}

// upload new application sends the files to backend for temporary storage.
export async function uploadNewApplication(selected_files: SelectedFile[]): Promise<UploadVersionResp> {
	const form_data = new FormData();
	selected_files.forEach((sf)=> {
		form_data.append( 'app_dir', sf.file, sf.rel_path );
	});

	const resp_data = await post('/application', form_data);
	const resp = <UploadVersionResp>resp_data;

	return resp;
}

export async function commitNewApplication(key:string): Promise<App> {
	const resp_data = await post('/application?key='+key, undefined);
	const app = new App;
	app.setFromRaw(resp_data);
	return app;
}

export async function uploadNewAppVersion(app_id:number, selected_files: SelectedFile[]): Promise<UploadVersionResp> {
	const form_data = new FormData();
	selected_files.forEach((sf)=> {
		form_data.append( 'app_dir', sf.file, sf.rel_path );
	});

	const resp_data = await post('/application/'+app_id+'/version', form_data);
	const resp = <UploadVersionResp>resp_data;

	return resp;
}

export async function commitNewAppVersion(app_id:number, key:string): Promise<App> {
	const resp_data = await post('/application/'+app_id+'/version?key='+key, undefined);
	const app = new App;
	app.setFromRaw(resp_data);
	return app;
}

