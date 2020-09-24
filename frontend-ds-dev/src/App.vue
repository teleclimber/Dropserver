<template>
	<h3 class="text-gray-600">Dropserver App Dev<br>-app {{baseData.app_path}}<br>-appspace {{baseData.appspace_path}}</h3>

	<div class="grid grid-cols-2 my-8">
		<div class="border-l-4 border-gray-800">
			<h4 class="bg-gray-800 px-2 text-white inline-block">App:</h4>
			<h1 class="text-2xl px-2">{{baseData.app_name}} <span class="bg-gray-400 px-2 rounded-md">{{baseData.app_version}}</span></h1>
			<div class="px-2">
				Schema: <span>{{baseData.app_version_schema}}</span>
			</div>
		</div>
		<div class="border-l-4 border-gray-800">
			<h4 class="bg-gray-800 px-2 text-white inline-block">Appspace:</h4>
			<h1 class="text-2xl px-2">[baseData.appspace_name]</h1>
			<div class="px-2">
				Schema: <span>{{baseData.appspace_schema}}</span>
			</div>
		</div>
	</div>

	<AppspaceControl></AppspaceControl>

	<RouteHits></RouteHits>

</template>

<script lang="ts">
import { defineComponent } from 'vue';
import baseData from './models/base-data';

import AppspaceControl from './components/AppspaceControl.vue';
import RouteHits from './components/RouteHits.vue';

export default defineComponent({
	name: 'DropserverAppDev',
	components: {
		AppspaceControl,
		RouteHits,
	},
	setup(props, context) {
		return {
			baseData
		};
	}
});
</script>

