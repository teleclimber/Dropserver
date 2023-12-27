<script setup lang="ts">
import { reactive, ref, computed, watch, onMounted } from 'vue';

import baseData from '../models/base-data';
import LiveLog from '../models/appspace-log-data';
import appData from '../models/app-data';
import type { Warning } from '../models/app-data'

import Log from './Log.vue';
import AppRoutes from './AppRoutes.vue';
import MigrationsGrid from './MigrationsGrid.vue';
import DataDef from './ui/DataDef.vue';
import SmallMessage from './ui/SmallMessage.vue';

const show_process_log = ref(false);

const appLog = reactive(new LiveLog) as LiveLog;	// have to cast to LiveLog to make TS happy.
appLog.subscribeAppLog(11, "");	// send anything. The ds-dev backend always retunrs logs for the subject app.

const p_event = computed( () => appData.last_processing_event );
const app_icon = ref("app-icon?"+Date.now());

const accent_color = computed( () => {
	return appData.manifest?.accent_color;
});

onMounted( () => {
	if( p_event.value.errors.length ) show_process_log.value = true;
});
watch( p_event, () => {
	if( p_event.value.errors.length ) show_process_log.value = true;
	app_icon.value = "app-icon?"+Date.now();
});

const warnings = computed( () => {
	const ret :Record<string,Warning[]> = {};
	p_event.value.warnings.forEach(w => {
		const f = w.field;
		if( !ret[f] ) ret[f] = [];
		ret[f].push(w);
	});
	return ret;
});
const bad_values = computed( () => {
	const ret :Record<string,string> = {};
	p_event.value.warnings.forEach(w => {
		if( w.bad_value ) ret[w.field] = w.bad_value;
	});
	return ret;
});

const none_classes = ['italic', 'text-gray-500'];

