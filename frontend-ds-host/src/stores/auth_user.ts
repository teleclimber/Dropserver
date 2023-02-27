import { ref, computed } from 'vue';
import { defineStore } from 'pinia';
import {ax, get} from '../controllers/userapi';
import { LoadState } from './types';

export const useAuthUserStore = defineStore('authenticated-user', () => {
	const load_state = ref(LoadState.NotLoaded);
	const is_loaded = computed( () => load_state.value === LoadState.Loaded );

	const logged_in = ref(true);	// assumed true until a request fails.

	const user_id = ref(-1);
	const email = ref("");
	const is_admin = ref(false);

	async function fetch() {
		if( load_state.value === LoadState.NotLoaded ) {
			load_state.value = LoadState.Loading;
			const resp_data = await get('/user/');
			setFromRaw(resp_data);
			load_state.value = LoadState.Loaded;
		}
	}

	function setFromRaw(raw:any) {
		user_id.value = Number(raw.user_id);
		email .value= raw.email+"";
		is_admin.value = !!raw.is_admin;
	}

	function setUnauthorized() {
		logged_in.value = false;
	}

	// It's possible the backend will reject an email that the frontend will accept
	// like if email is in use, or if the email validation on backend is more strict.
	// If changeEmail returns an empty string, the email was changed successfully.
	// Oherwise the string is a user-friendly indication of the problem
	async function changeEmail(new_email:string) :Promise<string> {
		if( !is_loaded.value ) throw new Error("trying to change email while user is not even loaded.");
		const resp = await ax.patch('/api/user/email/', {email:new_email});
		if( resp.status === 200 ) {
			if( typeof resp.data !== 'string' ) throw new Error("expected a string response");
			return resp.data;
		}
		else if( resp.status === 204 ) {	// "No content" means "Perfect, No Notes"
			email.value = new_email;
			return '';
		}
		else throw new Error("got unexpected response status "+resp.status);
	}

	async function changePassword(old_pw:string, new_pw:string) :Promise<string> {
		if( !is_loaded.value ) throw new Error("trying to change password while user is not even loaded.");
		const resp = await ax.patch('/api/user/password/', {old:old_pw, new:new_pw});
		if( resp.status === 200 ) {
			if( typeof resp.data !== 'string' ) throw new Error("expected a string response");
			return resp.data;
		}
		else if( resp.status === 204 ) return '';	// "No content" means "Perfect, No Notes"
		else throw new Error("got unexpected response status "+resp.status);
	}

	return {
		fetch,
		load_state,
		is_loaded,
		logged_in,
		user_id,
		email,
		is_admin,
		setUnauthorized,
		changeEmail,
		changePassword
	}

});