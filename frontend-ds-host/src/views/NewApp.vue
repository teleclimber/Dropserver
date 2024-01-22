<script lang="ts" setup>
import { ref, Ref } from "vue";
import { useRouter } from 'vue-router';
import { useAppsStore } from '@/stores/apps';
import DataDef from '@/components/ui/DataDef.vue';
import ViewWrap from '../components/ViewWrap.vue';
import URLInput, {ValidatedURL} from "@/components/ui/URLInput.vue";
import SelectFiles from '../components/ui/SelectFiles.vue';

const router = useRouter();

const appsStore = useAppsStore();
appsStore.loadData();

const from_url :Ref<ValidatedURL>= ref({
	url: "",
	valid: false,
	message: "Please enter a link"
});

function urlChanged(data:ValidatedURL) {
	from_url.value = data;
}

async function submitFromURL() {
	if( !from_url.value.valid ) return;
	router.push({name: 'new-app-from-url', params:{url:from_url.value.url}});
}

</script>

<template>
	<ViewWrap>
		<!-- add links to existing in-process apps if any? -->
		<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
				<h3 class="text-lg leading-6 font-medium text-gray-900">
					Get App From Link
				</h3>
				<p class="mt-1 max-w-2xl text-sm text-gray-500">
					Paste a link to the app
				</p>
			</div>
			<form @submit.prevent="submitFromURL">
				<DataDef field="Link:" class="py-5">
					<URLInput @changed="urlChanged"></URLInput>
					<p class="mt-2 py-1 px-3 rounded-lg bg-gray-100" >
						{{ from_url.message }}
					</p>
				</DataDef>
				<div class="px-4 pb-5 sm:px-6 flex justify-end items-center">
					<input type="submit"
						class="btn-blue"
						:disabled="!from_url.valid"
						value="Fetch App" />
				</div>
			</form>
		</div>
		<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
				<h3 class="text-lg leading-6 font-medium text-gray-900">Upload New App</h3>
				<p class="mt-1 max-w-2xl text-sm text-gray-500">
					Choose a Dropserver app package on your local file system.
				</p>
			</div>
			<div class="px-4 pt-5 sm:px-6 ">
				<SelectFiles></SelectFiles>
			</div>
		</div>
	</ViewWrap>
</template>

