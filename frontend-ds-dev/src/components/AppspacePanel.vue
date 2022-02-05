<template>
	<div class="m-4 flex justify-between">
		<AppspaceControl></AppspaceControl>
	</div>
	<div class="m-4">
		<p>Working Dir: ??</p>
		<p>Appspace Schema: {{appspaceStatus.appspace_schema}}</p>
	</div>
	<!-- here in future I imagine we'd have a rudimentary file viewer? -->
</template>

<script lang="ts">
import { defineComponent, reactive } from 'vue';

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
			appspaceStatus,
		}
	},
});
</script>