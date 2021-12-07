<template>
	<h3 class="text-gray-600">Dropserver App Dev<br>-app {{baseData.app_path}}<br>-appspace {{baseData.appspace_path}}</h3>

	<div class="grid grid-cols-2 my-8">
		<div class="border-l-4 border-gray-800">
			<h4 class="bg-gray-800 px-2 text-white inline-block">App:</h4>
			<h1 class="text-2xl px-2">{{baseData.name}} <span class="bg-gray-400 px-2 rounded-md">{{baseData.version}}</span></h1>
			<div class="px-2">
				Schema: <span>{{appspaceStatus.app_version_schema}}</span>
			</div>
		</div>
		<div class="border-l-4 border-gray-800">
			<h4 class="bg-gray-800 px-2 text-white inline-block">Appspace:</h4>
			<h1 class="text-2xl px-2">[baseData.appspace_name]</h1>
			<div class="px-2">
				Schema: <span>{{appspaceStatus.appspace_schema}}</span>
			</div>
		</div>
	</div>

	<AppspaceControl></AppspaceControl>

	<UserControl></UserControl>

	<SandboxLog title="App" :live_log="appLog"></SandboxLog>

	<SandboxLog title="Appspace" :live_log="appspaceLog"></SandboxLog>

	<MigrationJobs></MigrationJobs>

	<RouteHits></RouteHits>

	<AppRoutes></AppRoutes>

</template>

<script lang="ts">
import { defineComponent, reactive, watch } from 'vue';
import baseData from './models/base-data';
import appspaceStatus from './models/appspace-status';
import LiveLog from './models/appspace-log-data';

import AppspaceControl from './components/AppspaceControl.vue';
import UserControl from './components/UserControl.vue';
import SandboxLog from './components/AppspaceLog.vue';
import MigrationJobs from './components/MigrationJobs.vue';
import RouteHits from './components/RouteHits.vue';
import AppRoutes from './components/AppRoutes.vue';

export default defineComponent({
	name: 'DropserverAppDev',
	components: {
		AppspaceControl,
		UserControl,
		SandboxLog,
		MigrationJobs,
		RouteHits,
		AppRoutes,
	},
	setup(props, context) {
		const appLog = <LiveLog>reactive(new LiveLog);	// have to <LiveLog> to make TS happy.
		appLog.subscribeAppLog(11, "");	// send anything. The ds-dev backend always retunrs logs for the subject app.
		
		const appspaceLog = <LiveLog>reactive(new LiveLog);
		appspaceLog.subscribeAppspaceLog(15);	// 15 is designated hard-coded appspace id in ds-dev.
		return {
			baseData,
			appspaceStatus,
			appLog,
			appspaceLog,
		};
	}
});
</script>

