<script lang="ts" setup>
import { ComputedRef, ref, reactive, computed } from 'vue';
import { useRouter } from 'vue-router';

import { useAppsStore } from '@/stores/apps';
import { useAppspacesStore } from '@/stores/appspaces';
import { Appspace, LoadState } from '@/stores/types';
import { getLoadState } from '@/stores/loadable';

import ViewWrap from '../components/ViewWrap.vue';
import BigLoader from '../components/ui/BigLoader.vue';
import DataDef from '@/components/ui/DataDef.vue';
import MessageSad from '@/components/ui/MessageSad.vue';

import AppLicense from '@/components/app/AppLicense.vue';
import AppAuthorsSummary from '@/components/app/AppAuthorsSummary.vue';
import AppLinksCompact from '@/components/app/AppLinksCompact.vue';

const router = useRouter();

const props = defineProps<{
	app_id: number
}>();

const appsStore = useAppsStore();
appsStore.loadData();	// TODO No, this should be loadApp, but that requires some rethink in appsStore.
appsStore.loadAppVersions(props.app_id);

const appspacesStore = useAppspacesStore();
appspacesStore.loadData();

const app = computed( () => {
	if( !appsStore.is_loaded ) return undefined;
	const a = appsStore.getApp(Number(props.app_id));
	if( a ) return a.value;
	return undefined;
});
const app_icon_error = ref(false);
const app_icon = computed( () => {
	if( app_icon_error.value || !app.value?.cur_ver ) return "";
	return `/api/application/${app.value.app_id}/version/${app.value.cur_ver}/file/app-icon`;
});
const release_date = computed( () => {
	if( !app.value?.ver_data?.release_date ) return;
	return new Date(app.value?.ver_data.release_date).toLocaleDateString(undefined, {
		dateStyle:'medium'
	});
});
const app_versions = computed( () => {
	return appsStore.mustGetAppVersions(props.app_id);
});

const app_appspaces :ComputedRef<Appspace[]> = computed( () => {
	if( app.value === undefined ) return [];
	return appspacesStore.getAppspacesForApp(app.value.app_id).map( a => a.value );
}); 

const version_appspaces :ComputedRef<Map<string, Appspace[]>> = computed( () => {
	const ret: Map<string, Appspace[]> = new Map;
		app_appspaces.value.forEach( as => {
		const v = as.app_version
		if( !ret.has(v) ) ret.set(v, []);
		ret.get(v)!.push(as);
	});
	return ret;
});

const deleting_versions :Set<string> = reactive(new Set);

async function deleteVersion(version:string) {
	const v_as = version_appspaces.value.get(version);
	if( v_as && v_as.length !== 0 ) {
		alert("Can't delete an app version that is used by appspaces");
		return;
	}

	deleting_versions.add(version);
	await appsStore.deleteAppVersion(props.app_id, version);
}

const delete_app_ok = computed( () => {
	if( !appspacesStore.is_loaded ) return false;
	if( !app.value ) return false;
	return appspacesStore.getAppspacesForApp(app.value.app_id).length === 0;
});

const deleting_app = ref(false);

async function delApp() {
	deleting_app.value = true;
	await appsStore.deleteApp(props.app_id);
	router.push({name: 'apps'});
}

const last_check_str = computed( () => {
	if( !app.value?.url_data ) return '';
	const hours_ago = Math.round((Date.now() - app.value.url_data.last_dt.getTime())/1000/60/60);
	if( hours_ago <= 24 ) return hours_ago + ' hours ago';
	return app.value.url_data.last_dt.toLocaleDateString();
});

const loading_automatic = ref(false);
async function setAutomatic(auto :boolean) {
	if( app.value === undefined ) return;
	loading_automatic.value = true;
	await appsStore.changeAutomaticListingFetch(app.value.app_id, auto);
	loading_automatic.value = false;
}

</script>

