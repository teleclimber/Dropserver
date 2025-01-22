import { reactive, ref, shallowRef, ShallowRef, computed } from 'vue';
import { defineStore } from 'pinia';
import { ax } from '../controllers/userapi';
import { LoadState, AppspaceUser, AppspaceUserAuth } from './types';

export enum AvatarState {
	Preserve = "preserve",
	Delete = "delete",
	Replace = "replace"
}
export type PostAppspaceUser = {
	auth_type: string,
	auth_id: string,
	display_name: string,
	avatar: AvatarState,
	permissions: string[]
}

function userAuthFromRaw(raw:any) :AppspaceUserAuth{
	return {
		type: raw.type+'',
		identifier: raw.identifier+'',
		created: new Date(raw.created),
		last_seen: raw.last_seen ? new Date(raw.last_seen) : undefined
	}
}

function userFromRaw(raw:any) :AppspaceUser {
	return {
		appspace_id: Number(raw.appspace_id),
		proxy_id: raw.proxy_id+'',
		auths: Array.isArray(raw.auths) ? raw.auths.map(userAuthFromRaw) : [],
		display_name: raw.display_name+'',
		avatar: raw.avatar+'',
		//permissions = raw.permissions;
		created_dt: new Date(raw.created_dt),
		last_seen: raw.last_seen ? new Date(raw.last_seen) : undefined
	};
}

export const useAppspaceUsersStore = defineStore('appspace-users', () => {
	const load_state :Map<number,LoadState> = reactive(new Map);

	const appspace_users : ShallowRef<Map<number,ShallowRef<Array<ShallowRef<AppspaceUser>>>>> = shallowRef(new Map());

	function isLoaded(appspace_id: number) {
		const l = load_state.get(appspace_id);
		return l === undefined ? false : l === LoadState.Loaded;
	}

	async function loadData(appspace_id: number) {
		const l = load_state.get(appspace_id);
		if( !l ) {	// || l === LoadState.NotLoaded ) {
			load_state.set(appspace_id, LoadState.Loading);
			const resp = await ax.get('/api/appspace/'+appspace_id+'/user');
			if( !Array.isArray(resp.data) ) throw new Error("expected response to be array");
			const users = resp.data.map( (raw:any) => shallowRef(userFromRaw(raw)));
			appspace_users.value.set(appspace_id, shallowRef(users));	// This is possibly wrong? In case of reloading, need to replave value of shallow ref.
			appspace_users.value = new Map(appspace_users.value);
			load_state.set(appspace_id, LoadState.Loaded);
		}
	}
	async function reloadData(appspace_id: number) {
		const l = load_state.get(appspace_id);
		if( l === LoadState.Loading ) return;	// its' already loading so don't reload
		load_state.delete(appspace_id);
		loadData(appspace_id);
	}

	function getUsers(appspace_id: number) {
		if( isLoaded(appspace_id) ) return appspace_users.value.get(appspace_id);
	}
	function mustGetUsers(appspace_id: number) {
		const users = getUsers(appspace_id);
		if( !users ) throw new Error("expected users to be loaded");
		return users;
	}

	function getUser(appspace_id: number, proxy_id:string) {
		const users = getUsers(appspace_id);
		if( users === undefined ) return;
		return users.value.find( u => u.value.proxy_id === proxy_id );
	}
	function mustGetUser(appspace_id: number, proxy_id:string) {
		const user = getUser(appspace_id, proxy_id);
		if( user === undefined ) throw new Error(`expected user to exist in appspace id ${appspace_id} proxy id: ${proxy_id}`);
		return user;
	}

	async function addNewUser(appspace_id:number, data :PostAppspaceUser, avatarData:Blob|null ) {
		const users = mustGetUsers(appspace_id);
		const resp = await ax.post('/api/appspace/'+appspace_id+'/user', getFormData(data, avatarData));
		const new_user = userFromRaw(resp.data);
		users.value.push(shallowRef(new_user));
		users.value = Array.from(users.value);
	}
	
	async function updateUserMeta(appspace_id:number, proxy_id:string, data:PostAppspaceUser, avatarData:Blob|null) {
		const user = mustGetUser(appspace_id, proxy_id);
		const resp = await ax.patch('/api/appspace/'+appspace_id+'/user/'+proxy_id, getFormData(data, avatarData));
		const new_user = userFromRaw(resp.data);
		user.value = new_user;
	}

	return { 
		isLoaded,
		loadData,
		reloadData,
		getUsers,
		mustGetUsers,
		getUser,
		mustGetUser,
		addNewUser,
		updateUserMeta
	};
});

export function  getAvatarUrl(u: AppspaceUser) {
	if( u.avatar ) return `/api/appspace/${u.appspace_id}/user/${u.proxy_id}/avatar/${u.avatar}`;
	else return "";
}


function getFormData(data:PostAppspaceUser, avatarData:Blob|null) :FormData {
	const formData = new FormData();
	if( avatarData !== null ) formData.append('avatar', avatarData);

	const json = JSON.stringify(data);
	const json_blob = new Blob([json], {
		type: 'application/json'
	});

	formData.append('metadata', json_blob);

	return formData;
}