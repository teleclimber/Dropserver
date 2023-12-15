<script lang="ts" setup>
import { computed } from "vue";
import { useAppsStore } from '@/stores/apps';
import ViewWrap from '../components/ViewWrap.vue';
import NewAppVersionFromURL from "@/components/app/NewAppVersionFromURL.vue";
import SelectFiles from '../components/ui/SelectFiles.vue';

const props = defineProps<{
	app_id: number,
	version?: string
}>();

const appsStore = useAppsStore();
appsStore.loadApp(props.app_id);

const app = computed( () => {
	const a = appsStore.getApp(props.app_id);
	if( a ) return a.value;
	return undefined;
});

</script>

<template>
	<ViewWrap>
		<!-- add links to existing in-process apps if any? -->
		<div v-if="!app">
			loading...
		</div>
		<NewAppVersionFromURL v-else-if="app.url_data" :app_id="app_id" :version="version"></NewAppVersionFromURL>
		<div v-else class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
				<h3 class="text-lg leading-6 font-medium text-gray-900">Upload New Version</h3>
				<p class="mt-1 max-w-2xl text-sm text-gray-500">
					Choose a new version of the app package on your local file system.
				</p>
			</div>
			<div class="px-4 pt-5 sm:px-6 ">
				<SelectFiles :app_id="props.app_id" ></SelectFiles>
			</div>
		</div>
	</ViewWrap>
</template>

