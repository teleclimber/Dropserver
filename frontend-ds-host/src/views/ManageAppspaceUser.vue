<script lang="ts" setup>
import { ref, Ref, computed, onMounted, onUnmounted, watchEffect, reactive } from 'vue';
import { useRouter } from 'vue-router';

import type { TSNetPeerUser } from '@/stores/types';
import { useAppspacesStore } from '@/stores/appspaces';
import { useAppspaceUsersStore, AvatarState, getAvatarUrl, PostAuth } from '@/stores/appspace_users';

import ViewWrap from '../components/ViewWrap.vue';
import DataDef from '../components/ui/DataDef.vue';
import Avatar from '../components/ui/Avatar.vue';
import AuthListItem from '@/components/contacts/AuthListItem.vue';

const props = defineProps<{
	appspace_id: number,
	proxy_id?: string
}>();

const router = useRouter();

const appspacesStore = useAppspacesStore();
appspacesStore.loadAppspace(props.appspace_id);
const appspace = computed( () => {
	const a = appspacesStore.getAppspace(props.appspace_id);
	if( a === undefined ) return;
	return Object.assign({}, a.value);
});
onUnmounted( () => {
	appspacesStore.unWatchTSNetPeerUsers(props.appspace_id);
});

const appspaceUsersStore = useAppspaceUsersStore();
appspaceUsersStore.loadData(props.appspace_id);

const user = computed( () => {
	if( props.proxy_id === undefined || !appspaceUsersStore.isLoaded(props.appspace_id) ) return;
	return appspaceUsersStore.getUser(props.appspace_id, props.proxy_id );
});

const display_name_input :Ref<HTMLInputElement|undefined> = ref();
onMounted( () => {
	if( display_name_input.value === undefined ) return;
	display_name_input.value.focus();
});

const display_name = ref("");

const show_add_auth = ref(false);
const edit_auths:PostAuth[] = reactive([]);

watchEffect( () => {
	if( user.value === undefined ) {
		show_add_auth.value = true;
	}
	else {
		display_name.value = user.value.display_name;
		// avatar?
		edit_auths.splice(0);
		user.value.auths.forEach( auth => {
			edit_auths.push({
				type: auth.type,
				identifier: auth.identifier,
				extra_name: auth.extra_name,
				op:'',
			});
		});
		show_add_auth.value = false;
	}
});

let avatar_state = AvatarState.Preserve;
let avatar :Blob|null = null;

async function avatarChanged(ev:any) {
	if( ev ) {
		avatar = ev;
		avatar_state = AvatarState.Replace;
	}
	else {
		avatar = null;
		avatar_state = AvatarState.Delete;
	}
}

function handleAuthRemove(auth_type:string, auth_id: string, remove:boolean) {
	const i = findAuthI(auth_type, auth_id);
	if( i === -1 ) return;
	const auth = edit_auths[i];
	if( !auth ) return;
	if( remove ) {
		if( auth.op === 'add' ) edit_auths.splice(i, 1);
		else auth.op = 'remove';
	}
	else if( auth.op === 'remove' ) auth.op = '';
}
function findAuthI(auth_type:string, auth_id: string) {
	return edit_auths.findIndex( (a) => a.type === auth_type && a.identifier === auth_id);
}

const add_auth_type = ref('dropid');
const add_auth_dropid = ref('');
const add_auth_tsnetid = ref('');

const num_tsnet_peers = computed( () => {
	if( appspace.value?.tsnet_status.state !== 'Running') return;
	const peers = appspacesStore.watchTSNetPeerUsers(props.appspace_id);
	if( peers === undefined ) return;
	return peers.value.length;
});

// Here we want all peers that are unmatched
const tsnet_peer_users = computed( () => {
	if( !appspace.value?.tsnet_status.usable ) return;
	const peers = appspacesStore.watchTSNetPeerUsers(props.appspace_id);
	if( peers === undefined ) return;
	const ret : TSNetPeerUser[] = [];
	peers.value.forEach( (pu) => {
		if( appspaceUsersStore.findByAuth(props.appspace_id, 'tsnetid', pu.full_id) ) return;
		if( edit_auths.some(a => a.type === 'tsnetid' && a.identifier === pu.full_id) ) return;
		ret.push( pu );
	});
	return ret;
});

const invalid_add_auth = computed( () => {
	if( !show_add_auth.value ) return "";
	let identifier = '';
	if( add_auth_type.value === 'dropid' ) {
		// this is a sloppy validation of dropid. Must revisit on both frontend and backend.
		identifier = add_auth_dropid.value.trim().toLowerCase();
		if( identifier.length == 0 ) return ".";
		if( identifier.length < 3 ) return "too short";
		if( identifier.length > 200 ) return "too long";
		const pieces = identifier.split("/");
		if( !/^([a-zA-Z0-9]{1}[a-zA-Z0-9-]{0,62})(\.[a-zA-Z0-9]{1}[a-zA-Z0-9-]{0,62})*?(\.[a-zA-Z]{1}[a-zA-Z0-9]{0,62})\.?$/.test(pieces[0]) ) return "not a valid dropid";
	}
	else if( add_auth_type.value === 'tsnetid' ) {
		if( add_auth_tsnetid.value === '' ) return "can not be empty";
		identifier = add_auth_tsnetid.value;
	}
	else {
		throw new Error("invalid auth type "+add_auth_type.value)
	}

	// check not duplicate in appspcae:
	if( appspaceUsersStore.findByAuth(props.appspace_id, add_auth_type.value, identifier) ) return "already used";

	return "";
});

