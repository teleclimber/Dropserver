<template>
	<div>
		<div class="bg-gray-100 p-4">
			<div class="flex justify-between" @click.stop.prevent="show_process_log = !show_process_log">
				<span>App Processing Results:</span>
				<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
					<path fill-rule="evenodd" d="M15.707 4.293a1 1 0 010 1.414l-5 5a1 1 0 01-1.414 0l-5-5a1 1 0 011.414-1.414L10 8.586l4.293-4.293a1 1 0 011.414 0zm0 6a1 1 0 010 1.414l-5 5a1 1 0 01-1.414 0l-5-5a1 1 0 111.414-1.414L10 14.586l4.293-4.293a1 1 0 011.414 0z" clip-rule="evenodd" />
				</svg>
			</div>
			<div v-if="show_process_log">

				<!-- error message, processing steps coming from backend. -->

				<div class="border-l-4 border-gray-800 flex flex-col ">
					<div>
						<h4 class="bg-gray-800 px-2 py-1 text-white inline-block">App Log:</h4>
						<span v-if="!appLog.log_open" class="ml-2 px-2 rounded-sm inline-block bg-yellow-700 text-white text-sm font-bold">Log Closed</span>
					</div>
					<div class="h-64">
						<Log title="App" :live_log="appLog"></Log>
					</div>
				</div>
			</div>
		</div>

		<div v-if="app_ok" class="m-4">
			<h2 class="text-2xl my-2">Application Data:</h2>
			<div class="my-4">
				<p>Name: {{baseData.name}}</p>
				<p>Version: {{baseData.version}}</p>
				<p>Schema: {{baseData.schema}}</p>
				<p>Migrations: {{migrations_str}}</p>
			</div>
			<AppRoutes></AppRoutes>
			<!-- should also have registered migrations -->
		</div>

	</div>
</template>

<script lang="ts">
import { defineComponent, reactive, ref, computed } from 'vue';

import LiveLog from '../models/appspace-log-data';
import baseData from '../models/base-data';

import Log from './Log.vue';
import AppRoutes from './AppRoutes.vue';

export default defineComponent({
	components: {
		Log,
		AppRoutes
	},
	setup() {
		const show_process_log = ref(false);
		const app_ok = ref(true);	//temporary. 

		const appLog = <LiveLog>reactive(new LiveLog);	// have to <LiveLog> to make TS happy.
		appLog.subscribeAppLog(11, "");	// send anything. The ds-dev backend always retunrs logs for the subject app.
		
		const migrations_str = computed( () => {
			const m = baseData.possible_migrations;
			if( m.length === 0 ) return "n/a";
			return `${m[0]} to ${m[m.length -1]}`;	// whatever. Migrations should have more metadata (like description) and should be listed individually
		});
		return {
			show_process_log, app_ok,
			baseData,
			migrations_str,
			appLog
		}
	},
});
</script>