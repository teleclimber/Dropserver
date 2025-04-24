import { ref, computed, ShallowRef, shallowRef, triggerRef } from 'vue';
import { defineStore } from 'pinia';
import { ax } from '@/controllers/userapi';
import { LoadState, User } from '../types';
import { userFromRaw } from '../helpers';

export const useAdminAllUsersStore = defineStore('admin-all-users', () => {
	const load_state = ref(LoadState.NotLoaded);
	const is_loaded = computed( () => load_state.value === LoadState.Loaded );

	const users : ShallowRef<Map<number,ShallowRef<User>>> = shallowRef(new Map());
	
	async function fetch() {
		if( load_state.value === LoadState.NotLoaded ) {
			load_state.value = LoadState.Loading;
			const resp = await ax.get('/api/admin/user/');
			if( !Array.isArray(resp.data) ) throw new Error("expected array for admin all users, got "+typeof resp.data);
			resp.data.forEach( (raw:any) => {
				const u = userFromRaw(raw);
				users.value.set(u.user_id, shallowRef(u));
			});
			users.value = new Map(users.value);
			load_state.value = LoadState.Loaded;
		}
	}

	async function createWithTSNet(tsnet_id:string) {
		const resp = await ax.post(`/api/admin/user/`, {tsnet_id: tsnet_id});
		const u = userFromRaw(resp.data);
		users.value.set(u.user_id, shallowRef(u));
		triggerRef(users);
	}

	async function updateTSNet(user_id: number, tsnet_id :string) {
		const user = users.value.get(user_id);
		if( !user ) throw new Error("user not found");
		const resp = await ax.post(`/api/admin/user/${user_id}/tsnet`, {tsnet_id: tsnet_id});
		user.value.tsnet_identifier = resp.data.tsnet_identifier;
		user.value.tsnet_extra_name = resp.data.tsnet_extra_name;
		triggerRef(user);
	}

	async function deleteTSNet(user_id: number) {
		const user = users.value.get(user_id);
		if( !user ) throw new Error("user not found");
		const resp = await ax.delete(`/api/admin/user/${user_id}/tsnet`);
		user.value.tsnet_identifier = "";
		user.value.tsnet_extra_name = "";
		triggerRef(user);
	}

	return {is_loaded, fetch, users, createWithTSNet, updateTSNet, deleteTSNet}
});

