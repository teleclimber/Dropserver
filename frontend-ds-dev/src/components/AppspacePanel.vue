<template>
	<div class="m-4">
		<AppspaceControl></AppspaceControl>
	</div>
	<div class="m-4">
		<h2 class="text-2xl my-2">Appspace Data:</h2>
		<p>Working Dir: {{baseData.appspace_working_dir}}</p>
		<p>Appspace Schema: {{appspaceStatus.appspace_schema}}</p>
	</div>
	<!-- here in future I imagine we'd have a rudimentary file viewer? -->
</template>

<script lang="ts">
import { defineComponent } from 'vue';

import baseData from '../models/base-data';
import appspaceStatus from '../models/appspace-status';

import AppspaceControl from './AppspaceControl.vue';
import Log from './Log.vue';

export default defineComponent({
	components: {
		AppspaceControl,
		Log,
	},
	setup() {
		return {
			baseData,
			appspaceStatus,
		}
	},
});
</script>