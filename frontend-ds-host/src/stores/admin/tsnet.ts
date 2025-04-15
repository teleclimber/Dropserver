import { ref, Ref, ShallowRef, shallowRef } from 'vue';
import { defineStore } from 'pinia';
import { ax } from '../../controllers/userapi';
import { on } from '../../sse';
import { TSNetStatus, TSNetPeerUser } from '../types';
import { tsnetStatusFromRaw, tsnetPeerUsersFromRaw } from '../helpers/tsnet';

export const useAdminTSNetStore = defineStore('admin-tsnet', () => {
	const tsnet_status :Ref<TSNetStatus> = ref(tsnetStatusFromRaw({}));
	const peer_users: ShallowRef<TSNetPeerUser[]> = shallowRef([]);

	on('UserTSNetStatus', (raw) => {
		tsnet_status.value = tsnetStatusFromRaw(raw);
	});
	on('UserTSNetPeers', async (raw) => {
		loadTSNetPeerUsers();
	});

	async function loadTSNetStatus() {
		const resp = await ax.get(`/api/admin/tsnet`);
		tsnet_status.value = tsnetStatusFromRaw(resp.data);
	}
	async function loadTSNetPeerUsers() {
		const resp = await ax.get(`/api/admin/tsnet/peerusers`);
		if( !Array.isArray(resp.data) ) throw new Error("expected peerusers to be array");
		peer_users.value = tsnetPeerUsersFromRaw(resp.data);
	}

	return {
		tsnet_status, loadTSNetStatus,
		peer_users, loadTSNetPeerUsers
	}
});