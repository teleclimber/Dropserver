<template>
	<ViewWrap>
		<p>Appsace: {{appspace.subdomain}}</p>
		<div v-if="migration_job">
			<p>There is a migration job: {{migration_job.job_id}} {{migration_job.finished === null ? '' : 'Finished'}}
				{{migration_job.started === null ? '' : 'Started'}}
				{{ migration_job.to_version }}
				<router-link :to="{name: 'migrate-appspace', params:{id:appspace.id}}">Close</router-link>
			</p>
		</div>
		<div v-else>
			<p>Current App Version: {{cur_app_version.app_name}}, version: {{cur_app_version.version}} (schema: {{cur_app_version.schema}}, API: {{cur_app_version.api_version}})</p>
			<p v-if="to_version">To App Version: {{to_app_version.app_name}}, version: {{to_app_version.version}} (schema: {{to_app_version.schema}}, API: {{to_app_version.api_version}})</p>
			<p v-else>No version chosen</p>
			<p v-if="!show_all_versions">
				<button @click="showAllVersions(true)">Choose different version</button>
			</p>
		</div>
		<div v-if="to_version && !show_all_versions && !migration_job">
			<p v-if="cur_app_version.schema !== to_app_version.schema">
				This version change requires migrating your data from schema {{cur_app_version.schema}} to {{to_app_version.schema}}.
			</p>
			<p>[Optionally schedule migration]</p>
			<button @click="migrate()">Start Migration</button>
		</div>

		<div v-if="show_all_versions">
			<p><button @click="showAllVersions(false)">Close</button></p>
			<ul>
				<li v-for="ver in app.versions" :key="ver.version" :class="{'bg-yellow-100': ver.version === appspace.app_version}">
					{{ver.version}} created {{ver.created_dt.toLocaleString()}}
					<router-link v-if="ver.version !== appspace.app_version" :to="{name: 'migrate-appspace', params:{id:appspace.id}, query:{to_version:ver.version}}">Choose Version</router-link>
				</li>
			</ul>
			<p v-if="!app.loaded">Loading application versions</p>
		</div>
		
	</ViewWrap>
</template>

<script lang="ts">
import {useRoute} from 'vue-router';
import { defineComponent, ref, Ref, reactive, computed, onUnmounted, watchEffect, isReactive } from 'vue';

import { Appspace } from '../models/appspaces';
import { App } from '../models/apps';
import { AppVersion, AppVersionCollector } from '../models/app_versions';
import { MigrationJob, MigrationJobs, createMigrationJob } from '../models/migration_jobs';

import { AppspaceStatus } from '../twine-services/appspace_status';

import ViewWrap from '../components/ViewWrap.vue';

export default defineComponent({
	name: 'MigrateAppspace',
	components: {
		ViewWrap
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

		watchEffect( async () => {
			sticky_job_id.value = undefined;

			const appspace_id = Number(route.params.id);
			if( isNaN(appspace_id) ) return;

			await appspace.fetch(appspace_id);
						
			status.connectStatus(appspace_id);

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
			app,
			migrate,
			migration_job,
		}
	}
});

</script>