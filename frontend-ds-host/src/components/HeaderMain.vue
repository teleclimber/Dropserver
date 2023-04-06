<script lang="ts" setup>
import { computed } from 'vue';
import { useRoute } from 'vue-router';

import { appspaceIdParam, appIdParam } from '../router/index';

import { useAppspacesStore } from '@/stores/appspaces';
import { useAppsStore } from '@/stores/apps';

import { openNav, openUserMenu } from '../controllers/nav';

const route = useRoute();

const appspacesStore = useAppspacesStore();
const appsStore = useAppsStore();

const head = computed( () => {
	if( route.path.startsWith('/appspace/') ) {
		const appspace_id = appspaceIdParam(route);
		const appspace = appspacesStore.getAppspace(appspace_id);
		if( appspace !== undefined ) return appspace.value.domain_name;
	}
	if( route.path.startsWith('/app/') ) {
		const app_id = appIdParam(route);
		const app = appsStore.getApp(app_id);
		if( app !== undefined ) return app.value.name;
	}
	return route.meta.title;
});
</script>

<template>
	<header class="fixed w-full md:w-auto md:relative border-b bg-white grid ds-header-phone md:ds-header-full">
		<a class="md:hidden justify-self-center self-center" href="#" @click.stop.prevent="openNav()">
			<svg class="w-8 h8" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
			</svg>
		</a>
		<h1 class="text-xl md:text-4xl py-4 md:py-6 md:pl-6 font-bold text-gray-800 flex-no-wrap whitespace-nowrap overflow-hidden overflow-ellipsis">
			{{ head }}
		</h1>

		<div class="justify-self-end self-center pr-4 md:pr-6 flex-initial ">
			<div class="w-8 h-8 md:w-12 md:h-12 rounded-full bg-blue-100 border-2 border-blue-300 text-blue-400 flex justify-center items-end cursor-pointer hover:bg-blue-200" @click="openUserMenu">
				<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" class="w-6 h-6 md:w-10 md:h-10">
					<path fill-rule="evenodd" d="M7.5 6a4.5 4.5 0 119 0 4.5 4.5 0 01-9 0zM3.751 20.105a8.25 8.25 0 0116.498 0 .75.75 0 01-.437.695A18.683 18.683 0 0112 22.5c-2.786 0-5.433-.608-7.812-1.7a.75.75 0 01-.437-.695z" clip-rule="evenodd" />
				</svg>
			</div>
		</div>
	</header>
</template>

<style scoped>
	header {
		grid-area: header;
	}
</style>
