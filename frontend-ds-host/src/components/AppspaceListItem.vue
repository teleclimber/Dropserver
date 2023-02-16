<script setup lang="ts">
import { ref, watchEffect } from 'vue';
import { useAppsStore } from '@/stores/apps';

import type {Appspace} from '../models/appspaces';
import {AppVersionCollector } from '../models/app_versions';

const props = defineProps<{
	appspace: Appspace
}>();

if( !props.appspace.loaded ) console.error("appspace not loaded yet.");
const app_version = AppVersionCollector.get(props.appspace.app_id, props.appspace.app_version);

const protocol = props.appspace.no_tls ? 'http' : 'https';
const display_link = ref(protocol+'://'+props.appspace.domain_name+props.appspace.port_string)

const enter_link = ref("/appspacelogin?appspace="+encodeURIComponent(props.appspace.domain_name));

const appsStore = useAppsStore();
appsStore.fetchForOwner();
const app_name = ref('');

watchEffect( () => {
	if( appsStore.isLoaded ) {
		const app = appsStore.apps.get(props.appspace.app_id);
		if( app === undefined ) return;
		app_name.value = app.value.name;
	}
});

</script>

<template>
	<div class="bg-white overflow-hidden border-b border-b-gray-300 px-4 py-4 ">
		<h3 class="text-xl md:text-2xl font-medium text-gray-900">
			{{appspace.domain_name}}
		</h3>
		<p><a :href="enter_link" class="text-blue-700 underline hover:text-blue-500 overflow-hidden text-ellipsis">{{ display_link }}</a></p>
		<p class="mt-4">
			<router-link :to="{name: 'manage-app', params: {id:appspace.app_id}}" class="font-medium text-blue-800 hover:underline ">{{app_name}}</router-link> 
			<span class="bg-green-200 rounded-full px-2 ml-2 text-sm">{{app_version.version}}</span>
			<span v-if="appspace.upgrade" class="bg-blue-200 text-blue-600 px-2">
				Upgrade available: {{appspace.upgrade.version}}
			</span>
		</p>
		<p>Owner DropID: {{ appspace.dropid }}</p>
		<div class="pt-4 flex justify-end items-baseline">
			<span v-if="appspace.paused" class="mr-2 bg-pink-200 text-pink-800 px-2 text-xs font-bold uppercase">
				<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-4 h-4 inline-block">
					<path d="M5.75 3a.75.75 0 00-.75.75v12.5c0 .414.336.75.75.75h1.5a.75.75 0 00.75-.75V3.75A.75.75 0 007.25 3h-1.5zM12.75 3a.75.75 0 00-.75.75v12.5c0 .414.336.75.75.75h1.5a.75.75 0 00.75-.75V3.75a.75.75 0 00-.75-.75h-1.5z" />
				</svg>

				Paused
			</span>
			<router-link :to="{name: 'manage-appspace', params:{id:appspace.id}}" class="btn">Manage appspace</router-link>
		</div>
	</div>
</template>