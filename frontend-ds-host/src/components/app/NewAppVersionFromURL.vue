<script lang="ts" setup>
import { ref, Ref, watch, computed } from "vue";
import { useRouter } from 'vue-router';
import { useAppsStore, AppGetMeta } from '@/stores/apps';
import MessageSad from "@/components/ui/MessageSad.vue";
import BigLoader from "@/components/ui/BigLoader.vue";
import AppCard from "./AppCard.vue";
import Manifest from "./Manifest.vue";

const props = defineProps<{
	app_id: number,
	version?: string
}>();

const router = useRouter();

const appsStore = useAppsStore();

const getMeta :Ref<AppGetMeta|undefined> = ref();
const manifest_error = ref("");
watch( props, async () => {
	manifest_error.value = "";
	try {
		getMeta.value = await appsStore.fetchVersionManifest(props.app_id, props.version);
	}
	catch(e) {
		manifest_error.value = "error fetching listing versions"
	}
}, {immediate: true});

const listing_versions :Ref<string[]|undefined> = ref();
const listing_error = ref("");
watch( props, async () => {
	try {
		const resp = await fetch(`/api/application/${props.app_id}/listing-versions`);
		listing_versions.value = <string[]>await resp.json();
	}
	catch(e) {
		listing_error.value = "error fetching listing versions"
	}
}, {immediate: true});

const picked_version = ref("latest");
watch( () => props.version, async () => {
	picked_version.value = props.version || "latest";
}, {immediate: true});

watch( picked_version, () => {
	if( picked_version.value === "latest" /*&& props.version*/ ) {
		router.replace({name:'new-app-version', query:{version:undefined}})
	}
	else if( picked_version.value !== props.version ) {
		router.replace({name:'new-app-version', query:{version:picked_version.value}})
	}
}, {immediate: true});

const has_error = computed( () => {	
	if( listing_error.value !== "" ) return true;
	if( manifest_error.value !== "" ) return true;

	if( getMeta.value?.errors.length ) return true;
});

const submitting = ref(false);
async function doInstall() {
	let version = props.version || getMeta.value?.version_manifest?.version;
	if( !version ) return;
	submitting.value = true;
	const app_get_key = await appsStore.getNewVersionFromURL(props.app_id, version);
	submitting.value = false;
	router.push({name: 'new-app-in-process', params:{app_get_key}});
}

async function cancel() {
	router.push( {name:'manage-app', params:{id:props.app_id}});
}

</script>

<template>
	<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
		<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
			<h3 class="text-lg leading-6 font-medium text-gray-900">
				Install version from application website:
			</h3>
			<div class="mt-2">
				Select version: 
				<select ref="pick_version" v-model="picked_version" class="shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
					<option value="latest">Latest</option>
					<option v-for="v in listing_versions" :key="'version-'+v" :value="v">{{v}}</option>
				</select>
			</div>
		</div>

		<MessageSad v-if="listing_error" head="Problem fetching listing" class="m-6">
			{{ listing_error }}
		</MessageSad>
		<MessageSad v-else-if="manifest_error" head="Problem fetching manifest" class="m-6">
			{{ manifest_error }}
		</MessageSad>
		<BigLoader v-else-if="listing_versions === undefined"></BigLoader>

		<MessageSad v-if="getMeta?.errors.length" head="Unable to install this version" class="m-6">
			<p v-for="e in getMeta.errors">{{ e }}</p>
		</MessageSad>

		<AppCard v-if="getMeta?.version_manifest" :manifest="getMeta.version_manifest" :icon_url="''"></AppCard>
		<Manifest v-if="getMeta?.version_manifest" :manifest="getMeta.version_manifest" :warnings="getMeta.warnings"></Manifest>

		<form @submit.prevent="doInstall" @keyup.esc="cancel">
			<div class="px-4 py-5 sm:px-6 flex justify-between">
				<input type="button" class="btn" @click="cancel" value="Cancel" />
				<input
					ref="create_button"
					type="submit"
					class="btn-blue"
					:disabled="has_error || submitting"
					value="Install" />
			</div>
		</form>

	</div>
</template>

