import { ref, Ref, shallowRef, ShallowRef, triggerRef, shallowReactive, computed, version, isReactive } from 'vue';
import { defineStore } from 'pinia';
import axios from 'axios';
import { ax } from '../controllers/userapi';
import type {AxiosResponse, AxiosError} from 'axios';

import twineClient from '../twine-services/twine_client';
import {SentMessageI} from 'twine-web';
import { LoadState, App, AppVersionUI, AppManifest, AppUrlData } from './types';
import { Loadable, attachLoadState, setLoadState, getLoadState } from './loadable';

export function appUrlDataFromRaw(raw:any) :AppUrlData {
	return {
		app_id: Number(raw.app_id),
		url: raw.url+'',
		automatic: !!raw.automatic,
		last_dt: new Date(raw.last_dt),
		last_result: raw.last_result+'',
		new_url: raw.new_url + '',
		new_url_dt: raw.new_url_dt ? new Date(raw.new_url_dt) : undefined,
		listing_dt: new Date(raw.listing_dt),
		latest_version: raw.latest_version+''
	}
}

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
		url_data: raw.url_data ? appUrlDataFromRaw(raw.url_data) : undefined,
		cur_ver,
		ver_data: cur_ver ? appVersionUIFromRaw(raw.ver_data) : undefined
	}
}

export type PostNewAppResp = {
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
	async function loadApp(app_id:number) {
		const resp = await ax.get('/api/application/'+app_id);
		const app_in = appFromRaw(resp.data);
		const app_ex = apps.value.get(app_id);
		if( app_ex === undefined ) {
			apps.value.set(app_id, shallowRef(app_in));
			apps.value = new Map(apps.value);
		}
		else {
			app_ex.value = app_in;
		}
	}
	function setReload() {
		// ugh.
		load_state.value = LoadState.NotLoaded;
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
		if( !app_versions.has(app_id) ) app_versions.set(app_id, shallowReactive(attachLoadState([], LoadState.Loading)));
		const av = app_versions.get(app_id)!;
		const resp = await ax.get(`/api/application/${app_id}/version`);
		av.splice(0, av.length);	//empty it
		resp.data.forEach( (raw:any) => {
			av.push(appVersionUIFromRaw(raw));
		});
		av.reverse();
		setLoadState(av, LoadState.Loaded);
	}

	// upload new application sends the files to backend for temporary storage.
	async function uploadNewApplication(package_file: File): Promise<string> {
		const form_data = new FormData();
		form_data.append('package', package_file, package_file.name);

		const resp = await ax.post('/api/application', form_data);
		const data = <PostNewAppResp>resp.data;

		return data.app_get_key;
	}

	// fetch new app from the passed URL
	async function getNewAppFromURL(url: string, auto_refresh_listing:boolean, version?: string) :Promise<string> {
		const resp = await ax.post('/api/application', {url, auto_refresh_listing, version});
		const data = <PostNewAppResp>resp.data;
		return data.app_get_key;
	}
	async function getNewVersionFromURL(app_id :number, version: string) :Promise<string> {
		const resp = await ax.post('/api/application/'+app_id+'/version', {version});
		const data = <PostNewAppResp>resp.data;
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
		const data = <PostNewAppResp>resp.data;

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

		// When deleting a version, the cur_ver may change.
		const app  = mustGetApp(app_id);
		if( app.value.cur_ver === version ) {
			app.value.cur_ver = "";
			app.value.ver_data = undefined;
			if( av.length ) {
				app.value.cur_ver = av[0].version;
				app.value.ver_data = av[0];
			}
			app.value = Object.assign({}, app.value);
		}
	}
	
	async function deleteApp(app_id: number) {
		await ax.delete('/api/application/'+app_id);
		apps.value.delete(app_id);
		apps.value = new Map(apps.value);
		app_versions.delete(app_id);
	}

	async function refreshListing(app_id:number) :Promise<string> {
		const resp = await ax.post('/api/application/'+app_id+'/refresh-listing');
		loadApp(app_id);
		return resp.data+'';
	}

	async function changeAutomaticListingFetch(app_id: number, automatic:boolean) {
		const app = mustGetApp(app_id);
		if( !app.value.url_data ) return;	 // maybe an error would be better?
		await ax.post('/api/application/'+app_id+'/automatic-listing-fetch', {automatic});
		app.value.url_data.automatic = automatic;
	}

	async function fetchVersionManifest(app_id:number, version:string|undefined ) :Promise<AppGetMeta> {
		const resp = await ax.get(`/api/application/${app_id}/fetch-version-manifest?version=${version || ""}`);
		const data = <AppGetMeta>resp.data;
		data.version_manifest = rawToAppManifest(data.version_manifest);
		return data;
	}

	return {
		is_loaded, setReload,
		loadData, apps, loadApp, getApp, mustGetApp, deleteApp,
		loadAppVersions, getAppVersions, mustGetAppVersions, deleteAppVersion,
		uploadNewApplication, getNewAppFromURL, getNewVersionFromURL, commitNewApplication, uploadNewAppVersion,
		refreshListing, changeAutomaticListingFetch,
		fetchVersionManifest
	};
});


// AppGetMeta is returned by the server when it reads the contents of a new app code
// (whether it's a new version or an all new app).
// It returns any errors / problems found in the files, and the app version data if passable.
export type AppGetMeta = {
	key: string, // key is used to commit the uploaded files to their "destination" (new app, new app version)
	prev_version: string,
	next_version: string,
	errors: string[],	// maybe array of strings?
	warnings: Warning[],
	version_manifest?: AppManifest,
	app_id: number
}

export type Warning = {
	field: string,
	problem: string,
	bad_value: string,
	message: string
}


export function rawToAppManifest(raw:any) :AppManifest|undefined {
	if( raw.version === "" ) return undefined;	// version is the only required part of manifest, if it's not set we just got a zero-value manifest.
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
	input: string,
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
					const ev = <AppGetEvent>JSON.parse(new TextDecoder('utf-8').decode(m.payload));
					m.sendOK();
					if( ev.done ) this.unsubscribeKey();
					if( ev.done || ev.input ) this.loadInProcess();
					else this.last_event.value = ev;
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
		this.meta.value = data.meta;
		if( this.done && !this.has_error ) {
			const appStore = useAppsStore();
			appStore.setReload();	// temporary until we can lazy-load apps.
		}
	}
	async unsubscribeKey() {
		if( !this.subMessage ) return;
		const m = this.subMessage;
		this.subMessage = undefined;
		await m.refSendBlock(13, undefined);
	}

	get done() :boolean {	// Note we changed what "done" means on the backend, sort of. It now means the process is completely finished.
		return !!this.last_event.value?.done
	}
	get expects_input() :boolean {
		return !!this.last_event.value?.input;
	}
	get must_confirm() :boolean {
		return !this.done && this.last_event.value?.input === "commit";
	}
	get has_error() :boolean {
		return this.meta.value?.errors.length !== 0;
	}
	get version_manifest() :AppManifest|undefined {
		return this.meta.value?.version_manifest;
	}

	async cancel() {
		if( this.not_found.value ) return;
		try {
			await this.unsubscribeKey();
			this.not_found.value = true;
			this.last_event.value = undefined;
			this.meta.value = undefined;
			await ax.delete('/api/application/in-process/'+this.key.value);
		}
		catch(e) {
			// no-op. Whatever the error, don't hold up the frontend UI because of it.
		}
	}
}
