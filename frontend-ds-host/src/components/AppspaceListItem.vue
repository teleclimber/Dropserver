<script setup lang="ts">
import { ref, computed } from 'vue';

import { useAppsStore } from '@/stores/apps';
import type { Appspace } from '@/stores/types';

const props = defineProps<{
	appspace: Appspace
}>();

const protocol = props.appspace.no_tls ? 'http' : 'https';
const display_link = ref(protocol+'://'+props.appspace.domain_name+props.appspace.port_string)

const enter_link = ref("/appspacelogin?appspace="+encodeURIComponent(props.appspace.domain_name));

const version_classes = ref(['bg-green-200', 'text-green-800']);
if( props.appspace.upgrade_version ) version_classes.value = ['bg-orange-200', 'text-orange-800'];

</script>

<template>
	<div class="bg-white overflow-hidden border-b border-b-gray-300 px-4 py-4 ">
		<h3 class="text-xl md:text-2xl font-medium text-gray-900">
			{{appspace.domain_name}}
		</h3>
		<p><a :href="enter_link" class="text-blue-700 underline hover:text-blue-500 overflow-hidden text-ellipsis">{{ display_link }}</a></p>
		<p class="mt-4">
			<router-link :to="{name: 'manage-app', params: {id:appspace.app_id}}" class="font-medium text-blue-800 hover:underline ">
				{{appspace.ver_data?.name}}
			</router-link> 
			<span class="text-sm font-medium rounded-full px-2 ml-2 " :class="version_classes">{{appspace.app_version}}</span>
			<span v-if="appspace.upgrade_version" class="bg-green-200 text-green-800 rounded-full ml-2 px-2 text-sm">
				Upgrade available: {{appspace.upgrade_version}}
			</span>
		</p>
		<p>Owner DropID: {{ appspace.dropid }}</p>
		<div class="pt-4 flex justify-end items-baseline">
			<span v-if="appspace.paused" class="mr-2 bg-pink-200 text-pink-800 px-2 text-xs font-bold uppercase">
				<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-4 h-4 inline-block">
					<path d="M5.75 3a.75.75 0 00-.75.75v12.5c0 .414.336.75.75.75h1.5a.75.75 0 00.75-.75V3.75A.75.75 0 007.25 3h-1.5zM12.75 3a.75.75 0 00-.75.75v12.5c0 .414.336.75.75.75h1.5a.75.75 0 00.75-.75V3.75a.75.75 0 00-.75-.75h-1.5z" />
				</svg>

				Paused
			</span>
			<router-link :to="{name: 'manage-appspace', params:{appspace_id:appspace.appspace_id}}" class="btn">Manage appspace</router-link>
		</div>
	</div>
</template>