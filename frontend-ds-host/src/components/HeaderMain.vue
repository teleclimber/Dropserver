<style scoped>
	header {
		grid-area: header;
	}
</style>

<template>
	<header class="fixed w-full md:w-auto md:relative border-b bg-white grid ds-header-phone md:ds-header-full">
		<a class="md:hidden justify-self-center self-center" href="#" @click.stop.prevent="openNav()">
			<svg class="w-8 h8" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
			</svg>
		</a>
		<h1 class="text-xl md:text-4xl py-4 md:py-6 md:pl-6 font-bold text-gray-800 flex-no-wrap whitespace-nowrap overflow-hidden overflow-ellipsis">
			{{ page_title ? page_title : getHead()}}
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

<script lang="ts">
import { defineComponent } from 'vue';
import {useRoute} from 'vue-router';

import {openNav, openUserMenu, page_title} from '../controllers/nav';

export default defineComponent({
	name: 'HeaderMain',
	components: {
		
	},
	setup() {
		const route = useRoute();
		function getHead() {
			switch(route.name) {
				case "home": return "Home";
				case "user": return "User Settings";

				case "apps": return "Apps";
				case "manage-app": return "Manage App";
				case "new-app": return "New Application";	// upload or get from URL
				case "new-app-in-process": return "New Application";

				case "new-app-version": return "Upload New Version";	// New versions are only for manually uploaded apps
				case "new-app-version-in-process": return "Upload New Version";
				
				case "appspaces": return "Appspaces";
				case "new-appspace": return "New Appspace";
				case "manage-appspace": return "Manage Appspace";	// this should actually reflect the appspace name, or something like that.
				case "migrate-appspace": return "Migrate Appspace";
				case "restore-appspace": return "Restore Appspace";

				case "new-remote-appspace": return "Join Appspace";
				case "manage-remote-appspace": return "Manage Remote Appspace";	// this gets overridden by remote address

				case 'contacts': return "Contacts";
				case 'new-contact': return "Add Contact";

				case "admin": return "Instance Dashboard";
				case "admin-users": return "Instance Users";
				case "admin-settings": return "Instance Settings";
			}
			return route.name;	// default
		}
		return {
			getHead,
			openNav,
			openUserMenu,
			page_title
		}
	}
});
</script>