<script lang="ts" setup>
import { computed, ref } from 'vue';

import type { App } from '@/stores/types';
import { useAppspacesStore } from '@/stores/appspaces';

import AppLicense from './app/AppLicense.vue';
import AppAuthorsSummary from './app/AppAuthorsSummary.vue';
import AppLinksCompact from './app/AppLinksCompact.vue';
import AppUrlData from './app/AppUrlData.vue';

const props = defineProps<{
	app: App
}>();

const appspacesStore = useAppspacesStore();
appspacesStore.loadData();

const appspaces = computed( () => {
	return appspacesStore.getAppspacesForApp(props.app.app_id);
});

const app_icon_error = ref(false);
const app_icon = computed( () => {
	if( app_icon_error.value || !props.app.cur_ver ) return "";
	return `/api/application/${props.app.app_id}/version/${props.app.cur_ver}/file/app-icon`;
});

</script>

<template>
	<div class="md:mb-6 my-6 bg-white shadow overflow-hidden border-t-8 " :style="'border-color:'+(app.ver_data?.color || 'rgb(135, 151, 164)')" >
		<div class="grid app-grid gap-x-2 gap-y-2 px-4 pt-4 sm:px-6">
			<img v-if="app_icon" :src="app_icon" @error="app_icon_error = true" class="w-20 h-20" />
			<div v-else class="w-20 h-20 text-gray-300 __border flex justify-center items-center">
				<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-14 h-14">
					<path stroke-linecap="round" stroke-linejoin="round" d="M2.25 15a4.5 4.5 0 004.5 4.5H18a3.75 3.75 0 001.332-7.257 3 3 0 00-3.758-3.848 5.25 5.25 0 00-10.233 2.33A4.502 4.502 0 002.25 15z" />
				</svg>
			</div>
			<div class="self-center">
				<h3 class="text-2xl leading-6 font-medium text-gray-900">{{app.ver_data?.name}}</h3>
				<p class="italic" v-if="app.ver_data?.short_desc">“{{app.ver_data?.short_desc}}”</p>
			</div>
			<div class="col-span-3 sm:col-start-2">
				<p>Version 
					<span class="bg-gray-200 text-gray-600 px-1 rounded-md">{{app.cur_ver}}</span>
					by <AppAuthorsSummary :authors="app.ver_data?.authors" class=""></AppAuthorsSummary>
				</p>
				<AppUrlData v-if="app.url_data" :d="app.url_data" :cur_ver="app.cur_ver"></AppUrlData>
				<p v-else>Manually uploaded</p>
				<AppLicense class="" :license="app.ver_data?.license"></AppLicense>
				<AppLinksCompact class="mt-4" :ver_data="app.ver_data"></AppLinksCompact>
			</div>
			<router-link 
				:to="{name: 'manage-app', params:{id:app.app_id}}" 
				class="btn btn-blue col-start-1 col-span-3 justify-self-end  sm:self-start sm:row-start-1 sm:col-start-3">
				Manage
			</router-link>
		</div>
		<div class="flex justify-between items-center mt-4 px-4 sm:px-6 py-2 border-t" :class="[appspaces.length === 0 ? 'border-t' : '']">
			<span v-if="appspaces.length === 0" class="italic text-gray-500">No appspaces use this app.</span>
			<h4 v-else class="font-medium">Appspaces:</h4>
			<router-link :to="{name:'new-appspace', query:{app_id:app.app_id, version:app.cur_ver}}" class="btn">Create Appspace</router-link>
		</div>
		<div v-for="a in appspaces" class="flex justify-between items-baseline  border-t px-4 sm:px-6">
			<span class=" py-2 font-medium">{{ a.value.domain_name }}</span>
			<span>
				<span class="pr-4"><span class="bg-gray-200 text-gray-600 px-1 rounded-md">{{ a.value.app_version }}</span></span>
				<span class="">
					<router-link :to="{name: 'manage-appspace', params:{appspace_id:a.value.appspace_id}}" class="btn">Manage</router-link>
				</span>
			</span>
		</div>
	</div>
</template>

<style scoped>
.app-grid {
	grid-template-columns: 5rem 1fr max-content;
}
</style>