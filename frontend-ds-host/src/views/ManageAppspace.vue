<script lang="ts" setup>
import { ref, Ref, reactive, computed, onMounted, onUnmounted, watch } from 'vue';

import { useAppspacesStore } from '@/stores/appspaces';
import { useAppsStore } from '@/stores/apps';

import { fetchAppspaceSummary } from '../models/usage';
import type {SandboxSums} from '../models/usage';
import { LiveLog } from '../models/log';
import {setTitle} from '../controllers/nav';

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

const props = defineProps<{
	appspace_id: number
}>();

const appspacesStore = useAppspacesStore();
appspacesStore.loadData();
const appspace = computed( () => {
	if( appspacesStore.is_loaded ) return appspacesStore.mustGetAppspace(props.appspace_id).value;
});
if( appspace.value ) setTitle(appspace.value.domain_name);
else watch( appspace, () => {
	if( appspace.value ) setTitle(appspace.value.domain_name);
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

onUnmounted( async () => {
	status.disconnect();
	setTitle("");
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
				<div class="px-4 py-5 sm:px-6">
					<DataDef field="Appsace Address:">
						<a :href="enter_link" class="text-blue-700 underline hover:text-blue-500">{{display_link}}</a>
					</DataDef>

					<DataDef field="Owner DropID:">{{appspace.dropid}}</DataDef>

					<DataDef field="Created:">{{appspace.created_dt.toLocaleString()}}</DataDef>

					<DataDef field="Application:">{{app ? app.name : "loading..." }}</DataDef>

					<DataDef field="App Version:">
						{{app_version ? app_version.version : "..."}}
						<router-link v-if="appspace.upgrade_version" :to="{name: 'migrate-appspace', params:{id:appspace.appspace_id}, query:{to_version:appspace.upgrade_version}}" class="btn">
							<svg class="inline align-bottom w-6 h-6" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
								<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-8.707l-3-3a1 1 0 00-1.414 0l-3 3a1 1 0 001.414 1.414L9 9.414V13a1 1 0 102 0V9.414l1.293 1.293a1 1 0 001.414-1.414z" clip-rule="evenodd" />
							</svg>
							{{appspace.upgrade_version}} available
						</router-link>
						<router-link v-else :to="{name: 'migrate-appspace', params:{id:appspace.appspace_id}}" class="btn">Show other versions</router-link>
					</DataDef>

					<DataDef field="Data Schema:">
						{{app_version ? app_version.schema : "..."}}
						<router-link v-if="app_version && app_version.schema !== status.appspace_schema" :to="{name: 'migrate-appspace', params:{id:appspace.appspace_id}, query:{to_version:appspace.app_version}}" class="btn">
							<svg class="inline align-bottom w-6 h-6" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
								<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-8.707l-3-3a1 1 0 00-1.414 0l-3 3a1 1 0 001.414 1.414L9 9.414V13a1 1 0 102 0V9.414l1.293 1.293a1 1 0 001.414-1.414z" clip-rule="evenodd" />
							</svg>
							migrate!
						</router-link>
					</DataDef>

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
