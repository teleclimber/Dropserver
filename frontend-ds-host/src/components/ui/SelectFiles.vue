<script lang="ts" setup>
import {Ref, ref, computed, ShallowRef, onMounted} from 'vue';
import { useRouter } from 'vue-router';
import { useAppsStore } from '@/stores/apps';

const props = defineProps<{
	app_id?: number

}>();

const emits = defineEmits<{
	(e:'uploading') :void
}>();

const router = useRouter();

const appsStore = useAppsStore();
appsStore.loadData();

onMounted( () => {
	focusInput();
});

const input_elem :Ref<HTMLInputElement|undefined> = ref();
function focusInput() {
	if( input_elem.value === undefined ) return;
	input_elem.value.focus();
}
const selected_file :ShallowRef<File|undefined> = ref();
function selected() {
	if( input_elem.value?.files?.length === 1 ){
		selected_file.value = input_elem.value.files[0];
	} 
	else {
		selected_file.value = undefined;
	}
	focusInput();
}

const invalid = computed( () => {
	if( selected_file.value === undefined ) return true;
	if( selected_file.value.type !== 'application/x-gzip' ) return true;
	return false;
});

const uploading = ref(false);
async function submit() {
	if( input_elem.value === undefined ) return;
	
	const files = input_elem.value.files as FileList;
	if( files.length !== 1 ) return;

	const f = files[0];

	uploading.value = true;

	if( props.app_id === undefined ) {
		uploading.value = true;
		const app_get_key = await appsStore.uploadNewApplication(f);
		uploading.value = false;
		router.push({name: 'new-app-in-process', params:{app_get_key}});
	}
	else {
		// upload version.
		const app_get_key = await appsStore.uploadNewAppVersion(props.app_id, f);
		uploading.value = false;
		router.push({name: 'new-app-version-in-process', params:{id:props.app_id, app_get_key}});
	}
}

</script>
<template>
	<form @submit.prevent="submit">  <!--  @keyup.esc="cancel" causes back when escaping out of file select dialog :( -->
				
		<div class="">
			<input type="file"
				accept=".tgz, .tar.gz, application/x-gzip"
				name="app_dir"
				ref="input_elem"
				@input="selected"
				class="border-4 rounded-3xl border-blue-300 border-dashed bg-blue-50  py-16 pl-4 w-full
					outline-offset-2 outline-2 outline-blue-500" />
		</div>

		<p v-if="selected_file !== undefined && selected_file.type !== 'application/x-gzip'" 
			class="border-2 p-1 mt-4 rounded-lg bg-red-50 border-red-400 text-red-700">
			Incorrect file type: {{ selected_file.type }}
		</p>
		<p v-else class="border-2 p-1 mt-4 rounded-lg border-transparent bg-gray-50">&nbsp;</p>

		<div class="py-5 flex items-baseline justify-end">
			<input
				type="submit"
				class="btn-blue"
				:disabled="invalid"
				value="Upload" />
		</div>
	
	</form>
</template>

