<template>
	<ViewWrap>
		<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
				<h3 class="text-lg leading-6 font-medium text-gray-900">Upload New Version</h3>
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


<script lang="ts">
import { defineComponent, ref, reactive, onMounted, onUnmounted, computed } from 'vue';
import type { Ref } from 'vue';
import {useRoute} from 'vue-router';
import router from '../router';

import { App, uploadNewAppVersion } from '../models/apps';
import type { SelectedFile, AppGetMeta} from '../models/apps';

import {setTitle} from '../controllers/nav';

import ViewWrap from '../components/ViewWrap.vue';
import SelectFiles from '../components/ui/SelectFiles.vue';

export default defineComponent({
	name: 'NewAppVersion',
	components: {
		ViewWrap,
		SelectFiles,
	},
	setup() {
		const route = useRoute();
		const app = reactive(new App);
		onMounted( async () => {
			await app.fetch(Number(route.params.id));
			setTitle(app.name);
		});
		onUnmounted( () => {
			setTitle("");
		});

		const uploading = ref(false);
		const file_list :Ref<SelectedFile[]> = ref([]);

		function filesChanged(event_data:SelectedFile[]) {
			file_list.value = event_data;
		}

		async function doUpload() {
			if( !app.loaded ) return;
			if( file_list.value.length === 0 ) return;
			
			uploading.value = true;
			const app_get_key = await uploadNewAppVersion(app.app_id, file_list.value);
			uploading.value = false;

			router.push({name: 'new-app-version-in-process', params:{id:app.app_id, app_get_key}});
		}

		function resetUpload() {
			file_list.value = [];
		}
		
		return {
			app,
			filesChanged,
			doUpload,
			resetUpload
		};
	},
});


</script>