<style scoped>
	.log-grid {
		display: grid;
		grid-template-columns: 8rem 6rem 4fr 7rem 5fr;
	}
</style>

<template>
	<div class="border-l-4 border-gray-800  my-8">
		<h4 class="bg-gray-800 px-2 text-white inline-block">Appspace Routes:</h4>
		<div class="overflow-y-scroll h-64 bg-gray-100">
			<div class="log-grid">
				<template  v-for="route in appspaceRoutesData.sorted">
					<Route :route="route"></Route>
				</template>
			</div>
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, reactive } from 'vue';
import appspaceRoutesData from '../models/appspace-routes-data';

import Route from './Route.vue';

export default defineComponent({
	name: 'AppspaceRoutes',

	components: {
		Route
	},
	setup(props, context) {
		return {
			appspaceRoutesData,
		};
	},

});
</script>