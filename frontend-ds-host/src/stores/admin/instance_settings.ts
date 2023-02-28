import { ref, computed } from 'vue';
import { defineStore } from 'pinia';
import { ax } from '@/controllers/userapi';
import { LoadState } from '../types';

export const useInstanceSettingsStore = defineStore('admin-instance-settings', () => {
	const load_state = ref(LoadState.NotLoaded);
	const is_loaded = computed( () => load_state.value === LoadState.Loaded );

	const registration_open = ref(false);

	async function loadData() {
		const resp_data = await ax.get('/api/admin/settings');
		registration_open.value = !!(resp_data.data as any).registration_open;
		load_state.value = LoadState.Loaded;
	}

	async function setRegistrationOpen(open:boolean) {
		await ax.patch('/api/admin/settings', {registration_open: open});
		registration_open.value = open;
	}

	return {is_loaded, loadData, registration_open, setRegistrationOpen};
});