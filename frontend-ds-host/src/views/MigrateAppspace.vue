<script lang="ts" setup>
import { computed, onMounted, onUnmounted, watch, reactive, ref, Ref } from 'vue';

import { useAppspacesStore } from '@/stores/appspaces';
import { useAppsStore } from '@/stores/apps';
import { useAppspaceMigrationJobsStore } from '@/stores/migration_jobs';
import type { AppspaceMigrationJob, App } from '@/stores/types';

import { AppspaceStatus } from '../twine-services/appspace_status';

import ViewWrap from '../components/ViewWrap.vue';
import PickVersion from '../components/PickAppVersion.vue';
import MessageProcessing from '@/components/ui/MessageProcessing.vue';
import { useRouter } from 'vue-router';
import DataDef from '@/components/ui/DataDef.vue';
import UnderConstruction from '@/components/ui/UnderConstruction.vue';
import AppLicense from '@/components/app/AppLicense.vue';
import SmallMessage from '@/components/ui/SmallMessage.vue';
import MinimalAppUrlData from '@/components/appspace/MinimalAppUrlData.vue';

const props = defineProps<{
	appspace_id: number,
	to_version: string,
	migrate_only: boolean,
	job_id?: number
}>();

const router = useRouter();

const appspacesStore = useAppspacesStore();

const appsStore = useAppsStore();

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
	const a = appspacesStore.getAppspace(props.appspace_id);
	if( a === undefined ) return a;
	return a.value;
});

watch( appspace, () => {
	if( appspace.value ) appsStore.loadAppVersions(appspace.value.app_id);
});

const app :Ref<App|undefined> = ref();
watch( () => appspace.value?.app_id, async () => {
	app.value = undefined;
	if( appspace.value === undefined ) return;
	await appsStore.loadApp(appspace.value.app_id);
	const a = appsStore.getApp(appspace.value.app_id);
	if( a !== undefined ) app.value = a.value;
}, {immediate: true});

const app_versions = computed( () => {
	if( !appspace.value ) return;
	return appsStore.getAppVersions(appspace.value.app_id);
});

const cur_app_version = computed( () => {
	if( app_versions.value === undefined ) return;
	return app_versions.value.find( v => appspace.value?.app_version === v.version );
});

const cur_app_icon_error = ref(false);
const cur_app_icon = computed( () => {
	if( cur_app_icon_error.value || !cur_app_version.value ) return "";
	return `/api/application/${cur_app_version.value.app_id}/version/${cur_app_version.value.version}/file/app-icon`;
});

const data_schema_mismatch = computed( ()=> {
	return cur_app_version.value !== undefined && status.loaded && cur_app_version.value.schema !== status.appspace_schema;
});
const show_migrate_only = computed( () => {
	return data_schema_mismatch.value && props.migrate_only;
});

const to_app_version = computed( () => {
	if( app_versions.value === undefined || props.to_version === '' ) return;
	return app_versions.value.find( v => props.to_version === v.version);
});
const to_app_icon_error = ref(false);
const to_app_icon = computed( () => {
	if( to_app_icon_error.value || !to_app_version.value ) return "";
	return `/api/application/${to_app_version.value.app_id}/version/${to_app_version.value.version}/file/app-icon`;
});
const to_release_date = computed( () => {
	if( !to_app_version.value?.release_date ) return;
	return new Date(to_app_version.value.release_date).toLocaleDateString(undefined, {
		dateStyle:'medium'
	});
});

const to_down = computed( () => {
	if( !app_versions.value || !appspace.value || props.to_version === '' ) return;
	const cur_i = app_versions.value.findIndex( v => appspace.value?.app_version === v.version );
	const to_i = app_versions.value.findIndex( v => props.to_version === v.version );
	return to_i > cur_i;
});

const to_changelog = ref("");
watch( to_app_version, async () => {
	to_changelog.value = "";
	if( to_app_version.value === undefined ) return;
	const resp = await fetch(`/api/application/${to_app_version.value.app_id}/version/${to_app_version.value.version}/changelog`);
	to_changelog.value = await resp.text();
}, { immediate: true });

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

const small_msg_classes = ['inline-block', 'mt-1'];

</script>

