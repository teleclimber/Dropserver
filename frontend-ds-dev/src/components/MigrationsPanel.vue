<template>
	<div class="m-4 flex justify-between">
		<div class="flex items-center">
			<span class="mr-2">Current schema: 
				<span class="font-mono">{{appspaceStatus.appspace_schema}}</span>
			</span>
			<div class="bg-gray-200 self-stretch flex items-center">
				<select class="px-1 mx-1 text-lg " v-model="migrate_to_schema">
					<option v-for="m in baseData.possible_migrations.reverse()" :value="m" :key="'migrate-to-schema-'+m">{{m}}</option>
				</select>
			</div>
			<UiButton class="flex items-center" type="submit" :disabled="migrate_to_schema === appspaceStatus.appspace_schema" @click.stop.prevent="runMigrationClicked()">
				<svg class="w-6 h-6 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
					<path d="M5 12a1 1 0 102 0V6.414l1.293 1.293a1 1 0 001.414-1.414l-3-3a1 1 0 00-1.414 0l-3 3a1 1 0 001.414 1.414L5 6.414V12zM15 8a1 1 0 10-2 0v5.586l-1.293-1.293a1 1 0 00-1.414 1.414l3 3a1 1 0 001.414 0l3-3a1 1 0 00-1.414-1.414L15 13.586V8z" />
				</svg>
				Migrate
			</UiButton>
		</div>
		<Sandbox></Sandbox>
	</div>
	<div class="m-4">
		<MigrationJobs></MigrationJobs>
	</div>
</template>

<script lang="ts">
import { defineComponent, ref, computed } from 'vue';
import baseData, { runMigration } from '../models/base-data';
import appspaceStatus from '../models/appspace-status';
import sandboxControl from '../models/sandbox-control';

import Sandbox from './Sandbox.vue';
import MigrationJobs from './MigrationJobs.vue';
import UiButton from './ui/UiButton.vue';

export default defineComponent({
	name: 'MigrationsPanel',
	components: {
		Sandbox,
		MigrationJobs,
		UiButton
	},
	setup(props, context) {
		const migrate_to_schema = ref(0);

		function runMigrationClicked() {
			console.log('runMigrationClicked');
			runMigration(migrate_to_schema.value);
		}

		function stopSandbox() {
			sandboxControl.stopSandbox();
		}

		return {
			baseData, appspaceStatus,
			migrate_to_schema, 
			runMigrationClicked,
			stopSandbox,
		};
	}
});
</script>