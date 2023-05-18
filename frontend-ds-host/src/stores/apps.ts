import { ref, shallowRef, ShallowRef, computed } from 'vue';
import { defineStore } from 'pinia';
import axios from 'axios';
import { ax } from '../controllers/userapi';
import type {AxiosResponse, AxiosError} from 'axios';

import twineClient from '../twine-services/twine_client';
import {SentMessageI} from 'twine-web';
import { LoadState, App, AppVersion, SelectedFile } from './types';

function appVersionFromRaw(raw:any) {
	return {
		app_id: Number(raw.app_id),
		version: raw.version+'',
		app_name: raw.app_name+'',
		api_version: Number(raw.api_version),
		schema: Number(raw.schema),
		created_dt: new Date(raw.created_dt)
	}
}
function appFromRaw(raw:any) :App {
	const app_id = Number(raw.app_id);
	const name = raw.name + '';
	const created_dt = new Date(raw.created_dt);

	let versions = [];
	if( Array.isArray(raw.versions) ) versions = raw.versions.reverse().map(appVersionFromRaw);
	
	return {
		app_id,
		name,
		created_dt,
		versions
	}
}

export type UploadResp = {
	app_get_key: string
}

export const useAppsStore = defineStore('apps', () => {
	const load_state = ref(LoadState.NotLoaded);

	const apps : ShallowRef<Map<number,ShallowRef<App>>> = shallowRef(new Map());

	const is_loaded = computed( () => {
		return load_state.value === LoadState.Loaded;
	});

	async function loadData() {
		if( load_state.value === LoadState.NotLoaded ) {
			load_state.value = LoadState.Loading;
			const resp = await ax.get('/api/application');
			resp.data.apps.forEach( (raw:any) => {
				const app = appFromRaw(raw);
				apps.value.set(app.app_id, shallowRef(app));
			});
			apps.value = new Map(apps.value);
			load_state.value = LoadState.Loaded;
		}
	}

	function getApp(app_id: number) :ShallowRef<App> | undefined {
		return apps.value.get(app_id);
	}
	function mustGetApp(app_id: number) :ShallowRef<App> {
		const app = getApp(app_id);
		if( app === undefined ) throw new Error("could not et app: "+app_id);
		return app;
	}

	// upload new application sends the files to backend for temporary storage.
	async function uploadNewApplication(package_file: File): Promise<string> {
		const form_data = new FormData();
		form_data.append('package', package_file, package_file.name);

		const resp = await ax.post('/api/application', form_data);
		const data = <UploadResp>resp.data;

		return data.app_get_key;
	}

	async function commitNewApplication(key:string): Promise<number> {
		const resp = await ax.post('/api/application/in-process/'+key, undefined);
		// mark local data as stale now
		// Other options: route could return app data, or we could manually load it here.
		load_state.value = LoadState.NotLoaded;
		return resp.data.app_id;
	}

	async function uploadNewAppVersion(app_id:number, package_file: File): Promise<string> {
		const form_data = new FormData();
		form_data.append('package', package_file, package_file.name);

		const resp = await ax.post('/api/application/'+app_id+'/version', form_data);
		const data = <UploadResp>resp.data;

		return data.app_get_key;
	}

	async function deleteAppVersion(app_id: number, version:string) {
		const app = mustGetApp(app_id);
		await ax.delete('/api/application/'+app_id+'/version/'+version);

		const v_index = app.value.versions.findIndex( v => v.version === version );
		const v = app.value.versions[v_index];
		if( v === undefined ) throw new Error("did not find the version");
		app.value.versions.splice(v_index, 1);
		app.value = Object.assign({}, app.value);
	}
	
	async function deleteApp(app_id: number) {
		await ax.delete('/api/application/'+app_id);
		apps.value.delete(app_id);
		apps.value = new Map(apps.value);
	}

	return {is_loaded, loadData, apps, getApp, mustGetApp, deleteAppVersion, deleteApp, uploadNewApplication, commitNewApplication, uploadNewAppVersion};
});


// NewAppVersionResp is returned by the server when it reads the contents of a new app code
// (whether it's a new version or an all new app).
// It returns any errors / problems found in the files, and the app version data if passable.
export type AppGetMeta = {
	key: string, // key is used to commit the uploaded files to their "destination" (new app, new app version)
	prev_version: string,
	next_version: string,
	errors: string[],	// maybe array of strings?
	version_manifest?: AppManifest
}
type MigrationStep = {
	direction: "up"|"down"
	schema: number
}
type AppManifest = {
	name :string,
	short_description: string,
	version :string,
	release_date: Date|undefined,
	main: string,	// do we care here?
	schema: number,
	migrations: MigrationStep[],
	lib_version: string,	//semver
	signature: string,	//later
	code_state: string,	 // ? later
	icon: string,	// how to reference icon? app version should have  adefault path so no need to reference it here? Except to know if there is one or not
	
	authors: string[],	// later, 
	description: string,	// actually a reference to a long description. Later.
	release_notes: string,	// ref to a file or something...
	code: string,	// URL to code repo. OK.
	homepage: string,	//URL to home page for app
	help: string,	// URL to help
	license: string,	// SPDX format of license
	license_file: string,	// maybe this is like icon, lets us know it exists and can use the link to the file.
	funding: string,	// URL for now, but later maybe array of objects? Or...?

	size: number	// bytes of what? compressed package? 
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
export class AppGetter {
	key = ref("");
	not_found = ref(false);
	last_event :ShallowRef<AppGetEvent | undefined> = shallowRef();
	meta :ShallowRef<AppGetMeta | undefined> = shallowRef();

	private subMessage :SentMessageI|undefined;

	async updateKey(key :string) {
		this.key.value = key;

		await this.loadInProcess();

		if( this.done ) return;

		const payload = new TextEncoder().encode(key);

		await twineClient.ready();
		this.subMessage = await twineClient.twine.send(13, 11, payload);

		for await (const m of this.subMessage.incomingMessages()) {
			switch (m.command) {
				case 11:	//event
					this.last_event.value = <AppGetEvent>JSON.parse(new TextDecoder('utf-8').decode(m.payload));
					m.sendOK();
					if(this.last_event.value.done && this.meta.value === undefined ) {
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
			resp = await ax.get('/api/application/in-process/'+this.key.value);
		}
		catch(error: any | AxiosError) {
			if( axios.isAxiosError(error) && error.response && error.response.status == 404 ) {
				this.not_found.value = true;
				return;
			}
			throw error;
		}
		if( resp?.data === undefined ) return;
		const data = <InProcessResp>resp.data;
		this.last_event.value = data.last_event;
		if( this.last_event.value.done ) this.meta.value = data.meta;
	}
	async unsubscribeKey() {
		if( !this.subMessage ) return;
		const m = this.subMessage;
		this.subMessage = undefined;
		await m.refSendBlock(13, undefined);
	}

	get done() :boolean {
		if( this.meta.value ) return true;
		else return !!this.last_event.value?.done;
	}
	get canCommit() :boolean {
		return !!this.meta.value && this.meta.value.errors.length === 0;
	}

	async cancel() {
		if( this.not_found ) return;
		try {
			await ax.delete('/api/application/in-process/'+this.key.value);
		}
		catch(e) {
			// no-op. Whatever the error, don't hold up the frontend UI because of it.
		}
	}
}
