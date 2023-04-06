<script lang="ts" setup>
import { ComputedRef, ref, reactive, watch, onUnmounted, computed } from 'vue';
import { useRouter } from 'vue-router';

import { useAppsStore } from '@/stores/apps';
import { useAppspacesStore } from '@/stores/appspaces';
import { Appspace } from '@/stores/types'

import ViewWrap from '../components/ViewWrap.vue';
import BigLoader from '../components/ui/BigLoader.vue';

const router = useRouter();

const props = defineProps<{
	app_id: number
}>();

const appsStore = useAppsStore();
appsStore.loadData();

const appspacesStore = useAppspacesStore();
appspacesStore.loadData();

const app = computed( () => {
	if( !appsStore.is_loaded ) return undefined;
	const a = appsStore.getApp(Number(props.app_id));
	if( a ) return a.value;
	return undefined;
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

const latest_version = computed( () => {
	return app.value?.versions[0].version;	// some chance there are zero versions in app??
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

</script>

<template>
	<ViewWrap>
		<template v-if="app">
			<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
				<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
					<h3 class="text-lg leading-6 font-medium text-gray-900">Application</h3>
				</div>
				<div class="px-4 py-2 sm:px-6">
					<p>Name: {{app.name}}</p>
				</div>
				<div class="px-4 sm:px-6">
					<p>Created: {{app.created_dt.toLocaleString()}}</p>
				</div>
				<div class="px-4 py-5 sm:px-6">
					<h4 class="text-xl font-medium">Appspaces:</h4>
					<template v-if="app_appspaces.length !== 0">
						<ul >
							<li v-for="appspace in app_appspaces" :key="'appspace-'+appspace.appspace_id">
								<span class="text-xl">{{appspace.domain_name}}</span>
								({{appspace.app_version}})
								<router-link :to="{name: 'manage-appspace', params:{appspace_id:appspace.appspace_id}}" class="btn">Manage</router-link>
							</li>
						</ul>
						<div class="flex justify-end pt-3 ">
							<router-link :to="{name:'new-appspace', query:{app_id:app.app_id, version:latest_version}}" class="btn">Create Appspace</router-link>
						</div>
					</template>
					<div v-else class="bg-red-50 px-4 py-2 rounded flex justify-between items-baseline">
						<p class="">App not used by any appspaces.</p>
						<router-link :to="{name:'new-appspace', query:{app_id:app.app_id, version:latest_version}}"
							class="btn whitespace-nowrap ">
							Create Appspace
						</router-link>
					</div>
				</div>
			</div>

			<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
				<div class="px-4 py-5 sm:px-6 flex justify-between">
					<h3 class="text-lg leading-6 font-medium text-gray-900">Versions</h3>
					<div >
						<router-link :to="{name: 'new-app-version', params:{id:app.app_id}}" class="btn btn-blue">Upload New Version</router-link>
					</div>
				</div>

				<ul class="border-t border-b border-gray-200 divide-y divide-gray-200">
					<li v-for="ver in app.versions" :key="ver.version" class="px-4 sm:px-6 py-3 flex items-center justify-between text-sm">
						<div class="w-0 flex-1 flex items-center">
							<span class="ml-2 flex-1 w-0 font-bold">
								{{ver.version}}
							</span>
						</div>
						<div class="w-0 flex-1 flex items-center">
							<span class="ml-2 flex-1 w-0 ">
								{{ver.schema}}
							</span>
						</div>
						<div class="w-0 flex-1 flex items-center">
							<span class="ml-2 flex-1 w-0 ">
								{{ver.api_version}}
							</span>
						</div>
						<div class="w-0 flex-1 flex items-center">
							<span class="ml-2 flex-1 w-0">
								{{ver.created_dt.toLocaleString()}}
							</span>
						</div>
						<div class="">
							<span v-if="deleting_versions.has(ver.version)">deleting</span>
							<span v-else-if="version_appspaces.get(ver.version)?.length">in use</span>
							<button v-else @click.stop.prevent="deleteVersion(ver.version)" class="btn text-red-700">
								<svg xmlns="http://www.w3.org/2000/svg" class="inline align-bottom h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
									<path fill-rule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z" clip-rule="evenodd" />
								</svg>
								<span class="hidden sm:inline-block">delete</span>
							</button>
						</div>

					</li>
				</ul>

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

