<script setup lang="ts">
import { computed } from 'vue';

import appspaceStatus from '../models/appspace-status';

const status_string = computed( () => {
	if( appspaceStatus.problem ) return "problem";
	if( appspaceStatus.app_version_schema !== appspaceStatus.appspace_schema ) return "migrate";
	if( appspaceStatus.paused ) return "paused";
	if( appspaceStatus.temp_paused ) return "busy";
	return "ready";
});
</script>
<template>
	<div class="w-48 h-8 text-center">
		<div v-if="status_string === 'problem'" class="bg-red-300 h-full flex justify-center items-center">Problem</div>
		<div v-else-if="appspaceStatus.temp_paused" class="bg-yellow-400 h-full flex justify-center items-center">
			<svg class="w-6 h-6 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
				<path fill-rule="evenodd" d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1zm.008 9.057a1 1 0 011.276.61A5.002 5.002 0 0014.001 13H11a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0v-2.101a7.002 7.002 0 01-11.601-2.566 1 1 0 01.61-1.276z" clip-rule="evenodd" />
			</svg>
			{{appspaceStatus.temp_pause_reason}}
		</div>
		<div v-else-if="status_string === 'migrate'" class="bg-yellow-400 h-full flex justify-center items-center">
			<svg class="w-6 h-6 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
				<path d="M5 12a1 1 0 102 0V6.414l1.293 1.293a1 1 0 001.414-1.414l-3-3a1 1 0 00-1.414 0l-3 3a1 1 0 001.414 1.414L5 6.414V12zM15 8a1 1 0 10-2 0v5.586l-1.293-1.293a1 1 0 00-1.414 1.414l3 3a1 1 0 001.414 0l3-3a1 1 0 00-1.414-1.414L15 13.586V8z" />
			</svg>
			Migration required
		</div>
		<div v-else-if="status_string === 'ready'" class="bg-green-200 text-green-800 h-full flex justify-center items-center">
			<svg class="w-6 h-6 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
				<path d="M2 10.5a1.5 1.5 0 113 0v6a1.5 1.5 0 01-3 0v-6zM6 10.333v5.43a2 2 0 001.106 1.79l.05.025A4 4 0 008.943 18h5.416a2 2 0 001.962-1.608l1.2-6A2 2 0 0015.56 8H12V4a2 2 0 00-2-2 1 1 0 00-1 1v.667a4 4 0 01-.8 2.4L6.8 7.933a4 4 0 00-.8 2.4z" />
			</svg>
			Ready
		</div>
		<div v-else-if="status_string === 'paused'" class="bg-pink-200 text-pink-800 h-full flex justify-center items-center">
			<svg class="w-6 h-6 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
				<path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zM7 8a1 1 0 012 0v4a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v4a1 1 0 102 0V8a1 1 0 00-1-1z" clip-rule="evenodd" />
			</svg>
			Paused
		</div>
		<div v-else-if="status_string === 'busy'" class="bg-pink-200 text-pink-800 h-full flex justify-center items-center">
			<svg class="w-6 h-6 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
				<path fill-rule="evenodd" d="M13.477 14.89A6 6 0 015.11 6.524l8.367 8.368zm1.414-1.414L6.524 5.11a6 6 0 018.367 8.367zM18 10a8 8 0 11-16 0 8 8 0 0116 0z" clip-rule="evenodd" />
			</svg>
			Please wait...
		</div>
	</div>
</template>