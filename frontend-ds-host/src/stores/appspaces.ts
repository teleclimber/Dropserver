import { ref, shallowRef, ShallowRef, computed } from 'vue';
import { defineStore } from 'pinia';
import { ax } from '../controllers/userapi';
import { LoadState, Appspace } from './types';

type NewAppspaceData = {
	app_id:number,
	app_version:string,
	domain_name: string,
	subdomain: string,
	dropid: string
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
		upgrade_version: raw.uprade_version ? raw.upgrade_version+'' : undefined
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
			load_state.value = LoadState.Loaded;
		}
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

	async function createAppspace(data:NewAppspaceData) :Promise<number> {
		load_state.value = LoadState.NotLoaded;
		const resp = await ax.post('/api/appspace', data);
		return Number(resp.data.appspace_id);
	}

	return {is_loaded, loadData, appspaces, getAppspacesForApp, getAppspacesForAppVersion, createAppspace }
});