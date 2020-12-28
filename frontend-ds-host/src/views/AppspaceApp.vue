<template>
	<div v-if="app_version.loaded">

	</div>
	<div v-else>
		Loading...
	</div>
</template>

<script lang="ts">
import { defineComponent } from 'vue';
import { app_versions } from '../models/apps';

export default defineComponent({
	name: 'AppspaceApp',
	components: {
	
	},
	props: {
		app_id: {
			type: Number,
			required: true
		},
		version: {
			type: String,
			required: true
		}
	},
	setup(props) {
		const app_version = app_versions.getAppVersion(props.app_id, props.version);
		return {app_version}
	}
});
</script>