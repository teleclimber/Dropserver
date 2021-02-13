<template>
	<ViewWrap>
		<div v-if="step === 'pick'" class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
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

		<div v-if="step === 'review' && !uploadResp.errors" class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
				<h3 class="text-lg leading-6 font-medium text-gray-900">Review</h3>
			</div>
			<div class="px-4 py-5 sm:px-6">
				<dl class="border border-gray-200 rounded divide-y divide-gray-200">
					<DataDef field="App Name">{{uploadResp.version_metadata.name}}</DataDef>
					<DataDef field="Version">{{uploadResp.version_metadata.version}}</DataDef>
					<DataDef field="App Schema">{{uploadResp.version_metadata.schema}}</DataDef>
					<DataDef field="DropServer API">{{uploadResp.version_metadata.api_version}}</DataDef>
				</dl>
				
				<div class="pt-5 flex justify-between">
					<button @click="resetUpload" class="btn">Back to Upload</button>
					<button @click="doCommit" class="btn btn-blue">Create Application</button>
				</div>
			</div>
		</div>

		<div v-if="step === 'review' && uploadResp.errors" class="md:mb-6 my-6 bg-red-100 shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
				<h3 class="text-lg leading-6 font-medium text-gray-900">Errors Found</h3>
			</div>
			<div class="px-4 py-5 sm:px-6">
				<p v-for="err in uploadResp.errors">{{err}}</p>

				<div class="pt-5 flex">
					<button @click="resetUpload" class="btn">Back to Upload</button>
				</div>

			</div>
		</div>
	</ViewWrap>
</template>


<script lang="ts">
import { defineComponent, ref, reactive, onMounted } from 'vue';
import type { Ref } from 'vue';
import router from '../router';

import { App, uploadNewApplication, commitNewApplication } from '../models/apps';
import type { SelectedFile, UploadVersionResp } from '../models/apps';

import ViewWrap from '../components/ViewWrap.vue';
import SelectFiles from '../components/ui/SelectFiles.vue';
import DataDef from '../components/ui/DataDef.vue';

export default defineComponent({
	name: 'NewApp',
	components: {
		ViewWrap,
		SelectFiles,
		DataDef
	},
	setup() {
		const step = ref("pick");
		const uploading = ref(false);
		const file_list :Ref<SelectedFile[]> = ref([]);
		const uploadResp :Ref<UploadVersionResp|null> = ref(null);

		function filesChanged(event_data:SelectedFile[]) {
			console.log("files changed event data", event_data);
			file_list.value = event_data;
		}

		async function doUpload() {
			if( file_list.value.length === 0 ) return;
			
			uploading.value = true;
		
			uploadResp.value = await uploadNewApplication(file_list.value);

			uploading.value = false;
			step.value = "review";
		}
		async function doCommit() {
			if( uploadResp.value === null ) return
			const app = await commitNewApplication(uploadResp.value.key);
			console.log('app', app);
			router.push({name: 'manage-app', params:{id: app.app_id}});
		}

		function resetUpload() {
			step.value = "pick";
			file_list.value = [];
			uploadResp.value = null;
		}
		
		return {
			step,
			filesChanged,
			doUpload,
			uploadResp,
			doCommit,
			resetUpload
		};
	},
});

</script>