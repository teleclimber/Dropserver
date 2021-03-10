<template>
	<div class="grid grid-cols-4 items-stretch justify-center border border-gray-200 rounded-md">
		<span class="bg-gray-200 text-center">version</span>
		<span class="bg-gray-200 text-center">schema</span>
		<span class="bg-gray-200 text-center">API</span>
		<span class="bg-gray-200 text-right pr-2">
			<button @click="$emit('close')" class="btn">
				Close
				<svg class="align-text-bottom inline w-4 h-4" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
					<path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" />
				</svg>
			</button>
		</span>
		<div v-for="ver in versions" :key="ver.version" class="contents" >
			<div class="px-4 py-2 font-bold text-center border-t border-gray-200">
				<div v-if="ver.version === current" class="text-gray-400 text-lg">{{ver.version}}</div>
				<button v-else @click="$emit('version', ver.version)" class="btn text-lg">
					{{ver.version}}
				</button>

			</div>
			<div class="border-t border-gray-200 flex justify-center items-center">{{ver.schema}}</div>
			<div class="border-t border-gray-200 flex justify-center items-center">{{ver.api_version}}</div>
			<div class="border-t border-gray-200 flex justify-center items-center">details...</div>
			
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, PropType } from 'vue';

import type {AppVersion} from '../models/app_versions';

export default defineComponent({
	name: 'PickVersion',
	components: {
		
	},
	props: {
		versions: {
			type: Object as PropType<AppVersion[]>,
			required: true
		},
		current: String
	},
	emits: ['version', 'close'],
	
});
</script>