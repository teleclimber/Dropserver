<script lang="ts" setup>
import { ref, Ref, reactive, computed, onMounted, onUnmounted, watch } from 'vue';

import { useAppspacesStore } from '@/stores/appspaces';
import { useAppsStore } from '@/stores/apps';
import { useAppspaceUsersStore } from '@/stores/appspace_users';

import { fetchAppspaceSummary } from '../models/usage';
import type {SandboxSums} from '../models/usage';
import { LiveLog } from '../models/log';

import { AppspaceStatus } from '../twine-services/appspace_status';

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

const props = defineProps<{
	appspace_id: number
}>();

const appspacesStore = useAppspacesStore();
appspacesStore.loadAppspace(props.appspace_id);
const appspace = computed( () => {
	const a = appspacesStore.getAppspace(props.appspace_id);
	if( a === undefined ) return;
	return a.value;
});

onMounted( () => {
	appspacesStore.loadAppspace(props.appspace_id);
});

const appsStore = useAppsStore();
appsStore.loadData();
const app = computed( () => {
	if( appsStore.is_loaded && appspace.value ) return appsStore.mustGetApp(appspace.value.app_id).value;
});
const app_version = computed( () => {
	if( !appspace.value || !app.value ) return;
	return app.value.versions.find( v => v.version === appspace.value?.app_version );
});

const status = reactive(new AppspaceStatus) as AppspaceStatus;
status.connectStatus(props.appspace_id);

const appspaceUsersStore = useAppspaceUsersStore();

watch( () => status.temp_paused, (p, old_p) => {
	// Reload appspace after a temp_paused period finishes.
	// This is a hack to get the app version of the appspace (and other data)
	// updated after a migration job finishes.
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

const data_schema_mismatch = computed( ()=> {
	return !!app_version.value && app_version.value.schema !== status.appspace_schema;
});

onUnmounted( async () => {
	status.disconnect();
});

</script>
<template>
	<ViewWrap>
		<template v-if="appspace !== undefined">
			<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
				<div class="px-4 py-5 sm:px-6 border-b border-gray-200 flex justify-between">
					<h3 class="text-lg leading-6 font-medium text-gray-900">Appspace</h3>
					<div class="flex items-stretch">
						<AppspaceStatusVisualizer :status="status" class="mr-4 flex items-center"></AppspaceStatusVisualizer>
						<button @click.stop.prevent="togglePause()" :disabled="pausing" class="btn btn-blue">
							{{ appspace.paused ? 'Unpause' : 'Pause'}}
						</button>
					</div>
				</div>
				<div class="my-5">
					<DataDef field="Appsace Address:">
						<a :href="enter_link" class="text-blue-700 underline hover:text-blue-500">{{display_link}}</a>
					</DataDef>

					<DataDef field="Owner DropID:">{{appspace.dropid}}</DataDef>

					<DataDef field="Created:">{{appspace.created_dt.toLocaleString()}}</DataDef>

					<DataDef field="Application:">{{app ? app.name : "loading..." }}</DataDef>

					<DataDef field="App Version:">
						{{app_version ? app_version.version : "..."}}
						<router-link v-if="appspace.upgrade_version" :to="{name: 'migrate-appspace', params:{appspace_id:appspace.appspace_id}, query:{to_version:appspace.upgrade_version}}" class="btn">
							<svg class="inline align-bottom w-6 h-6" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
								<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-8.707l-3-3a1 1 0 00-1.414 0l-3 3a1 1 0 001.414 1.414L9 9.414V13a1 1 0 102 0V9.414l1.293 1.293a1 1 0 001.414-1.414z" clip-rule="evenodd" />
							</svg>
							{{appspace.upgrade_version}} available
						</router-link>
						<router-link v-else :to="{name: 'migrate-appspace', params:{appspace_id:appspace.appspace_id}}" class="btn">change version</router-link>
					</DataDef>

					<DataDef field="Data Schema:">
						<div v-if="data_schema_mismatch" class="data-schema-grid grid gap-x-4">
							<p>App version {{ app_version?.version }}:</p>
							<span class="font-bold">{{ app_version?.schema }}</span>
							<p>Appspace Data:</p>
							<span class="flex items-center">
								<span class="font-bold">{{ status.appspace_schema }}</span>
								<router-link :to="{name: 'migrate-appspace', params:{appspace_id:appspace.appspace_id}, query:{migrate_only:'true'}}" class="ml-4 btn flex items-center">
									<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-5 h-5">
										<path fill-rule="evenodd" d="M2.24 6.8a.75.75 0 001.06-.04l1.95-2.1v8.59a.75.75 0 001.5 0V4.66l1.95 2.1a.75.75 0 101.1-1.02l-3.25-3.5a.75.75 0 00-1.1 0L2.2 5.74a.75.75 0 00.04 1.06zm8 6.4a.75.75 0 00-.04 1.06l3.25 3.5a.75.75 0 001.1 0l3.25-3.5a.75.75 0 10-1.1-1.02l-1.95 2.1V6.75a.75.75 0 00-1.5 0v8.59l-1.95-2.1a.75.75 0 00-1.06-.04z" clip-rule="evenodd" />
									</svg>

									migrate data
								</router-link>
							</span>
						</div>
						<template v-else>
							{{ status.appspace_schema }}
						</template>
					</DataDef>

					<MessageSad head="Data Schema Mismatch" v-if="data_schema_mismatch" class="my-4 md:rounded-xl md:mx-6">
						<p>The application expects the data saved in the appspace to have a schema version of {{ app_version?.schema }}.
						However the schema of the appspace is currently {{ status.appspace_schema }}.</p>
						<p>Hit the "Migrate" link to bring the appspace data to the correct schema for the application,
							or change the app version to match the data schema.</p>
					</MessageSad>
				</div>
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