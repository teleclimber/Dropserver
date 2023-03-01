import { ref, computed } from 'vue';
import { defineStore } from 'pinia';
import { ax } from '@/controllers/userapi';
import { LoadState } from './types';

export const useInstanceMetaStore = defineStore('instance-meta', () => {
	const load_state = ref(LoadState.NotLoaded);
	const is_loaded = computed( () => load_state.value === LoadState.Loaded );

	const ds_host_version = ref("");

	async function loadData() {
		const resp_data = await ax.get('/api/instance/');
		ds_host_version.value = (resp_data.data as any).ds_host_version;
		load_state.value = LoadState.Loaded;
	}

	return {is_loaded, loadData, ds_host_version};
});