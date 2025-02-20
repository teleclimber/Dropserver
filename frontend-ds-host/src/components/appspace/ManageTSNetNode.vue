<script lang="ts" setup>
import { ref, Ref, reactive, computed, onMounted, onUnmounted, watch } from 'vue';

import type { AppspaceTSNetData, AppspaceTSNetStatus, AppspaceUser } from '@/stores/types';

import { useAppspacesStore } from '@/stores/appspaces';
import { useAppspaceUsersStore } from '@/stores/appspace_users';

import DataDef from '@/components/ui/DataDef.vue';
import MessageWarn from '@/components/ui/MessageWarn.vue';
import SmallMessage from '@/components/ui/SmallMessage.vue';

const props = defineProps<{
	appspace_id: number,
	tsnet_data: AppspaceTSNetData | undefined,
	tsnet_status: AppspaceTSNetStatus
	suggested_name: string
}>();

const appspacesStore = useAppspacesStore();
const appspaceUsersStore = useAppspaceUsersStore();

// tsnet stuff:
const show_edit_tsnet_config = ref(false);
const tsnet_control_url = ref('');
const tsnet_hostname = ref('');
const tsnet_auth_key = ref('');
const tsnet_connect = ref(true);
const tsnet_tags = ref('');

function showEditTSNetConfig() {
	tsnet_control_url.value = props.tsnet_data?.control_url || '';
	tsnet_hostname.value = props.tsnet_data?.hostname || props.suggested_name;
	tsnet_connect.value = props.tsnet_data ? props.tsnet_data.connect : true;
	show_edit_tsnet_config.value = true;
}

async function saveTSNetConfig() {
	await appspacesStore.setTSNetData(props.appspace_id, {
		control_url:tsnet_control_url.value,
		hostname: tsnet_hostname.value,
		auth_key: tsnet_auth_key.value,
		connect: tsnet_connect.value,
		tags: tagsFromString(tsnet_tags.value)
	});
	show_edit_tsnet_config.value = false;
}
function tagsFromString(str :string) :string[] {
	return str.split(/, /).map( s => s.trim() ).filter( s => s !== '' );
}

async function tsnetSetConnect(connect:boolean) {
	if( !props.tsnet_data ) return;
	await appspacesStore.setTSNetData(props.appspace_id, {
		control_url: props.tsnet_data.control_url,
		hostname: props.tsnet_data.hostname,
		connect: connect
	});
}

