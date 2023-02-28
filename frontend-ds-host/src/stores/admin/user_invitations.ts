import { ref, computed, reactive } from 'vue';
import { defineStore } from 'pinia';
import {ax} from '@/controllers/userapi';
import { LoadState } from '../types';

interface Invite {
	email: string
}

function inviteFromRaw(raw:any) :Invite {
	return {
		email: raw.email+"",
	};
}

export const useAdminUserInvitesStore = defineStore('admin-user-invites', () => {
	const load_state = ref(LoadState.NotLoaded);
	const is_loaded = computed( () => load_state.value === LoadState.Loaded );

	const invites : Array<Invite> = reactive([]);
	
	async function fetch() {
		if( load_state.value === LoadState.NotLoaded ) {
			load_state.value = LoadState.Loading;
			const resp = await ax.get('/api/admin/invitation/');
			if( !Array.isArray(resp.data) ) throw new Error("expected array for admin invitations, got "+typeof resp.data);
			resp.data.forEach( (r) => {
				invites.push(inviteFromRaw(r));
			});
			load_state.value = LoadState.Loaded;
		}
	}

	async function createInvitation(email:string) :Promise<string> {
		const resp = await ax.post('/api/admin/invitation', {email});
		if( resp.status === 200 ) {
			if( typeof resp.data !== 'string' ) throw new Error("expected a string response");
			return resp.data;
		}
		else if( resp.status === 204 ) {	// "No content" means "Perfect, No Notes"
			console.log("adding email")
			invites.push({email})
			return '';
		}
		else throw new Error("got unexpected response status "+resp.status);
	}

	return {is_loaded, fetch, invites, createInvitation}
});