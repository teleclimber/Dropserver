<template>
	<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
		<div class="px-4 py-5 sm:px-6 flex flex-col sm:flex-row sm:justify-between">
			<div>
				<h3 class="text-2xl leading-6 font-medium text-gray-900">
					{{appspace.subdomain}}.something.sometld
					<svg class="inline align-bottom w-7 h-7 text-indigo-600" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
						<path d="M11 3a1 1 0 100 2h2.586l-6.293 6.293a1 1 0 101.414 1.414L15 6.414V9a1 1 0 102 0V4a1 1 0 00-1-1h-5z" />
						<path d="M5 5a2 2 0 00-2 2v8a2 2 0 002 2h8a2 2 0 002-2v-3a1 1 0 10-2 0v3H5V7h3a1 1 0 000-2H5z" />
					</svg>	
				</h3>
				<p class="mt-1 max-w-2xl text-sm text-gray-500">
					{{app_version.app_name}} v. {{app_version.version}}
					<span v-if="appspace.upgrade" class="bg-blue-200 text-blue-600 px-2">Upgrade available: {{appspace.upgrade.version}}</span>
				</p>
			</div>
			<div>
				<div v-if="appspace.paused">
					Paused
				</div>
				<div v-else class="">
					<span class="text-green-700 bg-green-200 px-2">
						<svg class="inline align-bottom w-7 h-7 mr-2" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
							<path d="M2 10.5a1.5 1.5 0 113 0v6a1.5 1.5 0 01-3 0v-6zM6 10.333v5.43a2 2 0 001.106 1.79l.05.025A4 4 0 008.943 18h5.416a2 2 0 001.962-1.608l1.2-6A2 2 0 0015.56 8H12V4a2 2 0 00-2-2 1 1 0 00-1 1v.667a4 4 0 01-.8 2.4L6.8 7.933a4 4 0 00-.8 2.4z" />
						</svg>
						<span>Ready</span>
					</span>
				</div>
			</div>
		</div>
		<div class="px-4 py-5 sm:px-6 flex justify-end border-t border-gray-200">
			<router-link :to="{name: 'manage-appspace', params:{id:appspace.id}}" class="btn btn-blue">Manage</router-link>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, PropType } from 'vue';

import type {Appspace} from '../models/appspaces';
import {AppVersionCollector } from '../models/app_versions';

export default defineComponent({
	name: 'AppspaceListItem',
	components: {
	},
	props: {
		appspace: {
			type: Object as PropType<Appspace>,
			required: true
		}
	},
	setup(props) {
		// this will bomb if appspace is not loaded yet.
		if( !props.appspace.loaded ) console.error("appspace not loaded yet.");
		const app_version = AppVersionCollector.get(props.appspace.app_id, props.appspace.app_version);

		return {
			app_version
		}
	}
	
});

// Here I would like to load all kinds of things, like app version, potential upgrade, ...
// But tehre may be many list items being created at the same time, so a request per is bad
// Need to batch requests.
// If there was a global AppVerionsCollection, then I could "ask" for it, 
//  ..and it would automatically load a bunch at a time.
</script>