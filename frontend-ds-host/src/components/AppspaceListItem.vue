<template>
	<div class="md:mb-6">
		<h2 class="text-xl">{{appspace.subdomain}}.something.sometld</h2>
		<p>Created: {{appspace.created_dt.toLocaleString()}}</p>
		<p>App Version: {{app_version.app_name}}, version: {{appspace.app_version}} (schema: {{app_version.schema}}, API: {{app_version.api_version}})</p>
		<p>{{ appspace.paused ? 'Paused' : 'Not Paused (ready)' }}</p>
		<p>Upgrade available?</p>
		<p><router-link :to="{name: 'manage-appspace', params:{id:appspace.id}}">Manage</router-link></p>
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