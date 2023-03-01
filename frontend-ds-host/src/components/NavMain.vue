<script setup lang="ts">
import { watch, computed } from 'vue';
import {useRoute} from 'vue-router';

import {nav_open, closeNav} from '../controllers/nav';

import DropserverLogo from './DropserverLogo.vue';
import NavItem from './NavItem.vue';

const path2section = [
	{section: "appspaces",		paths: ["/appspace", "/new-appspace", "/remote-appspace", "/new-remote-appspace"]},
	{section: "apps", 			paths: ["/app", "/new-app"]},
	{section: "contacts",		paths: ["/contact"]},
	{section: "admin-users",	paths: ["/admin/users"]},
	{section: "admin-settings", paths: ["/admin/settings"]},
	{section: "admin-home", 	paths: ["/admin"]},
];

const route = useRoute();
watch( () => route.name, () => {
	closeNav();
});

const in_admin = computed( () => {
	return route.path.startsWith('/admin');
});

const active_section = computed( () => {
	const p = route.path;
	const s = path2section.find( sec => sec.paths.find( pre => p.startsWith(pre)) );
	if( s ) return s.section;
	return '';
});
</script>

<style scoped>
	aside {
		grid-area: nav;
	}
</style>

<template>
	<aside class="fixed md:relative block md:block w-screen md:w-auto z-10 h-screen md:shadow-xl bg-gray-800" :class="{hidden:!nav_open}">
		<div class="flex justify-between">
			<router-link to="/"><DropserverLogo class="p-6 text-4xl" :dark="true"></DropserverLogo></router-link>
			<a href="#" @click.stop.prevent="closeNav" class="p-6 md:hidden">
				<svg class=" h-10 w-10 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
				</svg>
			</a>
		</div>
		<nav>
			<ul v-if="in_admin" class="">
				<NavItem to="/admin" :active="active_section === 'admin-home'">Admin Home</NavItem>
				<NavItem to="/admin/users" :active="active_section === 'admin-users'">Users</NavItem>
				<NavItem to="/admin/settings" :active="active_section === 'admin-settings'">Settings</NavItem>
			</ul>
			<ul v-else class="">
				<NavItem to="/appspace" :active="active_section === 'appspaces'">Appspaces</NavItem>
				<NavItem to="/app" :active="active_section === 'apps'">Apps</NavItem>
				<NavItem to="/contact" :active="active_section === 'contacts'">Contacts</NavItem>
			</ul>
		</nav>
	</aside>
</template>
