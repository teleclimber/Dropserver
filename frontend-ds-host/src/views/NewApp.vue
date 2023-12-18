<script lang="ts" setup>
import { ref, computed } from "vue";
import { useRouter } from 'vue-router';
import { useAppsStore } from '@/stores/apps';
import DataDef from '@/components/ui/DataDef.vue';
import ViewWrap from '../components/ViewWrap.vue';
import SelectFiles from '../components/ui/SelectFiles.vue';

const router = useRouter();

const appsStore = useAppsStore();
appsStore.loadData();

const from_url = ref("");

const from_url_normalized = computed( () => {
	if( from_url.value === "" ) return "";
	let u = from_url.value.trim().toLowerCase();
	if( !u.startsWith("http://") && !u.startsWith("https://") ) u = "https://"+u;
	return u;
});

const from_url_valid = computed( () => {
	if( from_url_normalized.value === "" ) return "";
	let u :URL|undefined;
	try {
		u = new URL(from_url_normalized.value);
	}
	catch {
		return "Please check the link, it appears to be invalid.";
	}
	if( u.protocol !== "https:" ) {
		return "Please use a secure https:// URL.";
	}
	return "";
});

const url_message = computed( () => {
	if( from_url.value.trim() === "" ) return "Please enter a link";
	if( from_url_valid.value !== "" ) return from_url_valid.value;
	if( from_url.value.toLowerCase() !== from_url_normalized.value ) return "OK: "+from_url_normalized.value;
	return "OK";
});

async function submitFromURL() {
	if( from_url_valid.value !== "" ) return;
	router.push({name: 'new-app-from-url', params:{url:from_url.value}});
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
					<input type="text" v-model="from_url" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
					<p class="mt-2 py-1 px-3 rounded-lg bg-gray-100" >
						{{ url_message }}
					</p>
				</DataDef>
				<div class="px-4 pb-5 sm:px-6 flex justify-end items-center">
					<input type="submit"
						class="btn-blue"
						:disabled="from_url === '' || from_url_valid !== ''"
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