async function tsnetDeleteConfig() {
	if( confirm("Delete tailscale node configuration data?") ) {
		await appspacesStore.deleteTSNetData(props.appspace_id); 
	}
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

const show_tsnet_users = ref(false);

</script>

<template>
	<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
		<div class="px-4 py-5 sm:px-6 border-b border-gray-200 flex justify-between">
			<h3 class="text-lg leading-6 font-medium text-gray-900">Tailscale</h3>
			<div>
				<span v-if="tsnet_status.transitory == 'connecting'" class="p-2 bg-gray-200 text-gray-700">
					Connecting...
				</span>
				<span v-else-if="tsnet_status.transitory == 'disconnecting'" class="p-2 bg-gray-200 text-gray-700">
					Disconnecting...
				</span>
				<button v-else-if="!tsnet_data" @click.stop.prevent="showEditTSNetConfig()" :disabled="show_edit_tsnet_config" class="btn btn-blue">
					Create Node
				</button>
				<span v-else-if="tsnet_status.state == '' || tsnet_status.state == 'Off'" class="p-2 bg-red-200 text-red-800">
					Off
				</span>
				<span v-else-if="tsnet_status.state == 'NeedsLogin'" class="p-2 bg-orange-100 text-orange-600">
					Needs Authentication
				</span>
				<span v-else-if="tsnet_status.tags.length===0" class="p-2 bg-orange-100 text-orange-600">
					No tag
				</span>
				<span v-else-if="tsnet_status.state == 'Running'" class="p-2 bg-green-200 text-green-800">
					Connected
				</span>
				<span v-else class="p-2 bg-orange-50">
					{{ tsnet_status.state }}
				</span>
			</div>
		</div>

		<MessageWarn head="TSNet Error Message" v-if="tsnet_status.err_message">{{ tsnet_status.err_message }}</MessageWarn>
		<MessageWarn head="TSNet Warnings" v-if="tsnet_status.warnings.length">
			<ul v-for="warn in tsnet_status.warnings">
				<li>
					<h3>{{ warn.title }}</h3>
					<p>{{ warn.text }}</p>
					<p>severity: {{  warn.severity }} impacts connectivity: {{ warn.impacts_connectivity ? 'yes' : 'no' }}</p>
				</li>
			</ul>
		</MessageWarn>
		
		<div v-if="tsnet_status.browse_to_url !== '' && !tsnet_status.login_finished && tsnet_status.transitory != 'disconnecting'" class="px-4 sm:px-6 my-5">
			<p>The node needs to be authenticated. Click this link and follow the instructions:</p>
			<p><a class="text-blue-700 hover:text-blue-500 underline" :href="tsnet_status.browse_to_url" target="_blank">
				{{ tsnet_status.browse_to_url }}
			</a></p>
			<div class="flex justify-start mt-4">
				<button @click.stop.prevent="tsnetSetConnect(false)" class="btn btn-blue">
					Cancel
				</button>
			</div>
		</div>
		<div v-else-if="tsnet_status.transitory" class="px-4 sm:px-6 my-5">
			<p v-if="tsnet_status.transitory == 'connecting'">Connecting...</p>
			<p v-else-if="tsnet_status.transitory == 'disconnecting'">Disconnecting...</p>
		</div>
		<div v-else-if="tsnet_data" class="px-4 sm:px-6 my-5">
			<template v-if="tsnet_status.state == '' || tsnet_status.state == 'Off'">
				<div class="flex justify-between">
					<button @click.stop.prevent="tsnetDeleteConfig()" class="btn text-red-700">
						<svg xmlns="http://www.w3.org/2000/svg" class="inline align-bottom h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
							<path fill-rule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z" clip-rule="evenodd" />
						</svg>
						<span class="hidden sm:inline-block">delete configuration</span>
					</button>
					<button @click.stop.prevent="tsnetSetConnect(true)" class="btn btn-blue">
						Connect
					</button>
				</div>
				
			</template>
			<template v-else-if="tsnet_status.url">
				<MessageWarn v-if="tsnet_status.tags.length === 0 " head="No Tags">
					An appspace's Tailscale node must have a tag. 
					Open the admin panel for {{ tsnet_status.control_url }}
					to add an appropriate tag and disable node expiration (see docs).
				</MessageWarn>
				<DataDef field="Appspace Address:">
					<a class="text-blue-700 hover:text-blue-500 underline" :href="tsnet_status.url">{{tsnet_status.url}}</a>
					<SmallMessage mood="info" v-if="!tsnet_status.magic_dns_enabled" class="my-2">
						Enable MagicDNS in Tailscale admin panel to get a nicer address.
					</SmallMessage>
					<SmallMessage mood="info" v-if="!tsnet_status.https_available" class="my-2">
						Recommended: enable HTTPS in the Tailscale admin panel.
					</SmallMessage>
				</DataDef>
				<DataDef field="Tailnet:">
					{{ tsnet_status.tailnet }}
					<span v-if="tsnet_data.control_url ==''">(Tailscale)</span>
					<span v-else>({{ tsnet_data.control_url }})</span>
				</DataDef>
				<DataDef v-if="tsnet_status.key_expiry" field="Key Expiry:">
					{{ tsnet_status.key_expiry.toLocaleDateString() }}
					<span class="text-orange-600 block italic">
						Recommended: disable Key Expiry for this node 
						in the admin panel for {{ tsnet_status.control_url }}.
					</span>
				</DataDef>
				<template v-if="tsnet_status.usable">
					<DataDef field="Users:">
						{{ tsnet_peer_matched_users.size }} of {{ tsnet_peer_users?.length || 0 }}
						peers are users of this appspace
						<a href="#" @click.stop.prevent="show_tsnet_users = !show_tsnet_users" class=btn>
							{{ show_tsnet_users?"hide" : "show" }} peers
						</a>
					</DataDef>
					<ul v-if="show_tsnet_users" class="border-gray-200 border-t my-4">
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
				<div class="flex justify-end mt-4">
					<button v-if="tsnet_data" @click.stop.prevent="tsnetSetConnect(!tsnet_data.connect)" class="btn btn-blue">
						{{ tsnet_data.connect ? 'Disconnect' : 'Connect'}}
					</button>
				</div>
			</template>
		</div>
		<div v-else-if="show_edit_tsnet_config" class="px-4 sm:px-6 my-5">
			<p>Connect this appspace to a tailnet.
				This will create a node on the tailnet with its own address.
				You can also connect to alternative control servers such as a Headscale instance.</p>
			<form @submit.prevent="saveTSNetConfig" @keyup.esc="show_edit_tsnet_config = !show_edit_tsnet_config">
				<DataDef field="Hostname:">
					<input type="text" v-model="tsnet_hostname"
						class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
				</DataDef>
				<DataDef field="Control URL:">
					<input type="text" v-model="tsnet_control_url"
						class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
					<p>Leave blank to use Tailscale.com. Otherwise enter your Headscale (or other) URL.</p>
				</DataDef>
				<DataDef field="Auth Key:">
					<input type="text" v-model="tsnet_auth_key"
						class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
				</DataDef>
				<DataDef field="Tags:">
					<input type="text" v-model="tsnet_tags"
						class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
					<p>Node must have at least one tag.</p>
				</DataDef>
				<div class="flex justify-between">
					<input type="button" class="btn py-2" @click="show_edit_tsnet_config = !show_edit_tsnet_config" value="Cancel" />
					<input
						type="submit"
						class="btn-blue"
						value="Connect" />
				</div>
			</form>
		</div>
	</div>
</template>