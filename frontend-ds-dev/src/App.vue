<template>
	<div class="flex flex-col h-screen">
		<AppHead></AppHead>

		<div class="flex border-b-4 border-black px-4 ">
			<Tab tab="app" class="relative flex items-center">
				App
				<span v-if="app_error" class="absolute -top-1 right-2 flex h-3 w-3 ">
					<span v-if="app_control.tab !== 'app'" class="animate-ping absolute inline-flex h-full w-full rounded-full bg-red-400 opacity-75"></span>
					<span class="inline-flex rounded-full h-3 w-3 bg-red-500 "></span>
				</span>
			</Tab>
			<Tab tab="appspace">
				Appspace
			</Tab>
			<Tab tab="users">
				Users
			</Tab>
			<Tab tab="route-hits">
				Route Hits
			</Tab>
		</div>

		<div class="flex-shrink flex-grow overflow-y-scroll">
			<AppPanel v-if="app_control.tab === 'app'" ></AppPanel>
			<AppspacePanel v-else-if="app_control.tab === 'appspace'"></AppspacePanel>
			<RouteHitsPanel v-else-if="app_control.tab === 'route-hits'"></RouteHitsPanel>
			<UsersPanel v-else-if="app_control.tab === 'users'"></UsersPanel>
		</div>

		<AppspaceLogPanel v-if="app_control.tab !== 'app'"></AppspaceLogPanel>

	</div>
</template>

<script lang="ts">
import { defineComponent, computed, watch } from 'vue';
import baseData from './models/base-data';
import appData from './models/app-data';
import appspaceStatus from './models/appspace-status';

import AppHead from './components/AppHead.vue';
import Tab from './components/Tab.vue';
import AppPanel from './components/AppPanel.vue';
import AppspacePanel from './components/AppspacePanel.vue';
import RouteHitsPanel from './components/RouteHitsPanel.vue';
import UsersPanel from './components/UsersPanel.vue';
import AppspaceLogPanel from './components/AppspaceLogPanel.vue';

import {app_control} from './main';

export default defineComponent({
	name: 'DropserverAppDev',
	components: {
		AppHead,
		Tab,
		AppPanel,
		AppspacePanel,
		RouteHitsPanel,
		UsersPanel,
		AppspaceLogPanel
	},
	setup(props, context) {
		const app_error = computed( () => {
			return !appData.last_processing_event.processing && appData.last_processing_event.errors.length
		});
		return {
			baseData,
			appspaceStatus,
			app_control,
			app_error
		};
	}
});
</script>

