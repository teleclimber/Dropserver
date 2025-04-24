import { ref, Ref, computed } from 'vue';
import { defineStore } from 'pinia';
import { ax, get } from '../controllers/userapi';
import { LoadState, User } from './types';
import { userFromRaw } from './helpers';

export const useAuthUserStore = defineStore('authenticated-user', () => {
	const load_state = ref(LoadState.NotLoaded);
	const is_loaded = computed( () => load_state.value === LoadState.Loaded );

	const logged_in = ref(true);	// assumed true until a request fails.

	const user :Ref<User> = ref({
		email: "",
		has_password: false,
		is_admin: false,
		tsnet_extra_name: '',
		tsnet_identifier:'',
		user_id:-1,
	});

	const user_id = ref(-1);

	async function fetch() {
		if( load_state.value === LoadState.NotLoaded ) {
			load_state.value = LoadState.Loading;
			const resp_data = await get('/user/');
			user_id.value = Number(resp_data.user_id);
			user.value = userFromRaw(resp_data);
			load_state.value = LoadState.Loaded;
		}
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
			user.value.email = new_email;//todo check thsi is reactive
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
		user_id, user,
		setUnauthorized,
		changeEmail,
		changePassword
	}

});