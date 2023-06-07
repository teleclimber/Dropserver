<script lang="ts" setup>
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import type { AppVersion } from '../stores/types';

import MessageSad from '@/components/ui/MessageSad.vue';

const router = useRouter();

const props = defineProps<{
	appspace_id: number,
	versions: AppVersion[],
	current: string
}>();

const pick_versions = computed( () => {
	return props.versions.filter( v => v.version !== props.current );
});

function pickVersion(v:string) {
	router.replace({name: 'migrate-appspace', params:{appspace_id:props.appspace_id}, query:{to_version:v}});
}

</script>

<template>
	<div class="grid grid-cols-4 items-stretch justify-center border border-gray-200 rounded-md">
		<span class="bg-gray-200 text-center">version</span>
		<span class="bg-gray-200 text-center">schema</span>
		<span class="bg-gray-200 text-center">API</span>
		<span class="bg-gray-200">&nbsp;</span>
		<div v-for="ver in pick_versions" :key="ver.version" class="contents" >
			<div class="px-4 py-2 font-bold text-center border-t border-gray-200">
				<div v-if="ver.version === current" class="text-gray-400 text-lg">{{ver.version}}</div>
				<button v-else @click="pickVersion(ver.version)" class="btn text-lg">
					{{ver.version}}
				</button>
			</div>
			<div class="border-t border-gray-200 flex justify-center items-center">{{ver.schema}}</div>
			<div class="border-t border-gray-200 flex justify-center items-center">0...</div>
			<div class="border-t border-gray-200 flex justify-center items-center">details...</div>
		</div>
		<MessageSad v-if="pick_versions.length === 0" head="No Other Versions" class="col-span-4">
			There are no other versions of this app on the system.
		</MessageSad>
	</div>
</template>
