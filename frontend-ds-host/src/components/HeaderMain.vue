<style scoped>
	@layer base {
		@responsive {
			.ds-header-full {
				grid-template-columns: 1fr 1fr;
			}
		}
		.ds-header-phone {
			grid-template-columns: 4rem 1fr 1fr;
		}
	}
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
		<h1 class="text-xl md:text-4xl py-4 md:py-6 md:pl-6 font-bold text-gray-800">{{getHead()}}</h1>

		<div class="justify-self-end self-center pr-2">[user name]</div>
	</header>
</template>

<script lang="ts">
import { defineComponent } from 'vue';
import {useRoute} from 'vue-router';

import {openNav} from '../controllers/nav';

export default defineComponent({
	name: 'HeaderMain',
	components: {
		
	},
	setup() {
		const route = useRoute();
		function getHead() {
			switch(route.name) {
				case "home": return "Home";
				case "apps": return "Apps";
				case "manage-app": return "Manage App";
				case "new-app": return "New App";	// upload or get from URL
				case "new-app-version": return "Upload New Version";	// New versions are only for manually uploaded apps
				case "appspaces": return "Appspaces";
				case "new-appspace": return "New Appspace";
				case "manage-appspace": return "Manage Appspace";	// this should actually reflect the appspace name, or something like that.
			}
			return route.name;	// default
		}
		return {
			getHead,
			openNav
		}
	}
});
</script>