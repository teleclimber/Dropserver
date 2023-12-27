<script lang="ts" setup>
import { ref, computed } from 'vue';
import { useRouter } from 'vue-router';

import { useAppspacesStore } from '@/stores/appspaces';
import type { Appspace } from '../../stores/types';

import DataDef from '../../components/ui/DataDef.vue';

const props = defineProps<{
	appspace: Appspace
}>();

const router = useRouter();

const appspacesStore = useAppspacesStore();

const domain_check = ref("");

const domain_checked = computed(() => {
	return domain_check.value.toLowerCase() === props.appspace.domain_name.toLowerCase();
});

const deleting = ref(false);

async function del() {
	if( !domain_checked.value) return;

	deleting.value = true;
	await appspacesStore.deleteAppspace(props.appspace.appspace_id);

	router.replace({name:"appspaces"});
}
</script>

<template>
	<div class="md:mb-6 my-6 bg-yellow-100 shadow overflow-hidden sm:rounded-lg">
		<div class="px-4 py-5 sm:px-6 border-b border-yellow-200">
			<h3 class="text-lg leading-6 font-medium text-gray-900">Delete Appspace</h3>
			<p class="mt-1 max-w-2xl text-sm text-gray-700">
				Deleting this appspace will delete all the data (including backups!)
			</p>
		</div>
		<div class="pb-5">
			<DataDef field="Enter domain name of appspace:">
				<input type="text" v-model="domain_check" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
			</DataDef>
			<div class="px-4 sm:px-6 flex justify-end">
				<button v-if="!deleting" @click.stop.prevent="del" class="btn btn-blue" :disabled="!domain_checked">delete</button>
				<span v-else>Deleting...</span>
			</div>
		</div>
	</div>
</template>

