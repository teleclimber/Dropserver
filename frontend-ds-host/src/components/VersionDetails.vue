<template>
	<div class="grid grid-cols-4 items-center justify-center border border-gray-200 rounded-md">
		<span class="bg-gray-200 text-center">version</span>
		<span class="bg-gray-200 text-center">schema</span>
		<span class="bg-gray-200 text-center">API</span>
		<span class="bg-gray-200">&nbsp;</span>
		<div class="px-4 py-2 font-bold text-lg text-center">{{app_version.version}}</div>
		<div class=" text-center">{{app_version.schema}}</div>
		<div class=" text-center">{{app_version.api_version}}</div>
		<div>details...</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, PropType } from 'vue';

import type {AppVersion} from '../models/app_versions';

export default defineComponent({
	name: 'VersionDetails',
	components: {
		
	},
	props: {
		app_version: {
			type: Object as PropType<AppVersion>,
			required: true
		}
	}
	
});
</script>