<template>
	<div class="h-64 overflow-scroll bg-gray-50">
		<pre v-if="live_log.loaded" class="text-sm">{{live_log.content}}</pre>
		<span v-else class="">Loading log...</span>
	</div>
</template>

<script lang="ts">
import { defineComponent, PropType } from 'vue';

import type {LiveLog} from '../../models/log';

export default defineComponent({
	name: 'LogViewer',
	components: {
		
	},
	props: {
		live_log: {
			type: Object as PropType<LiveLog>,
			required: true
		}
	},
});
</script>