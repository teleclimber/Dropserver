<script lang="ts" setup>
import { computed, onMounted, onUnmounted, watch, reactive } from 'vue';

import { useAppspacesStore } from '@/stores/appspaces';
import { useAppsStore } from '@/stores/apps';
import { useAppspaceMigrationJobsStore } from '@/stores/migration_jobs';
import type { AppspaceMigrationJob } from '@/stores/types';

import { AppspaceStatus } from '../twine-services/appspace_status';

import ViewWrap from '../components/ViewWrap.vue';
import VersionDetails from '../components/VersionDetails.vue';
import PickVersion from '../components/PickAppVersion.vue';
import MessageProcessing from '@/components/ui/MessageProcessing.vue';
import { useRouter } from 'vue-router';
import DataDef from '@/components/ui/DataDef.vue';
import UnderConstruction from '@/components/ui/UnderConstruction.vue';

const props = defineProps<{
	appspace_id: number,
	to_version: string,
	migrate_only: boolean,
	job_id?: number
}>();

const router = useRouter();

const appspacesStore = useAppspacesStore();
appspacesStore.loadData();

const appsStore = useAppsStore();
appsStore.loadData();

const migrationJobsStore = useAppspaceMigrationJobsStore();
watch( () => migrationJobsStore.isLoaded(props.appspace_id), () => {
	if( !migrationJobsStore.isLoaded(props.appspace_id) ) return;
	migrationJobsStore.connect(props.appspace_id);
});

onMounted( () => {
	migrationJobsStore.reloadData(props.appspace_id);
	appspacesStore.loadAppspace(props.appspace_id);
});

const status = reactive(new AppspaceStatus) as AppspaceStatus;
status.connectStatus(props.appspace_id);

const appspace = computed( () => {
	if( !appspacesStore.is_loaded ) return;
	return appspacesStore.mustGetAppspace(props.appspace_id).value;
});

const app = computed( () => {
	if( appspace.value === undefined || !appsStore.is_loaded ) return;
	return appsStore.mustGetApp(appspace.value.app_id).value;
});

const cur_app_version = computed( () => {
	if( app.value === undefined ) return;
	return app.value.versions.find( v => appspace.value?.app_version === v.version );
});

const data_schema_mismatch = computed( ()=> {
	return cur_app_version.value !== undefined && status.loaded && cur_app_version.value.schema !== status.appspace_schema;
});
const show_migrate_only = computed( () => {
	return data_schema_mismatch.value && props.migrate_only;
});

const to_app_version = computed( () => {
	if( app.value === undefined || props.to_version === '' ) return;
	return app.value.versions.find( v => props.to_version === v.version);
});

const running_migration_job = computed( () => {
	let job : AppspaceMigrationJob|undefined;
	const jobs = migrationJobsStore.getJobs(props.appspace_id);
	if( jobs === undefined ) return;
	jobs.value.forEach( j => {
		if( !j.value.finished ) job = j.value;
	});
	console.log('current m job', job);
	return job;
});
watch( running_migration_job, (new_job, old_job) => {
	// if running job becomes undefined it means it finished. So reload appspace.
	if( old_job !== undefined && new_job === undefined ) appspacesStore.loadAppspace(props.appspace_id);
});
const migration_job = computed( () => {
	if( props.job_id === undefined ) return;
	const jobs = migrationJobsStore.getJobs(props.appspace_id);
	if( jobs === undefined ) return;
	const job = jobs.value.get(props.job_id);
	if( job === undefined ) return;
	return job.value;
});

const show_all_versions = computed( () => {
	return !props.to_version && !props.migrate_only;
});

const ok_to_migrate = computed( () => {
	if( show_migrate_only.value ) return true;
	return !!to_app_version.value 
		&& cur_app_version.value?.version !== to_app_version.value.version
		&& running_migration_job.value === undefined
		&& migration_job.value === undefined;
});

async function migrate() {
	if( !ok_to_migrate.value || appspace.value === undefined ) return;
	const to_version = show_migrate_only.value ? appspace.value.app_version : props.to_version;
	const job = await migrationJobsStore.createMigrationJob(props.appspace_id, to_version);
	router.replace({name:'migrate-appspace', query:{job_id:job.value.job_id} });
}

onUnmounted( () => {
	migrationJobsStore.disconnect(props.appspace_id);
});

</script>

<template>
	<ViewWrap>

		<MessageProcessing head="Migration Ongoing" v-if="running_migration_job && migration_job?.job_id !== running_migration_job.job_id">
			A migration is ongoing.
			<router-link :to="{name:'migrate-appspace', query:{job_id:running_migration_job.job_id}}" class="btn">show details</router-link>.
		</MessageProcessing>

		<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200 flex justify-between">
				<h3 class="text-lg leading-6 font-medium text-gray-900">{{ migrate_only ? 'Migrate Appspace Data' : 'Change Application Version' }}</h3>
			</div>

			<div v-if="migration_job" class="px-4 py-5 sm:px-6">
				<MessageProcessing v-if="migration_job.finished === null" head="Migration Job Running">
					<!-- get wording correct for migration versus app version change.-->
					<p>Migrating to version {{ migration_job.to_version }}.</p>
				</MessageProcessing>
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
			
			<div v-else-if="show_migrate_only" class="px-4 py-5 sm:px-6">
				Migrate appspace data from schema {{ status.appspace_schema }} to {{ cur_app_version?.schema }}.
				<UnderConstruction 
					head="This Might Not Work"
					class="my-5">
					Dropserver doesn't handle data schema mismatches correctly.
					You can try running this migration, but it may fail.
					If so, your data will be reverted back.
				</UnderConstruction>
			</div>
			<div v-else>
				<DataDef field="Current Version:">
					<VersionDetails v-if="cur_app_version" :app_version="cur_app_version" class="col-span-3"></VersionDetails>
					<span v-else>(none)</span>
				</DataDef>
				<DataDef v-if="show_all_versions" field="Choose a Version:">
					<PickVersion 
						v-if="app && appspace"
						:appspace_id="appspace_id"
						:versions="app.versions"
						:current="appspace.app_version"
						class="col-span-3"
					></PickVersion>
				</DataDef>
				<DataDef v-if="to_version" field="Selected Version:">
					<VersionDetails v-if="to_app_version" :app_version="to_app_version" class="col-span-3"></VersionDetails>
					<p v-if="!show_all_versions" class="flex justify-end col-span-4 py-1">
						<router-link class="btn" :to="{name:'migrate-appspace',params:{appspace_id:appspace_id}}" :replace="true">Choose different version</router-link>
					</p>
				</DataDef>

				<UnderConstruction 
					v-if="data_schema_mismatch && to_app_version && status.appspace_schema !== to_app_version?.schema" 
					head="This Might Not Work"
					class="my-5">
					Dropserver doesn't handle data schema mismatches correctly.
					You can try running this migration, but it may fail.
					If so, your data will be reverted back.
				</UnderConstruction>
				
				<p v-if="to_app_version && status.appspace_schema !== to_app_version.schema" class="px-4 py-5 sm:px-6">
					This version change requires migrating your data from schema {{status.appspace_schema}} to {{to_app_version.schema}}.
				</p>
			</div>

			<div class="border-t px-4 md:px-6 py-5 flex justify-between items-baseline">
				<router-link :to="{name: 'manage-appspace', params:{appspace_id:appspace_id}}" class="btn">back to appspace</router-link>
				<button @click="migrate()" class="btn btn-blue" :disabled="!ok_to_migrate" v-if="!migration_job">Start Migration</button>
			</div>
		</div>
	</ViewWrap>
</template>
