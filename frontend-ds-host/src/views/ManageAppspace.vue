<script lang="ts" setup>
import { ref, Ref, reactive, computed, onMounted, onUnmounted, watch } from 'vue';

import type { AppspaceUser } from '@/stores/types';
import { useAppspacesStore } from '@/stores/appspaces';
import { useAppspaceUsersStore } from '@/stores/appspace_users';
import { useAppsStore } from '@/stores/apps';

import { fetchAppspaceSummary } from '../models/usage';
import type {SandboxSums} from '../models/usage';
import { LiveLog } from '../models/log';

import ViewWrap from '../components/ViewWrap.vue';
import BigLoader from '../components/ui/BigLoader.vue';
import AppspaceStatusVisualizer from '../components/AppspaceStatusVisualizer.vue';
import ManageAppspaceUsers from '../components/ManageAppspaceUsers.vue';
import ManageBackups from '../components/appspace/ManageBackups.vue';
import DeleteAppspace from '../components/appspace/DeleteAppspace.vue';
import DataDef from '../components/ui/DataDef.vue';
import UsageSummaryValue from '../components/UsageSummaryValue.vue';
import LogViewer from '../components/ui/LogViewer.vue';
import MessageSad from '@/components/ui/MessageSad.vue';
import MessageWarn from '@/components/ui/MessageWarn.vue';
import MinimalAppUrlData from '@/components/appspace/MinimalAppUrlData.vue';

const props = defineProps<{
	appspace_id: number
}>();

const appspacesStore = useAppspacesStore();
appspacesStore.loadAppspace(props.appspace_id);
const appspace = computed( () => {
	const a = appspacesStore.getAppspace(props.appspace_id);
	if( a === undefined ) return;
	return Object.assign({}, a.value);
});
const appsStore = useAppsStore();
watch( () => appspace.value?.app_id, () => {
	if( appspace.value ) appsStore.loadApp(appspace.value.app_id);
} );
const app = computed( () => {
	if( !appspace.value ) return;
	const a = appsStore.getApp(appspace.value.app_id);
	if( a ) return a.value;
});

onMounted( () => {
	appspacesStore.loadAppspace(props.appspace_id);
});
onUnmounted( () => {
	appspacesStore.unWatchTSNetPeerUsers(props.appspace_id);
});

const appspaceUsersStore = useAppspaceUsersStore();

watch( () => appspace.value?.status.temp_paused, (p, old_p) => {
	// Reload appspace after a temp_paused period finishes.
	// This is a hack to get the app version of the appspace (and other data)
	// updated after a migration job finishes.
	// -> this should be replaced with events from the backend that the store automatically responds to.
	// TODO in fact explore moving this to the store?
	if( old_p && !p ) {
		appspacesStore.loadAppspace(props.appspace_id);
		appspaceUsersStore.reloadData(props.appspace_id);
	}
});

const display_link = computed( () => {
	if( appspace.value ) {
		const a = appspace.value;
		const protocol = a.no_tls ? 'http' : 'https';
		return protocol+'://'+a.domain_name+a.port_string;
	}
	else return "https://...loading...";
});
const enter_link = computed( () => {
	if( appspace.value ) {
		return "/appspacelogin?appspace="+encodeURIComponent(appspace.value.domain_name);
	}
	else return "#";	// maybe return something that makes it clear that clicking is not going to work? or is that taken care of by display link
});

const appspaceLog = reactive(new LiveLog);// as LiveLog;
appspaceLog.initAppspaceLog(props.appspace_id);

fetchAppspaceSummary(props.appspace_id).then( (summary) => {
	usage.value = summary;
});

const usage :Ref<SandboxSums> = ref({tied_up_ms:0, cpu_usec: 0, memory_byte_sec: 0, io_bytes: 0, io_ops: 0});

const pausing = ref(false);
async function togglePause() {
	if( !appspace.value ) return;
	pausing.value = true;
	await appspacesStore.setPause(props.appspace_id, !appspace.value.paused);
	pausing.value = false;
}

