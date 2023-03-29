<script lang="ts" setup>
import type {LiveLog} from '../../models/log';

defineProps<{
	live_log: LiveLog
}>();

</script>

<template>
	<div class="h-64 overflow-scroll bg-gray-50">
		<pre v-if="live_log.loaded" class="text-sm">{{live_log.content}}</pre>
		<span v-else class="">Loading log...</span>
	</div>
</template>
