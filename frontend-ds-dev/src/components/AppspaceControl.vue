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
			{{importAndMigrate.cur_state}}
		</UiButton>

	</div>
</template>

<script lang="ts">
import { defineComponent, reactive, ref, computed, watch, watchEffect } from 'vue';
import { pauseAppspace, runMigration, ImportAndMigrate } from '../models/base-data';
import appspaceStatus from '../models/appspace-status';
import sandboxControl from '../models/sandbox-control';

import UiButton from './ui/UiButton.vue';

export default defineComponent({
	name: 'AppspaceControl',
	components: {
		UiButton,
	},
	setup(props, context) {
		const ui_paused = ref(false);
		watch( () => appspaceStatus.paused, () => {
			ui_paused.value = !!appspaceStatus.paused;
		});
		function togglePause() {
			ui_paused.value = !ui_paused.value;
			console.log("toggling pause to ", ui_paused.value);
			pauseAppspace(ui_paused.value);
		}

		const migrate_to_schema = ref(0);

		const ui_inspect_sandbox = ref(false);
		function toggleInspect() {
			ui_inspect_sandbox.value = !ui_inspect_sandbox.value;
			console.log( "setting inspect sandbox", ui_inspect_sandbox.value);
			sandboxControl.setInspect(ui_inspect_sandbox.value);
		};
		watch( () => sandboxControl.inspect, () => {
			ui_inspect_sandbox.value = sandboxControl.inspect;
		});

		const status_string = computed( () => {
			if( appspaceStatus.problem ) return "problem";
			if( appspaceStatus.app_version_schema !== appspaceStatus.appspace_schema ) return "migrate";
			if( appspaceStatus.paused ) return "paused";
			if( appspaceStatus.temp_paused ) return "busy";
			return "ready";
		});

		function runMigrationClicked() {
			runMigration(migrate_to_schema.value);
		}

		function stopSandbox() {
			sandboxControl.stopSandbox();
		}

		const importAndMigrate = reactive(new ImportAndMigrate);
		importAndMigrate.reset();

		return {
			appspaceStatus,
			ui_paused, migrate_to_schema, status_string,
			togglePause, runMigrationClicked,
			importAndMigrate,
		};
	}
});
</script>