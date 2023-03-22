<script lang="ts" setup>
import { ref, Ref, onMounted, watchEffect, onUnmounted } from 'vue';
import type {WatchStopHandle} from 'vue';
import { useRouter } from 'vue-router';
import { setTitle } from '@/controllers/nav';

import { useAppsStore } from '@/stores/apps';
import type { SelectedFile } from '@/stores/types';

import ViewWrap from '../components/ViewWrap.vue';
import SelectFiles from '../components/ui/SelectFiles.vue';

const router = useRouter();

const props = defineProps<{
	app_id?: number
}>();

const appsStore = useAppsStore();
appsStore.loadData();

let stopSetTitle :WatchStopHandle | undefined;
onMounted( () => {
	stopSetTitle = watchEffect( () => {
		if( props.app_id === undefined || !appsStore.is_loaded ) return;
		const app = appsStore.getApp(props.app_id);
		if( !app ) return;
		setTitle(app.value.name);
	});
});

onUnmounted( () => {
	if( stopSetTitle) stopSetTitle();
	setTitle("");
});



const uploading = ref(false);
const file_list :Ref<SelectedFile[]> = ref([]);

function filesChanged(event_data:SelectedFile[]) {
	file_list.value = event_data;
}

async function doUpload() {
	if( file_list.value.length === 0 ) return;

	if( props.app_id === undefined ) {
		uploading.value = true;
		const app_get_key = await appsStore.uploadNewApplication(file_list.value);
		uploading.value = false;
		router.push({name: 'new-app-in-process', params:{app_get_key}});
	}
	else {
		// upload version.
		const app_get_key = await appsStore.uploadNewAppVersion(props.app_id, file_list.value);
		uploading.value = false;
		router.push({name: 'new-app-version-in-process', params:{id:props.app_id, app_get_key}});
	}
}

</script>

<template>
	<ViewWrap>
		<!-- add links to existing in-process apps if any -->
		<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
				<h3 class="text-lg leading-6 font-medium text-gray-900">
					{{ app_id === undefined ? "Upload New App" : "Upload New Versoin" }}
				</h3>
				<p class="mt-1 max-w-2xl text-sm text-gray-500">
					Choose a directory on your local file system that contains the application code.
				</p>
			</div>
			<div class="px-4 py-5 sm:px-6 flex justify-between items-center">

				<SelectFiles @changed="filesChanged"></SelectFiles>

				<button @click="doUpload" class="btn btn-blue">Upload</button>
			</div>
		</div>
	</ViewWrap>
</template>

