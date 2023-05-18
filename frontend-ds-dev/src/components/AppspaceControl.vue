<template>
	<div class="flex items-stretch">
		<UiButton class="mr-2 flex items-center" type="submit" @click.stop.prevent="togglePause">
			<svg class="inline w-6 h-6 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
				<path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zM7 8a1 1 0 012 0v4a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v4a1 1 0 102 0V8a1 1 0 00-1-1z" clip-rule="evenodd" />
			</svg>
			{{ui_paused ? "Unpause" : "Pause"}}
		</UiButton>

		<UiButton class="flex items-center" type="submit" @click.stop.prevent="importAndMigrate.start()">
			<svg class="inline w-6 h-6 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
				<path fill-rule="evenodd" d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1zm.008 9.057a1 1 0 011.276.61A5.002 5.002 0 0014.001 13H11a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0v-2.101a7.002 7.002 0 01-11.601-2.566 1 1 0 01.61-1.276z" clip-rule="evenodd" />
			</svg>
			Reload Appspace
		</UiButton>

		<div class="flex items-center ml-2">
			<div class="bg-gray-200 self-stretch flex items-center pl-4">
				<span>
					Migrate from 
					<span class=" text-lg font-mono">{{appspaceStatus.appspace_schema}}</span>
					to
				</span>
				<select class="mx-1 text-lg font-mono" v-model="migrate_to_schema">
					<option v-for="m in possible_migrations" :value="m" :key="'migrate-to-schema-'+m">{{m}}</option>
				</select>
			</div>
			<UiButton class="flex items-center" type="submit" :disabled="migrate_to_schema === appspaceStatus.appspace_schema" @click.stop.prevent="runMigrationClicked()">
				<svg class="w-6 h-6 mr-1" :class="{'animate-spin':importAndMigrate.working}" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
					<path d="M5 12a1 1 0 102 0V6.414l1.293 1.293a1 1 0 001.414-1.414l-3-3a1 1 0 00-1.414 0l-3 3a1 1 0 001.414 1.414L5 6.414V12zM15 8a1 1 0 10-2 0v5.586l-1.293-1.293a1 1 0 00-1.414 1.414l3 3a1 1 0 001.414 0l3-3a1 1 0 00-1.414-1.414L15 13.586V8z" />
				</svg>
				Migrate
			</UiButton>
		</div>
		<span v-if="!last_job" class="flex items-center text-gray-500 ml-2">
			<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 mr-1" viewBox="0 0 20 20" fill="currentColor">
				<path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clip-rule="evenodd" />
			</svg>
			No recent migration
		</span>
		<span v-else-if="last_job.finished && last_job.error" class="flex items-center text-red-700 font-bold ml-2">
			<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 mr-1" viewBox="0 0 20 20" fill="currentColor">
				<path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clip-rule="evenodd" />
			</svg>
			Error running migration
		</span>
		<span v-else-if="last_job.finished" class="flex items-center text-green-700 ml-2">
			<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 mr-1" viewBox="0 0 20 20" fill="currentColor">
				<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
			</svg>
			Migration successful
		</span>
		<span v-else-if="!last_job.finished" class="flex items-center text-yellow-800 ml-2">
			<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 mr-1 animate-spin" viewBox="0 0 20 20" fill="currentColor">
				<path fill-rule="evenodd" d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1zm.008 9.057a1 1 0 011.276.61A5.002 5.002 0 0014.001 13H11a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0v-2.101a7.002 7.002 0 01-11.601-2.566 1 1 0 01.61-1.276z" clip-rule="evenodd" />
			</svg>
			Migration running...
		</span>
	</div>
	<div v-if="last_job && last_job.error" class="my-2 bg-red-50 p-1 text-red-800">
		<p>{{last_job.error}}</p>
	</div>
</template>

<script lang="ts">
import { defineComponent, reactive, ref, computed, onMounted } from 'vue';
import appData from '../models/app-data';
import migrationData from '../models/migration-data';
import appspaceStatus, { pauseAppspace, runMigration, ImportAndMigrate }  from '../models/appspace-status';

import UiButton from './ui/UiButton.vue';

export default defineComponent({
	name: 'AppspaceControl',
	components: {
		UiButton,
	},
	setup(props, context) {
		const ui_paused = ref(false);
		onMounted( () => {
			ui_paused.value = !!appspaceStatus.paused;
		});
		function togglePause() {
			ui_paused.value = !ui_paused.value;
			pauseAppspace(ui_paused.value);
		}

		const migrate_to_schema = ref(appData.schema);

		function runMigrationClicked() {
			runMigration(migrate_to_schema.value);
		}

		const importAndMigrate = reactive(new ImportAndMigrate);

		const last_job = computed( () => migrationData.last_job );

		const possible_migrations = computed( () => {
			const cur_schema = appspaceStatus.appspace_schema;
			const m = appData.migrations;
			const ret :number[] = [];
			let i = cur_schema;
			// up:
			while(true) {
				++i;
				if( m.find( s => s.direction === 'up' && s.schema === i) ) ret.push(i);
				else break
			}
			i = cur_schema;
			// down:
			while(true) {
				--i;
				if( m.find( s => s.direction === 'down' && s.schema === i) ) ret.unshift(i);
				else break
			}
			return ret;
		});

		return {
			appspaceStatus, appData,
			ui_paused, migrate_to_schema,
			possible_migrations,
			togglePause, runMigrationClicked,
			importAndMigrate,
			last_job,
		};
	}
});
</script>