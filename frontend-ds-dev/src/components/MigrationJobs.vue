<template>
	<div class="border-l-4 border-gray-800">
		<h4 class="bg-gray-800 px-2 text-white inline-block">Migration Jobs:</h4>

		<div class="overflow-y-scroll h-64 bg-gray-100" style="scroll-behavior: smooth" ref="scroll_container">
			<div class="my-4 px-2" v-for="job in migrationData.jobs" :key="job.job_id">
				<h5>Job #{{job.job_id}} {{job.started ? "Started: " + job.started.toLocaleString() : "not started"}}</h5>
				<p class="bg-red-200 my-2 p-2" v-if="job.err">{{job.err}}</p>
				<div class="bg-orange-400" v-if="job.started && !job.finished">Running...</div>
				<div class="bg-green-200 text-green-700 text-lg font-bold my-2 p-2" v-if="job.finished && !job.err">
					<svg class="inline w-6 h-6 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
						<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
					</svg>
					Finished</div>
				<div class="bg-red-200 text-red-700 text-lg font-bold my-2 p-2" v-if="job.finished && job.err">
					<svg class="inline w-6 h-6 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
						<path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clip-rule="evenodd" />
					</svg>
					Finished with error
				</div>

			</div>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent } from 'vue';
import migrationData from '../models/migration-data';

export default defineComponent({
	name: 'MigrationJobs',
	components: {
	},
	setup(props, context) {
		return {
			//migration_jobs: migrationData.jobs
			migrationData
		};
	},
	updated() {
	 	const elem = this.$refs.scroll_container as HTMLElement;
	 	elem.scrollTop = elem.scrollHeight;
	}
});
</script>