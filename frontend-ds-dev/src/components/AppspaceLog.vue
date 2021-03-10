<style scoped>
	.log-grid {
		display: grid;
		grid-template-columns: 10rem 1fr;
	}
</style>

<template>
	<div class="border-l-4 border-gray-800  my-8">
		<h4 class="bg-gray-800 px-2 text-white inline-block">Appspace Log:</h4>
		<div class="overflow-y-scroll h-64 bg-gray-100" style="scroll-behavior: smooth" ref="scroll_container">
			<div class="log-grid">
				<template  v-for="event in appspaceLogData.events">
					<span class="bg-gray-200 text-gray-700 pl-2 text-sm font-bold border-b border-gray-400">{{event.source}}</span>
					<p class="px-2 border-b border-gray-400" :class="{'bg-red-200': event.source.includes('stderr')}">
						{{event.message }}
					</p>
				</template>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, ref } from 'vue';
import appspaceLogData from '../models/appspace-log-data';
import type {AppspaceLogEvent} from '../models/appspace-log-data';

export default defineComponent({
	name: 'AppspaceLog',
	components: {
	},
	setup(props, context) {
		return {appspaceLogData};
	},
	updated() {
	 	const elem = this.$refs.scroll_container as HTMLElement;
	 	elem.scrollTop = elem.scrollHeight;
	},
});
</script>