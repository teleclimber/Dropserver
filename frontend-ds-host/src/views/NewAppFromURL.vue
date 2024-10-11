<script lang="ts" setup>
import { ref, Ref, watch, watchEffect, computed } from "vue";
import { useRouter } from 'vue-router';
import { useAppsStore, rawToAppManifest } from '@/stores/apps';
import type { AppGetMeta } from '@/stores/types';
import ViewWrap from "@/components/ViewWrap.vue";
import MessageSad from "@/components/ui/MessageSad.vue";
import BigLoader from "@/components/ui/BigLoader.vue";
import AppCard from "@/components/app/AppCard.vue";
import Manifest from "@/components/app/Manifest.vue";
import Changelog from "@/components/app/Changelog.vue";

const props = defineProps<{
	url: string,
	version?: string
}>();

const router = useRouter();

const appsStore = useAppsStore();

const listing_versions :Ref<string[]|undefined> = ref();
const listing_error = ref("");
watch( () => props.url, async () => {
	listing_versions.value = undefined;
	try {
		const resp = await fetch(`/api/application/fetch/${encodeURIComponent(props.url)}/listing-versions`);
		console.log(resp);
		if( !resp.ok ) throw new Error(await resp.text());
		listing_versions.value = <string[]>await resp.json();
	}
	catch(e:any) {
		listing_error.value = e.message;
	}
}, {immediate: true});

const picked_version = ref("latest");
watch( () => props.version, async () => {
	picked_version.value = props.version || "latest";
}, {immediate: true});

watch( picked_version, () => {
	if( picked_version.value === "latest" /*&& props.version*/ ) {
		router.replace({name:'new-app-from-url', query:{version:undefined}})
	}
	else if( picked_version.value !== props.version ) {
		router.replace({name:'new-app-from-url', query:{version:picked_version.value}})
	}
}, {immediate: true});

const getMeta :Ref<AppGetMeta|undefined> = ref();
const manifest_error = ref("");
watchEffect( async () => {
	manifest_error.value = "";
	if( listing_versions.value && listing_versions.value.length !== 0 ) {
		getMeta.value = undefined;
		try {
			const v = props.version ? "version="+encodeURIComponent(props.version) :'';
			const resp = await fetch(`/api/application/fetch/${encodeURIComponent(props.url)}/manifest?${v}`);
			if( !resp.ok ) throw new Error(await resp.text());
			const meta = <AppGetMeta>await resp.json();
			meta.version_manifest = rawToAppManifest(meta.version_manifest);
			getMeta.value = meta;
		}
		catch(e:any) {
			manifest_error.value = e.message;
		}
	}
	else getMeta.value = undefined;
});

const changelog = ref("");
const changelog_error = ref("");
watchEffect( async () => {
	changelog.value = "";
	changelog_error.value = "";
	if( getMeta.value?.version_manifest?.version ) {
		try {
			const v = "version="+encodeURIComponent(getMeta.value?.version_manifest?.version);
			const resp = await fetch(`/api/application/fetch/${encodeURIComponent(props.url)}/changelog?${v}`);
			if( !resp.ok ) throw new Error(await resp.text());
			changelog.value = await resp.text();
		}
		catch(e:any) {
			changelog_error.value = e.message;
		}
	}
});

const icon_url = computed( () => {
	if( getMeta.value?.version_manifest?.version ) {
		const v = "version="+encodeURIComponent(getMeta.value?.version_manifest?.version);
		return `/api/application/fetch/${encodeURIComponent(props.url)}/icon?${v}`
	} 
	return '';
});

const has_error = computed( () => {	
	if( listing_error.value !== "" ) return true;
	if( manifest_error.value !== "" ) return true;

	if( getMeta.value?.errors.length ) return true;
});

const auto_refresh_listing = ref(true);

const submitting = ref(false);
async function doInstall() {
	let version = props.version || getMeta.value?.version_manifest?.version;
	if( !version ) return;
	submitting.value = true;
	const app_get_key = await appsStore.getNewAppFromURL(props.url, auto_refresh_listing.value, version);
	submitting.value = false;
	router.push({name: 'new-app-in-process', params:{app_get_key}});
}

async function cancel() {
	router.push( {name:'new-app'});
}

</script>

<template>
	<ViewWrap>
		<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
				<h3 class="text-lg leading-6 font-medium text-gray-900">
					Install new app from website:
				</h3>
				<p>
					{{ props.url }}
				</p>
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

			<MessageSad v-if="getMeta?.errors.length" head="Unable to install this app" class="m-6">
				<p v-for="e in getMeta.errors">{{ e }}</p>
			</MessageSad>

			<template v-if="getMeta?.version_manifest">
				<AppCard  :manifest="getMeta.version_manifest" :icon_url="icon_url"></AppCard>

				<Changelog class="mb-6 mx-auto max-w-xl" :changelog="changelog" :error="changelog_error"></Changelog>

				<Manifest :manifest="getMeta.version_manifest" :warnings="getMeta.warnings"></Manifest>

				<form @submit.prevent="doInstall" @keyup.esc="cancel">
					<label class="block mx-4 my-5 sm:mx-6 py-2 px-4 border rounded">
						<input type="checkbox" v-model="auto_refresh_listing">
						Automatically check for new versions
					</label>
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
			</template>

		</div>
	</ViewWrap>
</template>

