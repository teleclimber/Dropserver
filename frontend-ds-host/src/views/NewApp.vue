<template>
	<ViewWrap>
		<!-- add links to existing in-process apps if any -->
		<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
				<h3 class="text-lg leading-6 font-medium text-gray-900">Upload</h3>
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
import { defineComponent, ref } from 'vue';
import type { Ref } from 'vue';
import router from '../router';

import { uploadNewApplication } from '../models/apps';
import type { SelectedFile } from '../models/apps';

import ViewWrap from '../components/ViewWrap.vue';
import SelectFiles from '../components/ui/SelectFiles.vue';

export default defineComponent({
	name: 'NewApp',
	components: {
		ViewWrap,
		SelectFiles,
	},
	setup() {
		const uploading = ref(false);
		const file_list :Ref<SelectedFile[]> = ref([]);

		function filesChanged(event_data:SelectedFile[]) {
			file_list.value = event_data;
		}

		async function doUpload() {
			if( file_list.value.length === 0 ) return;
			
			uploading.value = true;
		
			const app_get_key = await uploadNewApplication(file_list.value);

			uploading.value = false;

			router.push({name: 'new-app-in-process', params:{app_get_key}});
		}


		function resetUpload() {	//unused?
			file_list.value = [];
		}
		
		return {
			filesChanged,
			doUpload,
			resetUpload
		};
	},
});

</script>