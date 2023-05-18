<template>
	<div>
		<div class="bg-gray-100">
			<a href="#" class="p-4 flex justify-between hover:bg-yellow-50" @click.stop.prevent="show_process_log = !show_process_log">
				<span v-if="!p_event.processing && p_event.errors.length" class="flex items-center text-red-700 font-bold">
					<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 mr-1" viewBox="0 0 20 20" fill="currentColor">
						<path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clip-rule="evenodd" />
					</svg>
					Error Processing App
				</span>
				<span v-else-if="!p_event.processing" class="flex items-center text-green-700">
					<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 mr-1" viewBox="0 0 20 20" fill="currentColor">
						<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
					</svg>
					App Processed Successfully
				</span>
				<span v-if="p_event.processing" class="flex items-center text-yellow-800">
					<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 mr-1 animate-spin" viewBox="0 0 20 20" fill="currentColor">
						<path fill-rule="evenodd" d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1zm.008 9.057a1 1 0 011.276.61A5.002 5.002 0 0014.001 13H11a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0v-2.101a7.002 7.002 0 01-11.601-2.566 1 1 0 01.61-1.276z" clip-rule="evenodd" />
					</svg>
					<span class="font-bold mr-1">Processing App:</span> {{p_event.step}}
				</span>
				<div class="flex">
					<span class="text-sm uppercase pr-2">{{ show_process_log ? 'collapse' : 'expand' }}</span>
					<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 transition" :class="{'rotate-180':show_process_log}" viewBox="0 0 20 20" fill="currentColor">
						<path fill-rule="evenodd" d="M15.707 4.293a1 1 0 010 1.414l-5 5a1 1 0 01-1.414 0l-5-5a1 1 0 011.414-1.414L10 8.586l4.293-4.293a1 1 0 011.414 0zm0 6a1 1 0 010 1.414l-5 5a1 1 0 01-1.414 0l-5-5a1 1 0 111.414-1.414L10 14.586l4.293-4.293a1 1 0 011.414 0z" clip-rule="evenodd" />
					</svg>
				</div>
			</a>
			<div v-if="show_process_log" class="p-4 pt-0">

				<div v-if="p_event.errors.length" class="border-l-4 border-red-700">
					<h4 class="bg-red-700 px-2 py-1 text-white inline-block">Errors:</h4>
					<ul class="list-disc  mb-4 py-4 bg-gray-200">
						<li class="ml-6" v-for="err, i in p_event.errors" :key="'err-'+i">{{err}}</li>
					</ul>
				</div>

				<div class="border-l-4 border-gray-800 flex flex-col ">
					<div>
						<h4 class="bg-gray-800 px-2 py-1 text-white inline-block">App Log:</h4>
						<span v-if="!appLog.log_open" class="ml-2 px-2 rounded-sm inline-block bg-yellow-700 text-white text-sm font-bold">Log Closed</span>
					</div>
					<div >
						<Log title="App" :live_log="appLog"></Log>
					</div>
				</div>
			</div>
		</div>

		<div v-if="!show_process_log" class="m-4">
			<h2 class="text-2xl my-2">Application Data:</h2>
			<div class="my-4">
				<p>App dir: {{baseData.app_path}}</p>
				<p>Name: {{appData.name}}</p>
				<p>Version: {{appData.version}}</p>
				<p>Schema: {{appData.schema}}</p>
				<p>Migrations: {{migrations_str}}</p>
			</div>
			<AppRoutes></AppRoutes>
			<!-- should also have registered migrations -->
		</div>

	</div>
</template>

<script lang="ts">
import { defineComponent, reactive, ref, computed, watch, onMounted } from 'vue';

import baseData from '../models/base-data';
import LiveLog from '../models/appspace-log-data';
import appData from '../models/app-data';

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

		const appLog = reactive(new LiveLog) as LiveLog;	// have to cast to LiveLog to make TS happy.
		appLog.subscribeAppLog(11, "");	// send anything. The ds-dev backend always retunrs logs for the subject app.
		
		const migrations_str = computed( () => {
			// Maybe sort the array of migrations, and try to craft a string that makes sense?
			// would like to warn on missing migrations.
			const m = appData.migrations;
			if( m.length === 0 ) return "none";
			return m.map( s => (s.direction === 'up' ? '🔺' : '🔻') + s.schema ).join(', ');
		});

		const p_event = computed( () => appData.last_processing_event );

		onMounted( () => {
			if( p_event.value.errors.length ) show_process_log.value = true;
		});
		watch( p_event, () => {
			if( p_event.value.errors.length ) show_process_log.value = true;
		});

		return {
			baseData,
			show_process_log, app_ok,
			appData,
			migrations_str,
			appLog,
			p_event
		}
	},
});
</script>