const app_icon_error = ref(false);
const app_icon = computed( () => {
	if( app_icon_error.value || !appspace.value ) return "";
	return `/api/application/${appspace.value?.app_id}/version/${appspace.value?.app_version}/file/app-icon`;
});

const data_schema_mismatch = computed( ()=> {
	return appspace.value?.ver_data && appspace.value?.ver_data.schema !== appspace.value?.status.appspace_schema;
});

// tsnet stuff:
const show_edit_tsnet_config = ref(false);
const tsnet_control_url = ref('');
const tsnet_hostname = ref('');
const tsnet_auth_key = ref('');
const tsnet_connect = ref(true);
const tsnet_tags = ref('');

function showEditTSNetConfig() {
	tsnet_control_url.value = appspace.value?.tsnet_data?.control_url || '';
	tsnet_hostname.value = appspace.value?.tsnet_data?.hostname || appspace.value?.domain_name.split('.')[0] || '';
	tsnet_connect.value = appspace.value?.tsnet_data ? appspace.value.tsnet_data.connect : true;
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
	if( !appspace.value?.tsnet_data ) return;
	const td = appspace.value?.tsnet_data;
	await appspacesStore.setTSNetData(props.appspace_id, {
		control_url: td.control_url,
		hostname: td.hostname,
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
	<ViewWrap>
		<template v-if="appspace !== undefined">
			<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
				<div class="px-4 py-5 sm:px-6 border-b border-gray-200 flex justify-between">
					<h3 class="text-lg leading-6 font-medium text-gray-900">Appspace</h3>
					<div class="flex items-stretch">
						<AppspaceStatusVisualizer :status="appspace.status" class="mr-4 flex items-center"></AppspaceStatusVisualizer>
						<button @click.stop.prevent="togglePause()" :disabled="pausing" class="btn btn-blue">
							{{ appspace.paused ? 'Unpause' : 'Pause'}}
						</button>
					</div>
				</div>
				<div class="my-5">
					<DataDef field="Appspace Address:">
						<a :href="enter_link" class="text-blue-700 underline hover:text-blue-500">{{display_link}}</a>
					</DataDef>

					<DataDef field="Owner DropID:">{{appspace.dropid}}</DataDef>

					<DataDef field="Created:">{{appspace.created_dt.toLocaleString()}}</DataDef>

					<DataDef field="Application:">
						<span class="flex items-center">
							<span class="w-0">&nbsp;</span><!-- needed to make baseline allignment work -->
							<img v-if="app_icon" :src="app_icon" @error="app_icon_error = true" class="w-10 h-10" />
							<router-link :to="{name: 'manage-app', params:{id:appspace.app_id}}" class="font-medium text-lg text-blue-600 underline">
								{{appspace.ver_data?.name}}
							</router-link>
						</span>
					</DataDef>

					<DataDef field="App Version:">
						<span class="bg-gray-200 text-gray-600 px-1 rounded-md inline-block mr-1">{{appspace.app_version}}</span>
						<span v-if="appspace.upgrade_version">{{appspace.upgrade_version}} available </span>
						<router-link :to="{name: 'migrate-appspace', params:{appspace_id:appspace.appspace_id}}" class="btn">change version</router-link>
						<MinimalAppUrlData v-if="app?.url_data" :url_data="app?.url_data" :cur_ver="appspace.app_version"></MinimalAppUrlData>
					</DataDef>

					<DataDef field="Data Schema:">
						<div v-if="data_schema_mismatch" class="data-schema-grid grid gap-x-4">
							<p>App version {{ appspace?.ver_data?.schema }}:</p>
							<span class="font-bold">{{ appspace?.ver_data?.schema }}</span>
							<p>Appspace Data:</p>
							<span class="flex items-center">
								<span class="font-bold">{{ appspace.status.appspace_schema }}</span>
								<router-link :to="{name: 'migrate-appspace', params:{appspace_id:appspace.appspace_id}, query:{migrate_only:'true'}}" class="ml-4 btn flex items-center">
									<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-5 h-5">
										<path fill-rule="evenodd" d="M2.24 6.8a.75.75 0 001.06-.04l1.95-2.1v8.59a.75.75 0 001.5 0V4.66l1.95 2.1a.75.75 0 101.1-1.02l-3.25-3.5a.75.75 0 00-1.1 0L2.2 5.74a.75.75 0 00.04 1.06zm8 6.4a.75.75 0 00-.04 1.06l3.25 3.5a.75.75 0 001.1 0l3.25-3.5a.75.75 0 10-1.1-1.02l-1.95 2.1V6.75a.75.75 0 00-1.5 0v8.59l-1.95-2.1a.75.75 0 00-1.06-.04z" clip-rule="evenodd" />
									</svg>

									migrate data
								</router-link>
							</span>
						</div>
						<template v-else>
							{{ appspace.status.appspace_schema }}
						</template>
					</DataDef>

					<MessageSad head="Data Schema Mismatch" v-if="data_schema_mismatch" class="my-4 md:rounded-xl md:mx-6">
						<p>The application expects the data saved in the appspace to have a schema version of {{ appspace?.ver_data?.schema }}.
						However the schema of the appspace is currently {{ appspace.status.appspace_schema }}.</p>
						<p>Hit the "Migrate" link to bring the appspace data to the correct schema for the application,
							or change the app version to match the data schema.</p>
					</MessageSad>
				</div>
			</div>

			<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
				<div class="px-4 py-5 sm:px-6 border-b border-gray-200 flex justify-between">
					<h3 class="text-lg leading-6 font-medium text-gray-900">Tailscale</h3>
					<div>
						<span v-if="appspace.tsnet_status.transitory == 'connecting'" class="p-2 bg-gray-200 text-gray-700">
							Connecting...
						</span>
						<span v-else-if="appspace.tsnet_status.transitory == 'disconnecting'" class="p-2 bg-gray-200 text-gray-700">
							Disconnecting...
						</span>
						<button v-else-if="!appspace.tsnet_data" @click.stop.prevent="showEditTSNetConfig()" :disabled="show_edit_tsnet_config" class="btn btn-blue">
							Create Node
						</button>
						<span v-else-if="appspace.tsnet_status.state == '' || appspace.tsnet_status.state == 'Off'" class="p-2 bg-red-200 text-red-800">
							Off
						</span>
						<span v-else-if="appspace.tsnet_status.state == 'NeedsLogin'" class="p-2 bg-orange-100 text-orange-600">
							Needs Authentication
						</span>
						<span v-else-if="appspace.tsnet_status.tags.length===0" class="p-2 bg-orange-100 text-orange-600">
							No tag
						</span>
						<span v-else-if="appspace.tsnet_status.state == 'Running'" class="p-2 bg-green-200 text-green-800">
							Connected
						</span>
						<span v-else class="p-2 bg-orange-50">
							{{ appspace.tsnet_status.state }}
						</span>
					</div>
				</div>
				
				<div v-if="appspace.tsnet_status.browse_to_url !== '' && !appspace.tsnet_status.login_finished && appspace.tsnet_status.transitory != 'disconnecting'" class="px-4 sm:px-6 my-5">
					<p>The node needs to be authenticated. Click this link and follow the instructions:</p>
					<p><a class="text-blue-700 hover:text-blue-500 underline" :href="appspace.tsnet_status.browse_to_url" target="_blank">
						{{ appspace.tsnet_status.browse_to_url }}
					</a></p>
					<div class="flex justify-start mt-4">
						<button @click.stop.prevent="tsnetSetConnect(false)" class="btn btn-blue">
							Cancel
						</button>
					</div>
				</div>
				<div v-else-if="appspace.tsnet_status.transitory" class="px-4 sm:px-6 my-5">
					<p v-if="appspace.tsnet_status.transitory == 'connecting'">Connecting...</p>
					<p v-else-if="appspace.tsnet_status.transitory == 'disconnecting'">Disconnecting...</p>
				</div>
				<div v-else-if="appspace.tsnet_data" class="px-4 sm:px-6 my-5">
					<div v-if="appspace.tsnet_status.state == '' || appspace.tsnet_status.state == 'Off'">
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
						
					</div>
					<div v-else-if="appspace.tsnet_status.url">
						<MessageWarn v-if="appspace.tsnet_status.tags.length === 0 " head="No Tags">
							An appspace's Tailscale node must have a tag. 
							Open the admin panel for {{ appspace.tsnet_status.control_url }}
							to add an appropriate tag and disable node expiration (see docs).
						</MessageWarn>
						<DataDef field="Appspace Address:">
							<a class="text-blue-700 hover:text-blue-500 underline" :href="appspace.tsnet_status.url">{{appspace.tsnet_status.url}}</a>
						</DataDef>
						<DataDef field="Tailnet:">
							{{ appspace.tsnet_status.tailnet }}
							<span v-if="appspace.tsnet_data.control_url ==''">(Tailscale)</span>
							<span v-else>({{ appspace.tsnet_data.control_url }})</span>
						</DataDef>
						<DataDef v-if="appspace.tsnet_status.key_expiry" field="Key Expiry:">
							{{ appspace.tsnet_status.key_expiry.toLocaleDateString() }}
							<span class="text-orange-600 block italic">
								Recommended: disable Key Expiry for this node 
								in the admin panel for {{ appspace.tsnet_status.control_url }}.
							</span>
						</DataDef>
						<template v-if="appspace.tsnet_status.usable">
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
										<p>
											<span class="font-bold">{{ u.display_name }}</span>
											({{ u.login_name }})
										</p>
										<p class="italic text-gray-500" v-if="u.sharee">appspace node was shared with them</p>
										<p v-if="tsnet_peer_matched_users.has(u.id)">
											Appspace user "{{ tsnet_peer_matched_users.get(u.id)?.display_name }}"
											<router-link class="btn" :to="{name:'appspace-user', params:{appspace_id: props.appspace_id, proxy_id:tsnet_peer_matched_users.get(u.id)?.proxy_id}}">Edit</router-link>
										</p>
									</span>
									<span v-if="tsnet_peer_matched_users.has(u.id)" class="justify-self-end">
										matched
										<!-- maybe link to manage user? -->
										<!-- maybe the user display name if different? -->
									</span>
									<span v-else class="justify-self-end">
										add/match
									</span>
								</li>
							</ul>
						</template>
						<div class="flex justify-end">
							<button v-if="appspace.tsnet_data" @click.stop.prevent="tsnetSetConnect(!appspace.tsnet_data.connect)" class="btn btn-blue">
								{{ appspace.tsnet_data.connect ? 'Disconnect' : 'Connect'}}
							</button>
						</div>
					</div>
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
			<!-- tailscale temporary ebug output -->
			<div class="bg-slate-200 p-5">
				<DataDef field="State:">{{appspace.tsnet_status.state}}</DataDef>
				<DataDef field="Tailscale Address:">
					<a :href="appspace.tsnet_status.url" class="text-blue-700 underline hover:text-blue-500">
						{{appspace.tsnet_status.url}}
					</a>
					<p>IP4: {{ appspace.tsnet_status.ip4 }}</p>
					<p>IP6: {{ appspace.tsnet_status.ip6 }}</p>
				</DataDef>
				<DataDef field="https & dns:">
					<p>listening TLS: {{ appspace.tsnet_status.listening_tls }}</p>
					<p>TLS available: {{ appspace.tsnet_status.https_available }}</p>
					<p>Magic DNS: {{ appspace.tsnet_status.magic_dns_enabled }}</p>
				</DataDef>
				<DataDef field="tailnet:">{{appspace.tsnet_status.tailnet}} at {{  appspace.tsnet_status.control_url }}</DataDef>
				<DataDef field="Key expiry:">{{ appspace.tsnet_status.key_expiry?.toLocaleDateString() || "none" }}</DataDef>
				<DataDef field="name:">{{appspace.tsnet_status.name}}</DataDef>
				<DataDef field="tags:">{{ appspace.tsnet_status.tags?.join(", ") }}</DataDef>
				<DataDef field="err_message:">{{appspace.tsnet_status.err_message}}</DataDef>
				<DataDef field="browse_to_url:">
					<a :href="appspace.tsnet_status.browse_to_url" class="text-blue-700 underline hover:text-blue-500">
						{{appspace.tsnet_status.browse_to_url}}
					</a>
					<p>Login finished: {{appspace.tsnet_status.login_finished ? 'yes' : 'no'}}</p>
				</DataDef>
				<!-- warnings...-->
				<DataDef field="Warnings:">
					<p>{{ appspace.tsnet_status.warnings.length }} warnings.</p>
					<div v-for="warn in appspace.tsnet_status.warnings">
						<h3>{{ warn.title }}</h3>
						<p>{{ warn.text }}</p>
						<p>severity: {{  warn.severity }} impacts connectivity: {{ warn.impacts_connectivity ? 'yes' : 'no' }}</p>
					</div>
				</DataDef>
				<hr>
				<h4>Peer Users:</h4>
				<ul v-if="tsnet_peer_users">
					<li v-for="u in tsnet_peer_users">
						{{ u.display_name }} ({{ u.login_name }}) {{  u.control_url }}
						<span v-if="u.sharee">Shared in</span>
						<ul class="ml-2">
							<li v-for="d in u.devices">
								{{ d.name }} ({{  d.os }} {{ d.device_model }}) 
								<span v-if="d.online">online</span>
								<span v-else-if="d.last_seen">Last seen: {{  d.last_seen.toLocaleDateString() }}</span>
							</li>
						</ul>
					</li>
				</ul>
				<p v-else>No peer user data</p>
			</div>

			<ManageAppspaceUsers :appspace_id="appspace_id"></ManageAppspaceUsers>

			<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
				<div class="px-4 py-5 sm:px-6 border-b border-gray-200 flex items-baseline justify-between">
					<h3 class="text-lg leading-6 font-medium text-gray-900">Usage <span class="text-base text-gray-500">(last 30 days)</span></h3>
					<div class="flex items-baseline">
						<!-- usage drilldown... -->
					</div>
				</div>
				<div class="px-4 grid grid-cols-3">
					<UsageSummaryValue name="Tied Up time" :val="usage.tied_up_ms" unit="ms"></UsageSummaryValue>
					<UsageSummaryValue name="CPU time" :val="usage.cpu_usec" unit="usec"></UsageSummaryValue>
					<UsageSummaryValue name="Memory" :val="usage.memory_byte_sec" unit="byte-sec"></UsageSummaryValue>
					<UsageSummaryValue name="IO Bytes" :val="usage.io_bytes" unit="bytes"></UsageSummaryValue>
					<UsageSummaryValue name="IO Ops" :val="usage.io_ops" unit="ops"></UsageSummaryValue>
				</div>
			</div>

			<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
				<div class="px-4 py-5 sm:px-6 border-b border-gray-200 flex items-baseline justify-between">
					<h3 class="text-lg leading-6 font-medium text-gray-900">Logs</h3>
					<div class="flex items-baseline">
						<!-- log ctl.. -->
					</div>
				</div>
				<div class="px-2 ">
					<LogViewer :live_log="appspaceLog"></LogViewer>
				</div>
			</div>

			<ManageBackups :appspace_id="appspace.appspace_id"></ManageBackups>

			<DeleteAppspace :appspace="appspace"></DeleteAppspace>
			
		</template>
		<BigLoader v-else></BigLoader> 

	</ViewWrap>
</template>
<style scoped>
.data-schema-grid {
	grid-template-columns: auto 1fr;
}
</style>