<template>
	<ViewWrap>
		<template v-if="app">
			<div class="md:mb-6 my-6 pb-4 bg-white shadow overflow-hidden border-t-8 border-gray-200" :style="'border-color:'+(app.ver_data?.color || 'rgb(135, 151, 164)')" >
				<div class="grid app-grid gap-x-2 gap-y-2 px-4 py-4 sm:px-6">
					<img v-if="app_icon" :src="app_icon" @error="app_icon_error = true" class="w-20 h-20" />
					<div v-else class="w-20 h-20 text-gray-300  flex justify-center items-center">
						<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-14 h-14">
							<path stroke-linecap="round" stroke-linejoin="round" d="M2.25 15a4.5 4.5 0 004.5 4.5H18a3.75 3.75 0 001.332-7.257 3 3 0 00-3.758-3.848 5.25 5.25 0 00-10.233 2.33A4.502 4.502 0 002.25 15z" />
						</svg>
					</div>
					<div class="self-center">
						<h3 class="text-2xl leading-6 font-medium text-gray-900">{{app.ver_data?.name}}</h3>
						<p class="italic" v-if="app.ver_data?.short_desc">“{{app.ver_data?.short_desc}}”</p>
					</div>
					<div class="col-span-3 sm:col-start-2">
						<p class="">
							Version 
							<span class="bg-gray-200 text-gray-600 px-1 rounded-md">{{app.cur_ver}}</span>
							<span v-if="release_date"> released {{ release_date || '' }}</span>
						</p>
						<p class="">
							By: <AppAuthorsSummary :authors="app.ver_data?.authors" class=""></AppAuthorsSummary>
						</p>
						<AppLicense class="" :license="app.ver_data?.license"></AppLicense>
						<AppLinksCompact class="mt-3" :ver_data="app.ver_data"></AppLinksCompact>
					</div>
				</div>
			</div>

			<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
				<div class="px-4 pt-5 pb-2 sm:px-6 flex justify-between">
					<h3 class="text-lg leading-6 font-medium text-gray-900">Versions</h3>
					<div>
						<div v-if="app.url_data">
							<span class="italic text-gray-500 mr-2">last checked {{ last_check_str }}</span>
							<a href="#" class="btn" @click.stop.prevent="">check for upgrades</a>
						</div>
						<router-link v-else :to="{name: 'new-app-version', params:{id:app.app_id}}" class="btn">Upload New Version</router-link>
					</div>
				</div>
				<div v-if="app.url_data" class="px-4 sm:px-6 py-2">
					<p>This app is distributed from a website:</p>
					<p class="text-gray-800 italic">{{ app.url_data.url }}</p>
					<p v-if="app.url_data.last_result != 'ok'">
						Last attempt to get app versions resulted in a problem: {{ app.url_data.last_result }}
					</p>
					<p v-if="app.url_data.new_url">
						The app listing is now available at a new address: {{ app.url_data.new_url }}
						<!-- need a button to accept the new URL. -->
					</p>
					<p>
						<span v-if="app.url_data.automatic">
							The app listing is refreshed automatically to show new versions.
						</span>
						<span v-else>
							The app listing must be refreshed manually. Enable automatic refresh:
						</span>
						<button v-if="!loading_automatic" class="btn" @click.stop.prevent="setAutomatic(!app.url_data.automatic)">
							{{ app.url_data.automatic ? "disable" : "enable" }}
						</button>
						<span v-else class="text-gray-600 italic">hang on...</span>
					</p>
					<p v-if="app.cur_ver !== app.url_data.latest_version">
						New version is available: {{ app.url_data.latest_version }}
						<!-- it's likely that we have to block new version installation while "new_url" is there? -->
						<!-- We could also show new versions or any relevant version in the versions listing. That seems like the best place? -->
					</p>
				</div>
				<div class="grid grid-cols-4 items-stretch">
					<div></div>
					<div class="flex justify-center items-end font-medium text-center">installed:</div>
					<div class="flex justify-center items-end font-medium text-center">data schema:</div>
					<div></div>
					<template v-for="ver in app_versions" :key="ver.version">
						<div class="border-t py-1 pl-4 md:pl-6 flex flex-col md:flex-row items-start md:items-center justify-center md:justify-start">
							<span class=" font-medium bg-gray-200 text-gray-700 px-1 rounded-md">
								{{ver.version}}
							</span>
							<span v-if="ver.release_date" class="md:ml-1"> 
								({{ new Date(ver.release_date).toLocaleDateString(undefined, {dateStyle: 'short'}) }})
							</span>
						</div>
						<div class="border-t flex items-center justify-center">
							{{ver.created_dt.toLocaleDateString(undefined, {dateStyle: 'short'})}}
						</div>
						<div class="border-t py-2 flex items-center justify-center">
							{{ver.schema}}
						</div>
						<div class="border-t flex items-center justify-end pr-4 md:pr-6">
							<span v-if="deleting_versions.has(ver.version)">deleting</span>
							<span v-else-if="version_appspaces.get(ver.version)?.length" 
								class="bg-yellow-200 text-yellow-800 px-2 uppercase text-xs font-medium">in use</span>
							<button v-else @click.stop.prevent="deleteVersion(ver.version)" class="btn text-red-700">
								<svg xmlns="http://www.w3.org/2000/svg" class="inline align-bottom h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
									<path fill-rule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z" clip-rule="evenodd" />
								</svg>
								<span class="hidden sm:inline-block">delete</span>
							</button>
						</div>
					</template>
				</div>
			</div>

			<div class="md:mb-6 my-6 bg-white shadow sm:rounded-lg">
				<div class="px-4 py-5 sm:px-6 flex justify-between">
					<h3 class="text-lg leading-6 font-medium text-gray-900">Appspaces</h3>
					<div >
						<router-link :to="{name:'new-appspace', query:{app_id:app.app_id, version:app.cur_ver}}"
							class="btn whitespace-nowrap ">
							Create Appspace
						</router-link>
					</div>
				</div>

				<ul class="border-t border-b border-gray-200 divide-y divide-gray-200">
					<li v-for="appspace in app_appspaces" 
						:key="'appspace-'+appspace.appspace_id"
						class="py-2 px-4 md:px-6 flex flex-wrap items-baseline w-auto">
						<span class="mr-1 font-medium flex-shrink overflow-hidden text-ellipsis">{{appspace.domain_name}}</span>
						<span class="mr-1">({{appspace.app_version}})</span>
						<span class="flex-grow text-right">
							<router-link :to="{name: 'manage-appspace', params:{appspace_id:appspace.appspace_id}}" class="btn">Manage</router-link>
						</span>
					</li>
				</ul>

				<div class="">
					<MessageSad head="No Appspaces" v-if="app_appspaces.length === 0" class="">
						There are no appspaces using this app. 
						<router-link :to="{name:'new-appspace', query:{app_id:app.app_id, version:app.cur_ver}}"
								class="btn whitespace-nowrap ">
								Create one!
							</router-link>
					</MessageSad>
				</div>
			</div>

			<div class="md:mb-6 my-6 bg-yellow-100 shadow overflow-hidden sm:rounded-lg flex justify-between">
				<div class="px-4 py-5 sm:px-6 ">
					<h3 class="text-lg leading-6 font-medium text-gray-900">Delete App</h3>
					<p class="mt-1 max-w-2xl text-sm text-gray-700">
						<template v-if="delete_app_ok">
							Delete the app and all its versions.
						</template>
						<template v-else>
							Unable to delete: app is used by appspaces.
						</template>
					</p>
				</div>
				<div class="px-4 sm:px-6 flex justify-end">
					<button v-if="!deleting_app" @click.stop.prevent="delApp" class="btn btn-blue self-center" :disabled="!delete_app_ok">delete</button>
					<span v-else>Deleting...</span>
				</div>
			</div>
		</template>
		<BigLoader v-else></BigLoader> 
	</ViewWrap>
</template>


<style scoped>
.app-grid {
	grid-template-columns: 5rem 1fr max-content;
}
</style>