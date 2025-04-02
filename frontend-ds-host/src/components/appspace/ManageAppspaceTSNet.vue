<script lang="ts" setup>
import { computed } from 'vue';

import type { TSNetData, TSNetStatus, AppspaceUser, TSNetCreateConfig } from '@/stores/types';

import { useAppspacesStore } from '@/stores/appspaces';
import { useAppspaceUsersStore } from '@/stores/appspace_users';

import ManageTSNetNode from '../tsnet/ManageTSNetNode.vue';

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
				<li v-for="u in tsnet_peer_users" class="border-gray-200 border-b py-2 flex flex-col md:flex-row justify-between">
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
					<span v-else class="justify-self-end">
						add/match
					</span>
				</li>
			</ul>
		</template>
	</ManageTSNetNode>
</template>