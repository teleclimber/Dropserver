<template>
	<div class="flex my-4 items-stretch">
		<span class="w-48 text-center">
			<div v-if="status_string === 'problem'" class="bg-red-300 py-1">Problem</div>
			<div v-else-if="appspaceStatus.temp_paused" class="bg-yellow-400 py-1">
				<svg class="inline w-6 h-6 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
					<path fill-rule="evenodd" d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1zm.008 9.057a1 1 0 011.276.61A5.002 5.002 0 0014.001 13H11a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0v-2.101a7.002 7.002 0 01-11.601-2.566 1 1 0 01.61-1.276z" clip-rule="evenodd" />
				</svg>
				{{appspaceStatus.temp_pause_reason}}
			</div>
			<div v-else-if="status_string === 'migrate'" class="bg-yellow-400 py-1">
				<svg class="inline w-6 h-6 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
					<path d="M5 12a1 1 0 102 0V6.414l1.293 1.293a1 1 0 001.414-1.414l-3-3a1 1 0 00-1.414 0l-3 3a1 1 0 001.414 1.414L5 6.414V12zM15 8a1 1 0 10-2 0v5.586l-1.293-1.293a1 1 0 00-1.414 1.414l3 3a1 1 0 001.414 0l3-3a1 1 0 00-1.414-1.414L15 13.586V8z" />
				</svg>
				Migration required
			</div>
			<div v-else-if="status_string === 'ready'" class="bg-green-200 text-green-800 py-1">
				<svg class="inline w-6 h-6 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
					<path d="M2 10.5a1.5 1.5 0 113 0v6a1.5 1.5 0 01-3 0v-6zM6 10.333v5.43a2 2 0 001.106 1.79l.05.025A4 4 0 008.943 18h5.416a2 2 0 001.962-1.608l1.2-6A2 2 0 0015.56 8H12V4a2 2 0 00-2-2 1 1 0 00-1 1v.667a4 4 0 01-.8 2.4L6.8 7.933a4 4 0 00-.8 2.4z" />
				</svg>
				Ready
			</div>
			<div v-else-if="status_string === 'paused'" class="bg-pink-200 text-pink-800 py-1">
				<svg class="inline w-6 h-6 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
					<path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zM7 8a1 1 0 012 0v4a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v4a1 1 0 102 0V8a1 1 0 00-1-1z" clip-rule="evenodd" />
				</svg>
				Paused
			</div>
			<div v-else-if="status_string === 'busy'" class="bg-pink-200 text-pink-800 py-1">
				<svg class="inline w-6 h-6 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
					<path fill-rule="evenodd" d="M13.477 14.89A6 6 0 015.11 6.524l8.367 8.368zm1.414-1.414L6.524 5.11a6 6 0 018.367 8.367zM18 10a8 8 0 11-16 0 8 8 0 0116 0z" clip-rule="evenodd" />
				</svg>
				Please wait...
			</div>
		</span>


		<button class="bg-red-700 hover:bg-red-900 text-white py-1 px-2 mx-4 rounded" type="submit" @click.stop.prevent="togglePause">
			<svg class="inline w-6 h-6 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
				<path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zM7 8a1 1 0 012 0v4a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v4a1 1 0 102 0V8a1 1 0 00-1-1z" clip-rule="evenodd" />
			</svg>
			{{ui_paused ? "Unpause" : "Pause"}}
		</button>

		<button class="bg-red-700 hover:bg-red-900 text-white py-1 px-2 mr-4 rounded" :class="{'bg-red-500': ui_inspect_sandbox}" @click.stop.prevent="toggleInspect">
			<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" :class="{'text-red-800':!ui_inspect_sandbox, 'text-white': ui_inspect_sandbox}" viewBox="0 0 20 20" fill="currentColor">
				<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8 7a1 1 0 00-1 1v4a1 1 0 001 1h4a1 1 0 001-1V8a1 1 0 00-1-1H8z" clip-rule="evenodd" />
			</svg>
			Inspect
		</button>

		<span class="flex bg-gray-200 items-baseline">
			<select class="rounded-l border-2 text-lg" v-model="migrate_to_schema">
				<option v-for="m in baseData.possible_migrations" :value="m" :key="'migrate-to-schema-'+m">{{m}}</option>
			</select>
			<button class="bg-red-700 hover:bg-red-900 text-white py-1 px-2 rounded" type="submit" @click.stop.prevent="runMigrationClicked()">
				<svg class="inline w-6 h-6 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
					<path d="M5 12a1 1 0 102 0V6.414l1.293 1.293a1 1 0 001.414-1.414l-3-3a1 1 0 00-1.414 0l-3 3a1 1 0 001.414 1.414L5 6.414V12zM15 8a1 1 0 10-2 0v5.586l-1.293-1.293a1 1 0 00-1.414 1.414l3 3a1 1 0 001.414 0l3-3a1 1 0 00-1.414-1.414L15 13.586V8z" />
				</svg>
				Migrate
			</button>
		</span>

		<button class="bg-red-700 hover:bg-red-900 text-white py-1 px-2 mx-4 rounded" type="submit" @click.stop.prevent="stopSandbox()">
			<svg class="inline w-6 h-6 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
				<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
			</svg>
			Kill
		</button>

		<button class="bg-red-700 hover:bg-red-900 text-white py-1 px-2 mr-4 rounded" type="submit" @click.stop.prevent="importAndMigrate.start">
			<svg class="inline w-6 h-6 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
				<path fill-rule="evenodd" d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1zm.008 9.057a1 1 0 011.276.61A5.002 5.002 0 0014.001 13H11a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0v-2.101a7.002 7.002 0 01-11.601-2.566 1 1 0 01.61-1.276z" clip-rule="evenodd" />
			</svg>
			{{importAndMigrate.cur_state}}
		</button>

		<button class="bg-red-700 hover:bg-red-900 text-white py-1 px-2 rounded" type="submit" @click.stop.prevent="appRoutesData.reloadRoutes">
			<svg class="inline w-6 h-6 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
				<path fill-rule="evenodd" d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1zm.008 9.057a1 1 0 011.276.61A5.002 5.002 0 0014.001 13H11a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0v-2.101a7.002 7.002 0 01-11.601-2.566 1 1 0 01.61-1.276z" clip-rule="evenodd" />
			</svg>
			Routes
		</button>
	</div>
</template>

<script lang="ts">
import { defineComponent, reactive, ref, computed, watch, watchEffect } from 'vue';
import baseData, { pauseAppspace, runMigration, ImportAndMigrate } from '../models/base-data';
import appspaceStatus from '../models/appspace-status';
import sandboxControl from '../models/sandbox-control';
import appRoutesData from '../models/app-routes';

export default defineComponent({
	name: 'AppspaceControl',
	components: {
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
			baseData, appspaceStatus,
			ui_paused, migrate_to_schema, ui_inspect_sandbox, status_string,
			togglePause, toggleInspect, runMigrationClicked,
			stopSandbox,
			importAndMigrate,
			appRoutesData
		};
	}
});
</script>