function addAuth() {
	if( invalid_add_auth.value ) return;
	if( add_auth_type.value === 'dropid' ) {
		edit_auths.push({
			op: 'add',
			type: 'dropid',
			identifier: add_auth_dropid.value,
			extra_name: ''
		});
		add_auth_dropid.value = "";
		show_add_auth.value = false;
	}
	else if( add_auth_type.value === 'tsnetid' ) {
		const peers = appspacesStore.watchTSNetPeerUsers(props.appspace_id);
		const peer = peers?.value.find( p => p.id === add_auth_tsnetid.value);
		if( !peer ) return;
		edit_auths.push({
			op: 'add',
			type: 'tsnetid',
			identifier: peer.full_id,
			extra_name: peer.login_name
		});
		add_auth_tsnetid.value = '';
		show_add_auth.value = false;
	}
}

const invalid = computed( () => {
	if( display_name.value.trim() === "" ) return "display name can not be empty";
	if( display_name.value.length > 29 ) return "display name is too long";
	if( invalid_add_auth.value !== "" ) return invalid_add_auth.value
	if( !props.proxy_id && !edit_auths.some(a => a.op != "remove") ) {
		return "a new user must have at least one login method";
	}
	return "";
});

async function save() {
	console.log("saving", user.value, invalid.value);
	if( invalid.value !== "" ) return;

	if( props.proxy_id ) {
		await appspaceUsersStore.updateUserMeta(props.appspace_id, props.proxy_id, {
			display_name: display_name.value,
			permissions: [],
			avatar: avatar_state,
			auths: edit_auths,
		}, avatar);
	}
	else {
		await appspaceUsersStore.addNewUser(props.appspace_id, {
			display_name: display_name.value,
			permissions: [],
			avatar: avatar_state,
			auths: edit_auths
		}, avatar);
	}
	router.push({name: 'manage-appspace', params:{appspace_id: props.appspace_id}});
}

function cancel() {
	router.back();
}

</script>
<template>
	<ViewWrap>
		<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<form @submit.prevent="save" @keyup.esc="cancel">
				<div class="px-4 py-5 sm:px-6 border-b border-gray-200 flex items-baseline justify-between">
					<h3 class="text-lg leading-6 font-medium text-gray-900">
						{{ proxy_id ? "Manage Appspace User: "+user?.display_name : "New Appspace User" }}
					</h3>
				</div>
				<div class="py-6">
					<DataDef field="Display Name:">
						<input type="text" v-model="display_name" ref="display_name_input" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
					</DataDef>
					<DataDef field="Avatar:">
						<Avatar :current="user ? getAvatarUrl(user) : ''" @changed="avatarChanged"></Avatar>
					</DataDef>
				</div>
				<div class="px-4 sm:px-6 flex justify-between">
					<h4 class="font-medium">Login methods:</h4>
					<a href="#" class="btn" v-if="!show_add_auth" @click.prevent="show_add_auth = true">add login method</a>
				</div>
				<AuthListItem v-for="a in edit_auths"
					:auth="a" 
					:controls="!show_add_auth"
					:removed="a.op==='remove'"
					@remove="handleAuthRemove(a.type, a.identifier, $event)"
					class="border-t py-2 px-4 sm:px-6"></AuthListItem>
				<div v-if="show_add_auth" class="px-4 sm:px-6 py-3 bg-gray-100 flex flex-col sm:flex-row justify-between ">
					<span>
						<span class="font-medium mr-2">Type:</span>
						<select v-model="add_auth_type" class="mr-4">
							<option value="dropid">DropID</option>
							<option value="tsnetid">Tailscale</option>
						</select>
						<span v-if="add_auth_type==='dropid'">
							<span class="font-medium mr-2">DropID:</span>
							<input type="text" v-model="add_auth_dropid">
							<span v-if="invalid_add_auth" class="text-orange-700 mx-2 whitespace-nowrap italic">{{ invalid_add_auth }}</span>
						</span>
						<template v-else-if="add_auth_type==='tsnetid'">
							<span v-if="tsnet_peer_users===undefined" class="italic">
								Tailscale node is not connected.
							</span>
							<span v-else-if="num_tsnet_peers===0" class="italic">
								There are no peers in this network.
							</span>
							<span v-else-if="tsnet_peer_users?.length===0" class="italic">
								All peers are already appspace users.
							</span>
							<span v-else-if="add_auth_type==='tsnetid'">
								<select v-model="add_auth_tsnetid">
									<option value="">Choose peer...</option>
									<option v-for="peer in tsnet_peer_users" :value="peer.id">
										{{ peer.display_name }} ({{ peer.login_name }})
									</option>
								</select>
							</span>
						</template>
					</span>
					<span>
						<button @click.stop.prevent="show_add_auth = false" class="btn mr-2">cancel</button>
						<button @click.stop.prevent="addAuth" class="btn disabled:text-gray-400" :disabled="!!invalid_add_auth">add</button>
					</span>
				</div>
				<div class="py-5 px-4 sm:px-6 border-t flex items-baseline justify-between">
					<input type="button" class="btn" @click="cancel" value="Cancel" />
					<input
						type="submit"
						class="btn-blue"
						:disabled="!!invalid || show_add_auth"
						value="Save" />
				</div>
			</form>
		</div>
	</ViewWrap>
</template>
