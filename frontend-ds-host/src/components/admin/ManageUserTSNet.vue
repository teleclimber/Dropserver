<script lang="ts" setup>
import { computed, onMounted } from 'vue';

import type { UserForAdmin, TSNetCreateConfig } from '@/stores/types';

import { useInstanceSettingsStore } from '@/stores/admin/instance_settings';
import { useAdminTSNetStore } from '@/stores/admin/tsnet';


import ManageTSNetNode from '../tsnet/ManageTSNetNode.vue';

const instanceSettingsStore = useInstanceSettingsStore();
const adminTSNetStore = useAdminTSNetStore();

onMounted( () => {
	adminTSNetStore.loadTSNetStatus();
	adminTSNetStore.loadTSNetPeerUsers();
});

async function createTSNetNode(config :TSNetCreateConfig) {
	await instanceSettingsStore.setTSNetData(config);
}

async function tsnetSetConnect(connect:boolean) {
	await instanceSettingsStore.setTSNetConnect( connect );
}

async function tsnetDeleteConfig() {
	await instanceSettingsStore.deleteTSNetData(); 
}

const tsnet_peer_matched_users = computed( () => {
	const ret : Map<string,UserForAdmin> = new Map();
	adminTSNetStore.peer_users.forEach( (pu) => {
		// todo ...
		// const u = appspaceUsersStore.findByAuth(props.appspace_id, 'tsnetid', pu.full_id);
		// if( u ) ret.set(pu.id, u);
	});
	return ret;
});

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
				<li v-for="u in adminTSNetStore.peer_users" class="border-gray-200 border-b py-2 flex flex-col md:flex-row justify-between">
					<span>
						<span class="font-bold">{{ u.display_name }}</span>
						({{ u.login_name }})
						<span class="bg-amber-500 text-amber-50 px-2 text-sm rounded whitespace-nowrap" v-if="u.sharee">shared out</span>
						
					</span>
					<span v-if="tsnet_peer_matched_users.has(u.id)" class="justify-self-end flex items-baseline">
						<span class="italic mr-1">Appspace user:</span>
						"{{ tsnet_peer_matched_users.get(u.id)?.email }}"
						<!-- <router-link class="btn ml-2" :to="{name:'appspace-user', params:{appspace_id: props.appspace_id, proxy_id:tsnet_peer_matched_users.get(u.id)?.proxy_id}}">Edit</router-link> -->
						
						<!-- maybe link to manage user? -->
						<!-- maybe the user display name if different? -->
					</span>
					<span v-else class="justify-self-end">
						add/match
					</span>
				</li>
			</ul>
		</template>
	</ManageTSNetNode>
</template>