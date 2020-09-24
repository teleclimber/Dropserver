<template>
	<div class="flex my-4">
		<span>Status: {{statusString}}</span>
		<button class="bg-red-700 hover:bg-red-900 text-white py-1 px-2 rounded" type="submit" @click.stop.prevent="togglePause">(un)pause</button>
	</div>
</template>

<script lang="ts">
import { defineComponent } from 'vue';
import baseData, { pauseAppspace } from '../models/base-data';

export default defineComponent({
	name: 'AppspaceControl',
	data() {
		return {
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
		}
	}
});
</script>