<template>
	<div class="flex flex-col h-screen">
		<AppHead></AppHead>

		<div class="flex border-b-4 border-black px-4 ">
			<Tab tab="app">
				App
			</Tab>
			<Tab tab="appspace">
				Appspace
			</Tab>
			<Tab tab="migrations">
				Migrations
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
			<MigrationsPanel v-else-if="app_control.tab === 'migrations'"></MigrationsPanel>
			<UsersPanel v-else-if="app_control.tab === 'users'"></UsersPanel>
		</div>

		<AppspaceLogPanel v-if="app_control.tab !== 'app'"></AppspaceLogPanel>

	</div>
</template>

<script lang="ts">
import { defineComponent, reactive, watch } from 'vue';
import baseData from './models/base-data';
import appspaceStatus from './models/appspace-status';

import AppHead from './components/AppHead.vue';
import Tab from './components/Tab.vue';
import AppPanel from './components/AppPanel.vue';
import AppspacePanel from './components/AppspacePanel.vue';
import RouteHitsPanel from './components/RouteHitsPanel.vue';
import MigrationsPanel from './components/MigrationsPanel.vue';
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
		MigrationsPanel,
		UsersPanel,
		AppspaceLogPanel
	},
	setup(props, context) {
	
		return {
			baseData,
			appspaceStatus,
			app_control,
		};
	}
});
</script>

