<template>
	<div class="md:mb-6">
		<h3>{{app.name}}</h3>
		<p>Created: {{app.created_dt.toLocaleString()}}</p>
		<ul>
			<li v-for="ver in app.versions" :key="ver.version">
				{{ver.version}} created {{ver.created_dt.toLocaleString()}}
				<router-link :to="{name:'new-appspace', query:{app_id:app.app_id, version:ver.version}}">Create Appspace</router-link>
			</li>
		</ul>
		<p><router-link :to="{name: 'manage-app', params:{id:app.app_id}}">Manage</router-link></p>
	</div>
</template>

<script lang="ts">
import { defineComponent, PropType } from 'vue';

import type {App} from '../models/apps';

export default defineComponent({
	name: 'AppListItem',
	components: {
		
	},
	props: {
		app: {
			type: Object as PropType<App>,
			required: true
		}
	}
	
});
</script>