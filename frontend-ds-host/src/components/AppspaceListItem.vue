<template>
	<div class="md:mb-6">
		<h2 class="text-xl">{{appspace.subdomain}}.something.sometld</h2>
		<p>Created: {{appspace.created_dt.toLocaleString()}}</p>
		<p v-if="appspace.app_version.loaded">App: {{appspace.app_version.name}}, version: {{appspace.app_version.version}}</p>
		<p>{{ appspace.paused ? 'Paused' : 'Not Paused (ready)' }}</p>
		<p>Upgrade available?</p>
		<p><router-link :to="{name: 'manage-appspace', params:{id:appspace.id}}">Manage</router-link></p>
	</div>
</template>

<script lang="ts">
import { defineComponent, PropType } from 'vue';

import type {Appspace} from '../models/appspaces';

export default defineComponent({
	name: 'AppspaceListItem',
	components: {
		
	},
	props: {
		appspace: {
			type: Object as PropType<Appspace>,
			required: true
		}
	}
	
});
</script>