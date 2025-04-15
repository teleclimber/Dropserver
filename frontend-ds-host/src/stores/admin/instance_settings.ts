import { ref, Ref, computed } from 'vue';
import { defineStore } from 'pinia';
import { ax } from '@/controllers/userapi';
import { LoadState, TSNetData, TSNetCreateConfig } from '../types';
import { tsnetDataFromRaw } from '../helpers/tsnet';

export const useInstanceSettingsStore = defineStore('admin-instance-settings', () => {
	const load_state = ref(LoadState.NotLoaded);
	const is_loaded = computed( () => load_state.value === LoadState.Loaded );

	const registration_open = ref(false);

	const tsnet_data :Ref<TSNetData|undefined> = ref();

	async function loadData() {
		const resp_data = await ax.get('/api/admin/settings');
		registration_open.value = !!(resp_data.data as any).registration_open;
		tsnet_data.value = tsnetDataFromRaw(resp_data.data.tsnet);
		load_state.value = LoadState.Loaded;
	}

	async function setRegistrationOpen(open:boolean) {
		await ax.post('/api/admin/settings/registration', {registration_open: open});
		registration_open.value = open;
	}

	async function setTSNetData(data : TSNetCreateConfig) {
		await ax.post('/api/admin/settings/tsnet', data);
		tsnet_data.value = {
			control_url: data.control_url,
			hostname: data.hostname,
			connect: true
		};
	}
	async function setTSNetConnect(connect :boolean) {
		if( !tsnet_data.value ) throw new Error("expected tsnet data to exist");
		await ax.post('/api/admin/settings/tsnet/connect', {connect});
		tsnet_data.value.connect = connect;
	}
	async function deleteTSNetData() {
		await ax.delete('/api/admin/settings/tsnet');
		tsnet_data.value = undefined;
	}

	return {
		is_loaded, loadData,
		registration_open, setRegistrationOpen,
		tsnet_data, setTSNetData, setTSNetConnect, deleteTSNetData
	};
});