<template>
	<ViewWrap>

		<MessageProcessing head="Migration Ongoing" v-if="running_migration_job && migration_job?.job_id !== running_migration_job.job_id">
			A migration is ongoing.
		</MessageProcessing>

		<div class="md:mb-6 my-6 bg-white shadow overflow-hidden sm:rounded-lg">
			<div class="px-4 py-5 sm:px-6 border-b border-gray-200 flex flex-col sm:flex-row justify-between sm:items-center">
				<h3 class="text-lg leading-6 font-medium text-gray-900">{{ migrate_only ? 'Migrate Appspace Data' : 'Change Application Version' }}</h3>
				<router-link v-if="props.to_version" class="btn mt-2 sm:m-0" :to="{name:'migrate-appspace',params:{appspace_id:appspace_id}}" :replace="true">Choose different version</router-link>
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
			<div v-else-if="show_all_versions" class="px-4 my-5 sm:px-6">
				<div class="flex items-baseline flex-col md:flex-row my-5">
					<img v-if="cur_app_icon" :src="cur_app_icon" @error="cur_app_icon_error = true" class="w-10 h-10 self-center" />
					<div v-else class="w-10 h-10 text-gray-300  flex justify-center items-center">
						<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-14 h-14">
							<path stroke-linecap="round" stroke-linejoin="round" d="M2.25 15a4.5 4.5 0 004.5 4.5H18a3.75 3.75 0 001.332-7.257 3 3 0 00-3.758-3.848 5.25 5.25 0 00-10.233 2.33A4.502 4.502 0 002.25 15z" />
						</svg>
					</div>
						<router-link v-if="cur_app_version" :to="{name: 'manage-app', params:{id:cur_app_version?.app_id}}" class="font-medium text-xl text-blue-600 underline">
							{{cur_app_version?.name}}
						</router-link>
						<MinimalAppUrlData v-if="app?.url_data" :url_data="app.url_data" :cur_ver="cur_app_version?.version"></MinimalAppUrlData>
				</div>
				<div class="flex justify-between items-baseline">
					<h4 class="text-xl">Installed versions:</h4>
					<router-link v-if="app?.url_data" :to="{name:'new-app-version', params:{id:app.url_data.app_id}}" class="btn">
						get other versions
					</router-link>
				</div>
				<PickVersion 
						v-if="app_versions && appspace"
						:appspace_id="appspace_id"
						:versions="app_versions"
						:current="appspace.app_version"
					></PickVersion>
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
			<div v-else class=" sm:px-6 my-5">
				<div class="px-4" v-if="to_app_version">
					<div class="my-5 mx-auto max-w-xl pb-4 bg-white shadow overflow-hidden border-2" style="border-top-width:1rem" :style="'border-color:'+(to_app_version?.color || 'rgb(135, 151, 164)')" >
						<div class="grid app-grid gap-x-2 gap-y-2 px-2 py-2 ">
							<img v-if="to_app_icon" :src="to_app_icon" @error="to_app_icon_error = true" class="w-20 h-20" />
							<div v-else class="w-20 h-20 text-gray-300 flex justify-center items-center">
								<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-14 h-14">
									<path stroke-linecap="round" stroke-linejoin="round" d="M2.25 15a4.5 4.5 0 004.5 4.5H18a3.75 3.75 0 001.332-7.257 3 3 0 00-3.758-3.848 5.25 5.25 0 00-10.233 2.33A4.502 4.502 0 002.25 15z" />
								</svg>
							</div>
							<div class="self-center">
								<h3 class="text-2xl leading-6 font-medium text-gray-900">{{to_app_version.name}}</h3>
							</div>
							<div class="col-span-3 sm:col-start-2">
								<p class="text-lg font-medium">
									Version 
									<span class="bg-gray-200 text-gray-700 px-1 rounded-md">{{to_app_version.version}}</span>
									<span v-if="to_release_date"> released {{ to_release_date || '' }}</span>
								</p>
							</div>
						</div>
					</div>
				</div>

				<div class="px-4 sm:px-2 mx-auto max-w-xl font-medium mt-6">What's new:</div>
				<div class="bg-gray-100 px-4 sm:px-2 py-2 mx-auto max-w-xl max-h-48 overflow-y-scroll mb-6">
					<pre class="text-sm whitespace-pre-wrap">{{ to_changelog || "No changelog :(" }}</pre>
				</div>

				<div v-if="to_app_version && cur_app_version">
					<DataDef field="App name:">
						<p class="font-medium text-lg">{{ to_app_version.name }}</p>
						<SmallMessage v-if="cur_app_version.name !== to_app_version.name" mood="warn" :class="small_msg_classes">
							The name has changed.
							It was: “<span class="font-medium">{{ cur_app_version.name }}</span>”
						</SmallMessage>
					</DataDef>

					<DataDef field="Version:">
						<p class="">
							<span class="font-medium text-lg bg-gray-200 text-gray-700 px-1 rounded-md">{{ to_app_version.version }}</span>
							<span v-if="to_release_date"> released {{ to_release_date || '' }}</span>
						</p>
						<SmallMessage mood="info" v-if="to_down"  :class="small_msg_classes">
							Changing to an earlier version of the app.
						</SmallMessage>
					</DataDef>

					<DataDef field="Data schema:">
						<p class="font-medium text-lg">{{ to_app_version.schema }}</p>
						<SmallMessage v-if="cur_app_version.schema !== to_app_version.schema" mood="info" :class="small_msg_classes">
							This version change requires migrating your data from schema {{status.appspace_schema}} to {{to_app_version.schema}}.
						</SmallMessage>
					</DataDef>

					<DataDef field="License:">
						<p><AppLicense :license="to_app_version.license" ></AppLicense></p>
						<SmallMessage v-if="cur_app_version.license !== to_app_version.license"  mood="warn" :class="small_msg_classes">
							The license has changed. It was:
							<AppLicense :license="cur_app_version.license"></AppLicense>
						</SmallMessage>
					</DataDef>
				</div>

				<UnderConstruction 
					v-if="data_schema_mismatch && to_app_version && status.appspace_schema !== to_app_version?.schema" 
					head="This Might Not Work"
					class="my-5">
					Dropserver doesn't handle data schema mismatches correctly.
					You can try running this migration, but it may fail.
					If so, your data will be reverted back.
				</UnderConstruction>
			</div>

			<div class="border-t px-4 md:px-6 py-5 flex justify-between items-baseline">
				<router-link :to="{name: 'manage-appspace', params:{appspace_id:appspace_id}}" class="btn">back to appspace</router-link>
				<button @click="migrate()" class="btn btn-blue" :disabled="!ok_to_migrate" v-if="!migration_job && !show_all_versions">
					{{ migrate_only ? "Start Migration" : "Change Version" }}</button>
			</div>
		</div>
	</ViewWrap>
</template>

<style scoped>
.app-grid {
	grid-template-columns: 5rem 1fr max-content;
}
</style>