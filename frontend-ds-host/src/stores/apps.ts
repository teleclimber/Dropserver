import { ref, Ref, shallowRef, ShallowRef, triggerRef, shallowReactive, computed, version, isReactive } from 'vue';
import { defineStore } from 'pinia';
import axios from 'axios';
import { ax } from '../controllers/userapi';
import type {AxiosResponse, AxiosError} from 'axios';

import twineClient from '../twine-services/twine_client';
import {SentMessageI} from 'twine-web';
import { LoadState, App, AppVersion, AppVersionUI, AppManifest } from './types';
import { Loadable, attachLoadState, setLoadState, getLoadState } from './loadable';

export function appVersionUIFromRaw(raw:any) :AppVersionUI {
	return {
		app_id: Number(raw.app_id),
		created_dt: new Date(raw.created_dt),
		color: raw.color ? raw.color+'' : undefined,
		name: raw.name+'',
		schema: Number(raw.schema),
		short_desc: raw.short_desc+'',
		version: raw.version+'',
		authors: raw.authors ? raw.authors.map( (a:any) => ({name:a.name+'', email: a.email+'', url: a.url+''})) : [],
		website: raw.website + '',
		code: raw.code + '',
		funding: raw.funding + '',
		release_date: raw.release_date+'',
		license: raw.license+''
	}
}
function appFromRaw(raw:any) :App {
	const app_id = Number(raw.app_id);
	const created_dt = new Date(raw.created_dt);
	const cur_ver = raw.cur_ver ? raw.cur_ver+'' : undefined;
	return {
		app_id,
		created_dt,
		cur_ver,
		ver_data: cur_ver ? appVersionUIFromRaw(raw.ver_data) : undefined
	}
}

export type UploadResp = {
	app_get_key: string
}

export const useAppsStore = defineStore('apps', () => {
	const load_state = ref(LoadState.NotLoaded);

	const apps : ShallowRef<Map<number,ShallowRef<App>>> = shallowRef(new Map());
	const app_versions :Map<number, Loadable<AppVersionUI[]>> = new Map;

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

	// app versions:
	function getAppVersions(app_id:number) :Loadable<AppVersionUI[]> | undefined {
		return app_versions.get(app_id);
	}
	function mustGetAppVersions(app_id:number) :Loadable<AppVersionUI[]> {
		const ret = getAppVersions(app_id);
		if( ret === undefined ) throw new Error('no versions for app: '+app_id);
		return ret;
	}
	async function loadAppVersions(app_id:number) :Promise<void> {
		if( app_versions.has(app_id) ) return;
		const av :Loadable<AppVersionUI[]> = shallowReactive(attachLoadState([], LoadState.Loading));
		console.log("is reactive at creation:", isReactive(av));
		app_versions.set(app_id, av);
		const resp = await ax.get(`/api/application/${app_id}/version`);
		resp.data.forEach( (raw:any) => {
			av.push(appVersionUIFromRaw(raw));
		});
		setLoadState(av, LoadState.Loaded);
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
		app_versions.delete(resp.data.app_id);	// delete any versions to force reload.
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
		await ax.delete('/api/application/'+app_id+'/version/'+version);

		const av = mustGetAppVersions(app_id);
		if( getLoadState(av) !== LoadState.Loaded ) throw new Error("trying to delete version in not-yet loaded versions store");
		const v_index = av.findIndex( v => v.version === version );
		const v = av[v_index];
		if( v === undefined ) throw new Error("did not find the version");
		av.splice(v_index, 1);
	}
	
	async function deleteApp(app_id: number) {
		await ax.delete('/api/application/'+app_id);
		apps.value.delete(app_id);
		apps.value = new Map(apps.value);
		app_versions.delete(app_id);
	}

	return {
		is_loaded, 
		loadData, apps, getApp, mustGetApp, deleteApp,
		loadAppVersions, getAppVersions, mustGetAppVersions, deleteAppVersion,
		uploadNewApplication, commitNewApplication, uploadNewAppVersion
	};
});


// NewAppVersionResp is returned by the server when it reads the contents of a new app code
// (whether it's a new version or an all new app).
// It returns any errors / problems found in the files, and the app version data if passable.
export type AppGetMeta = {
	key: string, // key is used to commit the uploaded files to their "destination" (new app, new app version)
	prev_version: string,
	next_version: string,
	errors: string[],	// maybe array of strings?
	warnings: Record<string, string>,
	version_manifest?: AppManifest
}


function rawToAppManifest(raw:any) :AppManifest {
	const ret = Object.assign({}, raw);
	Object.keys(ret).filter( k => k.includes("-") ).forEach( k => {
		const new_k = k.replaceAll("-", "_");
		ret[new_k] = ret[k];
		delete ret[k];
	});
	if( ret.authors ) {
		ret.authors = ret.authors.map( (a:any) => ({name:a.name+'', email: a.email+'', url: a.url+''}))
	}
	else ret.authors = [];
	return ret;
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
		if( data.meta.version_manifest ) data.meta.version_manifest = rawToAppManifest(data.meta.version_manifest);
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
