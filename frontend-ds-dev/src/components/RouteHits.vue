<style scoped>
	.route-grid {
		display: grid;
		grid-template-columns: 1fr 1fr 4fr 7rem 5fr;
	}
</style>

<template>
	<div class="border-l-4 border-gray-800 my-8">
		<h4 class="bg-gray-800 px-2 text-white inline-block">Route Hits:</h4>

		<div class="overflow-y-scroll h-64 bg-gray-100">
			<div class="route-grid">
				<template v-for="r in routeEvents.hit_events">
					<span class="text-sm bold pl-2" style="font-variant-caps: all-small-caps">
						owner
					</span>
					<span class="px-2" style="font-variant-caps: all-small-caps">{{r.request.method}}</span>
					<span class="pl-2 text-sm">
						{{r.request.url}}
					</span>
					<span class="font-mono text-center" :class="{'bg-red-400': r.status>=500, 'bg-red-200': r.status >= 400, 'bg-green-200': r.status < 300}">
						{{r.status}}	
					</span><span></span>
					<Route v-if="r.route_config" :route="r.route_config"></Route>
					<template v-else>
						<span class="border-b border-gray-400" style="grid-column: span 2;" >&nbsp;</span>
						<span class="border-b border-gray-400 italic text-sm px-2 text-gray-600">No matching route</span>
						<span class="border-b border-gray-400" style="grid-column: span 2;"></span>
					</template>
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