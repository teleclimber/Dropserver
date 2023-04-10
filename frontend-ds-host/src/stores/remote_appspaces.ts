import { ref, shallowRef, ShallowRef, computed } from 'vue';
import { defineStore } from 'pinia';
import { ax } from '../controllers/userapi';
import { LoadState, RemoteAppspace } from './types';

function appspaceFromRaw(raw:any) :RemoteAppspace {
	return {
		domain_name: raw.domain_name+'',
		owner_dropid: raw.owner_dropid+'',
		user_dropid: raw.user_dropid+'',
		no_tls: !!raw.no_tls,
		port_string: raw.port_string+'',
		created_dt: new Date(raw.created_dt)
	};
}


type RemoteAppspacePost = {
	check_only: boolean,
	domain_name: string,
	user_dropid: string,
}
export type RemoteAppspacePostResp = {
	inputs_valid: boolean,
	domain_message: string,
	remote_message: string
}

export const useRemoteAppspacesStore = defineStore('user-remote-appspaces', () => {
	const load_state = ref(LoadState.NotLoaded);

	const appspaces : ShallowRef<Map<string,ShallowRef<RemoteAppspace>>> = shallowRef(new Map());

	const is_loaded = computed( () => {
		return load_state.value === LoadState.Loaded;
	});

	async function loadData() {
		if( load_state.value === LoadState.NotLoaded ) {
			load_state.value = LoadState.Loading;
			const as :Map<string,ShallowRef<RemoteAppspace>> = new Map;
			const resp = await ax.get('/api/remoteappspace');
			if( !Array.isArray(resp.data) ) throw new Error("expected remote appspaces to be array");
			resp.data.forEach( (raw:any) => {
				const r = appspaceFromRaw(raw);
				as.set(r.domain_name, shallowRef(r));
			});
			appspaces.value = as;
			load_state.value = LoadState.Loaded;
		}
	}

	function get(domain:string) {
		return appspaces.value.get(domain);
	}

	async function create(domain_name:string, user_dropid: string) :Promise<RemoteAppspacePostResp> {
		const payload:RemoteAppspacePost = {
			check_only: false,
			domain_name,
			user_dropid
		}
	
		const ret = await ax.post('/api/remoteappspace', payload);

		appspaces.value = new Map();
		load_state.value = LoadState.NotLoaded;
	
		return <RemoteAppspacePostResp>ret.data;
	}

	async function deleteAppspace(domain_name:string) :Promise<void> {
		await ax.delete('/api/remoteappspace/'+domain_name);
		appspaces.value.delete(domain_name);
		appspaces.value = new Map(appspaces.value);
	}

	return { is_loaded, loadData, get, appspaces, create, deleteAppspace };
});