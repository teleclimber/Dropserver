<style scoped>
	.route-grid {
		display: grid;
		grid-template-columns: max-content max-content 4fr 7rem 5fr;
		align-items: baseline;
	}
</style>

<template>
	<div class="bg-gray-50 h-full" style="scroll-behavior: smooth" ref="scroll_container">
		<div v-if="routeEvents.hit_events.length !== 0" class="route-grid gap-x-2">
			<template v-for="r, i in routeEvents.hit_events" :key="'route-hit-'+i">
				<span v-if="r.user" class="pl-2 font-bold text-gray-700">
					{{r.user.display_name}}
				</span>
				<span v-else class="pl-2 text-gray-500 italic text-sm">
					no authentication
				</span>
				<span class="text-right text-lg" style="font-variant-caps: all-small-caps">{{r.request.method}}</span>
				<span class="pl-2 text-sm">
					{{r.request.url}}
				</span>
				<div class="text-right">
					<span class="font-mono text-center px-2 rounded-full" :class="{'bg-red-400': r.status>=500, 'bg-red-200': r.status >= 400, 'bg-green-200': r.status < 300}">
						{{r.status}}	
					</span>
				</div>
				<span></span>
				<template v-if="r.v0_route_config">
					<span v-if="r.v0_route_config.auth.allow == 'public'" class="pl-2">
						<span class="text-white text-xs px-2 bg-yellow-500 rounded-full">PUBLIC</span>
					</span>
					<span v-else-if="r.authorized && r.v0_route_config.auth.permission" class="pl-2">
						<span class="text-white text-xs px-2 bg-green-600">{{ r.v0_route_config.auth.permission }}</span>
					</span>
					<span v-else-if="r.authorized" class="pl-2 text-green-600 flex items-baseline">
						<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 self-center" viewBox="0 0 20 20" fill="currentColor">
							<path fill-rule="evenodd" d="M6.267 3.455a3.066 3.066 0 001.745-.723 3.066 3.066 0 013.976 0 3.066 3.066 0 001.745.723 3.066 3.066 0 012.812 2.812c.051.643.304 1.254.723 1.745a3.066 3.066 0 010 3.976 3.066 3.066 0 00-.723 1.745 3.066 3.066 0 01-2.812 2.812 3.066 3.066 0 00-1.745.723 3.066 3.066 0 01-3.976 0 3.066 3.066 0 00-1.745-.723 3.066 3.066 0 01-2.812-2.812 3.066 3.066 0 00-.723-1.745 3.066 3.066 0 010-3.976 3.066 3.066 0 00.723-1.745 3.066 3.066 0 012.812-2.812zm7.44 5.252a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
						</svg>
						authorized
					</span>
					<span v-else class="pl-2 text-red-500 italic flex items-baseline">
						<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 self-center" viewBox="0 0 20 20" fill="currentColor">
							<path fill-rule="evenodd" d="M13.477 14.89A6 6 0 015.11 6.524l8.367 8.368zm1.414-1.414L6.524 5.11a6 6 0 018.367 8.367zM18 10a8 8 0 11-16 0 8 8 0 0116 0z" clip-rule="evenodd" />
						</svg>
						unauthorized
					</span>
					<span class=" text-sm text-gray-500">match:</span>
					<span class=" text-gray-900 pl-2 text-sm ">
						{{r.v0_route_config.path.path}}
						<span v-if="!r.v0_route_config.path.end">**</span>
					</span>

					<template v-if="r.v0_route_config.type === 'static'">
						<span class="text-sm italic text-right">Serve files:</span>
						<span class="font-mono pl-2">{{r.v0_route_config.options.path}}</span>
					</template>
					<template v-else-if="r.v0_route_config.type === 'function'">
						<span class="text-sm italic text-right">Handler:</span>
						<div class=" ">
							<span class="italic font-mono px-2 rounded bg-yellow-100 text-yellow-800">{{r.v0_route_config.options.name}}()</span>
						</div>
					</template>
					<template v-else>
						<span class="bg-yellow-500 text-yellow-100 px-1 text-sm">
							{{r.v0_route_config.type}}
						</span>
						<span class="">not implemented</span>
					</template>

					<div style="grid-column: span 5 " class="border-b border-gray-400"></div>

				</template>
				<span v-else style="grid-column: span 5">
					No match
				</span>
				
			</template>
		</div>
		<div v-else class="flex justify-center items-center h-full">
			No route hits yet.
		</div>
	</div>
</template>

<script lang="ts">
import { defineComponent, onUpdated, ref, Ref } from 'vue';
import routeEvents from '../models/route-hits';

export default defineComponent({
	name: 'Routehits',
	components: {
	},
	setup(props, context) {
		const scroll_container:Ref<undefined|HTMLElement> = ref(undefined);
		onUpdated( () => {
			if( !scroll_container.value ) return;
			scroll_container.value.scrollTop = scroll_container.value.scrollHeight;
		});
		return {
			scroll_container,
			routeEvents
		};
	},
});
</script>