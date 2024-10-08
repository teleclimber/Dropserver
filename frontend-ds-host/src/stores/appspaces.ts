import { ref, shallowRef, ShallowRef, computed } from 'vue';
import { defineStore } from 'pinia';
import { ax } from '../controllers/userapi';
import { on } from '../sse';
import { appVersionUIFromRaw } from './apps';
import { LoadState, Appspace, AppspaceStatus } from './types';

type NewAppspaceData = {
	app_id:number,
	app_version:string,
	domain_name: string,
	subdomain: string,
	dropid: string
}

function appspaceStatusFromRaw(raw:any) :AppspaceStatus {
	return {
		owner_id: Number(raw.owner_id),
		appspace_id: Number(raw.appspace_id),
		paused: !!raw.paused,
		temp_paused: raw.temp_paused,
		temp_pause_reason: raw.temp_pause_reason+'',
		appspace_schema: Number(raw.appspace_schema),
		app_version_schema: Number(raw.app_version_schema),
		problem: !!raw.problem
	}
}

function appspaceFromRaw(raw:any) :Appspace {
	return {
		appspace_id: Number(raw.appspace_id),
		domain_name: raw.domain_name+'',
		no_tls: !!raw.no_tls,
		port_string: raw.port_string+'',
		dropid: raw.dropid+'',
		created_dt: new Date(raw.created_dt),
		paused: !!raw.paused,
		app_id: Number(raw.app_id),
		app_version: raw.app_version+'',
		status: appspaceStatusFromRaw(raw.status),
		upgrade_version: raw.upgrade_version ? raw.upgrade_version+'' : undefined,
		ver_data: raw.ver_data ? appVersionUIFromRaw(raw.ver_data) : undefined
	}
}

export const useAppspacesStore = defineStore('user-appspaces', () => {
	const load_state = ref(LoadState.NotLoaded);

	const appspaces : ShallowRef<Map<number,ShallowRef<Appspace>>> = shallowRef(new Map());

	const is_loaded = computed( () => {
		return load_state.value === LoadState.Loaded;
	});

	async function loadData() {
		if( load_state.value === LoadState.NotLoaded ) {
			load_state.value = LoadState.Loading;
			const resp = await ax.get('/api/appspace');
			if( !Array.isArray(resp.data) ) throw new Error("expected appspaces to be array");
			resp.data.forEach( (raw:any) => {
				const as = appspaceFromRaw(raw);
				appspaces.value.set(as.appspace_id, shallowRef(as));
			});
			appspaces.value = new Map(appspaces.value);
			load_state.value = LoadState.Loaded;
		}
	}
	async function loadAppspace(appspace_id:number) {
		const resp = await ax.get('/api/appspace/'+appspace_id);
		const as_in = appspaceFromRaw(resp.data);
		const as_ex = appspaces.value.get(appspace_id);
		if( as_ex === undefined ) {
			appspaces.value.set(appspace_id, shallowRef(as_in));
			appspaces.value = new Map(appspaces.value);
		}
		else {
			as_ex.value = as_in;
		}
	}

	// connect to events that are relevant to appspace
	// - appspace status, ...
	on('AppspaceStatus', (raw) => {
		const status = appspaceStatusFromRaw(raw);
		const as = getAppspace(status.appspace_id);
		if( !as ) return;

		const old_status = as.value.status;
		let reload = false;
		if( old_status.paused !== status.paused ) reload = true;	// if status changes for pause, means pause was toggled.
		// other traps for reload

		if( reload ) loadAppspace(status.appspace_id);
		else as.value = Object.assign({}, as.value, {status});
	});

	function getAppspace(appspace_id:number) {
		return appspaces.value.get(appspace_id);
	}
	function mustGetAppspace(appspace_id:number) {
		const a = getAppspace(appspace_id);
		if( !a ) throw new Error("appspace not found "+appspace_id);
		return a;
	}

	function getAppspacesForApp(app_id:number) :ShallowRef<Appspace>[] {
		if( !is_loaded.value ) return [];
		const resp :ShallowRef<Appspace>[] = [];
		appspaces.value.forEach( a => {
			if( a.value.app_id === app_id ) resp.push(a);
		});
		return resp;
	}
	function getAppspacesForAppVersion(app_id:number, version:string) :ShallowRef<Appspace>[] {
		if( !is_loaded.value ) return [];
		const resp :ShallowRef<Appspace>[] = [];
		appspaces.value.forEach( a => {
			if( a.value.app_id === app_id && a.value.app_version === version ) resp.push(a);
		});
		return resp;
	}

	async function createAppspace(data:NewAppspaceData) :Promise<{appspace_id:number, job_id:number}> {
		load_state.value = LoadState.NotLoaded;
		const resp = await ax.post('/api/appspace', data);
		return {
			appspace_id: Number(resp.data.appspace_id),
			job_id: Number(resp.data.job_id)
		}
	}

	async function setPause(appspace_id: number, pause :boolean) {
		const a = mustGetAppspace(appspace_id);
		const data = await ax.post('/api/appspace/'+appspace_id+'/pause', {pause});
		// check that it returned OK!
		a.value.paused = pause;
	}

	async function deleteAppspace(appspace_id: number) {
		mustGetAppspace(appspace_id);	 //throws is appspace not found.
		await ax.delete('/api/appspace/'+appspace_id);
		appspaces.value.delete(appspace_id);
		appspaces.value = new Map(appspaces.value);
	}

	return {
		is_loaded,
		loadData,
		loadAppspace,
		appspaces,
		getAppspace,
		mustGetAppspace,
		getAppspacesForApp,
		getAppspacesForAppVersion,
		createAppspace,
		setPause,
		deleteAppspace
	}
});