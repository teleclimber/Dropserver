<script lang="ts" setup>
import { computed } from 'vue';

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

</script>

<template>
	<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
		<div class="px-4 py-5 sm:px-6 flex justify-between">
			<div>
				<h3 class="text-2xl leading-6 font-medium text-gray-900">{{app.name}}</h3>
				<p class="mt-1 max-w-2xl">
					Created: {{app.created_dt.toLocaleString()}}
				</p>
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

