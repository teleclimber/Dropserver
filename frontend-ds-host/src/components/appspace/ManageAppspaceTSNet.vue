<script lang="ts" setup>
import { ref, Ref, shallowRef, ShallowRef, computed, nextTick } from 'vue';

import type { TSNetData, TSNetStatus, AppspaceUser, TSNetCreateConfig } from '@/stores/types';

import { useAppspacesStore } from '@/stores/appspaces';
import { AvatarState, PostAuth, useAppspaceUsersStore } from '@/stores/appspace_users';

import ManageTSNetNode from '../tsnet/ManageTSNetNode.vue';
import DataDef from '../ui/DataDef.vue';

const props = defineProps<{
	appspace_id: number,
	tsnet_data: TSNetData | undefined,
	tsnet_status: TSNetStatus
	suggested_name: string
}>();

const appspacesStore = useAppspacesStore();
const appspaceUsersStore = useAppspaceUsersStore();

async function saveTSNetConfig(config :TSNetCreateConfig) {
	await appspacesStore.createTSNetNode(props.appspace_id, config);
}

async function tsnetSetConnect(connect:boolean) {
	if( !props.tsnet_data ) return;
	await appspacesStore.connectTSNetNode(props.appspace_id, connect);
}

async function tsnetDeleteConfig() {
	await appspacesStore.deleteTSNetData(props.appspace_id); 
}

const tsnet_peer_users = computed( () => {
	const pu = appspacesStore.watchTSNetPeerUsers(props.appspace_id);
	if( pu === undefined ) return undefined;
	return pu.value;
});

const tsnet_peer_matched_users = computed( () => {
	const ret : Map<string,AppspaceUser> = new Map();
	tsnet_peer_users.value?.forEach( (pu) => {
		const u = appspaceUsersStore.findByAuth(props.appspace_id, 'tsnetid', pu.full_id);
		if( u ) ret.set(pu.id, u);
	});
	return ret;
});


const select_user_input :Ref<HTMLSelectElement|undefined> = ref();
const show_select_id = ref('');
const select_options :ShallowRef<AppspaceUser[]> = shallowRef([]);
const selected_proxy_id :Ref<string>= ref("");
function showSelect(peer_user_id:string) {
	if( show_select_id.value !== '' ) return;
	selected_proxy_id.value = "";
	select_options.value = [];
	const cur_user = tsnet_peer_matched_users.value.get(peer_user_id);
	if( cur_user === undefined ) {
		appspaceUsersStore.getUsers(props.appspace_id)?.forEach( u => {
			select_options.value.push(u);
		});
	}
	show_select_id.value = peer_user_id;
	nextTick( () => {
		// the select elem is in a v-for, so the ref might be an array.
		const sels = select_user_input.value;
		if( Array.isArray(sels) ) sels[0].focus();
		else if( sels?.focus ) sels.focus();
	});
}

function cancelSelect() {
	show_select_id.value = '';
}

async function saveSelect() {
	if( selected_proxy_id.value == "" ) return;
	const tsnet_peer = tsnet_peer_users.value?.find( pu => pu.id === show_select_id.value);
	if( !tsnet_peer ) return;

	const auth :PostAuth = {
		op: "add",
		type: "tsnetid",
		identifier: tsnet_peer.full_id,
		extra_name: tsnet_peer.login_name
	};

	if( selected_proxy_id.value === "new-user" ) {
		await appspaceUsersStore.addNewUser(props.appspace_id, {
			display_name: tsnet_peer.display_name,
			auths: [auth],
			avatar: AvatarState.Preserve,
			permissions: []
		}, null);
	}
	else {
		await appspaceUsersStore.updateUserAuth(props.appspace_id, selected_proxy_id.value, auth);
	}
	show_select_id.value = '';
}

</script>

<template>
	<ManageTSNetNode
		:for_appspace="true"
		:tsnet_data="tsnet_data" 
		:tsnet_status="tsnet_status" 
		:suggested_name="suggested_name"
		:num_peers="tsnet_peer_users?.length || 0"
		:num_matched_peers="tsnet_peer_matched_users.size"
		@create-node="saveTSNetConfig"
		@set-connect="tsnetSetConnect"
		@delete="tsnetDeleteConfig">

		<template v-slot:users>
			<ul class="border-gray-200 border-t my-4">
				<template v-for="u in tsnet_peer_users">
					<li v-if="show_select_id == u.id" class="border border-yellow-200 p-3 bg-yellow-100">
						<form @submit.prevent="saveSelect" @keyup.esc="cancelSelect">
							<DataDef field="Tailnet User:">
								<span class="font-bold">{{ u.display_name }}</span>
								({{ u.login_name }})
								<span class="bg-amber-500 text-amber-50 px-2 text-sm rounded whitespace-nowrap" v-if="u.sharee">shared out</span>
							</DataDef>
							<DataDef field="Appspace User:">
								<select v-model="selected_proxy_id" ref="select_user_input">
									<option value="">Select Appspace user</option>
									<option v-for="o in select_options" :value="o.proxy_id">{{ o.display_name }}</option>
									<option value="new-user">Create new user</option>
								</select>
							</DataDef>
							<div class="flex justify-between pt-2">
								<input type="button" class="btn" @click="cancelSelect" value="Cancel" />
								<input
									type="submit"
									class="btn-blue"
									:disabled="selected_proxy_id == ''"
									value="Save" />
							</div>
						</form>
					</li>
					<li v-else class="border-gray-200 border-b py-2 flex flex-col md:flex-row justify-between">
						<span>
							<span class="font-bold">{{ u.display_name }}</span>
							({{ u.login_name }})
							<span class="bg-amber-500 text-amber-50 px-2 text-sm rounded whitespace-nowrap" v-if="u.sharee">shared out</span>
							
						</span>
						<span v-if="tsnet_peer_matched_users.has(u.id)" class="justify-self-end flex items-baseline">
							<span class="italic mr-1">Appspace user:</span>
							"{{ tsnet_peer_matched_users.get(u.id)?.display_name }}"
							<router-link class="btn ml-2" :to="{name:'appspace-user', params:{appspace_id: props.appspace_id, proxy_id:tsnet_peer_matched_users.get(u.id)?.proxy_id}}">Edit</router-link>
							
							<!-- maybe link to manage user? -->
							<!-- maybe the user display name if different? -->
						</span>
						<button v-else-if="show_select_id == ''" class="justify-self-end btn ml-4" @click.stop.prevent="showSelect(u.id)">select</button>
					</li>
				</template>
			</ul>
		</template>
	</ManageTSNetNode>
</template>