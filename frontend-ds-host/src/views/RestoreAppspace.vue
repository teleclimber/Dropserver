<template>
	<ViewWrap>
		<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200 flex justify-between">
				<h3 class="text-lg leading-6 font-medium text-gray-900">Restore Appspace from Backup</h3>
				<div>
					<a href="#" @click.stop.prevent="cancel" class="btn">Cancel</a>
				</div>
			</div>

			<div v-if="step === 'start'">
				<form ref="form_elem">
					<label v-for="file in appspaceBackups.files" :key="file.name" class="px-4 py-3 sm:px-6 border-b border-gray-200 flex items-baseline">
						<input type="radio" class="mr-4" name="select_backup" :value="file.name" />
						<div class="flex-grow">{{file.name}}</div>
						
						<div class="px-4">[size]</div>
						
					</label>
					<div v-if="appspaceBackups.files.length === 0" class="px-4 py-3 sm:px-6 border-b border-gray-200">No backup files. Please upload one</div>
					<label class="px-4 py-3 sm:px-6 border-b border-gray-200 flex items-baseline">
						<input type="radio" class="mr-4" name="select_backup" value="upload" />
						<span class="mr-2">Upload:</span>
						<input type="file" name="backup_file" ref="upload_input_elem" @input="fileSelected" @changed="fileSelected" accept=".zip"/>
					</label>
				</form>

				<div class="px-4 py-5 sm:px-6 flex justify-between items-baseline">
					<a href="#" @click="cancel" class="btn">cancel</a>
					<button @click="toProcessingStep()" class="btn btn-blue">Next</button>
				</div>
			</div>

			<div v-else-if="step === 'processing' && !restore_data.loaded" class="px-4 py-3 sm:px-6 italic">
				Please wait...
			</div>
			<div v-else-if="step === 'processing'">
				<div class="border-b border-gray-200">
					<h4 class="px-4 py-5 sm:px-6">Data to Restore:</h4>
					<DataDef field="Schema:">
						{{restore_data.schema}}
					</DataDef>
				</div>
				<div class="px-4 py-5 sm:px-6 flex justify-between items-baseline">
					<a href="#" @click="cancel" class="btn">cancel</a>
					<button @click="commit()" class="btn btn-blue">Restore Now</button>
				</div>
			</div>

		</div>
	</ViewWrap>
</template>

<script lang="ts">
import { defineComponent, ref, Ref, reactive, computed, onUnmounted, watchEffect, isReactive } from 'vue';
import router from '../router';

import { Appspace, uploadRestoreZip, selectRestoreBackup, commitRestore } from '../models/appspaces';
import type { AppspaceRestoreData } from '../models/appspaces';
import {AppspaceBackups} from '../models/appspace_backups';

import {setTitle} from '../controllers/nav';

import ViewWrap from '../components/ViewWrap.vue';
import DataDef from '../components/ui/DataDef.vue';

export default defineComponent({
	name: 'RestoreAppspace',
	components: {
		ViewWrap,
		DataDef
	},
	props: {
		appspace_id: {
			type: Number,
			required: true
		}
	},
	setup(props) {
		const upload_input_elem :Ref<HTMLInputElement|null> = ref(null);
		const form_elem :Ref<HTMLFormElement|null> = ref(null);

		const step = ref("start");	// start, processing, confirm, progress

		const restore_data :Ref<AppspaceRestoreData> = ref({
			loaded: false,
			token:"",
			schema: 0
		});

		const appspace = reactive(new Appspace);
		appspace.fetch(props.appspace_id);

		watchEffect( () => {
			setTitle(appspace.domain_name);
		});
		onUnmounted( () => {
			setTitle("");
		});

		const appspaceBackups = reactive(new AppspaceBackups(props.appspace_id));
		appspaceBackups.fetchForAppspace();

		function fileSelected() {
			if( upload_input_elem.value === null ) return;

			if( form_elem.value === null ) return;
			const radio_elem = form_elem.value.querySelector('input[name="select_backup"][value="upload"]');
			if( !radio_elem ) return;
			(<HTMLInputElement>radio_elem).checked = true;
		}

		async function toProcessingStep() {
			if( form_elem.value === null ) return;
			const sel_elem = form_elem.value.querySelector('input[name="select_backup"]:checked');
			if( sel_elem === null ) return;
			const selected_file = (<HTMLInputElement>sel_elem).value;
			if( !selected_file ) return;
			if( selected_file === "upload" ) {
				if( upload_input_elem.value === null ) return;
				const files = <FileList>upload_input_elem.value.files;
				if( files.length === 0 ) return;
				step.value = "processing";
				restore_data.value = await uploadRestoreZip(appspace.id, files[0]);
			}
			else {
				// send that filename
				step.value = "processing";
				restore_data.value = await selectRestoreBackup(appspace.id, selected_file);
			}
		}

		async  function commit() {
			if( !restore_data.value.loaded ) return;
			step.value = "restoring";
			await commitRestore(appspace.id, restore_data.value.token);
			// assuming everythign went well, go back to manage appspace page
			router.push({name: 'manage-appspace', params:{id:appspace.id}});
		}

		function cancel() {
			router.push({name: 'manage-appspace', params:{id:appspace.id}});
		}

		return {
			form_elem, upload_input_elem, fileSelected,
			step, toProcessingStep, commit, cancel,
			restore_data,
			appspace,
			appspaceBackups,
		}
	}
});

</script>