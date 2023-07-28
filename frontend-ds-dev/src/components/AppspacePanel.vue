<script setup lang="ts">
import baseData from '../models/base-data';
import appspaceStatus from '../models/appspace-status';

import AppspaceControl from './AppspaceControl.vue';

</script>
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