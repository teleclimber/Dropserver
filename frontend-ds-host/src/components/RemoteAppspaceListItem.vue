<script setup lang="ts">
import { ref } from 'vue';

import type { RemoteAppspace } from '@/stores/types';

const props = defineProps<{
	remote_appspace: RemoteAppspace
}>();

const protocol = props.remote_appspace.no_tls ? 'http' : 'https';
const display_link = ref(protocol+'://'+props.remote_appspace.domain_name+props.remote_appspace.port_string);
const enter_link = ref("/appspacelogin?appspace="+encodeURIComponent(props.remote_appspace.domain_name));

</script>

<template>
	<div class="bg-white overflow-hidden border-b border-b-gray-300 px-4 py-4 ">
		<h3 class="text-xl md:text-2xl font-medium text-gray-900">
			{{remote_appspace.domain_name}}
		</h3>
		<p><a :href="enter_link" class="text-blue-700 underline hover:text-blue-500 overflow-hidden text-ellipsis">{{ display_link }}</a></p>
		<p class="mt-4">Remote Appspace provided by [owner dropid].</p>

		<div class="pt-4 flex justify-end">
			<router-link :to="{name: 'manage-remote-appspace', params:{domain:remote_appspace.domain_name}}" class="btn">Manage appspace</router-link>
		</div>
	</div>
</template>