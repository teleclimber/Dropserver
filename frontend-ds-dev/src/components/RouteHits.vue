<style scoped>
	.route-grid {
		display: grid;
		grid-template-columns: 1fr 1fr 4fr 7rem 5fr;
	}
</style>

<template>
	<div class="border-l-4 border-gray-800">
		<h4 class="bg-gray-800 px-2 text-white inline-block">Route Hits:</h4>

		<div class="overflow-y-scroll h-64 bg-gray-100">
			<div class="route-grid">
				<template v-for="r in routeEvents.hit_events">
					<span>
						(owner)
						<svg class="inline text-green-600 w-4 h-4 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
							<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
						</svg>
					</span>
					<span class="px-2" style="font-variant-caps: all-small-caps">{{r.request.method}}</span>
					<span class="bg-gray-200 text-gray-900 pl-2 text-sm font-bold">
						{{r.request.url}}
						<svg v-if="r.route_config" class="inline text-green-600 w-4 h-4 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
							<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
						</svg>
						<svg v-else class="inline text-red-600 w-4 h-4 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
							<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
						</svg>
					</span>
					<span></span><span></span>
					<Route v-if="r.route_config" :route="r.route_config"></Route>
					<div class="bg-gray-400" style="grid-column: 1 / -1; height: 5px" > </div>
				</template>
				
			</div>
		</div>
	</div>

</template>

<script lang="ts">
import { defineComponent } from 'vue';
import routeEvents from '../models/route-hits';
import Route from './Route.vue';

export default defineComponent({
	name: 'Routehits',
	components: {
		Route
	},
	setup(props, context) {
		return {
			routeEvents
		};
	}
});
</script>