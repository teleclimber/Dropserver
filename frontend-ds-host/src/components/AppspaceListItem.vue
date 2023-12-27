<script setup lang="ts">
import { ref, Ref, computed } from 'vue';
import type { Appspace, App } from '@/stores/types';
import { useAppsStore } from '@/stores/apps';

import MinimalAppUrlData from './appspace/MinimalAppUrlData.vue';
import { RouterLink } from 'vue-router';

const props = defineProps<{
	appspace: Appspace
}>();

const appsStore = useAppsStore();
const app :Ref<App|undefined> = computed( () => {
	const a = appsStore.getApp(props.appspace.app_id);
	if( a !== undefined ) return a.value;
	return undefined;
});

const enter_link = ref("/appspacelogin?appspace="+encodeURIComponent(props.appspace.domain_name));

const app_icon_error = ref(false);
const app_icon = computed( () => {
	if( app_icon_error.value || !props.appspace ) return "";
	return `/api/application/${props.appspace.app_id}/version/${props.appspace.app_version}/file/app-icon`;
});

</script>

<template>
	<div class="bg-white overflow-hidden border-b border-l-4 border-b-gray-300 px-4 py-4 mb-4" :style="'border-left-color:'+(appspace?.ver_data?.color || 'rgb(135, 151, 164)')">
		<a :href="enter_link" class="block text-xl md:text-2xl mb-2 font-medium text-gray-900  hover:text-blue-500 overflow-hidden text-ellipsis ">
			{{appspace.domain_name}}
			<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-5 h-5 inline-block">
				<path fill-rule="evenodd" d="M3 4.25A2.25 2.25 0 015.25 2h5.5A2.25 2.25 0 0113 4.25v2a.75.75 0 01-1.5 0v-2a.75.75 0 00-.75-.75h-5.5a.75.75 0 00-.75.75v11.5c0 .414.336.75.75.75h5.5a.75.75 0 00.75-.75v-2a.75.75 0 011.5 0v2A2.25 2.25 0 0110.75 18h-5.5A2.25 2.25 0 013 15.75V4.25z" clip-rule="evenodd" />
				<path fill-rule="evenodd" d="M6 10a.75.75 0 01.75-.75h9.546l-1.048-.943a.75.75 0 111.004-1.114l2.5 2.25a.75.75 0 010 1.114l-2.5 2.25a.75.75 0 11-1.004-1.114l1.048-.943H6.75A.75.75 0 016 10z" clip-rule="evenodd" />
			</svg>

		</a>
		
		<p class="mt-3 flex items-center">
			<div class="w-10 h-10 flex items-center justify-center">
				<div class="w-7 h-7 rounded-full bg-gray-300  flex justify-center content-center text-gray-400">
					<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-6 h-6 self-end">
						<path stroke-linecap="round" stroke-linejoin="round" d="M15.75 6a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0zM4.501 20.118a7.5 7.5 0 0114.998 0A17.933 17.933 0 0112 21.75c-2.676 0-5.216-.584-7.499-1.632z" />
					</svg>
				</div>
			</div>
			<span class="pl-1">{{ appspace.dropid  }}</span>
			<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-5 h-5 text-yellow-500">
				<path fill-rule="evenodd" d="M8 7a5 5 0 113.61 4.804l-1.903 1.903A1 1 0 019 14H8v1a1 1 0 01-1 1H6v1a1 1 0 01-1 1H3a1 1 0 01-1-1v-2a1 1 0 01.293-.707L8.196 8.39A5.002 5.002 0 018 7zm5-3a.75.75 0 000 1.5A1.5 1.5 0 0114.5 7 .75.75 0 0016 7a3 3 0 00-3-3z" clip-rule="evenodd" />
			</svg>
		</p>

		<div class=" flex">
			<div class="my-2 __border-b __border-l-2 pr-2 flex items-center" :style="'border-color:'+(appspace?.ver_data?.color || 'rgb(135, 151, 164)')" >
				<img v-if="app_icon" :src="app_icon" @error="app_icon_error = true" class="w-10 h-10 self-start" />
				<div v-else class="w-10 h-10 text-gray-300 __border flex justify-center items-center">
					<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-12 h-12">
						<path stroke-linecap="round" stroke-linejoin="round" d="M2.25 15a4.5 4.5 0 004.5 4.5H18a3.75 3.75 0 001.332-7.257 3 3 0 00-3.758-3.848 5.25 5.25 0 00-10.233 2.33A4.502 4.502 0 002.25 15z" />
					</svg>
				</div>
				<div class="flex flex-col md:flex-row items-baseline">
					<p class="ml-1">
						<router-link :to="{name: 'manage-app', params:{id:appspace.app_id}}" class="font-medium text-lg text-blue-600 underline">
							{{appspace.ver_data?.name}}
						</router-link>
						<span class="bg-gray-200 text-gray-600 px-1 rounded-md ml-1">{{appspace.app_version}}</span>
					</p>
					<MinimalAppUrlData v-if="app?.url_data" :cur_ver="appspace.app_version" :url_data="app.url_data" class="self-baseline"></MinimalAppUrlData>
					<span v-else-if="appspace.upgrade_version" class="ml-1">
						Upgrade available: <span class="bg-gray-200 text-gray-600 px-1 rounded-md">{{appspace.upgrade_version}}</span>
					</span>
				</div>
				
			</div>
		</div>
		<div class="pt-4 flex justify-end items-baseline">
			<span v-if="appspace.paused" class="mr-2 bg-pink-200 text-pink-800 px-2 text-xs font-bold uppercase">
				<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-4 h-4 inline-block">
					<path d="M5.75 3a.75.75 0 00-.75.75v12.5c0 .414.336.75.75.75h1.5a.75.75 0 00.75-.75V3.75A.75.75 0 007.25 3h-1.5zM12.75 3a.75.75 0 00-.75.75v12.5c0 .414.336.75.75.75h1.5a.75.75 0 00.75-.75V3.75a.75.75 0 00-.75-.75h-1.5z" />
				</svg>

				Paused
			</span>
			<router-link :to="{name: 'manage-appspace', params:{appspace_id:appspace.appspace_id}}" class="btn">Manage appspace</router-link>
		</div>
	</div>
</template>