<template>
	<ViewWrap>
		<template v-if="step === 'pick'">
			<h3>Upload a New Version for {{app.name}}</h3>

			<SelectFiles @changed="filesChanged"></SelectFiles>

			<button @click="doUpload">Upload</button>
		</template>

		<template v-if="step === 'review'">
			<h3>Review New Version for {{app.name}}</h3>

			<p>{{uploadResp.version_metadata.name}}</p>

			<button @click="doCommit">Create</button>
		</template>
	</ViewWrap>
</template>


<script lang="ts">
import { defineComponent, ref, reactive, onMounted } from 'vue';
import type { Ref } from 'vue';
import {useRoute} from 'vue-router';
import router from '../router';

import { App, uploadNewAppVersion, commitNewApplication } from '../models/apps';
import type { SelectedFile, UploadVersionResp } from '../models/apps';

import ViewWrap from '../components/ViewWrap.vue';
import SelectFiles from '../components/ui/SelectFiles.vue';

export default defineComponent({
	name: 'NewAppVersion',
	components: {
		ViewWrap,
		SelectFiles
	},
	setup() {
		const route = useRoute();
		const app = reactive(new App);
		onMounted( async () => {
			await app.fetch(Number(route.params.id));
		});

		const step = ref("pick");
		const uploading = ref(false);
		const file_list :Ref<SelectedFile[]> = ref([]);
		const uploadResp :Ref<UploadVersionResp|null> = ref(null);

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
			console.log('app', app);
			router.push({name: 'manage-app', params:{id: app.app_id}});
		}
		
		return {
			app,
			step,
			filesChanged,
			doUpload,
			uploadResp,
			doCommit,
		};
	},
});


</script>