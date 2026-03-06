import { ref, ShallowRef, shallowRef, triggerRef } from 'vue';
import { defineStore } from 'pinia';
import { ax } from '@/controllers/userapi';
import { LoadState, InstanceUser } from './types';

export const useInstanceUsersStore = defineStore('instance-users', () => {
	const load_state = ref(LoadState.NotLoaded);

	const users : ShallowRef<Map<number, ShallowRef<InstanceUser>>> = shallowRef(new Map());

	async function fetch() {
		if( load_state.value !== LoadState.NotLoaded ) return;
		load_state.value = LoadState.Loading;
		const resp = await ax.get('/api/instance-users/');
		if( !Array.isArray(resp.data) ) throw new Error("expected array for instance users, got "+typeof resp.data);
		let changed = false;
		resp.data.forEach( (raw:any) => {
			const existing = users.value.get(raw.user_id);
			if( existing ) {
				existing.value = {
					user_id: raw.user_id,
					display_name: raw.display_name,
					display_image: raw.display_image
				};
				triggerRef(existing);
			} else {
				users.value.set(raw.user_id, shallowRef({
					user_id: raw.user_id,
					display_name: raw.display_name,
					display_image: raw.display_image
				}));
				changed = true;
			}
		});
		if( changed ) triggerRef(users);
		load_state.value = LoadState.Loaded;
	}

	function getUser(user_id: number) :ShallowRef<InstanceUser> {
		if( load_state.value === LoadState.NotLoaded ) fetch();
		let u = users.value.get(user_id);
		if( !u ) {
			u = shallowRef({
				user_id: user_id,
				display_name: "...",
				display_image: ""
			});
			users.value.set(user_id, u);
			triggerRef(users);
		}
		return u;
	}

	return { load_state, users, fetch, getUser };
});
