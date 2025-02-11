import { reactive, shallowReactive, ShallowReactive } from 'vue';
import { defineStore } from 'pinia';
import { ax } from '../controllers/userapi';
import { LoadState, AppspaceUser, AppspaceUserAuth } from './types';

export enum AvatarState {
	Preserve = "preserve",
	Delete = "delete",
	Replace = "replace"
}
export type PostAuth = {
	op: '' | 'add' | 'remove',
	type: string,
	identifier: string,
	extra_name: string
}
export type PostAppspaceUser = {
	display_name: string,
	avatar: AvatarState,
	permissions: string[],
	auths: PostAuth[]
}

function userAuthFromRaw(raw:any) :AppspaceUserAuth{
	return {
		type: raw.type+'',
		identifier: raw.identifier+'',
		extra_name: raw.extra_name+'',
		created: new Date(raw.created)
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
	};
}

export const useAppspaceUsersStore = defineStore('appspace-users', () => {
	const load_state :Map<number,LoadState> = reactive(new Map);

	const appspace_users : ShallowReactive<Map<number,ShallowReactive<Array<AppspaceUser>>>> = shallowReactive(new Map());

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
			const users = resp.data.map( (raw:any) => userFromRaw(raw));
			appspace_users.set(appspace_id, shallowReactive(users));
			// Note: since we are not replacing individual values the component has to 
			// fetch an individual user in a computed so that it gets updated on change!
			load_state.set(appspace_id, LoadState.Loaded);
		}
	}
	async function reloadData(appspace_id: number) {
		const l = load_state.get(appspace_id);
		if( l === LoadState.Loading ) return;	// it's already loading so don't reload
		load_state.delete(appspace_id);
		loadData(appspace_id);
	}

	function getUsers(appspace_id: number) {
		if( isLoaded(appspace_id) ) return appspace_users.get(appspace_id);
	}
	function mustGetUsers(appspace_id: number) {
		const users = getUsers(appspace_id);
		if( !users ) throw new Error("expected users to be loaded");
		return users;
	}

	function getUser(appspace_id: number, proxy_id:string) {
		const users = getUsers(appspace_id);
		if( users === undefined ) return;
		return users.find( u => u.proxy_id === proxy_id );
	}
	function mustGetUser(appspace_id: number, proxy_id:string) {
		const user = getUser(appspace_id, proxy_id);
		if( user === undefined ) throw new Error(`expected user to exist in appspace id ${appspace_id} proxy id: ${proxy_id}`);
		return user;
	}
	function findByAuth(appspace_id: number, auth_type:string, auth_id: string ) :AppspaceUser|undefined {
		const users = getUsers(appspace_id);
		return users?.find( u => {
			return u.auths.some( a => a.type === auth_type && a.identifier === auth_id );
		});
	}

	async function addNewUser(appspace_id:number, data :PostAppspaceUser, avatarData:Blob|null ) {
		const users = mustGetUsers(appspace_id);
		const resp = await ax.post('/api/appspace/'+appspace_id+'/user', getFormData(data, avatarData));
		const new_user = userFromRaw(resp.data);
		users.push(new_user);
	}
	
	async function updateUserMeta(appspace_id:number, proxy_id:string, data:PostAppspaceUser, avatarData:Blob|null) {
		const users = mustGetUsers(appspace_id);
		const userI = users.findIndex( u => u.proxy_id == proxy_id );
		if( userI == -1 ) throw new Error(`expected user to exist in appspace id ${appspace_id} proxy id: ${proxy_id}`);
		const resp = await ax.patch('/api/appspace/'+appspace_id+'/user/'+proxy_id, getFormData(data, avatarData));
		const new_user = userFromRaw(resp.data);
		users[userI] = new_user;
	}

	return { 
		isLoaded,
		loadData,
		reloadData,
		getUsers,
		mustGetUsers,
		getUser,
		mustGetUser,
		findByAuth,
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