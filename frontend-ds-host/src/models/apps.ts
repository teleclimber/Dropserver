import {reactive} from 'vue';
import axios from 'axios';
import {ax, get, post, del} from '../controllers/userapi';
import type {AxiosResponse, AxiosError} from 'axios';

import twineClient from '../twine-services/twine_client';
import {SentMessageI} from 'twine-web';

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
	// need an error, in case of fetching an id that is wrong.
	
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
	loaded = false;
	apps : Map<number,App> = new Map();
	
	async fetchForOwner() {
		const resp_data = await get('/application');
		resp_data.apps.forEach( (raw:any) => {
			const app = new App;
			app.setFromRaw(raw);
			this.apps.set(app.app_id, app);
		});
		this.loaded = true;
	}

	get asArray() : App[] {
		// maybe this should return an empty array if all_loaded === false
		// Otherwise, some views might load some appspaces, then the appspace view will render a partial list.
		return Array.from(this.apps.values());
	}
}

export type SelectedFile = {
	file: File,
	rel_path: string
}

// NewAppVersionResp is returned by the server when it reads the contents of a new app code
// (whether it's a new version or an all new app).
// It returns any errors / problems found in the files, and the app version data if passable.
export type AppGetMeta = {
	key: string, // key is used to commit the uploaded files to their "destination" (new app, new app version)
	prev_version: string,
	next_version: string,
	errors: string[],	// maybe array of strings?
	version_metadata?: VersionMetadata
}
type VersionMetadata = {
	name :string,
	version :string,
	api_version :number,
	schema :number,
	migrations :number[],
	// user permissions
}

// upload new application sends the files to backend for temporary storage.
export async function uploadNewApplication(selected_files: SelectedFile[]): Promise<string> {
	const form_data = new FormData();
	selected_files.forEach((sf)=> {
		form_data.append( 'app_dir', sf.file, sf.rel_path );
	});

	const resp_data = await post('/application', form_data);
	const resp = <UploadResp>resp_data;

	return resp.app_get_key;
}

export async function commitNewApplication(key:string): Promise<App> {
	const resp_data = await post('/application?key='+key, undefined);
	const app = new App;
	app.setFromRaw(resp_data);
	return app;
}

export type UploadResp = {
	app_get_key: string
}

export async function uploadNewAppVersion(app_id:number, selected_files: SelectedFile[]): Promise<string> {
	const form_data = new FormData();
	selected_files.forEach((sf)=> {
		form_data.append( 'app_dir', sf.file, sf.rel_path );
	});

	const resp_data = await post('/application/'+app_id+'/version', form_data);
	const resp = <UploadResp>resp_data;

	return resp.app_get_key;
}

export async function deleteAppVersion(app_id: number, version:string) {
	await del('/application/'+app_id+'/version/'+version);
}

export async function deleteApp(app_id: number) {
	await del('/application/'+app_id);
}


// type InProcessResp struct {
//     LastEvent domain.AppGetEvent `json:"last_event"`
//     Meta      domain.AppGetMeta  `json:"meta"`
// }
// type AppGetEvent struct {
//     Key   AppGetKey `json:"key"`
//     Done  bool      `json:"done"`
//     Error bool      `json:"error"`
//     Step  string    `json:"step"`
// }
type AppGetEvent = {
	key: string,
	done: boolean,
	error: boolean,
	step: string
}
type InProcessResp = {
	last_event: AppGetEvent,
	meta: AppGetMeta
}
type CommitResp = {
	app_id: number,
	version: string
}
export class AppGetter {
	key = "";
	not_found = false;
	last_event: AppGetEvent | undefined;
	meta :AppGetMeta | undefined;

	private subMessage :SentMessageI|undefined;

	async updateKey(key :string) {
		this.key = key;

		await this.loadInProcess();

		if( this.done ) return;

		const payload = new TextEncoder().encode(key);

		await twineClient.ready();
		this.subMessage = await twineClient.twine.send(13, 11, payload);

		for await (const m of this.subMessage.incomingMessages()) {
			switch (m.command) {
				case 11:	//event
					this.last_event = <AppGetEvent>JSON.parse(new TextDecoder('utf-8').decode(m.payload));
					m.sendOK();
					if(this.last_event.done && this.meta === undefined ) {
						this.loadInProcess();
						this.unsubscribeKey();
					}
					break;
			
				default:
					m.sendError("What is this command?");
					throw new Error("what is this command? "+m.command);
			}
		}
	}
	async loadInProcess() {
		let resp :AxiosResponse|undefined;
		try {
			resp = await ax.get('/api/application/in-process/'+this.key);
		}
		catch(error: any | AxiosError) {
			if( axios.isAxiosError(error) && error.response && error.response.status == 404 ) {
				this.not_found = true;
				return;
			}
			throw error;
		}
		if( resp?.data === undefined ) return;
		const data = <InProcessResp>resp.data;
		this.last_event = data.last_event;
		if( this.last_event.done ) this.meta = data.meta;
	}
	async unsubscribeKey() {
		if( !this.subMessage ) return;
		const m = this.subMessage;
		this.subMessage = undefined;
		await m.refSendBlock(13, undefined);
	}

	get done() :boolean {
		if( this.meta ) return true;
		else return !!this.last_event?.done;
	}
	get canCommit() :boolean {
		return !!this.meta && this.meta.errors.length === 0;
	}

	async commit() :Promise<CommitResp> {
		const resp_data = <CommitResp>await post('/application/in-process/'+this.key, undefined);

		return resp_data;
	}
	async cancel() {
		if( this.not_found ) return;
		try {
			await del('/application/in-process/'+this.key);
		}
		catch(e) {
			// no-op. Whatever the error, don't hold up the frontend UI because of it.
		}
	}
}