</script>
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
				<span v-else-if="!p_event.processing && Object.keys(p_event.warnings).length" class="flex items-center text-orange-500">
					<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" class="w-5 h-5 mr-1">
							<path fill-rule="evenodd" d="M9.401 3.003c1.155-2 4.043-2 5.197 0l7.355 12.748c1.154 2-.29 4.5-2.599 4.5H4.645c-2.309 0-3.752-2.5-2.598-4.5L9.4 3.003zM12 8.25a.75.75 0 01.75.75v3.75a.75.75 0 01-1.5 0V9a.75.75 0 01.75-.75zm0 8.25a.75.75 0 100-1.5.75.75 0 000 1.5z" clip-rule="evenodd" />
						</svg>
					App Processed with Warnings
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
			<h2 class="text-2xl mt-2">Application Data:</h2>
			<p>App dir: {{baseData.app_path}}</p>
			<div class="my-4 ">
				<DataDef field="Name:">
					“{{appData.name}}”
					<SmallMessage mood="warn" v-for="w in warnings['name']">{{ w.message }}</SmallMessage>
				</DataDef>
				<DataDef field="Version:">
					{{ appData.version || bad_values['name'] }}
					<SmallMessage mood="warn" v-for="w in warnings['version']">{{ w.message }}</SmallMessage>
				</DataDef>
				<DataDef field="Schema:">
					{{ appData.schema }}
					<SmallMessage mood="warn" v-for="w in warnings['schema']">{{ w.message }}</SmallMessage>
				</DataDef>
				<DataDef field="Entrypoint:">
					{{ appData.entrypoint || bad_values['entrypoint'] }}
					<SmallMessage mood="warn" v-for="w in warnings['entrypoint']">{{ w.message }}</SmallMessage>
				</DataDef>
				<DataDef field="Migrations:">
					<MigrationsGrid :migrations="appData.migrations"></MigrationsGrid>
					<SmallMessage mood="warn" v-for="w in warnings['migrations']">{{ w.message }}</SmallMessage>
				</DataDef>
				<DataDef field="App Icon:">
					{{ appData.manifest?.icon|| "(none)" }}
					<img v-if="appData.manifest?.icon" :src="app_icon" class="border border-gray-300 h-20 w-20"/>
					<SmallMessage mood="warn" v-for="w in warnings['icon']">{{ w.message }}</SmallMessage>
				</DataDef>
				<DataDef field="Accent Color:">
					<span v-if="accent_color" class="rounded inline-block w-20 leading-3" :style="'background-color:'+accent_color">&nbsp;</span>
					<span v-else-if="bad_values['accent-color']">{{ bad_values['accent-color'] }}</span>
					<span v-else :class="none_classes">(none)</span>
					<SmallMessage mood="warn" v-for="w in warnings['accent-color']">{{ w.message }}</SmallMessage>
				</DataDef>
				<DataDef field="Short Description:">
					“{{ appData.manifest?.short_description }}”
					<SmallMessage mood="warn" v-for="w in warnings['short-description']">{{ w.message }}</SmallMessage>
				</DataDef>
				<DataDef field="Authors:">
					<div v-for="a in appData.manifest?.authors">
						{{ a.name }}
						&lt;<a class="text-blue-600 underline" :href="'mailto:'+a.email">{{ a.email }}</a>&gt;
						<a class="text-blue-600 underline" :href="a.url">{{ a.url }}</a>
					</div>
					<div v-if="!appData.manifest?.authors?.length" :class="none_classes">(none)</div>
					<SmallMessage mood="warn" v-for="w in warnings['authors']">{{ w.message }}</SmallMessage>
				</DataDef>
				<DataDef field="Website:">
					<a v-if="appData.manifest?.website" :href="appData.manifest.website" class="text-blue-600 underline">
						{{ appData.manifest.website }}
					</a>
					<span v-else-if="bad_values['website']">{{ bad_values['website'] }}</span>
					<span v-else :class="none_classes">(none)</span>
					<SmallMessage mood="warn" v-for="w in warnings['website']">{{ w.message }}</SmallMessage>
				</DataDef>
				<DataDef field="Code Repo:">
					<a v-if="appData.manifest?.code" :href="appData.manifest.code" class="text-blue-600 underline">
						{{ appData.manifest.code }}
					</a>
					<span v-else-if="bad_values['code']">{{ bad_values['code'] }}</span>
					<span v-else :class="none_classes">(none)</span>
					<SmallMessage mood="warn" v-for="w in warnings['code']">{{ w.message }}</SmallMessage>
				</DataDef>
				<DataDef field="Funding:">
					<a v-if="appData.manifest?.funding" :href="appData.manifest.funding" class="text-blue-600 underline">
						{{ appData.manifest.funding }}
					</a>
					<span v-else-if="bad_values['funding']">{{ bad_values['funding'] }}</span>
					<span v-else :class="none_classes">(none)</span>
					<SmallMessage mood="warn" v-for="w in warnings['funding']">{{ w.message }}</SmallMessage>
				</DataDef>
				<DataDef field="License:">
					<span v-if="appData.manifest?.license">
						{{ appData.manifest.license }}
					</span>
					<span v-else :class="none_classes">(none specified)</span>
					<span v-if="appData.manifest?.license_file" class="pl-2">
						File: {{ appData.manifest.license_file || bad_values['license-file'] }}
					</span>
					<span v-else class="pl-2" :class="none_classes">(no license file specified)</span>
					<SmallMessage mood="warn" v-for="w in warnings['license']">{{ w.message }}</SmallMessage>
					<SmallMessage mood="warn" v-for="w in warnings['license-file']">{{ w.message }}</SmallMessage>
				</DataDef>
				<DataDef field="Changelog:">
					<span v-if="appData.manifest?.changelog" class="pr-2">
						File: {{ appData.manifest.changelog || bad_values['changelog'] }}
					</span>
					<span v-else class="pr-2" :class="none_classes">(none specified)</span>
					<SmallMessage mood="warn" v-for="w in warnings['changelog']">{{ w.message }}</SmallMessage>
				</DataDef>
			</div>
			<div class="border-l-4 border-gray-800  my-8">
				<h4 class="bg-gray-800 px-2 py-1 text-white inline-block">Changelog:</h4>
				<div class="bg-gray-100 p-2 max-h-48 overflow-y-scroll">
					<pre v-if="appData.changelog_text" class="text-sm whitespace-pre-wrap">{{ appData.changelog_text }}</pre>
					<span v-else class="italic text-gray-600">No changelog</span>
				</div>
			</div>
			<AppRoutes></AppRoutes>
		</div>

	</div>
</template>
