<script lang="ts" setup>
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import type { AppVersionUI } from '../stores/types';

import MessageSad from '@/components/ui/MessageSad.vue';

const router = useRouter();

const props = defineProps<{
	appspace_id: number,
	versions: AppVersionUI[],
	current: string
}>();

const no_versions = computed( () => {
	return props.versions.filter( v => v.version !== props.current ).length === 0;
});

function pickVersion(v:string) {
	router.push({name: 'migrate-appspace', params:{appspace_id:props.appspace_id}, query:{to_version:v}});
}

</script>

<template>
	<div class="grid grid-cols-4 items-stretch justify-center border border-gray-200 ">
		
		<span class="bg-gray-200 text-center">version</span>
		<span class="bg-gray-200 text-center">schema</span>
		<span class="bg-gray-200 text-center">API</span>
		<span class="bg-gray-200">&nbsp;</span>
		<div v-for="ver in props.versions" :key="ver.version" class="contents" >
			<div class="px-4 py-2 font-bold text-center border-t border-gray-200">
				<span class="font-medium bg-gray-200 text-gray-700 px-1 rounded-md">{{ ver.version }}</span>
			</div>
			<div class="border-t border-gray-200 flex justify-center items-center">{{ver.schema}}</div>
			<div class="border-t border-gray-200 flex justify-center items-center">0</div>
			<div class="border-t border-gray-200 flex justify-center items-center">
				<span v-if="ver.version === current" class="bg-yellow-200 text-yellow-800 px-2 uppercase text-xs font-medium">current</span>
				<button v-else @click="pickVersion(ver.version)" class="btn">select</button>
			</div>
		</div>
		<MessageSad v-if="no_versions" head="No Other Versions" class="col-span-4">
			There are no other versions of this app on the system.
		</MessageSad>
	</div>
</template>
