<template>
	<ViewWrap>
		<template v-if="step === 'pick-app'">
			<div class="flex justify-between">
				<h3 class="text-lg pl-4 sm:pl-0">Pick Application:</h3>
				<router-link to="/new-app" class="btn btn-blue">Upload New App</router-link>
			</div>

			<!-- sort, search for apps -->

			<div v-for="app in apps.asArray" :key="app.app_id" class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
				<div class="px-4 py-5 sm:px-6 flex flex-col sm:flex-row justify-between">
					<div>
						<h3 class="text-2xl leading-6 font-medium text-gray-900">{{app.name}}</h3>
						<p class="mt-1 max-w-2xl text-sm text-gray-500">
							Created: {{app.created_dt.toLocaleString()}} [app detail string]
						</p>
					</div>
					<div class="pt-4 sm:pt-0 self-end sm:self-center">
						<router-link :to="{name:'new-appspace', query:{app_id:app.app_id}}" class="btn btn-blue">
							Create Appspace
						</router-link>
					</div>
				</div>
			</div>

			<BigLoader v-if="!apps.loaded"></BigLoader>

			<MessageSad v-if="apps.loaded && apps.asArray.length === 0" head="There are no applications" class="mb-6 shadow sm:rounded-xl">
				You do not have any applications yet. Please visit the 
				<router-link class="text-blue-600 underline" to="/app">Apps</router-link>
				page and get some!
			</MessageSad>

		</template>

		<template v-if="step === 'settings'">
			<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
				<div class="px-4 py-5 sm:px-6 border-b border-gray-200 flex justify-between">
					<h3 class="text-lg leading-6 font-medium text-gray-900">Application: {{ app.loaded ? app.name : 'loading...' }}</h3>
					<div>
						<router-link :to="{name: 'new-appspace'}" class="btn">Back</router-link>
					</div>
				</div>

				<div v-if="!show_all_versions" class="px-4 py-5 sm:px-6 sm:grid sm:grid-cols-4">
					<div class="font-medium text-gray-500">
						Version:
					</div>
					<VersionDetails v-if="app_version !== undefined" :app_version="app_version" class="col-span-3"></VersionDetails>
					<p class="flex justify-end col-span-4 py-1">
						<button @click="show_all_versions = true" class="btn">Choose different version</button>
					</p>
				</div>
				<div v-else class="px-4 py-5 sm:px-6 sm:grid grid-cols-4">
					<div class="font-medium text-gray-500">
						Choose Version:
					</div>
					<PickVersion :versions="app.versions" class="col-span-3" @version="pickVersion" @close="show_all_versions = false"></PickVersion>
					<p v-if="!app.loaded">Loading application versions</p>
				</div>

				<div v-if="!show_all_versions" class="px-4 py-5 sm:px-6 sm:grid sm:grid-cols-4">
					<div class="font-medium text-gray-500">
						Auto-Update:
					</div>
					<div class="col-span-3">
						<div v-if="using_latest_version">[not implemented]</div>
						<div v-else>
							<button @click="pickLatestVersion()" class="btn">Pick latest version</button>
							to enable auto-update.
						</div>
					</div>
				</div>
			</div>

			<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
				<div class="px-4 py-5 sm:px-6 border-b border-gray-200">
					<h3 class="text-lg leading-6 font-medium text-gray-900">Appspace Settings:</h3>
				</div>

				<div class="px-4 py-5 sm:px-6 sm:grid sm:grid-cols-4">
					<div class="font-medium text-gray-500">
						[Sub]Domain:
					</div>
					<div class="col-span-3">
						[not implemented]
					</div>
				</div>
				<div class="px-4 py-5 sm:px-6 border-t border-gray-200 flex justify-between items-center">
					<router-link class="btn" to="/appspace">cancel</router-link>
					<button @click="create" class="btn-blue">Create</button>
				</div>
			</div>
			
		</template>
	</ViewWrap>
</template>


<script lang="ts">
import { defineComponent, ref, Ref, reactive, watch, watchEffect, computed } from 'vue';
import {useRoute} from 'vue-router';
import router from '../router';

import { App, Apps } from '../models/apps';
import { AppVersion } from '../models/app_versions';
import {createAppspace} from '../models/appspaces';

import ViewWrap from '../components/ViewWrap.vue';
import BigLoader from '../components/ui/BigLoader.vue';
import MessageSad from '../components/ui/MessageSad.vue';
import VersionDetails from '../components/VersionDetails.vue';
import PickVersion from '../components/PickAppVersion.vue';

export default defineComponent({
	name: 'NewAppspace',
	components: {
		ViewWrap,
		BigLoader,
		MessageSad,
		VersionDetails,
		PickVersion
	},
	setup() {
		const route = useRoute();
		const step = ref("");	// pick-app, settings

		const show_all_versions = ref(false);

		const apps = reactive(new Apps);
		const app :Ref<App|undefined> = ref();
		const app_version :Ref<AppVersion|undefined> = ref();

		watchEffect(() => {
			if( route.query.app_id !== undefined ) {
				app.value = reactive(new App);
				const app_id = Number(route.query.app_id);
				app.value.fetch(app_id);
				step.value = 'settings';
			}
			else {
				app.value = undefined;
				apps.fetchForOwner();
				step.value = 'pick-app';
			}
		});

		const picked_version = computed( () => {
			if( app.value != undefined && typeof route.query.version === 'string' && route.query.version !== '' ) {
				return route.query.version;
			} else {
				return '';
			}
		});

		const using_latest_version = computed( () => {
			if( app.value === undefined || !app.value.loaded ) return false;
			if( picked_version.value === '' ) return true;
			if( app.value.versions[0].version === picked_version.value ) return true;
			return false;
		});

		watchEffect( () => {
			if( app.value === undefined || !app.value.loaded ) return; // || !app.value.error
			if( picked_version.value !== '' ) {
				const av = app.value.versions.find( (v) => v.version === picked_version.value );
				if( av === undefined ) return;	//show error with version not found.
				app_version.value = av;
			}
			else {
				const avs = app.value.versions;
				if( avs.length === 0 ) return; // show error saying no versions in applicaion.
				app_version.value = avs[0];
			}
		});

		function pickVersion(version:string) {
			if( app.value === undefined ) return;
			show_all_versions.value = false;
			router.push({name: 'new-appspace', query:{app_id:app.value.app_id, version:version}});
		}
		function pickLatestVersion() {
			if( app.value === undefined  || !app.value.loaded ) return;
			router.push({name: 'new-appspace', query:{app_id:app.value.app_id, version:app.value.versions[0].version}});
		}

		async function create() {
			if( app.value === undefined || !app.value.loaded || app_version.value === undefined ) return;
			const appspace_id = await createAppspace(app.value.app_id, app_version.value.version);
			router.push({name: 'manage-appspace', params:{id: appspace_id+''}});
		}

		return {
			step,
			apps,
			app,
			picked_version,
			using_latest_version,
			app_version,
			create,
			show_all_versions,
			pickVersion,
			pickLatestVersion
		};
	},
});

</script>