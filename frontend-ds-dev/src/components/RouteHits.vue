<style scoped>
	.route-grid {
		display: grid;
		grid-template-columns: 10rem 3rem 4fr 7rem 5fr;
	}
</style>

<template>
	<div class="border-l-4 border-gray-800 my-8">
		<h4 class="bg-gray-800 px-2 text-white inline-block">Route Hits:</h4>

		<div class="overflow-y-scroll h-64 bg-gray-100" style="scroll-behavior: smooth" ref="scroll_container">
			<div class="route-grid">
				<template v-for="r in routeEvents.hit_events">
					<span v-if="r.user" class="pl-2 py-1 font-bold text-gray-700">
						{{r.user.display_name}}
					</span>
					<span v-else class="pl-2 py-1 text-gray-500 italic">
						Some rando...
					</span>
					<span class="px-2 py-1" style="font-variant-caps: all-small-caps">{{r.request.method}}</span>
					<span class="pl-2 py-1 text-sm">
						{{r.request.url}}
					</span>
					<span class="font-mono text-center py-1" :class="{'bg-red-400': r.status>=500, 'bg-red-200': r.status >= 400, 'bg-green-200': r.status < 300}">
						{{r.status}}	
					</span>
					<span></span>
					<template v-if="r.route_config">
						<span v-if="r.route_config.auth.allow == 'public'" class="pl-2 py-1 border-b border-gray-600">
							<span class="text-white text-xs px-2 bg-orange-500 rounded-full">PUBLIC</span>
						</span>
						<span v-else-if="r.authorized && r.route_config.auth.permission" class="pl-2 py-1 border-b border-gray-600">
							<span class="text-white text-xs px-2 bg-green-600">{{ r.route_config.auth.permission }}</span>
						</span>
						<span v-else-if="r.authorized" class="pl-2 py-1 border-b text-green-500 border-gray-600">
							authorized
						</span>
						<span v-else class="pl-2 py-1 border-b text-orange-500 border-gray-600 italic">
							unauthorized
						</span>
						<span class="border-b border-gray-600"></span>
						<span class="bg-gray-200 text-gray-900 pl-2 py-1 text-sm border-b border-gray-600">{{r.route_config.path}}</span>
	
						<template v-if="r.route_config.handler.type === 'file'">
							<span class="border-b border-gray-600 px-1 py-1 text-sm italic text-right">Serve files:</span>
							<span class="border-b border-gray-600 font-mono pl-2 py-1">{{r.route_config.handler.path}}</span>
						</template>
						<template v-else-if="r.route_config.handler.type === 'function'">
							<span class="border-b border-gray-600 px-1 py-1 text-sm italic text-right bg-orange-100">Call function:</span>
							<div class=" py-1 border-b border-gray-600 bg-orange-100">
								<span class="font-mono pl-2">{{r.route_config.handler.file}}</span>
								<span class="italic font-mono pl-2 text-yellow-800">{{r.route_config.handler.function}}()</span>
							</div>
						</template>
						<template v-else>
							<span class="border-b border-gray-400 bg-yellow-500 text-yellow-100 px-1 text-sm">
								{{r.route_config.handler.type}}
							</span>
							<span class="border-b border-gray-400">not implemented</span>
						</template>

					</template>
					<span v-else style="grid-column: span 5">
						No match
					</span>
					
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
	},
	updated() {
	 	const elem = this.$refs.scroll_container as HTMLElement;
	 	elem.scrollTop = elem.scrollHeight;
	}
});
</script>