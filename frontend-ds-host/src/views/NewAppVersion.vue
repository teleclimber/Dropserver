<template>
	<ViewWrap>
		<div v-if="step === 'pick'" class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
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

		<div v-if="step === 'review' && !uploadResp.errors" class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
				<h3 class="text-lg leading-6 font-medium text-gray-900">Review New Version</h3>
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
					<button @click="doCommit" class="btn btn-blue">Create Version</button>
				</div>
			</div>
		</div>

		<div v-if="step === 'review' && uploadResp.errors" class="md:mb-6 my-6 bg-red-100 shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
				<h3 class="text-lg leading-6 font-medium text-gray-900">Errors Found in New Version</h3>
			</div>
			<div class="px-4 py-5 sm:px-6">
				<p v-for="err in uploadResp.errors">{{err}}</p>

				<div class="pt-5 flex">
					<button @click="resetUpload" class="btn">Back to Upload</button>
				</div>

			</div>
		</div>

		<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6">
				<h3 class="text-lg leading-6 font-medium text-gray-900">Existing Versions</h3>
			</div>

			<ul class="border-t border-b border-gray-200 divide-y divide-gray-200">
				<li v-for="ver in versions" :key="ver.version" class="pl-3 pr-4 py-3 flex items-center justify-between text-sm" :class="{'bg-yellow-100':uploaded_version === ver.version}">
					<div class="w-0 flex-1 flex items-center">
						<span class="ml-2 flex-1 w-0 font-bold">
							{{ver.version}}
						</span>
					</div>
					<div class="w-0 flex-1 flex items-center">
						<span class="ml-2 flex-1 w-0 ">
							{{ver.schema}}
						</span>
					</div>
					<div class="w-0 flex-1 flex items-center">
						<span class="ml-2 flex-1 w-0 ">
							{{ver.api_version}}
						</span>
					</div>
					<div v-if="uploaded_version === ver.version" class="w-0 flex-1 flex items-center">
						<span class="ml-2 flex-1 w-0">
							UPLOADED
						</span>
					</div>
					<div v-else class="w-0 flex-1 flex items-center">
						<span class="ml-2 flex-1 w-0">
							{{ver.created_dt.toLocaleString()}}
						</span>
					</div>
				</li>
			</ul>
		</div>
	</ViewWrap>
</template>


<script lang="ts">
import { defineComponent, ref, reactive, onMounted, onUnmounted, computed } from 'vue';
import type { Ref } from 'vue';
import {useRoute} from 'vue-router';
import router from '../router';

import { App, uploadNewAppVersion, commitNewApplication } from '../models/apps';
import type { SelectedFile, UploadVersionResp } from '../models/apps';

import {setTitle} from '../controllers/nav';

import ViewWrap from '../components/ViewWrap.vue';
import SelectFiles from '../components/ui/SelectFiles.vue';
import DataDef from '../components/ui/DataDef.vue';

export default defineComponent({
	name: 'NewAppVersion',
	components: {
		ViewWrap,
		SelectFiles,
		DataDef
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

		const step = ref("pick");
		const uploading = ref(false);
		const file_list :Ref<SelectedFile[]> = ref([]);
		const uploadResp :Ref<UploadVersionResp|null> = ref(null);
		const uploaded_version = computed( () => {
			if( uploadResp.value !== null && 
			uploadResp.value.errors === undefined && 
			uploadResp.value.version_metadata && 
			uploadResp.value.version_metadata.version ) {
				return uploadResp.value.version_metadata.version;
			}
			return "";
		});

		const versions = computed( () => {
			const ret = app.versions.slice();
			if( uploadResp.value && !uploadResp.value.errors && uploadResp.value.version_metadata !== undefined ) {
				const resp = uploadResp.value;
				if( resp.prev_version ) {
					const index = ret.findIndex( v => v.version === resp.prev_version );
					if( index !== -1 ) ret.splice(index, 0, uploadResp.value.version_metadata );
				}
				else if( resp.next_version ) {
					ret.push(uploadResp.value.version_metadata);
				}
			}
			return ret;
		});

		function filesChanged(event_data:SelectedFile[]) {
			console.log("files changed event data", event_data);
			file_list.value = event_data;
		}

		async function doUpload() {
			if( !app.loaded ) return;
			if( file_list.value.length === 0 ) return;
			
			uploading.value = true;
		
			uploadResp.value = await uploadNewAppVersion(app.app_id, file_list.value);

			uploading.value = false;
			step.value = "review";
		}
		async function doCommit() {
			if( uploadResp.value === null ) return
			const app = await commitNewApplication(uploadResp.value.key);
			router.push({name: 'manage-app', params:{id: app.app_id}});
		}

		function resetUpload() {
			step.value = "pick";
			file_list.value = [];
			uploadResp.value = null;
		}
		
		return {
			app,
			versions,
			step,
			filesChanged,
			doUpload,
			uploadResp,
			uploaded_version,
			doCommit,
			resetUpload
		};
	},
});


</script>