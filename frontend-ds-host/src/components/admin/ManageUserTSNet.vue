<script lang="ts" setup>
import { ref, Ref, shallowRef, ShallowRef, computed, onMounted, nextTick } from 'vue';

import type { User, TSNetCreateConfig } from '@/stores/types';

import { useAuthUserStore } from '@/stores/auth_user';
import { useInstanceSettingsStore } from '@/stores/admin/instance_settings';
import { useAdminTSNetStore } from '@/stores/admin/tsnet';
import { useAdminAllUsersStore } from '@/stores/admin/all_users';

import ManageTSNetNode from '../tsnet/ManageTSNetNode.vue';
import DataDef from '../ui/DataDef.vue';

const authUserStore = useAuthUserStore();
const instanceSettingsStore = useInstanceSettingsStore();
const adminTSNetStore = useAdminTSNetStore();
const adminUsersStore = useAdminAllUsersStore();

onMounted( () => {
	adminTSNetStore.loadTSNetStatus();
	adminTSNetStore.loadTSNetPeerUsers();
	adminUsersStore.fetch();
});

async function createTSNetNode(config :TSNetCreateConfig) {
	await instanceSettingsStore.setTSNetData(config);
}

async function tsnetSetConnect(connect:boolean) {
	if( !connect && authUserStore.using_tsnet ) {
		if( !confirm('You are connected through your tailnet right now. '
			+'If you disconnect you lose access to Dropserver '
			+'until you can log back in using a different method.') ) {
				return;
		}
	}
	await instanceSettingsStore.setTSNetConnect( connect );
}

async function tsnetDeleteConfig() {
	await instanceSettingsStore.deleteTSNetData(); 
}

const tsnet_peer_matched_users = computed( () => {
	const ret : Map<string,User> = new Map();
	adminTSNetStore.peer_users.forEach( (pu) => {
		const matched_user = adminUsersStore.users.values().find( u => u.value.tsnet_identifier === pu.full_id );
		if( matched_user ) ret.set(pu.id, matched_user.value);
	});
	return ret;
});

const select_user_input :Ref<HTMLSelectElement|undefined> = ref();
const show_select_id = ref('');
const select_options :ShallowRef<User[]> = shallowRef([]);
const selected_user_id :Ref<number>= ref(-99);
function showSelect(peer_user_id:string) {
	if( show_select_id.value !== '' ) return;
	selected_user_id.value = -99;
	select_options.value = [];
	const cur_user = tsnet_peer_matched_users.value.get(peer_user_id);
	if( cur_user === undefined ) {
		adminUsersStore.users.values().forEach( u => {
			if( !u.value.tsnet_identifier ) {
				select_options.value.push(u.value);
			}
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
	if( selected_user_id.value < -1 ) return;
	if( selected_user_id.value === -1 ) {
		await adminUsersStore.createWithTSNet(show_select_id.value);
	}
	else {
		await adminUsersStore.updateTSNet(selected_user_id.value, show_select_id.value);
	}
	show_select_id.value = '';
}

</script>

<template>
	<ManageTSNetNode
		:for_appspace="false"
		:tsnet_data="instanceSettingsStore.tsnet_data" 
		:tsnet_status="adminTSNetStore.tsnet_status" 
		suggested_name="ds-host"
		:num_peers="adminTSNetStore.peer_users.length"
		:num_matched_peers="tsnet_peer_matched_users.size"
		@create-node="createTSNetNode"
		@set-connect="tsnetSetConnect"
		@delete="tsnetDeleteConfig">

		<template v-slot:users>
			<ul class="border-gray-200 border-t my-4">
				<template v-for="u in adminTSNetStore.peer_users">
					<li v-if="show_select_id == u.id" class="border border-yellow-200 p-3 bg-yellow-100">
						<form @submit.prevent="saveSelect" @keyup.esc="cancelSelect">
							<DataDef field="Tailnet User:">
								<span class="font-bold">{{ u.display_name }}</span>
								({{ u.login_name }})
								<span class="bg-amber-500 text-amber-50 px-2 text-sm rounded whitespace-nowrap" v-if="u.sharee">shared out</span>
							</DataDef>
							<DataDef field="Dropserver User:">
								<select v-model="selected_user_id" ref="select_user_input">
									<option value="-99">Select Dropserver user</option>
									<option v-for="o in select_options" :value="o.user_id">{{ o.user_id }} {{ o.email }}</option>
									<option :value="-1">Create new user</option>
								</select>
							</DataDef>
							<div class="flex justify-between pt-2">
								<input type="button" class="btn" @click="cancelSelect" value="Cancel" />
								<input
									type="submit"
									class="btn-blue"
									:disabled="selected_user_id < -1"
									value="Save" />
							</div>
						</form>
					</li>
					<li v-else class="border-gray-200 border-b py-2 ">
						<div class="flex flex-col md:flex-row justify-between">
							<span>
								<span class="font-bold">{{ u.display_name }}</span>
								({{ u.login_name }})
								<span class="bg-amber-500 text-amber-50 px-2 text-sm rounded whitespace-nowrap" v-if="u.sharee">shared out</span>
							</span>
							<span v-if="tsnet_peer_matched_users.has(u.id)" class="justify-self-end flex items-baseline">
								<span class="italic mr-1">User:</span>
								{{ tsnet_peer_matched_users.get(u.id)?.user_id }} ({{ tsnet_peer_matched_users.get(u.id)?.email }})
								<router-link class="btn ml-4" :to="{name:'admin-user', params:{user_id:tsnet_peer_matched_users.get(u.id)?.user_id}}">
									view
								</router-link>
							</span>
							<button v-else-if="show_select_id == ''" class="justify-self-end btn ml-4" @click.stop.prevent="showSelect(u.id)">select</button>
						</div>
					</li>
				</template>
			</ul>
		</template>
	</ManageTSNetNode>
</template>