<script lang="ts" setup>
import { ref } from 'vue';
import router from '../../router/index';

import { useRemoteAppspacesStore } from '@/stores/remote_appspaces';

const props = defineProps<{
	domain: string
}>();

const remoteAppspaceStore = useRemoteAppspacesStore();

const deleting = ref(false);

async function del() {
	deleting.value = true;
	await remoteAppspaceStore.deleteAppspace(props.domain);
	router.replace({name:"appspaces"});
}

</script>

<template>
	<div class="md:mb-6 my-6 bg-yellow-100 shadow overflow-hidden sm:rounded-lg">
		<div class="px-4 py-5 sm:px-6 border-b border-yellow-200">
			<h3 class="text-lg leading-6 font-medium text-gray-900">Remove Remote Appspace</h3>
			<p class="mt-1 max-w-2xl text-sm text-gray-700">
				Removing this remote appspace means you will no longer be able to access it
			</p>
		</div>
		<div class="py-5">
			<div class="px-4 sm:px-6 flex justify-end">
				<button v-if="!deleting" @click.stop.prevent="del" class="btn btn-blue" >remove</button>
				<span v-else>Removing...</span>
			</div>
		</div>
	</div>
</template>

