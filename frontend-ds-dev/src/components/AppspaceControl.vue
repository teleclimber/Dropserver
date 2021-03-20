<template>
	<div class="flex my-4 items-stretch">
		<span class="w-48 text-center">
			<div v-if="statusString === 'problem'" class="bg-orange-300 py-1">Problem</div>
			<div v-if="statusString === 'migrating'" class="bg-yellow-400 py-1">
				<svg class="inline w-6 h-6 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
					<path fill-rule="evenodd" d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1zm.008 9.057a1 1 0 011.276.61A5.002 5.002 0 0014.001 13H11a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0v-2.101a7.002 7.002 0 01-11.601-2.566 1 1 0 01.61-1.276z" clip-rule="evenodd" />
				</svg>
				Migrating
			</div>
			<div v-if="statusString === 'migrate'" class="bg-yellow-400 py-1">
				<svg class="inline w-6 h-6 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
					<path d="M5 12a1 1 0 102 0V6.414l1.293 1.293a1 1 0 001.414-1.414l-3-3a1 1 0 00-1.414 0l-3 3a1 1 0 001.414 1.414L5 6.414V12zM15 8a1 1 0 10-2 0v5.586l-1.293-1.293a1 1 0 00-1.414 1.414l3 3a1 1 0 001.414 0l3-3a1 1 0 00-1.414-1.414L15 13.586V8z" />
				</svg>
				Migration required
			</div>
			<div v-if="statusString === 'ready'" class="bg-green-200 text-green-800 py-1">
				<svg class="inline w-6 h-6 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
					<path d="M2 10.5a1.5 1.5 0 113 0v6a1.5 1.5 0 01-3 0v-6zM6 10.333v5.43a2 2 0 001.106 1.79l.05.025A4 4 0 008.943 18h5.416a2 2 0 001.962-1.608l1.2-6A2 2 0 0015.56 8H12V4a2 2 0 00-2-2 1 1 0 00-1 1v.667a4 4 0 01-.8 2.4L6.8 7.933a4 4 0 00-.8 2.4z" />
				</svg>
				Ready
			</div>
			<div v-if="statusString === 'paused'" class="bg-pink-200 text-pink-800 py-1">
				<svg class="inline w-6 h-6 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
					<path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zM7 8a1 1 0 012 0v4a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v4a1 1 0 102 0V8a1 1 0 00-1-1z" clip-rule="evenodd" />
				</svg>
				Paused
			</div>
			<div v-if="statusString === 'busy'" class="bg-pink-200 text-pink-800 py-1">
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
			{{paused ? "Unpause" : "Pause"}}
		</button>

		<label class="bg-red-700 hover:bg-red-900 text-white py-1 px-2 mr-4 rounded" :class="{'bg-red-900': migrate_inspect}">
			<input type="checkbox" @change="setMigrateInspect()">
			Inspect
		</label>

		<span class="flex bg-gray-200 items-baseline">
			<span class="px-2">Migrate:</span>
			<select class="rounded-l border-2 text-lg" v-model="migrate_to_schema">
				<option v-for="m in baseData.possible_migrations" :value="m">{{m}}</option>
			</select>
			<button class="bg-red-700 hover:bg-red-900 text-white py-1 px-2 rounded" type="submit" @click.stop.prevent="runMigration()">
				<svg class="inline w-6 h-6 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
					<path d="M5 12a1 1 0 102 0V6.414l1.293 1.293a1 1 0 001.414-1.414l-3-3a1 1 0 00-1.414 0l-3 3a1 1 0 001.414 1.414L5 6.414V12zM15 8a1 1 0 10-2 0v5.586l-1.293-1.293a1 1 0 00-1.414 1.414l3 3a1 1 0 001.414 0l3-3a1 1 0 00-1.414-1.414L15 13.586V8z" />
				</svg>
				Migrate
			</button>
		</span>

		<button class="bg-red-700 hover:bg-red-900 text-white py-1 px-2 mx-4 rounded" type="submit" @click.stop.prevent="stopSandbox">
			<svg class="inline w-6 h-6 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
				<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
			</svg>
			Kill Sandbox
		</button>

		<button class="bg-red-700 hover:bg-red-900 text-white py-1 px-2 rounded" type="submit" @click.stop.prevent="importAndMigrate.start">
			<svg class="inline w-6 h-6 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
				<path fill-rule="evenodd" d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1zm.008 9.057a1 1 0 011.276.61A5.002 5.002 0 0014.001 13H11a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0v-2.101a7.002 7.002 0 01-11.601-2.566 1 1 0 01.61-1.276z" clip-rule="evenodd" />
			</svg>
			{{importAndMigrate.cur_state}}
		</button>
	</div>
</template>

<script lang="ts">
import { defineComponent, reactive } from 'vue';
import baseData, { pauseAppspace, runMigration, setInspect, stopSandbox, ImportAndMigrate } from '../models/base-data';

export default defineComponent({
	name: 'AppspaceControl',
	data() {
		return {
			migrate_to_schema: 0,
			migrate_inspect: false,
			paused: false,
		};
	},
	components: {
	},
	setup(props, context) {
		const importAndMigrate = reactive(new ImportAndMigrate);
		importAndMigrate.reset()
		return {
			baseData,
			importAndMigrate
		};
	},
	computed: {
		statusString() {
			if( this.baseData.problem ) return "problem";
			if( this.baseData.migrating ) return "migrating";
			if( this.baseData.schema !== this.baseData.appspace_schema ) return "migrate";
			if( this.baseData.paused ) return "paused";
			if( this.baseData.temp_paused ) return "busy";
			return "ready";
		}
	},
	methods: {
		togglePause() {
			this.paused = !this.paused;
			pauseAppspace(this.paused);
		},
		setMigrateInspect() {
			this.migrate_inspect = !this.migrate_inspect;
			setInspect(this.migrate_inspect);
		},
		runMigration() {
			runMigration(this.migrate_to_schema);
		},
		stopSandbox() {
			stopSandbox();
		}
	}
});
</script>