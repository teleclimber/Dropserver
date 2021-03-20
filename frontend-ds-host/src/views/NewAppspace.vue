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

			<MessageSad v-if="apps.loaded && apps.asArray.length === 0" head="There are no applications" class="my-6 shadow sm:rounded-xl">
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
						<router-link :to="{name: 'new-appspace'}" class="btn">Pick Application</router-link>
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
						Base Domain:
					</div>
					<div class="col-span-3">
						<select v-model="domain_name" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
							<option value="">Pick Domain Name</option>
							<option v-for="d in domain_names.for_appspace" :key="d.domain_name" :value="d.domain_name">{{d.domain_name}}</option>
						</select>
					</div>
				</div>
				<div class="px-4 pb-5 sm:px-6 sm:grid sm:grid-cols-4">
					<div class="font-medium text-gray-500">
						Subdomain:
					</div>
					<div class="col-span-3">
						<input type="text" v-model="subdomain" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
						<p class="pt-2 flex justify-between">
							<div class="text-lg font-medium">{{full_domain}}</div>
							<div>{{domain_valid}}</div>
						</p>
					</div>
				</div>
				<div class="px-4 py-5 sm:px-6 sm:grid sm:grid-cols-4">
					<div class="font-medium text-gray-500">
						DropID:
					</div>
					<div class="col-span-3">
						<select v-model="dropid" class="w-full shadow-sm border border-gray-300 focus:ring-indigo-500 focus:border-indigo-500 rounded-md">
							<option value="">Pick DropID</option>
							<option v-for="d in dropids.dropids" :key="d.key" :value="d">{{d.key}}</option>
						</select>
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
import { DomainNames, checkAppspaceDomain } from '../models/domainnames';
import { createAppspace } from '../models/appspaces';
import type { NewAppspaceData } from '../models/appspaces';

import ViewWrap from '../components/ViewWrap.vue';
import BigLoader from '../components/ui/BigLoader.vue';
import MessageSad from '../components/ui/MessageSad.vue';
import VersionDetails from '../components/VersionDetails.vue';
import PickVersion from '../components/PickAppVersion.vue';
import { DropIDs, DropID } from '../models/dropids';

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
		const app = ref(new App);
		const app_version :Ref<AppVersion|undefined> = ref();

		const domain_names = reactive(new DomainNames);
		domain_names.fetchForOwner();

		const domain_name = ref("");
		const subdomain = ref("");

		const full_domain = computed( () => {
			let ret = domain_name.value;
			if( subdomain.value != "" ) ret = subdomain.value + "."+ domain_name.value;
			return ret;
		});

		const domain_valid = ref("");
		watch( [domain_name, subdomain], async () => {
			subdomain.value = subdomain.value.trim();

			if( domain_name.value === '' ) {
				domain_valid.value = '';
				return;
			}

			const domain_data = domain_names.for_appspace.find( d => d.domain_name === domain_name.value );
			if( domain_data === undefined ) return;

			if( subdomain.value === "" && domain_data.appspace_subdomain_required ) {
				domain_valid.value = 'subdomain required';
				return;
			}
			if( subdomain.value.length > 62 ) {
				domain_valid.value = 'long';
				return;
			}
			// check for bad chars
			
			// Here we query the server to see if the id already exists.
			// Note this is a pretty poor way to do this.
			domain_valid.value = 'checking';
			const check = await checkAppspaceDomain(domain_name.value, subdomain.value)
			if( !check.valid ) domain_valid.value = "Invalid: "+check.message;
			else if( !check.available ) domain_valid.value = "Unavailable: "+check.message;
			else domain_valid.value = "OK";
		});

		// Dropid
		const dropid :Ref<DropID|undefined> = ref();
		const dropids = reactive(new DropIDs);
		dropids.fetchForOwner();


		watchEffect(() => {
			if( route.query.app_id !== undefined ) {
				const app_id = Number(route.query.app_id);
				app.value.fetch(app_id);
				step.value = 'settings';
			}
			else {
				app.value = new App;
				apps.fetchForOwner();
				step.value = 'pick-app';
			}
		});

		const picked_version = computed( () => {
			if( app.value.loaded && typeof route.query.version === 'string' && route.query.version !== '' ) {
				return route.query.version;
			} else {
				return '';
			}
		});

		const using_latest_version = computed( () => {
			if( !app.value.loaded ) return false;
			if( picked_version.value === '' ) return true;
			if( app.value.versions[0].version === picked_version.value ) return true;
			return false;
		});

		watchEffect( () => {
			if( !app.value.loaded ) return; // || !app.value.error
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
			if( !app.value.loaded ) return;
			show_all_versions.value = false;
			router.push({name: 'new-appspace', query:{app_id:app.value.app_id, version:version}});
		}
		function pickLatestVersion() {
			if( !app.value.loaded ) return;
			router.push({name: 'new-appspace', query:{app_id:app.value.app_id, version:app.value.versions[0].version}});
		}

		async function create() {
			if( !app.value.loaded || app_version.value === undefined ) return;
			// TODO also bail if domain is not valid, etc...
			if( dropid.value === undefined ) return;

			const data :NewAppspaceData = {
				app_id: app.value.app_id,
				app_version: app_version.value.version,
				domain_name: domain_name.value,
				subdomain: subdomain.value,
				dropid: dropid.value.key,
			};
			const appspace_id = await createAppspace(data);
			router.push({name: 'manage-appspace', params:{id: appspace_id+''}});
		}

		return {
			step,
			apps,
			app,
			picked_version,
			using_latest_version,
			app_version,
			domain_names: domain_names,
			domain_name, subdomain, full_domain, domain_valid,
			dropid, dropids,
			create,
			show_all_versions,
			pickVersion,
			pickLatestVersion
		};
	},
});

</script>