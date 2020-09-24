<template>
	<div class="border-l-4 border-gray-800">
		<h4 class="bg-gray-800 px-2 text-white inline-block">Route Hits:</h4>

		<div class="overflow-y-scroll h-64 bg-gray-100">
			<div class="my-4 px-2" v-for="r in routeEvents.hit_events">
				<h5>{{r.request.method}} {{r.request.url}}</h5>
				<div v-if="r.route_config">
					<div class="flex items-center">
						<svg class="inline text-green-600 w-4 h-4 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
							<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
						</svg>
						Match: {{r.route_config.methods.join(', ')}} {{r.route_config.path}}
					</div>
					<div>Auth: {{r.route_config.auth.type}}</div>
					<div>
						Handler: {{r.route_config.handler.type}}
						<span v-if="r.route_config.handler.type === 'file'">{{r.route_config.handler.path}}</span>
						<!-- also do the other handlers -->
					</div>
				</div>
				<div v-else class="flex items-center">
					<svg class="inline text-red-600 w-4 h-4 mr-1" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
						<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
					</svg>
					<span>No hit</span>
				</div>
			</div>
		</div>
	</div>

</template>

<script lang="ts">
import { defineComponent } from 'vue';
import routeEvents from '../models/route-hits';

export default defineComponent({
	name: 'Routehits',
	components: {
	},
	setup(props, context) {
		return {
			routeEvents
		};
	}
});
</script>