<script lang="ts" setup>
import { ComputedRef, ref, Ref, reactive, computed } from 'vue';
import { useRouter } from 'vue-router';

import { useAppsStore } from '@/stores/apps';
import { useAppspacesStore } from '@/stores/appspaces';
import { Appspace } from '@/stores/types';

import ViewWrap from '../components/ViewWrap.vue';
import BigLoader from '../components/ui/BigLoader.vue';
import MessageSad from '@/components/ui/MessageSad.vue';

import URLInput, {ValidatedURL} from '@/components/ui/URLInput.vue';

import AppLicense from '@/components/app/AppLicense.vue';
import AppAuthorsSummary from '@/components/app/AppAuthorsSummary.vue';
import AppLinksCompact from '@/components/app/AppLinksCompact.vue';
import DataDef from '@/components/ui/DataDef.vue';

const router = useRouter();

const props = defineProps<{
	app_id: number
}>();

const appsStore = useAppsStore();
appsStore.loadApp(props.app_id);
appsStore.loadAppVersions(props.app_id);

const appspacesStore = useAppspacesStore();
appspacesStore.loadData();

const app = computed( () => {
	const a = appsStore.getApp(props.app_id);
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

const show_change_url = ref(false);
const new_url = ref("");
function showChangeURL() {
	if( !app.value?.url_data ) return;
	const ud = app.value?.url_data;
	new_url.value = ud.url;
	if( ud.new_url ) new_url.value = ud.new_url;
	submitting_url_change.value = false;
	show_change_url.value = true;
}
const from_url :Ref<ValidatedURL>= ref({
	url: "",
	valid: false,
	message: "Please enter a link"
});

function urlChanged(data:ValidatedURL) {
	from_url.value = data;
}
function cancelChangeURL() {
	show_change_url.value = false;
}
const submitting_url_change = ref(false);
async function submitChangeURL() {
	if( !from_url.value.valid ) return;
	submitting_url_change.value = true;
	const problem = await appsStore.changeAppURL(props.app_id, from_url.value.url);
	submitting_url_change.value = false;
	if( problem !== '' ) alert(problem);
	else show_change_url.value = false;
}

const last_check_str = computed( () => {
	if( !app.value?.url_data ) return '';
	let time_ago = Math.round((Date.now() - app.value.url_data.last_dt.getTime())/1000/60);
	if( time_ago < 60 ) return time_ago + ' minutes ago';
	time_ago = Math.round(time_ago/60);
	if( time_ago <= 24 ) return time_ago + ' hours ago';
	return app.value.url_data.last_dt.toLocaleDateString();
});

async function refreshListing() {
	if( app.value === undefined ) return;
	const err = await appsStore.refreshListing(app.value.app_id);
	if( err ) alert(err);
}

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
						<router-link v-if="!app.url_data" :to="{name: 'new-app-version', params:{id:app.app_id}}" class="btn">Upload New Version</router-link>
						<template v-else-if="!app.url_data.new_url && app.url_data.last_result !== 'error'">
							<p v-if="app.cur_ver !== app.url_data.latest_version">
								<span class="bg-yellow-100 px-1">New version: {{ app.url_data.latest_version }}</span>
								<router-link :to="{name:'new-app-version', query:{version:app.url_data.latest_version}}"
									class="btn whitespace-nowrap ">
									Get it
								</router-link>
							</p>
							<p v-else>
								<router-link :to="{name:'new-app-version', query:{version:undefined}}"
									class="btn whitespace-nowrap ">
									Get a different version
								</router-link>
							</p>
						</template>
					</div>
				</div>
				<div v-if="app.url_data" class="my-2">
					<template v-if="show_change_url">
						<form @submit.prevent="submitChangeURL" @keyup.esc="cancelChangeURL">
							<DataDef field="Change Link:" class="py-5">
								<div class="bg-blue-100 p-2 rounded">
									<URLInput @changed="urlChanged" :initial_value="new_url" ></URLInput>
									<p class="my-2 py-1 px-3 rounded-lg bg-gray-100" >
										{{ from_url.message }}
									</p>
									<div class="flex justify-between items-center">
										<input type="button" class="btn" @click="cancelChangeURL" value="Cancel" />
										<span v-if="submitting_url_change" class="text-gray-600">
											hang on...
										</span>
										<input type="submit"
											v-else
											class="btn-blue"
											:disabled="!from_url.valid"
											value="Change Link" />
									</div>
								</div>
							</DataDef>
						</form>
					</template>
					<template v-else>
						<DataDef field="App Listing Address:">
							<p class="italic">
								{{ app.url_data.url }}
								<button v-if="!app.url_data.new_url" class="btn" @click.stop.prevent="showChangeURL">change</button>
							</p>
							<template v-if="app.url_data.new_url">
								<p class=" text-orange-600">
									<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-5 h-5 inline-block align-bottom">
										<path fill-rule="evenodd" d="M8.485 2.495c.673-1.167 2.357-1.167 3.03 0l6.28 10.875c.673 1.167-.17 2.625-1.516 2.625H3.72c-1.347 0-2.189-1.458-1.515-2.625L8.485 2.495zM10 5a.75.75 0 01.75.75v3.5a.75.75 0 01-1.5 0v-3.5A.75.75 0 0110 5zm0 9a1 1 0 100-2 1 1 0 000 2z" clip-rule="evenodd" />
									</svg>
									The app listing is now available at a new address:
								</p>
								<p class="italic">
									{{ app.url_data.new_url }}
									<button class="btn" @click.stop.prevent="showChangeURL">accept</button>
								</p>
							</template>
						</DataDef>
						<DataDef field="Last Refreshed:">
							{{ last_check_str }}
							<button href="#" class="btn" @click.stop.prevent="refreshListing">refresh listing</button>
							<p v-if="app.url_data.last_result === 'error'" class=" text-orange-600">
								<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-5 h-5 inline-block align-bottom">
									<path fill-rule="evenodd" d="M8.485 2.495c.673-1.167 2.357-1.167 3.03 0l6.28 10.875c.673 1.167-.17 2.625-1.516 2.625H3.72c-1.347 0-2.189-1.458-1.515-2.625L8.485 2.495zM10 5a.75.75 0 01.75.75v3.5a.75.75 0 01-1.5 0v-3.5A.75.75 0 0110 5zm0 9a1 1 0 100-2 1 1 0 000 2z" clip-rule="evenodd" />
								</svg>
								Last attempt to get app versions encountered an error.
							</p>
						</DataDef>
						<DataDef field="Automatic Refresh:">
							{{ app.url_data.automatic ? "automatic" : "manual" }}
							<button v-if="!loading_automatic" class="btn" @click.stop.prevent="setAutomatic(!app?.url_data?.automatic)">
								{{ app.url_data.automatic ? "disable" : "enable" }}
							</button>
							<span v-else class="text-gray-600 italic">hang on...</span>
						</DataDef>
					</template>
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