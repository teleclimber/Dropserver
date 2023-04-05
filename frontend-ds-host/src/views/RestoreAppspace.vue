<script lang="ts" setup>
import { ref, Ref, reactive, computed } from 'vue';
import { useRouter } from 'vue-router';
import { ax } from '../controllers/userapi';

import { useAppspacesStore } from '@/stores/appspaces';

import { AppspaceBackups } from '../models/appspace_backups';

import ViewWrap from '../components/ViewWrap.vue';
import DataDef from '../components/ui/DataDef.vue';
import MessageSad from '../components/ui/MessageSad.vue';

const props = defineProps<{
	appspace_id: number
}>();

const router = useRouter();

const appspacesStore = useAppspacesStore();
appspacesStore.loadData();

const upload_input_elem :Ref<HTMLInputElement|undefined> = ref();

const selected = ref("");

const step = ref("start");	// start, processing, confirm, progress

type AppspaceRestoreData = {
	loaded: boolean,
	err: {missing_files:string[], zip_files:string[]} | null,
	token: string,
	schema: number,
	//ds_api :number,
	// other stuff later
}

const restore_data :Ref<AppspaceRestoreData> = ref({
	loaded: false,
	err: null,
	token:"",
	schema: 0
});

const appspaceBackups = reactive(new AppspaceBackups(props.appspace_id));
appspaceBackups.fetchForAppspace();

function fileSelected() {
	if( upload_input_elem.value === null ) return;
	selected.value = 'upload';
}

const ok_to_next_step = computed( () => {
	if( selected.value === undefined ) return false;
	if( selected.value === "upload" ) {
		if( upload_input_elem.value === undefined ) return false;
		const files = upload_input_elem.value.files as FileList;
		if( files.length === 0 ) return false;
	}
	return true;
});

async function toProcessingStep() {
	if( !ok_to_next_step.value ) return;
	if( selected.value === "upload" ) {
		if( upload_input_elem.value === undefined ) return;
		const files = upload_input_elem.value.files as FileList;
		if( files.length === 0 ) return;
		step.value = "processing";
		restore_data.value = await uploadRestoreZip(props.appspace_id, files[0]);
		console.log("restore_data.err", restore_data.value.err);
	}
	else {
		// send that filename
		step.value = "processing";
		restore_data.value = await selectRestoreBackup(props.appspace_id, selected.value);
	}
}

async function commit() {
	if( !restore_data.value.loaded ) return;
	step.value = "restoring";
	await commitRestore(props.appspace_id, restore_data.value.token);
	appspacesStore.loadAppspace(props.appspace_id);
	// assuming everythign went well, go back to manage appspace page
	router.push({name: 'manage-appspace', params:{appspace_id:props.appspace_id}});
}

function backToStart() {
	step.value = "start";
	restore_data.value = {
		loaded: false,
		err: null,
		token:"",
		schema: 0
	};
}
function cancel() {
	router.push({name: 'manage-appspace', params:{appspace_id:props.appspace_id}});
}

async function uploadRestoreZip(appspace_id:number, zipFile :File) :Promise<AppspaceRestoreData> {	//return token
	const formData = new FormData();
	formData.append("zip", zipFile);
	const resp = await ax.postForm('/api/appspace/'+appspace_id+'/restore/upload', formData);

	const ret :AppspaceRestoreData = {
		loaded: true,
		err: resp.data.err || null,
		schema: Number(resp.data.schema),
		token:resp.data.token +''
	}
	return ret;
}

async function selectRestoreBackup(appspace_id:number, backup_file: string) :Promise<AppspaceRestoreData> {
	const resp = await ax.post('/api/appspace/'+appspace_id+'/restore/', {backup_file});
	const ret :AppspaceRestoreData = {
		loaded: true,
		err: resp.data.err || null,
		schema: Number(resp.data.schema),
		token:resp.data.token +''
	}
	return ret;
}

async function commitRestore(appspace_id:number, token: string) {
	await ax.post('/api/appspace/'+appspace_id+'/restore/'+token);
}

</script>

<template>
	<ViewWrap>
		<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200 flex justify-between">
				<h3 class="text-lg leading-6 font-medium text-gray-900">Restore Appspace from Backup</h3>
				<div>
					<a href="#" @click.stop.prevent="cancel" class="btn">Cancel</a>
				</div>
			</div>

			<form v-if="step === 'start'" @submit.prevent="toProcessingStep" @keyup.esc="cancel">
				<div>
					<label v-for="file in appspaceBackups.files" :key="file.name" class="px-4 py-3 sm:px-6 border-b border-gray-200 flex items-baseline">
						<input type="radio" class="mr-4" name="select_backup" :value="file.name" v-model="selected" />
						<div class="flex-grow">{{file.name}}</div>
						
						<div class="px-4">&nbsp;</div>
						
					</label>
					<div v-if="appspaceBackups.files.length === 0" class="px-4 py-3 sm:px-6 border-b border-gray-200">No backup files. Please upload one</div>
					<label class="px-4 py-3 sm:px-6 border-b border-gray-200 flex items-baseline">
						<input type="radio" class="mr-4" name="select_backup" value="upload" v-model="selected" />
						<span class="mr-2">Upload:</span>
						<input type="file" name="backup_file" ref="upload_input_elem" @input="fileSelected" @changed="fileSelected" accept=".zip"/>
					</label>

					<div class="px-4 py-5 sm:px-6 flex justify-between items-baseline">
						<input type="button" class="btn" @click="cancel" value="Cancel" />
						<input
							type="submit"
							class="btn-blue"
							:disabled="!ok_to_next_step"
							value="Next" />
					</div>
				</div>
			</form>

			<div v-else-if="step === 'processing' && !restore_data.loaded" class="px-4 py-3 sm:px-6 italic">
				Please wait...
			</div>
			<div v-else-if="step === 'processing' && restore_data.err">
				<MessageSad head="Error Processing Zip File" class="mx-0 sm:mx-4 my-6 sm:rounded-xl shadow">
					<template v-if="restore_data.err.missing_files.length">
						<p>Some necessary appspace data files or folders 
							are missing from the top level of the zip:</p>
						<ul class="list-disc">
							<li v-for="f in restore_data.err.missing_files" :key="'missing-'+f" class="ml-6">{{f}}</li>
						</ul>
						<p class="mt-2">Here are the files we found in the top level of the zip file:</p>
						<ul class="list-disc">
							<li v-for="f in restore_data.err.zip_files" :key="'zip-'+f" class="ml-6">{{f}}</li>
						</ul>
					</template>
				</MessageSad>
				<div class="mt-4 px-4 py-5 sm:px-6 border-t border-gray-200 flex justify-between items-baseline">
					<a href="#" @click="backToStart" class="btn">go back</a>
				</div>
			</div>
			<div v-else-if="step === 'processing'">
				<div class="border-b border-gray-200">
					<h4 class="px-4 py-5 sm:px-6">Data to Restore:</h4>
					<DataDef field="Schema:">
						{{restore_data.schema}}
					</DataDef>
				</div>
				<div class="px-4 py-5 sm:px-6 flex justify-between items-baseline">
					<a href="#" @click="backToStart" class="btn">go back</a>
					<button @click="commit()" class="btn btn-blue">Restore Now</button>
				</div>
			</div>
		</div>
	</ViewWrap>
</template>
