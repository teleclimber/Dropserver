<template>
	<div class="flex my-4">
		<span>Status: {{statusString}}</span>
		<button class="bg-red-700 hover:bg-red-900 text-white py-1 px-2 rounded" type="submit" @click.stop.prevent="togglePause">{{paused ? "Unpause" : "Pause"}}</button>
		<span>
			Run migration:
			<select class="rounded-l border-2 py-1" v-model="migrate_to_schema">
				<option v-for="m in baseData.possible_migrations" :value="m">{{m}}</option>
			</select>
			<button class="bg-red-700 hover:bg-red-900 text-white py-1 px-2 rounded" type="submit" @click.stop.prevent="runMigration()">Migrate</button>
		</span>
	</div>
</template>

<script lang="ts">
import { defineComponent } from 'vue';
import baseData, { pauseAppspace, runMigration } from '../models/base-data';

export default defineComponent({
	name: 'AppspaceControl',
	data() {
		return {
			migrate_to_schema: 0,
			paused: false
		}
	},
	components: {
	},
	setup(props, context) {
		return {
			baseData
		};
	},
	computed: {
		statusString() {
			if( this.baseData.paused ) return "paused";
			else return "ready";//not true, just trying it out.
		}
	},
	methods: {
		togglePause() {
			this.paused = !this.paused;
			pauseAppspace(this.paused);
		},
		runMigration() {
			runMigration(this.migrate_to_schema);
		}
	}
});
</script>