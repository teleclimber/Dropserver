<script setup lang="ts">
import { computed } from 'vue';
import type { AppUrlData } from '@/stores/types';

const props = defineProps<{
	url_data: AppUrlData,
	cur_ver?: string
}>();

const new_ver = computed( () => {
	return props.url_data.latest_version !== props.cur_ver;
});

const last_check_str = computed( () => {
	let time_ago = Math.round((Date.now() - props.url_data.last_dt.getTime())/1000/60);
	if( time_ago < 60 ) return time_ago + ' minutes ago';
	time_ago = Math.round(time_ago/60);
	if( time_ago <= 24 ) return time_ago + ' hours ago';
	return props.url_data.last_dt.toLocaleDateString();
});

</script>

<template>
	<div class="flex">
		&nbsp;
		<svg v-if="url_data.last_result === 'error' || url_data.new_url" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-5 h-5 text-orange-600">
			<path fill-rule="evenodd" d="M8.485 2.495c.673-1.167 2.357-1.167 3.03 0l6.28 10.875c.673 1.167-.17 2.625-1.516 2.625H3.72c-1.347 0-2.189-1.458-1.515-2.625L8.485 2.495zM10 5a.75.75 0 01.75.75v3.5a.75.75 0 01-1.5 0v-3.5A.75.75 0 0110 5zm0 9a1 1 0 100-2 1 1 0 000 2z" clip-rule="evenodd" />
		</svg>
		<svg v-else xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5">
			<path stroke-linecap="round" stroke-linejoin="round" d="M12 21a9.004 9.004 0 008.716-6.747M12 21a9.004 9.004 0 01-8.716-6.747M12 21c2.485 0 4.5-4.03 4.5-9S14.485 3 12 3m0 18c-2.485 0-4.5-4.03-4.5-9S9.515 3 12 3m0 0a8.997 8.997 0 017.843 4.582M12 3a8.997 8.997 0 00-7.843 4.582m15.686 0A11.953 11.953 0 0112 10.5c-2.998 0-5.74-1.1-7.843-2.918m15.686 0A8.959 8.959 0 0121 12c0 .778-.099 1.533-.284 2.253m0 0A17.919 17.919 0 0112 16.5c-3.162 0-6.133-.815-8.716-2.247m0 0A9.015 9.015 0 013 12c0-1.605.42-3.113 1.157-4.418" />
		</svg>
		<span v-if="url_data.last_result === 'error'" class="text-orange-600">There was a problem fetching the app listing</span>
		<span v-else-if="url_data.new_url" class="text-orange-600">App has moved to a new address</span>
		<span v-else-if="new_ver">
			Latest published version: {{  url_data.latest_version }}
		</span>
		<span v-if="url_data.last_result !== 'error' && !new_ver" >
			Last checked for new versions {{ last_check_str }}
			{{ url_data.automatic ? '(automatic)' : '' }}
		</span>
	</div>
</template>