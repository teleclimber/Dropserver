<script lang="ts" setup>
import { computed, ref } from 'vue';

import type { App } from '@/stores/types';
import { useAppspacesStore } from '@/stores/appspaces';

import DataDef from './ui/DataDef.vue';

const props = defineProps<{
	app: App
}>();

const latest_version = computed( () => {
	return props.app.versions[0].version;	// some chance there are zero versions in app??
});

const appspacesStore = useAppspacesStore();
appspacesStore.loadData();

const appspaces = computed( () => {
	return appspacesStore.getAppspacesForApp(props.app.app_id);
});

const app_icon_error = ref(false);
const app_icon = computed( () => {
	if( app_icon_error.value || props.app.versions.length === 0 ) return "";
	const v = props.app.versions[0].version;
	return `/api/application/${props.app.app_id}/version/${v}/app-icon`;
});

</script>

<template>
	<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
		<div class="px-4 py-5 sm:px-6 flex justify-between items-center">
			<div class="flex items-center">
				<img v-if="app_icon" :src="app_icon" @error="app_icon_error = true" class="w-20 h-20" />
				<div v-else class="w-20 h-20 text-gray-300 border flex justify-center items-center">
					<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-10 h-10">
						<path stroke-linecap="round" stroke-linejoin="round" d="M21 7.5l-2.25-1.313M21 7.5v2.25m0-2.25l-2.25 1.313M3 7.5l2.25-1.313M3 7.5l2.25 1.313M3 7.5v2.25m9 3l2.25-1.313M12 12.75l-2.25-1.313M12 12.75V15m0 6.75l2.25-1.313M12 21.75V19.5m0 2.25l-2.25-1.313m0-16.875L12 2.25l2.25 1.313M21 14.25v2.25l-2.25 1.313m-13.5 0L3 16.5v-2.25" />
					</svg>
				</div>
				<div class="ml-4">
					<h3 class="text-2xl leading-6 font-medium text-gray-900">{{app.name}}</h3>
					<p class="mt-1 max-w-2xl">
						Created: {{app.created_dt.toLocaleString()}}
					</p>
				</div>
			</div>
			<div>
				<router-link :to="{name: 'manage-app', params:{id:app.app_id}}" class="btn btn-blue">Manage</router-link>
			</div>
		</div>
		<div class="border-t border-gray-200">
			<DataDef field="Versions:">
				{{ app.versions.length }}, latest: {{latest_version}}
			</DataDef>
			<DataDef field="Appspaces:" v-if="appspaces.length !== 0">
				<div v-for="a in appspaces">
						{{ a.value.domain_name }} ({{ a.value.app_version }})
						<router-link :to="{name: 'manage-appspace', params:{appspace_id:a.value.appspace_id}}" class="btn">Manage</router-link>
					</div>
			</DataDef>
			<div class="flex justify-end my-5 px-4 sm:px-6">
				<router-link :to="{name:'new-appspace', query:{app_id:app.app_id, version:latest_version}}" class="btn">Create Appspace</router-link>
			</div>
		</div>
	</div>
</template>

