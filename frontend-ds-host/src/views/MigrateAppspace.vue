<template>
	<ViewWrap>
		<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200 flex justify-between">
				<h3 class="text-lg leading-6 font-medium text-gray-900">Change Application Version</h3>
				<div>
					<router-link :to="{name: 'manage-appspace', params:{id:appspace.id}}" class="btn">Close</router-link>
				</div>
			</div>


			<div v-if="migration_job" class="px-4 py-5 sm:px-6">
				<div v-if="migration_job.finished === null" class="bg-yellow-100 py-5 flex rounded-xl">
					<div class="w-12 sm:w-16 flex justify-center">
						<svg class="w-8 h-8 text-yellow-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
						</svg>
					</div>
					<div>
						<h3 class="text-yellow-600 text-lg font-medium">Migration Job Running</h3>
						<p>Migrating to version {{ migration_job.to_version }}.</p>
						<!-- need a cancel button -->
					</div>
				</div>
				<div v-else-if="migration_job.finished && migration_job.error" class="bg-red-100 py-5 flex rounded-xl">
					<div class="w-12 sm:w-16 flex justify-center">
						<svg class="w-8 h-8 text-red-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
						</svg>
					</div>
					<div>
						<h3 class="text-red-700 text-lg font-medium">Migration Job Encountered an Error</h3>
						<p>{{migration_job.error}}</p>
					</div>
				</div>
				<div v-else class="bg-green-100 py-5 flex rounded-xl">
					<div class="w-12 sm:w-16 flex justify-center">
						<svg class="w-8 h-8 text-green-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
						</svg>
					</div>
					<div>
						<h3 class="text-green-700 text-lg font-medium">Migration Job Finished</h3>
						<p>Migrated to version {{ migration_job.to_version }}.</p>
					</div>
				</div>
			</div>
			<div v-else>
				<div class="px-4 py-5 sm:px-6 sm:grid sm:grid-cols-4">
					<div class="font-medium text-gray-500">Current Version:</div>
					<VersionDetails :app_version="cur_app_version" class="col-span-3"></VersionDetails>
				</div>
				<div v-if="to_version" class="px-4 py-5 sm:px-6 sm:grid sm:grid-cols-4">
					<div class="font-medium text-gray-500">Selected Version:</div>
					<VersionDetails :app_version="to_app_version" class="col-span-3"></VersionDetails>
					<p v-if="!show_all_versions" class="flex justify-end col-span-4 py-1">
						<button @click="showAllVersions(true)" class="btn">Choose different version</button>
					</p>
				</div>
			</div>

			<div v-if="to_version && !show_all_versions && !migration_job" class="border-t border-gray-200 px-4 py-5 sm:px-6">
				<p v-if="cur_app_version.schema !== to_app_version.schema">
					This version change requires migrating your data from schema {{cur_app_version.schema}} to {{to_app_version.schema}}.
				</p>
				<p>[Optionally schedule migration]</p>
				<p class="pt-5 flex justify-end">
					<button @click="migrate()" class="btn btn-blue">Start Migration</button>
				</p>
			</div>

			<!-- use PickVersion here -->
			<div class="px-4 py-5 sm:px-6 sm:grid grid-cols-4" v-if="show_all_versions">
				<div class="font-medium text-gray-500">
					Choose a version:
				</div>
				<PickVersion :versions="app.versions" :current="appspace.app_version" class="col-span-3" @version="pickVersion" @close="showAllVersions(false)"></PickVersion>
				<p v-if="!app.loaded">Loading application versions</p>
			</div>
		</div>
	</ViewWrap>
</template>

<script lang="ts">
import {useRoute} from 'vue-router';
import { defineComponent, ref, Ref, reactive, computed, onUnmounted, watchEffect, isReactive } from 'vue';
import router from '../router';

import { Appspace } from '../models/appspaces';
import { App } from '../models/apps';
import { AppVersion, AppVersionCollector } from '../models/app_versions';
import { MigrationJob, MigrationJobs, createMigrationJob } from '../models/migration_jobs';
import {setTitle} from '../controllers/nav';

import { AppspaceStatus } from '../twine-services/appspace_status';

import ViewWrap from '../components/ViewWrap.vue';
import VersionDetails from '../components/VersionDetails.vue';
import PickVersion from '../components/PickAppVersion.vue';

export default defineComponent({
	name: 'MigrateAppspace',
	components: {
		ViewWrap,
		VersionDetails,
		PickVersion
	},
	setup(props) {
		const to_version :Ref<string|undefined> = ref(undefined);

		const route = useRoute();
		const appspace = reactive(new Appspace);
		const cur_app_version = ref(new AppVersion);
		const to_app_version = ref(new AppVersion);

		const sticky_job_id :Ref<number|undefined> = ref(undefined);
		const migration_jobs = reactive(new MigrationJobs);
		watchEffect( () => {
			if( sticky_job_id.value !== undefined ) return;
			migration_jobs.jobs.forEach( j => {
				if( j.finished === null ) sticky_job_id.value = j.job_id;
			});
		})
		const migration_job = computed(() => {
			let job : MigrationJob|undefined;
			if( sticky_job_id.value !== undefined ) {
				job = migration_jobs.jobs.get(sticky_job_id.value);
			}
			return job;
		});

		const status = reactive(new AppspaceStatus);

		const app = reactive(new App);
		const show_all_versions = ref(false);

		function showAllVersions(show:boolean) {
			show_all_versions.value = show;
			if( show && !app.loaded ) {
				app.fetch(appspace.app_id);
			}
		}
		function pickVersion(version:string) {
			show_all_versions.value = false;
			router.push({name: 'migrate-appspace', params:{id:appspace.id}, query:{to_version:version}});
		}

		watchEffect( async () => {
			sticky_job_id.value = undefined;

			const appspace_id = Number(route.params.id);
			if( isNaN(appspace_id) ) return;

			await appspace.fetch(appspace_id);
						
			status.connectStatus(appspace_id);

			setTitle(appspace.domain_name);

			to_version.value = typeof route.query.to_version == 'string' && route.query.to_version !== appspace.app_version ? route.query.to_version : undefined; 
			if( to_version.value ) {
				to_app_version.value = AppVersionCollector.get(appspace.app_id, to_version.value);
			}
		
			showAllVersions(!to_version.value);

			await migration_jobs.disconnect();
			migration_jobs.connect(appspace_id);
		});

		watchEffect( () => {
			if( appspace.loaded ) cur_app_version.value = AppVersionCollector.get(appspace.app_id, appspace.app_version);
		});

		onUnmounted( () => {
			setTitle("");
		});

		async function migrate() {
			if(!to_version.value) return;
			createMigrationJob(appspace.id, to_version.value);
		}

		onUnmounted( async () => {
			status.disconnect();
			migration_jobs.disconnect();
		});

		return {
			appspace,
			cur_app_version,
			to_app_version,
			to_version,
			show_all_versions,
			showAllVersions,
			pickVersion,
			app,
			migrate,
			migration_job,
		}
	}
});

</script>