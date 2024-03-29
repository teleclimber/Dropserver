import { ref, computed, ShallowRef, shallowRef } from 'vue';
import { defineStore } from 'pinia';
import {ax} from '@/controllers/userapi';
import { LoadState, UserForAdmin } from '../types';

function userFromRaw(raw:any) :UserForAdmin {
	return {
		user_id: Number(raw.user_id),
		email: raw.email+"",
		is_admin: !!raw.is_admin
	};
}

export const useAdminAllUsersStore = defineStore('admin-all-users', () => {
	const load_state = ref(LoadState.NotLoaded);
	const is_loaded = computed( () => load_state.value === LoadState.Loaded );

	const users : ShallowRef<Map<number,ShallowRef<UserForAdmin>>> = shallowRef(new Map());
	
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

	return {is_loaded, fetch, users}
});

