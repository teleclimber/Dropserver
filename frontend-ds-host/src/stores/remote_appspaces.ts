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

export const useRemoteAppspacesStore = defineStore('user-remote-appspaces', () => {
	const load_state = ref(LoadState.NotLoaded);

	const appspaces : ShallowRef<Map<string,ShallowRef<RemoteAppspace>>> = shallowRef(new Map());

	const is_loaded = computed( () => {
		return load_state.value === LoadState.Loaded;
	});

	async function loadData() {
		if( load_state.value === LoadState.NotLoaded ) {
			load_state.value = LoadState.Loading;
			const resp = await ax.get('/api/remoteappspace');
			if( !Array.isArray(resp.data) ) throw new Error("expected remote appspaces to be array");
			resp.data.forEach( (raw:any) => {
				const as = appspaceFromRaw(raw);
				appspaces.value.set(as.domain_name, shallowRef(as));
			});
			load_state.value = LoadState.Loaded;
		}
	}

	return { is_loaded, loadData, appspaces };
});